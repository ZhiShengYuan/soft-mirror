package model

import "time"

// Platform represents an OS+arch combination
type Platform struct {
	OS   string
	Arch string
}

// Binary represents a stored binary file
type Binary struct {
	Program  string
	Version  string
	OS       string
	Arch     string
	Filename string // actual on-disk filename (e.g. "HyPlayer.msix")
	Size     int64
	ModTime  time.Time
}

// AllowedOS is the strict allowlist for OS values
var AllowedOS = map[string]bool{
	"linux":   true,
	"windows": true,
	"darwin":  true,
}

// AllowedArch is the strict allowlist for arch values
var AllowedArch = map[string]bool{
	"amd64": true,
	"arm64": true,
}
