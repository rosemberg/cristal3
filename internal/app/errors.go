// Package app contains application-layer orchestration for each CLI subcommand.
// Subcommand entrypoints here depend only on ports declared in internal/domain/ports.
package app

import "errors"

// ErrNotImplemented is returned by subcommand entrypoints that are still stubs.
var ErrNotImplemented = errors.New("not implemented")
