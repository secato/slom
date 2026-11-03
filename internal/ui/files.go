package ui

import "os"

func homeDir() (string, error)          { return os.UserHomeDir() }
func readFile(p string) ([]byte, error) { return os.ReadFile(p) }
