//go:build !ui

package ui

import (
	"embed"
	"net/http"
)

// DistFE is an empty filesystem when built without the 'ui' build tag.
// Use -tags ui (after running make build-ui) to embed the compiled assets.
var DistFE embed.FS

// Handler returns a 501 Not Implemented response when the UI assets have not
// been embedded.  Pass -tags ui at build time to serve real assets.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "UI not available (binary built without -tags ui)", http.StatusNotImplemented)
	})
}
