//go:build e2e

package main

// Built with -tags e2e, the app listens on the socket named by MYGO_AUTOMATION so that
// scripts/e2e.py can drive the real window. Release builds leave this out.
import _ "github.com/egoist/mygo/plugins/automation"
