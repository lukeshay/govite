package node

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/go-multierror"
	"github.com/lukeshay/govite/internal/logging"
	"github.com/lukeshay/govite/pkg/utils/nodejs"
	"github.com/rs/xid"
)

//go:embed runtime.js
var runtimeJs []byte

// VM is a Javascript Virtual Machine running on Node.js
type VM interface {
	Run(javascript string) (any, error)
	Close() error
}

// Options for VM
type Options struct {
	// Dir is the working directory for the VM. Default is the same working
	// directory and currently running Go process.
	Dir string
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

func spreadPointerDef[Type any](def *Type, values ...Type) *Type {
	if len(values) == 0 {
		return def
	}

	return &values[0]
}

type vmMessage struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

type vmResult struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Content any    `json:"content"`
}

type vmConnection struct {
	ID               string
	Channels         sync.Map
	Conn             net.Conn
	InitializedMutex *sync.Mutex
	Initialized      bool
	Messsages        chan vmMessage
	log              *slog.Logger
	ChannelCount     int64
}

func (c *vmConnection) Close() error {
	var result *multierror.Error

	result = multierror.Append(result, c.Conn.Close())

	return result.ErrorOrNil()
}

func (c *vmConnection) AddChannel(id string) chan vmResult {
	c.log.Debug("Adding channel", "channelId", id)
	channel := make(chan vmResult)

	c.Channels.Store(id, channel)

	count := atomic.LoadInt64(&c.ChannelCount)
	atomic.StoreInt64(&c.ChannelCount, count+1)

	return channel
}

func (c *vmConnection) RemoveChannel(id string) {
	c.log.Debug("Removing channel", "channelId", id)

	c.Channels.Delete(id)

	count := atomic.LoadInt64(&c.ChannelCount)
	atomic.StoreInt64(&c.ChannelCount, count-1)
}

func (c *vmConnection) GetPendingRequests() int64 {
	return atomic.LoadInt64(&c.ChannelCount)
}

func (c *vmConnection) WaitForInitialization() {
	c.log.Debug("Waiting for initialization")

	for {
		if c.Initialized {
			return
		}
	}
}

func (c *vmConnection) HandleMessages() {
	c.WaitForInitialization()

	c.log.Debug("Handling messages")

	for {
		message := <-c.Messsages

		c.log.Debug("Handling message", "message", message)

		marshalled, err := json.Marshal(message)
		if err != nil {
			c.log.Debug("Error marshalling message", "error", err)
			continue
		}

		c.Conn.Write(append(marshalled, '\n'))
	}
}

func (c *vmConnection) Initialize() {
	c.InitializedMutex.Lock()
	c.Initialized = true
	c.InitializedMutex.Unlock()
}

func (c *vmConnection) ListenForResultAndDispatch() error {
	c.log.Debug("Listening for results")

	buffer, err := bufio.NewReader(c.Conn).ReadBytes('\n')
	if err != nil {
		c.log.Debug("Error reading from connection", "error", err)

		return err
	}

	bufferWithoutNewline := buffer[:len(buffer)-1]

	var result vmResult
	if err := json.Unmarshal(bufferWithoutNewline, &result); err != nil {
		c.log.Debug("Error unmarshalling result", "error", err)

		return nil
	}

	if result.ID == "INITIALIZED" {
		c.log.Debug("Received initialization message")

		c.Initialize()
	} else {
		go func() {
			c.log.Debug("Dispatching result", "result", result)

			channel, _ := c.Channels.Load(result.ID)

			channel.(chan vmResult) <- result
		}()
	}

	return nil
}

type nodeJsVM struct {
	options          *Options
	cmds             []*exec.Cmd
	socket           net.Listener
	connectionsMutex sync.Mutex
	connections      sync.Map
	log              *slog.Logger
	requests         int64
}

