package http

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetClient(t *testing.T) {
	client := GetClient()
	assert.NotNil(t, client, "GetClient should return a non-nil client")
	
	client2 := GetClient()
	assert.Equal(t, client, client2, "GetClient should return the same client instance on multiple calls")
}

func TestGetClientWithConfig(t *testing.T) {
	config := ClientConfig{
		Timeout:            15 * time.Second,
		KeepAlive:          20 * time.Second,
		MaxIdleConns:       50,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:    45 * time.Second,
	}
	
	client := GetClientWithConfig(config)
	assert.NotNil(t, client, "GetClientWithConfig should return a non-nil client")
	
	assert.Equal(t, 15*time.Second, client.Timeout, "Client timeout should match the configured value")
	
	transport, ok := client.Transport.(*http.Transport)
	assert.True(t, ok, "Client transport should be of type *http.Transport")
	assert.Equal(t, 50, transport.MaxIdleConns, "MaxIdleConns should match the configured value")
	assert.Equal(t, 10, transport.MaxIdleConnsPerHost, "MaxIdleConnsPerHost should match the configured value")
	assert.Equal(t, 45*time.Second, transport.IdleConnTimeout, "IdleConnTimeout should match the configured value")
}

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig()
	
	assert.Equal(t, 30*time.Second, config.Timeout, "Default timeout should be 30 seconds")
	assert.Equal(t, 30*time.Second, config.KeepAlive, "Default keep-alive should be 30 seconds")
	assert.Equal(t, 100, config.MaxIdleConns, "Default max idle connections should be 100")
	assert.Equal(t, 20, config.MaxIdleConnsPerHost, "Default max idle connections per host should be 20")
	assert.Equal(t, 90*time.Second, config.IdleConnTimeout, "Default idle connection timeout should be 90 seconds")
}

func TestGetTransport(t *testing.T) {
	transport := GetTransport()
	assert.NotNil(t, transport, "GetTransport should return a non-nil transport")
	assert.Equal(t, 100, transport.MaxIdleConns, "Transport MaxIdleConns should match the expected value")
	assert.Equal(t, 20, transport.MaxIdleConnsPerHost, "Transport MaxIdleConnsPerHost should match the expected value")
	assert.Equal(t, 90*time.Second, transport.IdleConnTimeout, "Transport IdleConnTimeout should match the expected value")
}
