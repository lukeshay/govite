package engine

import (
	"fmt"
	"net/http"
	"strings"
)

type Error struct {
	error
}

type ErrorCreator struct {
	prefix string
}

func newErrorCreator(prefix string) *ErrorCreator {
	return &ErrorCreator{prefix: prefix}
}

func (ec *ErrorCreator) Create() error {
	return fmt.Errorf(ec.prefix)
}

func (ec *ErrorCreator) Format(message string) error {
	return fmt.Errorf("%s: %s", ec.prefix, message)
}

func (ec *ErrorCreator) FormatErr(err error) error {
	return ec.Format(err.Error())
}

func (ec *ErrorCreator) Is(err error) bool {
	return strings.HasPrefix(err.Error(), ec.prefix)
}

var (
	CreateTempFileError     = newErrorCreator("Could not create temp file")
	DistDirAbsError         = newErrorCreator("Could not get the absolute path of the dist directory")
	AppDirAbsError          = newErrorCreator("Could not get the absolute path of the app directory")
	SrcDirAbsError          = newErrorCreator("Could not get the absolute path of the src directory")
	IndexHtmlReadError      = newErrorCreator("Could not read index.html")
	ServerEntryWriteError   = newErrorCreator("Could not write server entry")
	CreateTempDirError      = newErrorCreator("Could not create temp dir")
	CreateNodeJSVMError     = newErrorCreator("Could not create Node.js VM")
	HashError               = newErrorCreator("Could not hash")
	JSONMarshalError        = newErrorCreator("Could not marshal JSON")
	JSONUnmarshalError      = newErrorCreator("Could not unmarshal JSON")
	WriteFileError          = newErrorCreator("Could not write file")
	ExecuteNodeJSCodeError  = newErrorCreator("Could not execute Node.js code")
	InterfaceCastError      = newErrorCreator("Could not cast interface")
	StartViteDevServerError = newErrorCreator("Could not start Vite dev server")
)

type RenderResult struct {
	Content     string
	ContentType string
	Headers     http.Header
}

type Engine interface {
	// Render renders the given url with the given props.
	Render(url string, props any) (*RenderResult, error)
	// Close closes the engine.
	Close() error
	// StaticPath returns the path to the static directory.
	StaticPath() string
}

func defaultString(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}

	return value
}

func addToHead(html string, head interface{}) string {
	return strings.Replace(html, "</head>", fmt.Sprintf("%s</head>", head), 1)
}
