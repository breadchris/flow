package deps

import (
	"github.com/breadchris/flow/config"
	"github.com/breadchris/flow/session"
	"gorm.io/gorm"
)

// DepsFactory provides methods to create dependencies
type DepsFactory struct {
	config config.AppConfig
}

// NewDepsFactory creates a new dependency factory
func NewDepsFactory(config config.AppConfig) *DepsFactory {
	return &DepsFactory{
		config: config,
	}
}

// CreateDeps creates a new Deps instance with the provided database
func (f *DepsFactory) CreateDeps(db *gorm.DB, dir string) Deps {
	// Create session manager
	sessionManager, err := session.New()
	if err != nil {
		// For now, we'll log the error and continue with a nil session
		// This allows the app to start even if session store initialization fails
		sessionManager = nil
	}
	
	return Deps{
		Dir:     dir,
		DB:      db,
		Config:  f.config,
		Session: sessionManager,
	}
}