package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/getlantern/systray"
	"github.com/pkg/browser"
)

type openRequest struct {
	URL string `json:"url"`
}

type openResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

var (
	mu     sync.Mutex
	server *http.Server
)

func startHTTPServer() {
	mu.Lock()
	server = newServer()
	s := server
	mu.Unlock()
	listenAndServe(s)
}

func newServer() *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/open", handleOpen)
	return &http.Server{Addr: ":8765", Handler: mux}
}

func listenAndServe(s *http.Server) {
	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("server error: %v", err)
		systray.SetTooltip("Port 8765 in use")
	}
}

func restartServer() {
	mu.Lock()
	s := server
	mu.Unlock()

	if s != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	}

	mu.Lock()
	server = newServer()
	s = server
	mu.Unlock()

	go listenAndServe(s)
}

func handleOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respond(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req openRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, "malformed JSON body")
		return
	}
	if req.URL == "" {
		respond(w, http.StatusBadRequest, "url field is missing or empty")
		return
	}

	if err := validateURL(req.URL); err != nil {
		respond(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := browser.OpenURL(req.URL); err != nil {
		log.Printf("browser.OpenURL error: %v", err)
		respond(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(openResponse{Status: "ok"})
}

func validateURL(raw string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https, got %q", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("URL has no host")
	}
	return nil
}

func respond(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(openResponse{Status: "error", Message: msg})
}
