package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lukeshay/govite/internal/logging"
	"github.com/lukeshay/govite/pkg/utils/nodejs"
)

const devServerJs = "import  '@govite/govite/server'"

type DevelopmentEngineOptions struct {
	// The relative or absolute path to the directory of your vite project.
	//
	// **Default**: `.`
	AppDir string
	// Flags are the additional startup flags that are provided to the "node"
	// process.
	Flags []string
	// Port is the port to run the websocket server on. This defaults to 6543.
	Port int
	// hmrPort is the port to run the HMR server on. This defaults to 26543.
	hmrPort int
	// ServerPort is the port of your Go HTTP server.
	ServerPort int
	// Stdout is the output writer for the VM. Default is os.Stdout.
	Stdout io.Writer
	// Stderr is the error writer for the VM. Default is os.Stderr.
	Stderr io.Writer
	// Env is the environment variables to be set for the VM.
	Env []string
	// Logger is the logger to be used for the VM.
	Logger *slog.Logger
}

type DevelopmentEngine struct {
	log    *slog.Logger
	cmd    *exec.Cmd
	port   int
	appDir string
}

// NewDevelopmentEngine Creates a new Engine instance to be utilized in
// production. This will use the files built by vite to render the page HTML on the server.
func NewDevelopmentEngine(options DevelopmentEngineOptions) (Engine, error) {
	appAbs, err := filepath.Abs(defaultString(options.AppDir, "app"))
	if err != nil {
		return nil, DistDirAbsError.FormatErr(err)
	}
	log := logging.NewDefaultLogger(options.Logger)

	port := options.Port
	if port == 0 {
		port = 6543
	}
	hmrPort := options.hmrPort
	if hmrPort == 0 {
		hmrPort = 26543
	}

	cmd := nodejs.NewNodeJSCommand(nodejs.NodeJSCommandOptions{
		Script: devServerJs,
		Dir:    appAbs,
		Flags:  options.Flags,
		Stdout: options.Stdout,
		Stderr: options.Stderr,
		Env: map[string]string{
			"NODE_PATH":   fmt.Sprintf("%s/node_modules", appAbs),
			"PORT":        fmt.Sprintf("%d", port),
			"HMR_PORT":    fmt.Sprintf("%d", hmrPort),
			"SERVER_PORT": fmt.Sprintf("%d", options.ServerPort),
		},
	})

	cmd.Env = append(cmd.Env, options.Env...)

	log.Debug("Starting Vite dev server", "port", port)

	if err := cmd.Start(); err != nil {
		log.Error("Error starting Vite dev server", "error", err.Error())
		return nil, CreateNodeJSVMError.FormatErr(err)
	}

	return &DevelopmentEngine{
		log:    log,
		cmd:    cmd,
		port:   port,
		appDir: appAbs,
	}, nil
}

// MustNewDevelopmentEngine is like New, but panics if an error occurs.
func MustNewDevelopmentEngine(options DevelopmentEngineOptions) Engine {
	engine, err := NewDevelopmentEngine(options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating engine: %s\n", err.Error())

		panic(err)
	}

	return engine
}

func (e *DevelopmentEngine) Render(path string, props any) (*RenderResult, error) {
	marshalledProps, err := json.Marshal(props)
	if err != nil {
		e.log.Debug("Could not marshal JSON", "error", err.Error())
		return nil, JSONMarshalError.FormatErr(err)
	}

	// urlendcode props
	requestURL, err := url.Parse(fmt.Sprintf("http://localhost:%d%s?props=%s", e.port, path, url.QueryEscape(string(marshalledProps))))
	if err != nil {
		e.log.Debug("Could not parse URL", "error", err.Error())
		// TODO
		return nil, err
	}

	e.log.Debug("Making request", "url", requestURL.String())

	res, err := http.Get(requestURL.String())
	if err != nil {
		e.log.Debug("Could not make request", "error", err.Error())
		// TODO
		return nil, err
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		e.log.Debug("Could not read response body", "error", err.Error())
		// TODO
		return nil, err
	}

	return &RenderResult{
		Content:     string(resBody),
		ContentType: res.Header.Get("Content-Type"),
		Headers:     res.Header,
	}, nil
}

func (e *DevelopmentEngine) Close() error {
	return e.cmd.Cancel()
}

func (e *DevelopmentEngine) StaticPath() string {
	return filepath.Join(e.appDir, "public")
}
