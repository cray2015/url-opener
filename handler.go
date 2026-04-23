package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
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
	mu       sync.Mutex
	server   *http.Server
	urlRegex = regexp.MustCompile(`https?://\S+`)
)

func startHTTPServer() {
	mu.Lock()
	server = newServer()
	s := server
	mu.Unlock()
	log.Printf("listening on :8765")
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

	log.Printf("restarting listener on :8765")
	go listenAndServe(s)
}

func handleOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respond(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respond(w, http.StatusBadRequest, "failed to read body")
		return
	}
	log.Printf("POST /open body: %s", body)

	var req openRequest
	if err := json.Unmarshal(body, &req); err != nil {
		respond(w, http.StatusBadRequest, "malformed JSON body")
		return
	}
	if req.URL == "" {
		respond(w, http.StatusBadRequest, "url field is missing or empty")
		return
	}

	text := req.URL

	cleanURL, err := extractURL(text)
	if err != nil {
		log.Printf("extractURL error: %v (input: %s)", err, text)
		respond(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := browser.OpenURL(cleanURL); err != nil {
		log.Printf("browser.OpenURL error: %v", err)
		respond(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("opened %s", cleanURL)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(openResponse{Status: "ok"})
}

// extractURL finds a valid http/https URL from text that may contain extra
// content (e.g. "Source: ZDNET\nhttps://example.com"). It URL-decodes the
// input, unescapes backslash-escaped slashes, tries the whole string first,
// then falls back to a regex scan for the first URL-shaped token.
func extractURL(text string) (string, error) {
	if decoded, err := url.QueryUnescape(text); err == nil {
		text = decoded
	}
	text = strings.ReplaceAll(text, `\/`, `/`)

	if candidate := strings.TrimSpace(text); validateURL(candidate) == nil {
		return candidate, nil
	}

	match := urlRegex.FindString(text)
	if match == "" {
		return "", fmt.Errorf("no valid URL found in body")
	}
	match = strings.TrimRight(match, `.,;:!?)"'`)

	if err := validateURL(match); err != nil {
		return "", fmt.Errorf("no valid URL found in body")
	}
	return match, nil
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
