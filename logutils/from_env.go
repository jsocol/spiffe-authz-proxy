package logutils

import (
	"fmt"
	"log/slog"
	"os"
)

func FromEnv() *slog.Logger {
	var logLevel slog.Level
	err := logLevel.UnmarshalText([]byte(os.Getenv("LOG_LEVEL")))
	if err != nil {
		fmt.Printf("%s, defaulting to %s\n", err, logLevel.Level())
	}

	logOptions := &slog.HandlerOptions{
		Level: logLevel,
	}
	var logHandler slog.Handler
	switch os.Getenv("LOG_FORMAT") {
	case "text":
		logHandler = slog.NewTextHandler(os.Stdout, logOptions)
	case "null":
		logHandler = slog.DiscardHandler
	case "json":
		logHandler = slog.NewJSONHandler(os.Stdout, logOptions)
	default:
		fmt.Printf("unknown log format: %s. supported values are [json, text]\n", os.Getenv("LOG_FORMAT"))
		logHandler = slog.NewJSONHandler(os.Stdout, logOptions)
	}
	logger := slog.New(logHandler)

	return logger
}
