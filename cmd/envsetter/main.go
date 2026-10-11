package main

import (
	"os"

	"github.com/SpaceTent/EnvSetter/app"
)

var version = "development"

func main() {
	os.Exit(app.Run(os.Args[1:], os.Stdout, os.Stderr, version))
}
