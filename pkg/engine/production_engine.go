package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/lukeshay/govite/internal/logging"
	"github.com/lukeshay/govite/pkg/node"
	"github.com/lukeshay/govite/pkg/utils/hash"
)

const serverJs = `
import { render } from "%s";
import { env } from "node:process"

export default render(%s)
`

const htmlInitialState = `<script>
  window.__INITIAL_STATE__ = %s
</script>
</head>`

type ProductionEngineOptions struct {
	// The relative or absolute path to the dist folder of your vite project.
	//
	// **Default**: `dist`
	DistDir string
	// Flags are the additional startup flags that are provided to the "node"
	// process.
	Flags []string
	// sort is the port to run the websocket server on. This defaults to 6543.
	Port int
	// Stdout is the output writer for the VM. Default is os.Stdout.
	Stdout io.Writer
	// Stderr is the error writer for the VM. Default is os.Stderr.
	Stderr io.Writer
	// Env is the environment variables to be set for the VM.
	Env []string
	// NodeProcesses is the number of node processes to run. Default is 5.
	NodeProcesses int
	// Logger is the logger to be used for the VM.
	Logger *slog.Logger
}

type ProductionEngine struct {
	htmlTemplate string
	log          *slog.Logger
	serverEntry  string
	tempDir      string
	distDir      string
	vm           node.VM
}

// NewProductionEngine Creates a new Engine instance to be utilized in
// production. This will use the files built by vite to render the page HTML on the server.
func NewProductionEngine(options ProductionEngineOptions) (Engine, error) {
	distAbs, err := filepath.Abs(defaultString(options.DistDir, "dist"))
	if err != nil {
		return nil, DistDirAbsError.FormatErr(err)
	}

	indexHtml := filepath.Join(distAbs, "client", "index.html")
	serverEntry := filepath.Join(distAbs, "server", "entry-server.js")

	htmlTemplate, err := os.ReadFile(indexHtml)
	if err != nil {
		return nil, IndexHtmlReadError.FormatErr(err)
	}

	tempDir, err := os.MkdirTemp("", "govite-*")
	if err != nil {
		return nil, CreateTempDirError.FormatErr(err)
	}

	log := logging.NewDefaultLogger(options.Logger)

	vm, err := node.NewNodeJS(node.Options{
		Dir:           distAbs,
		Env:           options.Env,
		Flags:         options.Flags,
		Logger:        options.Logger,
		NodeProcesses: options.NodeProcesses,
		Port:          options.Port,
		Stderr:        options.Stderr,
		Stdout:        options.Stdout,
	})
	if err != nil {
		return nil, CreateNodeJSVMError.FormatErr(err)
	}

	return &ProductionEngine{
		htmlTemplate: string(htmlTemplate),
		log:          log,
		serverEntry:  serverEntry,
		tempDir:      tempDir,
		distDir:      distAbs,
		vm:           vm,
	}, nil
}

// MustNewProductionEngine is like New, but panics if an error occurs.
func MustNewProductionEngine(options ProductionEngineOptions) Engine {
	engine, err := NewProductionEngine(options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating engine: %s\n", err.Error())

		panic(err)
	}

	return engine
}

func (e *ProductionEngine) Render(url string, props any) (*RenderResult, error) {
	hash, err := hash.Hash(props)
	if err != nil {
		return nil, HashError.FormatErr(err)
	}

	marshalledProps, err := json.Marshal(props)
	if err != nil {
		return nil, JSONMarshalError.FormatErr(err)
	}

	fileName := filepath.Join(e.tempDir, fmt.Sprintf("%s.mjs", hash))

	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		file, err := os.Create(filepath.Join(e.tempDir, fmt.Sprintf("%s.mjs", hash)))
		if err != nil {
			return nil, CreateTempFileError.FormatErr(err)
		}

		defer file.Close()

		_, err = file.Write([]byte(fmt.Sprintf(serverJs, e.serverEntry, marshalledProps)))
		if err != nil {
			return nil, WriteFileError.FormatErr(err)
		}
	}

	result, err := e.vm.Run(fileName)
	if err != nil {
		return nil, ExecuteNodeJSCodeError.FormatErr(err)
	}

	r, ok := result.(map[string]interface{})
	if !ok {
		return nil, InterfaceCastError.Format("Error casting result")
	}

	htmlValue := r["html"]
	headValue := r["head"]
	cssValue := r["css"]

	html := addToHead(e.htmlTemplate, fmt.Sprintf(htmlInitialState, marshalledProps))
	html = strings.Replace(html, "<div id=\"app\"></div>", fmt.Sprintf("<div id=\"app\">%s</div>", htmlValue), 1)

	e.log.Info("Rendering HTML", "html", html)

	if headValue != nil {
		html = addToHead(html, headValue)
	}

	if cssValue != nil {
		html = addToHead(html, fmt.Sprintf("<style>%s</style>", cssValue))
	}

	return &RenderResult{
		Content:     html,
		ContentType: "text/html",
	}, nil
}

func (e *ProductionEngine) Close() error {
	return e.vm.Close()
}

func (e *ProductionEngine) StaticPath() string {
	return filepath.Join(e.distDir, "client")
}
