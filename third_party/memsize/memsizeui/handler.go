package memsizeui

import "net/http"

// Handler is a minimal, compatible stub for github.com/fjl/memsize/memsizeui.
//
// The upstream implementation depends on runtime internals via go:linkname.
// For this codebase we only need the type to exist to satisfy go-ethereum's
// optional debug wiring.
type Handler struct{}

func (h *Handler) Add(_ string, _ interface{}) {}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "memsize is disabled in this build", http.StatusNotImplemented)
}
