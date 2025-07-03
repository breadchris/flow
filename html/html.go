package html

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"strings"
)

// Node represents an HTML element or text content
type Node struct {
	Name         string
	Attrs        map[string]string
	DynamicAttrs map[string]func(context.Context) string
	Children     []*Node
	transform    func(*Node, context.Context) *Node
	text         string
	raw          string
}

// Render returns the HTML string representation of the node
func (n *Node) Render() string {
	return n.RenderCtx(context.Background())
}

// RenderCtx renders the node with the given context
func (n *Node) RenderCtx(ctx context.Context) string {
	if n == nil {
		return ""
	}

	// Apply transform if present
	if n.transform != nil {
		n = n.transform(n, ctx)
		if n == nil {
			return ""
		}
	}

	// Handle text content
	if n.text != "" {
		return html.EscapeString(n.text)
	}

	// Handle raw content
	if n.raw != "" {
		return n.raw
	}

	// Handle HTML elements
	if n.Name == "" {
		// Fragment node - just render children
		var result strings.Builder
		for _, child := range n.Children {
			result.WriteString(child.RenderCtx(ctx))
		}
		return result.String()
	}

	var result strings.Builder
	result.WriteString("<")
	result.WriteString(n.Name)

	// Render attributes
	for key, value := range n.Attrs {
		result.WriteString(fmt.Sprintf(` %s="%s"`, key, html.EscapeString(value)))
	}

	// Render dynamic attributes
	for key, valueFunc := range n.DynamicAttrs {
		value := valueFunc(ctx)
		result.WriteString(fmt.Sprintf(` %s="%s"`, key, html.EscapeString(value)))
	}

	// Self-closing tags
	if isSelfClosing(n.Name) {
		result.WriteString(" />")
		return result.String()
	}

	result.WriteString(">")

	// Render children
	for _, child := range n.Children {
		result.WriteString(child.RenderCtx(ctx))
	}

	result.WriteString("</")
	result.WriteString(n.Name)
	result.WriteString(">")

	return result.String()
}

// RenderPage renders the node as a complete HTML page to an HTTP response
func (n *Node) RenderPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(n.RenderCtx(r.Context())))
}

// Init initializes the node (for compatibility)
func (n *Node) Init(node *Node) {}

// isSelfClosing returns true if the tag is self-closing
func isSelfClosing(tagName string) bool {
	selfClosingTags := map[string]bool{
		"meta": true, "link": true, "img": true, "br": true, "hr": true,
		"input": true, "area": true, "base": true, "col": true, "embed": true,
		"keygen": true, "param": true, "source": true, "track": true, "wbr": true,
	}
	return selfClosingTags[tagName]
}

// Basic HTML Elements

func Html(children ...*Node) *Node {
	return &Node{Name: "html", Children: children}
}

func Head(children ...*Node) *Node {
	return &Node{Name: "head", Children: children}
}

func Body(children ...*Node) *Node {
	return &Node{Name: "body", Children: children}
}

func Meta(attrs ...*Node) *Node {
	node := &Node{Name: "meta", Attrs: make(map[string]string)}
	for _, attr := range attrs {
		if attr.Attrs != nil {
			for k, v := range attr.Attrs {
				node.Attrs[k] = v
			}
		}
	}
	return node
}

func Title(children ...*Node) *Node {
	return &Node{Name: "title", Children: children}
}

func Link(attrs ...*Node) *Node {
	node := &Node{Name: "link", Attrs: make(map[string]string)}
	for _, attr := range attrs {
		if attr.Attrs != nil {
			for k, v := range attr.Attrs {
				node.Attrs[k] = v
			}
		}
	}
	return node
}

func Script(children ...*Node) *Node {
	node := &Node{Name: "script", Attrs: make(map[string]string)}
	var nodeChildren []*Node
	
	for _, child := range children {
		if child.Attrs != nil {
			for k, v := range child.Attrs {
				node.Attrs[k] = v
			}
		} else {
			nodeChildren = append(nodeChildren, child)
		}
	}
	
	node.Children = nodeChildren
	return node
}

func Style(children ...*Node) *Node {
	return &Node{Name: "style", Children: children}
}

func Div(children ...*Node) *Node {
	node := &Node{Name: "div", Attrs: make(map[string]string)}
	var nodeChildren []*Node
	
	for _, child := range children {
		if child.Attrs != nil {
			for k, v := range child.Attrs {
				node.Attrs[k] = v
			}
		} else {
			nodeChildren = append(nodeChildren, child)
		}
	}
	
	node.Children = nodeChildren
	return node
}

func Span(children ...*Node) *Node {
	node := &Node{Name: "span", Attrs: make(map[string]string)}
	var nodeChildren []*Node
	
	for _, child := range children {
		if child.Attrs != nil {
			for k, v := range child.Attrs {
				node.Attrs[k] = v
			}
		} else {
			nodeChildren = append(nodeChildren, child)
		}
	}
	
	node.Children = nodeChildren
	return node
}

