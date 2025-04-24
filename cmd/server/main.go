package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/amorin24/llmproxy/pkg/api"
	"github.com/amorin24/llmproxy/pkg/config"
	"github.com/amorin24/llmproxy/pkg/logging"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func main() {
	logging.SetupLogging()

	cfg := config.GetConfig()

	r := mux.NewRouter()

	handler := api.NewHandler()

	r.HandleFunc("/api/query", handler.QueryHandler).Methods("POST")
	r.HandleFunc("/api/status", handler.StatusHandler).Methods("GET")
	r.HandleFunc("/api/download", handler.DownloadHandler).Methods("POST")
	r.HandleFunc("/api/health", handler.HealthHandler).Methods("GET")

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./ui"))))

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join("ui", "templates", "index.html"))
	})

	port := cfg.Port
	logrus.Infof("Starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		logrus.Fatalf("Error starting server: %v", err)
		os.Exit(1)
	}
}
