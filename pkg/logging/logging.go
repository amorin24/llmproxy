package logging

import (
	"os"

	"github.com/sirupsen/logrus"
)

func SetupLogging() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)
}

func LogRequest(model string, query string) {
	logrus.WithFields(logrus.Fields{
		"model": model,
		"query": query,
	}).Info("LLM query request")
}

func LogResponse(model string, responseTime int64, cached bool, err string) {
	fields := logrus.Fields{
		"model":        model,
		"responseTime": responseTime,
		"cached":       cached,
	}
	if err != "" {
		fields["error"] = err
		logrus.WithFields(fields).Error("LLM query error")
	} else {
		logrus.WithFields(fields).Info("LLM query response")
	}
}
