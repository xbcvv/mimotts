package main

import (
	"os"
	"strings"
)

type Config struct {
	Addr               string
	DataDir            string
	AdminToken         string
	CorsOrigin         string
	LogLevel           string // debug, info, warn, error
	LogFormat          string // json, text
	CallLogDir         string // directory for call log JSONL files
	RequireExternalKey bool   // reject unauthenticated TTS calls when true
}

func LoadConfig() Config {
	dataDir := env("DATA_DIR", "./data")
	return Config{
		Addr:               env("ADDR", ":7117"),
		DataDir:            dataDir,
		AdminToken:         env("ADMIN_TOKEN", "mimotts"),
		CorsOrigin:         env("CORS_ORIGIN", ""),
		LogLevel:           env("LOG_LEVEL", "info"),
		LogFormat:          env("LOG_FORMAT", "json"),
		CallLogDir:         env("CALL_LOG_DIR", strings.TrimRight(dataDir, "/")+"/call-logs"),
		RequireExternalKey: envBool("REQUIRE_EXTERNAL_KEY", false),
	}
}

func envBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
