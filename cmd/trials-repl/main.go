package main

import (
	"fmt"
	"os"
)

// build time variables
var (
	AppName      string
	BuildDate    string
	BuildVersion string
	BuildSHA     string
)

func main() {
	code, err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", AppName, err) // nolint: gas
	}

	os.Exit(code)
}

func run() (int, error) {

	cli := setup(start)
	err := cli.Execute()
	if err != nil {
		return 1, err
	}

	return 0, nil
}
