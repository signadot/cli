package override

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/signadot/libconnect/common/override"
)

// createLogServer creates an HTTP server and listener for log consumption
// Returns the server, listener, and the actual port that was assigned
func createLogServer(sandboxName, localAddress string) (*http.Server, net.Listener, int) {
	mux := http.NewServeMux()

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("error listening on available port: %v", err)
	}

	// Get the actual port that was assigned
	listeningPort := ln.Addr().(*net.TCPAddr).Port

	mux.HandleFunc("/api/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read the log body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		var logEntry override.LogEntry
		if err := json.Unmarshal(body, &logEntry); err != nil {
			http.Error(w, "failed to unmarshal body", http.StatusInternalServerError)
			return
		}

		printFormattedLogEntry(&logEntry, sandboxName, localAddress)

		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Handler: mux,
	}

	return server, ln, listeningPort
}

// startLogServer starts an HTTP server with the provided listener
func startLogServer(server *http.Server, ln net.Listener) {
	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("log server error: %v", err)
		}
	}()
}
