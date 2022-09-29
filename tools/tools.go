//go:build tools

package tools

// This is a list of tools to be maintained; some non-main
// package in the same repo needs to be imported so they'll be managed in
// go.mod.

import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/google/pprof"
	_ "github.com/jackc/tern"
	_ "github.com/mailru/easyjson/easyjson"
	_ "github.com/tomwright/dasel/cmd/dasel"
	_ "golang.org/x/tools/cmd/godoc"
	_ "golang.org/x/tools/cmd/stringer"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
	_ "mvdan.cc/gofumpt"
)
