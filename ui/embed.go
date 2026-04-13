package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"sync"
)

//go:embed all:dist
var DistFE embed.FS

var (
	handler     http.Handler
	handlerOnce sync.Once
)

// Handler returns an http.Handler that serves the embedded UI.
func Handler() http.Handler {
	handlerOnce.Do(func() {
		sub, err := fs.Sub(DistFE, "dist")
		if err != nil {
			panic(err)
		}
		handler = http.FileServer(http.FS(sub))
	})
	return handler
}
