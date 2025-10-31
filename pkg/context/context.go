package context

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type RequestContext struct {
	RequestID string
	
	StartTime time.Time
	
	Tenant string
	
	MaxCostUSD *float64
	
	Context context.Context
}

func NewRequestContext(ctx context.Context) *RequestContext {
	return &RequestContext{
		RequestID: uuid.New().String(),
		StartTime: time.Now(),
		Tenant:    "internal",
		Context:   ctx,
	}
}

func NewRequestContextWithID(ctx context.Context, requestID string) *RequestContext {
	return &RequestContext{
		RequestID: requestID,
		StartTime: time.Now(),
		Tenant:    "internal",
		Context:   ctx,
	}
}

func (rc *RequestContext) WithMaxCost(maxCost float64) *RequestContext {
	rc.MaxCostUSD = &maxCost
	return rc
}

func (rc *RequestContext) WithTenant(tenant string) *RequestContext {
	rc.Tenant = tenant
	return rc
}

func (rc *RequestContext) ElapsedTime() time.Duration {
	return time.Since(rc.StartTime)
}

func (rc *RequestContext) ElapsedMilliseconds() int64 {
	return rc.ElapsedTime().Milliseconds()
}

func (rc *RequestContext) Done() <-chan struct{} {
	return rc.Context.Done()
}

func (rc *RequestContext) Err() error {
	return rc.Context.Err()
}
