package logging

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

type LogFields struct {
	Model           string
	Query           string
	Response        string
	ResponseTime    int64
	Cached          bool
	Error           string
	ErrorType       string
	StatusCode      int
	Timestamp       time.Time
	RequestID       string
	NumTokens       int
	NumRetries      int
	OriginalModel   string
	FallbackModel   string
}

func SetupLogging() {
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})
	
	logrus.SetOutput(os.Stdout)
	
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetLevel(level)
	}
}

func LogRequest(fields LogFields) {
	if fields.Timestamp.IsZero() {
		fields.Timestamp = time.Now()
	}
	
	logrus.WithFields(logrus.Fields{
		"model":       fields.Model,
		"query":       fields.Query,
		"timestamp":   fields.Timestamp,
		"request_id":  fields.RequestID,
		"event_type":  "llm_request",
	}).Info("LLM query request")
}

func LogResponse(fields LogFields) {
	logFields := logrus.Fields{
		"model":         fields.Model,
		"response_time": fields.ResponseTime,
		"cached":        fields.Cached,
		"timestamp":     fields.Timestamp,
		"request_id":    fields.RequestID,
		"event_type":    "llm_response",
	}
	
	if fields.NumTokens > 0 {
		logFields["num_tokens"] = fields.NumTokens
	}
	
	if fields.StatusCode > 0 {
		logFields["status_code"] = fields.StatusCode
	}
	
	if fields.Response != "" {
		truncationLimit := getTruncationLimit()
		if len(fields.Response) > truncationLimit {
			logFields["response"] = fields.Response[:truncationLimit] + "..."
			if logrus.GetLevel() == logrus.DebugLevel {
				logFields["full_response"] = fields.Response
			}
		} else {
			logFields["response"] = fields.Response
		}
	}
	
	if fields.NumRetries > 0 {
		logFields["num_retries"] = fields.NumRetries
	}
	
	if fields.OriginalModel != "" && fields.FallbackModel != "" {
		logFields["original_model"] = fields.OriginalModel
		logFields["fallback_model"] = fields.FallbackModel
	}
	
	if fields.Error != "" {
		logFields["error"] = fields.Error
		logFields["error_type"] = fields.ErrorType
		logrus.WithFields(logFields).Error("LLM query error")
	} else {
		logrus.WithFields(logFields).Info("LLM query response")
	}
}

func LogRouterActivity(originalModel, selectedModel string, taskType, reason string) {
	logrus.WithFields(logrus.Fields{
		"original_model": originalModel,
		"selected_model": selectedModel,
		"task_type":      taskType,
		"reason":         reason,
		"timestamp":      time.Now(),
		"event_type":     "router_activity",
	}).Info("Router model selection")
}
