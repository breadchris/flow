package code

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/breadchris/flow/deps"
	"github.com/evanw/esbuild/pkg/api"
)

type BuildCache struct {
	BuiltAt    time.Time `json:"builtAt"`
	SourcePath string    `json:"sourcePath"`
	BuildPath  string    `json:"buildPath"`
	Hash       string    `json:"hash"`
}

type FileInfo struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	IsDir        bool      `json:"isDir"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"lastModified"`
	FileCount    int       `json:"fileCount"` // Number of files in directory (0 for regular files)
}

type SaveFileRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func New(d deps.Deps) *http.ServeMux {
	m := http.NewServeMux()

	m.HandleFunc("/render/", func(w http.ResponseWriter, r *http.Request) {
		handleRenderComponent(d)(w, r)
	})

	m.HandleFunc("/module/", func(w http.ResponseWriter, r *http.Request) {
		handleServeModule(w, r)
	})

	return m
}

// buildDirectoryListing lists files in a specific directory with controlled depth
func buildDirectoryListing(baseDir, relativePath string, maxDepth int) ([]FileInfo, error) {
	var files []FileInfo

	// Build the target directory path
	targetDir := baseDir
	if relativePath != "" {
		// Validate and sanitize the relative path
		cleanPath := filepath.Clean(relativePath)
		if strings.Contains(cleanPath, "..") {
			return nil, fmt.Errorf("invalid path")
		}
		targetDir = filepath.Join(baseDir, cleanPath)
	}

	// Check if directory exists
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory not found")
	}

	// Read directory contents
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		// Skip hidden files and directories (starting with .)
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't read
		}

		// Build the relative path from the base src directory
		var relPath string
		if relativePath == "" {
			relPath = entry.Name()
		} else {
			relPath = filepath.ToSlash(filepath.Join(relativePath, entry.Name()))
		}

		fileInfo := FileInfo{
			Name:         entry.Name(),
			Path:         relPath,
			IsDir:        entry.IsDir(),
			Size:         info.Size(),
			LastModified: info.ModTime(),
			FileCount:    0, // Default to 0 for files
		}

		// If it's a directory, count the files inside it (one level deep)
		//if entry.IsDir() {
		//	fileInfo.FileCount = countFilesInDirectory(filepath.Join(targetDir, entry.Name()))
		//}

		files = append(files, fileInfo)

		// If it's a directory and we haven't reached max depth, recurse
		if entry.IsDir() && maxDepth >= 1 {
			childFiles, err := buildDirectoryListing(baseDir, relPath, maxDepth-1)
			if err == nil {
				files = append(files, childFiles...)
			}
		}
	}

	return files, nil
}

// handleRenderComponent builds and renders a React component in a simple HTML page
func handleRenderComponent(d deps.Deps) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract path from URL
		componentPath := strings.TrimPrefix(r.URL.Path, "/render/")
		if componentPath == "" {
			http.Error(w, "Component path is required", http.StatusBadRequest)
			return
		}

		// Get component name from query parameter (optional)
		componentName := r.URL.Query().Get("component")
		if componentName == "" {
			componentName = "App" // Default to App component
		}

		// Validate and sanitize the path
		cleanPath := filepath.Clean(componentPath)
		if strings.Contains(cleanPath, "..") {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		// Build source path
		srcPath := filepath.Join("./", cleanPath)

		// Check if source file exists
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			http.Error(w, "Source file not found", http.StatusNotFound)
			return
		}

		// Read the source code to build
		sourceCode, err := os.ReadFile(srcPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read source file: %v", err), http.StatusInternalServerError)
			return
		}

		// Build with esbuild to get the compiled JavaScript
		result := api.Build(api.BuildOptions{
			Stdin: &api.StdinOptions{
				Contents:   string(sourceCode),
				ResolveDir: filepath.Dir(srcPath),
				Sourcefile: filepath.Base(srcPath),
				Loader:     api.LoaderTSX,
			},
			Loader: map[string]api.Loader{
				".js":  api.LoaderJS,
				".jsx": api.LoaderJSX,
				".ts":  api.LoaderTS,
				".tsx": api.LoaderTSX,
				".css": api.LoaderCSS,
			},
			Format:          api.FormatESModule,
			Bundle:          true,
			Write:           false,
			TreeShaking:     api.TreeShakingTrue,
			Target:          api.ESNext,
			JSX:             api.JSXAutomatic,
			JSXImportSource: "react",
			LogLevel:        api.LogLevelSilent,
			External:        []string{"react", "react-dom", "react-player", "supabase-kv"},
			TsconfigRaw: `{
			"compilerOptions": {
				"jsx": "react-jsx",
				"allowSyntheticDefaultImports": true,
				"esModuleInterop": true,
				"moduleResolution": "node",
				"target": "ESNext",
				"lib": ["ESNext", "DOM", "DOM.Iterable"],
				"allowJs": true,
				"skipLibCheck": true,
				"strict": false,
				"forceConsistentCasingInFileNames": true,
				"noEmit": true,
				"incremental": true,
				"resolveJsonModule": true,
				"isolatedModules": true
			}
		}`,
		})

		// Check for build errors
		if len(result.Errors) > 0 {
			errorMessages := make([]string, len(result.Errors))
			for i, err := range result.Errors {
				errorMessages[i] = fmt.Sprintf("%s:%d:%d: %s", err.Location.File, err.Location.Line, err.Location.Column, err.Text)
			}

			// Use the BuildErrorPage helper
			w.WriteHeader(http.StatusBadRequest)
			BuildErrorPage(componentPath, errorMessages).RenderPage(w, r)
			return
		}

		// Verify build succeeded
		if len(result.OutputFiles) == 0 {
			http.Error(w, "No output generated from build", http.StatusInternalServerError)
			return
		}

		// Generate the HTML page using Go HTML format
		page := ReactComponentPage(d.Config, componentName,
			ComponentLoader(componentPath, componentName, true),
		)

		// Render the page
		page.RenderPage(w, r)
	}
}

