package nodejs

import (
	"io"
	"os"
	"os/exec"
)

type NodeJSCommandOptions struct {
	Script string
	Dir    string
	Env    map[string]string
	Flags  []string
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
}

func NewNodeJSCommand(options NodeJSCommandOptions) *exec.Cmd {
	flags := options.Flags

	flags = append(flags, "--experimental-detect-module", "--no-warnings", "--input-type=module")

	if _, err := os.Stat(options.Script); err != nil {
		flags = append(flags, "-e")
	}

	flags = append(flags, options.Script)

	cmd := exec.Command("node", flags...)

	for k, v := range options.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	cmd.Dir = options.Dir
	cmd.Stdout = options.Stdout
	cmd.Stderr = options.Stderr
	cmd.Stdin = options.Stdin

	return cmd
}
