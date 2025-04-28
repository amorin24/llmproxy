package monitoring

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type ResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

func (rw *ResponseWriter) WriteHeader(code int) {
	rw.StatusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		rw := &ResponseWriter{
			ResponseWriter: w,
			StatusCode:     http.StatusOK, // Default to 200 OK
		}
		
		next.ServeHTTP(rw, r)
		
		duration := time.Since(start)
		
		logrus.WithFields(logrus.Fields{
			"method":     r.Method,
			"path":       r.URL.Path,
			"status":     rw.StatusCode,
			"duration":   duration.Milliseconds(),
			"user_agent": r.UserAgent(),
		}).Info("Request processed")
		
		if r.URL.Path == "/api/query" {
			RequestsTotal.WithLabelValues("api", http.StatusText(rw.StatusCode)).Inc()
		}
	})
}

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		rw := &ResponseWriter{
			ResponseWriter: w,
			StatusCode:     http.StatusOK, // Default to 200 OK
		}
		
		next.ServeHTTP(rw, r)
		
		logrus.WithFields(logrus.Fields{
			"method":     r.Method,
			"path":       r.URL.Path,
			"status":     rw.StatusCode,
			"duration":   time.Since(start).Milliseconds(),
			"remote_ip":  r.RemoteAddr,
			"user_agent": r.UserAgent(),
		}).Info("HTTP Request")
	})
}

func RequestLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		rw := &ResponseWriter{
			ResponseWriter: w,
			StatusCode:     http.StatusOK, // Default to 200 OK
		}
		
		next.ServeHTTP(rw, r)
		
		duration := time.Since(start)
		
		logrus.WithFields(logrus.Fields{
			"method":     r.Method,
			"path":       r.URL.Path,
			"status":     rw.StatusCode,
			"duration":   duration.Milliseconds(),
			"remote_ip":  r.RemoteAddr,
			"user_agent": r.UserAgent(),
			"referer":    r.Referer(),
		}).Info("HTTP Request")
		
		if r.URL.Path == "/api/query" || r.URL.Path == "/api/parallel" {
			IncreaseActiveRequests("api")
			defer DecreaseActiveRequests("api")
			
			RequestDuration.WithLabelValues("api").Observe(duration.Seconds())
			
			RequestsTotal.WithLabelValues("api", http.StatusText(rw.StatusCode)).Inc()
		}
	})
}