// handleServeModule builds and serves a React component as an ES module
func handleServeModule(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract path from URL
	componentPath := strings.TrimPrefix(r.URL.Path, "/module/")
	if componentPath == "" {
		http.Error(w, "Component path is required", http.StatusBadRequest)
		return
	}

	// Validate and sanitize the path
	cleanPath := filepath.Clean(componentPath)
	if strings.Contains(cleanPath, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Build source path
	srcPath := filepath.Join("./", cleanPath)

	// Check if source file exists
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		http.Error(w, "Source file not found", http.StatusNotFound)
		return
	}

	// Read the source code to build
	sourceCode, err := os.ReadFile(srcPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read source file: %v", err), http.StatusInternalServerError)
		return
	}

	var loader api.Loader
	switch filepath.Ext(srcPath) {
	case ".js":
		loader = api.LoaderJS
	case ".jsx":
		loader = api.LoaderJSX
	case ".ts":
		loader = api.LoaderTS
	case ".tsx":
		loader = api.LoaderTSX
	}

	// Build with esbuild to get the compiled JavaScript as ES module
	result := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   string(sourceCode),
			ResolveDir: filepath.Dir(srcPath),
			Sourcefile: filepath.Base(srcPath),
			Loader:     loader,
		},
		Loader: map[string]api.Loader{
			".js":  api.LoaderJS,
			".jsx": api.LoaderJSX,
			".ts":  api.LoaderTS,
			".tsx": api.LoaderTSX,
			".css": api.LoaderCSS,
		},
		Format:          api.FormatESModule,
		Bundle:          true,
		Write:           false,
		TreeShaking:     api.TreeShakingTrue,
		Target:          api.ESNext,
		JSX:             api.JSXAutomatic,
		JSXImportSource: "react",
		LogLevel:        api.LogLevelSilent,
		External:        []string{"react", "react-dom", "react-dom/client", "react/jsx-runtime", "supabase-kv"},
		TsconfigRaw: `{
			"compilerOptions": {
				"jsx": "react-jsx",
				"allowSyntheticDefaultImports": true,
				"esModuleInterop": true,
				"moduleResolution": "node",
				"target": "ESNext",
				"lib": ["ESNext", "DOM", "DOM.Iterable"],
				"allowJs": true,
				"skipLibCheck": true,
				"strict": false,
				"forceConsistentCasingInFileNames": true,
				"noEmit": true,
				"incremental": true,
				"resolveJsonModule": true,
				"isolatedModules": true
			}
		}`,
	})

	// Check for build errors
	if len(result.Errors) > 0 {
		errorMessages := make([]string, len(result.Errors))
		for i, err := range result.Errors {
			errorMessages[i] = fmt.Sprintf("%s:%d:%d: %s", err.Location.File, err.Location.Line, err.Location.Column, err.Text)
		}

		errorResponse := map[string]interface{}{
			"error":   "Build failed",
			"details": errorMessages,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Get the compiled JavaScript
	if len(result.OutputFiles) == 0 {
		http.Error(w, "No output generated from build", http.StatusInternalServerError)
		return
	}

	compiledJS := string(result.OutputFiles[0].Contents)

	// Return the ES module code
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "no-cache") // Prevent caching during development
	w.Write([]byte(compiledJS))
}
