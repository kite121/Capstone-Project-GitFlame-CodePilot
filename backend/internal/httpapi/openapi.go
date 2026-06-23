package httpapi

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.json
var spec []byte

func (s *Server) openAPI(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(200)
	_, _ = w.Write(spec)
}
