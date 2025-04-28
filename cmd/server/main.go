package main

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/amorin24/llmproxy/pkg/api"
	"github.com/amorin24/llmproxy/pkg/config"
	"github.com/amorin24/llmproxy/pkg/logging"
	"github.com/amorin24/llmproxy/pkg/monitoring"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func main() {
	logging.SetupLogging()

	cfg := config.GetConfig()

	monitoring.InitMonitoring()

	r := mux.NewRouter()

	r.Use(monitoring.RequestLoggerMiddleware)
	r.Use(monitoring.MetricsMiddleware)

	handler := api.NewHandler()

	r.HandleFunc("/api/query", handler.QueryHandler).Methods("POST")
	r.HandleFunc("/api/query-parallel", handler.ParallelQueryHandler).Methods("POST")
	r.HandleFunc("/api/status", handler.StatusHandler).Methods("GET")
	r.HandleFunc("/api/download", handler.DownloadHandler).Methods("POST")
	r.HandleFunc("/api/health", handler.HealthHandler).Methods("GET")
	r.HandleFunc("/api/metrics", monitoring.MetricsHandler).Methods("GET")

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./ui"))))

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join("ui", "templates", "index.html"))
	})

	port := cfg.Port
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logrus.Infof("Starting server on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		logrus.Fatalf("Error starting server: %v", err)
		os.Exit(1)
	}
}
