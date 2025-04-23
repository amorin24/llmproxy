package api

import (
	"context"

	"github.com/amorin24/llmproxy/pkg/models"
)

type RouterInterface interface {
	RouteRequest(ctx context.Context, req models.QueryRequest) (models.ModelType, error)
	FallbackOnError(ctx context.Context, originalModel models.ModelType, req models.QueryRequest, err error) (models.ModelType, error)
	GetAvailability() models.StatusResponse
}

type CacheInterface interface {
	Get(req models.QueryRequest) (models.QueryResponse, bool)
	Set(req models.QueryRequest, resp models.QueryResponse)
}
