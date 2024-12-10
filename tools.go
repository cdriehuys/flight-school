//go:build tools
// +build tools

package main

import (
	_ "github.com/jackc/tern/v2"
	_ "github.com/sqlc-dev/sqlc/cmd/sqlc"
)
