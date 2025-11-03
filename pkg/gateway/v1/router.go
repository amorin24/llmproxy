package v1

import (
	"github.com/amorin24/llmproxy/pkg/jobs"
	"github.com/amorin24/llmproxy/pkg/pricing"
	"github.com/amorin24/llmproxy/pkg/router"
	"github.com/amorin24/llmproxy/pkg/streaming"
	"github.com/gorilla/mux"
)

func SetupV1Routes(r *mux.Router, routerInstance *router.Router, catalogLoader *pricing.CatalogLoader, jobStore *jobs.JobStore, jobWorker *jobs.JobWorker) {
	gatewayHandler := NewGatewayHandler(catalogLoader)
	sseHandler := streaming.NewSSEHandler(routerInstance, catalogLoader)
	wsHandler := streaming.NewWebSocketHandler(routerInstance, catalogLoader)
	jobHandler := jobs.NewJobHandler(jobStore, jobWorker, catalogLoader)

	r.HandleFunc("/query", gatewayHandler.QueryHandler).Methods("POST")
	r.HandleFunc("/cost-estimate", gatewayHandler.CostEstimateHandler).Methods("POST")
	r.HandleFunc("/dry-run", gatewayHandler.DryRunHandler).Methods("POST")
	
	r.HandleFunc("/stream", sseHandler.StreamQuery).Methods("POST")
	r.HandleFunc("/ws", wsHandler.HandleWebSocket).Methods("GET")
	
	r.HandleFunc("/jobs", jobHandler.SubmitJob).Methods("POST")
	r.HandleFunc("/jobs/{id}", jobHandler.GetJobStatus).Methods("GET")
	r.HandleFunc("/jobs/{id}/result", jobHandler.GetJobResult).Methods("GET")
}