func P(children ...*Node) *Node {
	node := &Node{Name: "p", Attrs: make(map[string]string)}
	var nodeChildren []*Node
	
	for _, child := range children {
		if child.Attrs != nil {
			for k, v := range child.Attrs {
				node.Attrs[k] = v
			}
		} else {
			nodeChildren = append(nodeChildren, child)
		}
	}
	
	node.Children = nodeChildren
	return node
}

func H1(children ...*Node) *Node {
	node := &Node{Name: "h1", Attrs: make(map[string]string)}
	var nodeChildren []*Node
	
	for _, child := range children {
		if child.Attrs != nil {
			for k, v := range child.Attrs {
				node.Attrs[k] = v
			}
		} else {
			nodeChildren = append(nodeChildren, child)
		}
	}
	
	node.Children = nodeChildren
	return node
}

func H2(children ...*Node) *Node {
	node := &Node{Name: "h2", Attrs: make(map[string]string)}
	var nodeChildren []*Node
	
	for _, child := range children {
		if child.Attrs != nil {
			for k, v := range child.Attrs {
				node.Attrs[k] = v
			}
		} else {
			nodeChildren = append(nodeChildren, child)
		}
	}
	
	node.Children = nodeChildren
	return node
}

func H3(children ...*Node) *Node {
	node := &Node{Name: "h3", Attrs: make(map[string]string)}
	var nodeChildren []*Node
	
	for _, child := range children {
		if child.Attrs != nil {
			for k, v := range child.Attrs {
				node.Attrs[k] = v
			}
		} else {
			nodeChildren = append(nodeChildren, child)
		}
	}
	
	node.Children = nodeChildren
	return node
}

func A(children ...*Node) *Node {
	node := &Node{Name: "a", Attrs: make(map[string]string)}
	var nodeChildren []*Node
	
	for _, child := range children {
		if child.Attrs != nil {
			for k, v := range child.Attrs {
				node.Attrs[k] = v
			}
		} else {
			nodeChildren = append(nodeChildren, child)
		}
	}
	
	node.Children = nodeChildren
	return node
}

func Img(attrs ...*Node) *Node {
	node := &Node{Name: "img", Attrs: make(map[string]string)}
	for _, attr := range attrs {
		if attr.Attrs != nil {
			for k, v := range attr.Attrs {
				node.Attrs[k] = v
			}
		}
	}
	return node
}

func Form(children ...*Node) *Node {
	node := &Node{Name: "form", Attrs: make(map[string]string)}
	var nodeChildren []*Node
	
	for _, child := range children {
		if child.Attrs != nil {
			for k, v := range child.Attrs {
				node.Attrs[k] = v
			}
		} else {
			nodeChildren = append(nodeChildren, child)
		}
	}
	
	node.Children = nodeChildren
	return node
}

func Input(attrs ...*Node) *Node {
	node := &Node{Name: "input", Attrs: make(map[string]string)}
	for _, attr := range attrs {
		if attr.Attrs != nil {
			for k, v := range attr.Attrs {
				node.Attrs[k] = v
			}
		}
	}
	return node
}

func Button(children ...*Node) *Node {
	node := &Node{Name: "button", Attrs: make(map[string]string)}
	var nodeChildren []*Node
	
	for _, child := range children {
		if child.Attrs != nil {
			for k, v := range child.Attrs {
				node.Attrs[k] = v
			}
		} else {
			nodeChildren = append(nodeChildren, child)
		}
	}
	
	node.Children = nodeChildren
	return node
}

func Label(children ...*Node) *Node {
	node := &Node{Name: "label", Attrs: make(map[string]string)}
	var nodeChildren []*Node
	
	for _, child := range children {
		if child.Attrs != nil {
			for k, v := range child.Attrs {
				node.Attrs[k] = v
			}
		} else {
			nodeChildren = append(nodeChildren, child)
		}
	}
	
	node.Children = nodeChildren
	return node
}

func Nav(children ...*Node) *Node {
	node := &Node{Name: "nav", Attrs: make(map[string]string)}
	var nodeChildren []*Node
	
	for _, child := range children {
		if child.Attrs != nil {
			for k, v := range child.Attrs {
				node.Attrs[k] = v
			}
		} else {
			nodeChildren = append(nodeChildren, child)
		}
	}
	
	node.Children = nodeChildren
	return node
}

func Ul(children ...*Node) *Node {
	node := &Node{Name: "ul", Attrs: make(map[string]string)}
	var nodeChildren []*Node
	
	for _, child := range children {
		if child.Attrs != nil {
			for k, v := range child.Attrs {
				node.Attrs[k] = v
			}
		} else {
			nodeChildren = append(nodeChildren, child)
		}
	}
	
	node.Children = nodeChildren
	return node
}

