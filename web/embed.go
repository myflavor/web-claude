package web

import "embed"

// StaticFS holds the built web UI (Vite output under static/).
//
//go:embed all:static
var StaticFS embed.FS
