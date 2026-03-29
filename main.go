package main

import (
	"os"

	"github.com/0funct0ry/xwebs/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
