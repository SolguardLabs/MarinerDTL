package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"marinerdtl/src/scenario"
)

type Server struct {
	mux *http.ServeMux
}

func NewServer() *Server {
	server := &Server{mux: http.NewServeMux()}
	server.routes()
	return server
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.health)
	s.mux.HandleFunc("POST /v1/scenario/run", s.runScenario)
	s.mux.HandleFunc("POST /v1/scenario/validate", s.validateScenario)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "MarinerDTL"})
}

func (s *Server) runScenario(w http.ResponseWriter, r *http.Request) {
	var body scenario.Scenario
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	includeEvents := strings.EqualFold(r.URL.Query().Get("events"), "true")
	result, err := body.Run(includeEvents)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result.Report)
}

func (s *Server) validateScenario(w http.ResponseWriter, r *http.Request) {
	var body scenario.Scenario
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := body.Run(false)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"issues": result.Service.Validate(),
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		fmt.Fprintf(w, `{"error":%q}`, err.Error())
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