// Returns a Javascript Virtual Machine running an isolated process of
// Node.js.
func NewNodeJS(options ...Options) (VM, error) {
	option := spreadPointerDef(&Options{
		Port: 6543,
		Dir:  ".",
		Env:  []string{},
	}, options...)

	if option.Port == 0 {
		option.Port = 6543
	}
	if option.Dir == "" {
		option.Dir = "."
	}

	flags := []string{"--experimental-detect-module", "--no-warnings", "--input-type=module", "-e", string(runtimeJs)}

	if option != nil {
		flags = append(flags, option.Flags...)
	}

	socket, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", option.Port))
	if err != nil {
		return nil, err
	}

	log := logging.NewDefaultLogger(option.Logger)

	cmds := []*exec.Cmd{}

	nodeProcesses := option.NodeProcesses
	if nodeProcesses == 0 {
		nodeProcesses = 5
	}

	log.Debug("Starting node processes", "processes", nodeProcesses, "dir", option.Dir)

	for i := 0; i < nodeProcesses; i++ {
		cmd := nodejs.NewNodeJSCommand(nodejs.NodeJSCommandOptions{
			Script: string(runtimeJs),
			Dir:    option.Dir,
			Stdout: option.Stdout,
			Stderr: option.Stderr,
			Env: map[string]string{
				"PORT": fmt.Sprintf("%d", option.Port),
			},
			Flags: flags,
		})

		cmd.Env = append(cmd.Env, option.Env...)

		if err := cmd.Start(); err != nil {
			log.Info("Error starting node process", "error", err)
			socket.Close()
			return nil, err
		}

		cmds = append(cmds, cmd)
	}

	vm := &nodeJsVM{
		options:     option,
		cmds:        cmds,
		socket:      socket,
		connections: sync.Map{},
		log:         log,
	}

	go vm.acceptConnections()

	return vm, nil
}

func MustNewNodeJS(options ...Options) VM {
	vm, err := NewNodeJS(options...)
	if err != nil {
		panic(err)
	}
	return vm
}

func (vm *nodeJsVM) GetPendingRequests() int64 {
	return atomic.LoadInt64(&vm.requests)
}

func (vm *nodeJsVM) addPendingRequest() {
	requests := atomic.LoadInt64(&vm.requests)
	atomic.AddInt64(&vm.requests, requests+1)
}

func (vm *nodeJsVM) removePendingRequest() {
	requests := atomic.LoadInt64(&vm.requests)
	atomic.AddInt64(&vm.requests, requests-1)
}

func (vm *nodeJsVM) Run(javascript string) (any, error) {
	vm.addPendingRequest()
	defer vm.removePendingRequest()

	message := vmMessage{
		ID:      xid.New().String(),
		Type:    "import",
		Content: javascript,
	}

	vm.log.Debug("Running javascript", "message", message)

	connection := vm.getOpenConnection()

	vm.log.Debug("Got open connection", "id", connection.ID)

	channel := connection.AddChannel(message.ID)

	vm.log.Debug("Added channel", "id", message.ID)

	go func() { connection.Messsages <- message }()

	vm.log.Debug("Sent message", "message", message)

	result := <-channel

	connection.RemoveChannel(message.ID)

	vm.log.Debug("Received result", "result", result)

	if result.Status == "error" {
		return "", fmt.Errorf(result.Content.(string))
	}

	return result.Content, nil
}

func (vm *nodeJsVM) Close() error {
	var result *multierror.Error

	for _, cmd := range vm.cmds {
		result = multierror.Append(result, cmd.Process.Kill())
	}

	vm.connectionsMutex.Lock()
	defer vm.connectionsMutex.Unlock()

	vm.connections.Range(func(_ any, value any) bool {
		connection := value.(*vmConnection)

		result = multierror.Append(result, connection.Close())

		return true
	})

	result = multierror.Append(result, vm.socket.Close())

	return result.ErrorOrNil()
}

func (vm *nodeJsVM) acceptConnections() {
	for {
		connection, err := vm.socket.Accept()
		if err != nil {
			return
		}

		vm.log.Debug("Accepted connection", "address", connection.RemoteAddr().String())

		id := xid.New().String()

		vmConn := &vmConnection{
			ID:               id,
			Conn:             connection,
			Messsages:        make(chan vmMessage),
			Channels:         sync.Map{},
			log:              vm.log.With("id", id),
			InitializedMutex: &sync.Mutex{},
		}

		go vm.handleConnection(vmConn)
	}
}

func (vm *nodeJsVM) handleConnection(connection *vmConnection) {
	vm.connections.Store(connection.ID, connection)

	go connection.HandleMessages()

	for {
		err := connection.ListenForResultAndDispatch()
		if err != nil {
			vm.log.Debug("Error listening for result and dispatching", "error", err)

			connection.Close()

			vm.connections.Delete(connection.ID)

			return
		}
	}
}

func (vm *nodeJsVM) getOpenConnection() *vmConnection {
	vm.log.Debug("Getting available connection")

	found := ""

	for {
		vm.connections.Range(func(key any, value any) bool {
			vmConnection := value.(*vmConnection)

			if vmConnection.Initialized && float64(vmConnection.GetPendingRequests()) <= math.Ceil(float64(vm.GetPendingRequests())/float64(len(vm.cmds))) {

				vm.log.Debug("Found open connection", "id", vmConnection.ID)

				found = key.(string)

				return false
			}

			return true
		})

		if found != "" {
			break
		}
	}

	connection, _ := vm.connections.Load(found)

	return connection.(*vmConnection)
}
