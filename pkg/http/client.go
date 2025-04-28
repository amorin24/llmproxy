package http

import (
	"net/http"
	"sync"
	"time"

	"github.com/amorin24/llmproxy/pkg/config"
	"github.com/sirupsen/logrus"
)

var (
	defaultClient     *http.Client
	defaultClientOnce sync.Once
	
	sharedTransport = &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
	}
)

type ClientConfig struct {
	Timeout            time.Duration
	KeepAlive          time.Duration
	MaxIdleConns       int
	MaxIdleConnsPerHost int
	IdleConnTimeout    time.Duration
}

func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Timeout:            30 * time.Second,
		KeepAlive:          30 * time.Second,
		MaxIdleConns:       100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:    90 * time.Second,
	}
}

func GetClient() *http.Client {
	defaultClientOnce.Do(func() {
		cfg := config.GetConfig()
		timeout := 30 * time.Second
		
		if cfg.HTTPTimeout > 0 {
			timeout = time.Duration(cfg.HTTPTimeout) * time.Second
		}
		
		defaultClient = &http.Client{
			Timeout:   timeout,
			Transport: sharedTransport,
		}
		
		logrus.WithFields(logrus.Fields{
			"timeout":             timeout,
			"max_idle_conns":      sharedTransport.MaxIdleConns,
			"max_idle_conns_host": sharedTransport.MaxIdleConnsPerHost,
			"idle_conn_timeout":   sharedTransport.IdleConnTimeout,
		}).Debug("Initialized shared HTTP client")
	})
	
	return defaultClient
}

func GetClientWithConfig(config ClientConfig) *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxIdleConnsPerHost,
		IdleConnTimeout:     config.IdleConnTimeout,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
	}
	
	return &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}
}

func GetTransport() *http.Transport {
	return sharedTransport
}
