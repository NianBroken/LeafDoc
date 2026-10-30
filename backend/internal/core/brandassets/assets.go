// Package brandassets owns the embedded default visual resources shared by all surfaces.
package brandassets

import "embed"

const (
	SiteName    = "LeafDoc"
	AvatarFile  = "leafdoc-avatar.webp"
	LogoFile    = "leafdoc-logo.webp"
	FaviconFile = "leafdoc-favicon.ico"
)

// Files contains browser-ready resources and excludes the full-size source artwork.
//
//go:embed files/*
var Files embed.FS

// Read returns a built-in asset without accepting filesystem paths.
func Read(name string) ([]byte, error) { return Files.ReadFile("files/" + name) }