func Li(children ...*Node) *Node {
	node := &Node{Name: "li", Attrs: make(map[string]string)}
	var nodeChildren []*Node
	
	for _, child := range children {
		if child.Attrs != nil {
			for k, v := range child.Attrs {
				node.Attrs[k] = v
			}
		} else {
			nodeChildren = append(nodeChildren, child)
		}
	}
	
	node.Children = nodeChildren
	return node
}

func Main(children ...*Node) *Node {
	node := &Node{Name: "main", Attrs: make(map[string]string)}
	var nodeChildren []*Node
	
	for _, child := range children {
		if child.Attrs != nil {
			for k, v := range child.Attrs {
				node.Attrs[k] = v
			}
		} else {
			nodeChildren = append(nodeChildren, child)
		}
	}
	
	node.Children = nodeChildren
	return node
}

func Section(children ...*Node) *Node {
	node := &Node{Name: "section", Attrs: make(map[string]string)}
	var nodeChildren []*Node
	
	for _, child := range children {
		if child.Attrs != nil {
			for k, v := range child.Attrs {
				node.Attrs[k] = v
			}
		} else {
			nodeChildren = append(nodeChildren, child)
		}
	}
	
	node.Children = nodeChildren
	return node
}

func Header(children ...*Node) *Node {
	node := &Node{Name: "header", Attrs: make(map[string]string)}
	var nodeChildren []*Node
	
	for _, child := range children {
		if child.Attrs != nil {
			for k, v := range child.Attrs {
				node.Attrs[k] = v
			}
		} else {
			nodeChildren = append(nodeChildren, child)
		}
	}
	
	node.Children = nodeChildren
	return node
}

// Text and Content Functions

// T creates escaped text content
func T(text string) *Node {
	return &Node{text: text}
}

// Text is an alias for T
func Text(text string) *Node {
	return T(text)
}

// Raw creates unescaped HTML content
func Raw(rawHTML string) *Node {
	return &Node{raw: rawHTML}
}

// Attribute Functions

func Id(value string) *Node {
	return &Node{Attrs: map[string]string{"id": value}}
}

func Class(value string) *Node {
	return &Node{Attrs: map[string]string{"class": value}}
}

func Src(value string) *Node {
	return &Node{Attrs: map[string]string{"src": value}}
}

func Href(value string) *Node {
	return &Node{Attrs: map[string]string{"href": value}}
}

func Type(value string) *Node {
	return &Node{Attrs: map[string]string{"type": value}}
}

func Name(value string) *Node {
	return &Node{Attrs: map[string]string{"name": value}}
}

func Content(value string) *Node {
	return &Node{Attrs: map[string]string{"content": value}}
}

func Charset(value string) *Node {
	return &Node{Attrs: map[string]string{"charset": value}}
}

func Rel(value string) *Node {
	return &Node{Attrs: map[string]string{"rel": value}}
}

func For(value string) *Node {
	return &Node{Attrs: map[string]string{"for": value}}
}

func Method(value string) *Node {
	return &Node{Attrs: map[string]string{"method": value}}
}

func Action(value string) *Node {
	return &Node{Attrs: map[string]string{"action": value}}
}

func Value(value string) *Node {
	return &Node{Attrs: map[string]string{"value": value}}
}

func Attr(key, value string) *Node {
	return &Node{Attrs: map[string]string{key: value}}
}

func Attrs(attrs map[string]string) *Node {
	return &Node{Attrs: attrs}
}

// Helper Functions

// Ch adds multiple children to a fragment node
func Ch(children []*Node) *Node {
	return &Node{Children: children}
}

// If returns the true node if condition is true, otherwise the false node
func If(condition bool, trueNode, falseNode *Node) *Node {
	if condition {
		return trueNode
	}
	return falseNode
}

// Nil returns an empty node
func Nil() *Node {
	return &Node{}
}

// DefaultLayout creates a standard HTML page layout
func DefaultLayout(children ...*Node) *Node {
	return Html(
		Head(
			Meta(Charset("UTF-8")),
			Meta(Name("viewport"), Content("width=device-width, initial-scale=1.0")),
		),
		Body(children...),
	)
}

// LoadReactModule creates a script that loads and initializes a React module
func LoadReactModule(modulePath, componentName string) *Node {
	script := fmt.Sprintf(`
		import { createRoot } from 'react-dom/client';
		import %s from '/coderunner/module/%s';
		const container = document.getElementById('root');
		if (container) {
			const root = createRoot(container);
			root.render(React.createElement(%s));
		}
	`, componentName, modulePath, componentName)
	
	return Script(Type("module"), Raw(script))
}