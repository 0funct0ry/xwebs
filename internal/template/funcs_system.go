package template

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (e *Engine) registerSystemFuncs() {
	if e.sandboxed {
		disabled := func(name string) func(...interface{}) (interface{}, error) {
			return func(args ...interface{}) (interface{}, error) {
				return nil, fmt.Errorf("function %s is disabled in sandbox mode", name)
			}
		}

		e.funcs["env"] = disabled("env")
		e.funcs["shell"] = disabled("shell")
		e.funcs["hostname"] = disabled("hostname")
		e.funcs["pid"] = disabled("pid")
		e.funcs["cwd"] = disabled("cwd")
		e.funcs["fileRead"] = disabled("fileRead")
		e.funcs["fileExists"] = disabled("fileExists")
		e.funcs["glob"] = disabled("glob")
		e.funcs["tempFile"] = disabled("tempFile")
		return
	}

	e.funcs["env"] = func(key string) string {
		return os.Getenv(key)
	}

	e.funcs["shell"] = func(command string) (string, error) {
		cmd := exec.Command("sh", "-c", command)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return string(output), err
		}
		return strings.TrimSpace(string(output)), nil
	}

	e.funcs["hostname"] = func() string {
		h, _ := os.Hostname()
		return h
	}

	e.funcs["pid"] = func() int {
		return os.Getpid()
	}

	e.funcs["cwd"] = func() string {
		c, _ := os.Getwd()
		return c
	}

	e.funcs["fileRead"] = func(path string) (string, error) {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	e.funcs["fileExists"] = func(path string) bool {
		_, err := os.Stat(path)
		return err == nil
	}

	e.funcs["glob"] = func(pattern string) ([]string, error) {
		return filepath.Glob(pattern)
	}

	e.funcs["tempFile"] = func(pattern string) (string, error) {
		f, err := os.CreateTemp("", pattern)
		if err != nil {
			return "", err
		}
		f.Close()
		return f.Name(), nil
	}
}
