package main

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/amorin24/llmproxy/pkg/api"
	"github.com/amorin24/llmproxy/pkg/config"
	v1 "github.com/amorin24/llmproxy/pkg/gateway/v1"
	"github.com/amorin24/llmproxy/pkg/jobs"
	"github.com/amorin24/llmproxy/pkg/llm"
	"github.com/amorin24/llmproxy/pkg/logging"
	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/amorin24/llmproxy/pkg/monitoring"
	"github.com/amorin24/llmproxy/pkg/pricing"
	"github.com/amorin24/llmproxy/pkg/router"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func main() {
	logging.SetupLogging()

	cfg := config.GetConfig()

	monitoring.InitMonitoring()

	mockMode := os.Getenv("MOCK_MODE") == "true"
	if mockMode {
		logrus.Info("Running in MOCK_MODE - using mock providers")
		llm.Factory = func(modelType models.ModelType) (llm.Client, error) {
			return llm.NewMockClient(modelType)
		}
	}

	catalogLoader, err := pricing.NewCatalogLoader("docs/price-catalog.json")
	if err != nil {
		logrus.WithError(err).Warn("Failed to load price catalog, cost tracking may be inaccurate")
	}

	routerInstance := router.NewRouter()

	jobStore := jobs.NewJobStore(1 * time.Hour)
	jobWorker := jobs.NewJobWorker(jobStore, routerInstance, catalogLoader, 10)
	jobWorker.Start()

	r := mux.NewRouter()

	r.Use(monitoring.RequestLoggerMiddleware)
	r.Use(monitoring.MetricsMiddleware)

	handler := api.NewHandler()

	r.HandleFunc("/api/query", handler.QueryHandler).Methods("POST")
	r.HandleFunc("/api/parallel", handler.ParallelQueryHandler).Methods("POST")
	r.HandleFunc("/api/status", handler.StatusHandler).Methods("GET")
	r.HandleFunc("/api/download", handler.DownloadHandler).Methods("POST")
	r.HandleFunc("/api/health", handler.HealthHandler).Methods("GET")
	r.HandleFunc("/api/metrics", monitoring.MetricsHandler).Methods("GET")

	v1Router := r.PathPrefix("/v1/gateway").Subrouter()
	v1.SetupV1Routes(v1Router, routerInstance, catalogLoader, jobStore, jobWorker)

	r.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(http.Dir("./frontend/dist/assets"))))

	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join("frontend", "dist", r.URL.Path)
		if _, err := os.Stat(path); err == nil {
			http.ServeFile(w, r, path)
			return
		}
		http.ServeFile(w, r, filepath.Join("frontend", "dist", "index.html"))
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
