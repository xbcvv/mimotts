package main

import (
	"net/http"
	"os"

	"mimotts/backend/calllog"
	"mimotts/backend/slogx"
	"mimotts/backend/store"
)

func main() {
	cfg := LoadConfig()

	// Configure structured logger
	os.Setenv("LOG_LEVEL", cfg.LogLevel)
	os.Setenv("LOG_FORMAT", cfg.LogFormat)
	slogx.SetOutput(os.Stderr)

	slogx.Info("MimoTTS starting", map[string]any{
		"addr":               cfg.Addr,
		"dataDir":            cfg.DataDir,
		"callLogDir":         cfg.CallLogDir,
		"logLevel":           cfg.LogLevel,
		"logFormat":          cfg.LogFormat,
		"requireExternalKey": cfg.RequireExternalKey,
	})
	if cfg.AdminToken == "change-me-now" {
		slogx.Warn("ADMIN_TOKEN is using default value; set a real ADMIN_TOKEN before exposing this service")
	}

	st, err := store.New(cfg.DataDir)
	if err != nil {
		slogx.Error("store init failed", map[string]any{"error": slogx.SanitizeErr(err.Error(), 300)})
		os.Exit(1)
	}

	cl, err := calllog.New(cfg.CallLogDir)
	if err != nil {
		slogx.Error("call log init failed", map[string]any{"error": slogx.SanitizeErr(err.Error(), 300)})
		os.Exit(1)
	}
	defer cl.Close()

	srv := &Server{cfg: cfg, store: st, callLog: cl}
	slogx.Info("MimoTTS listening", map[string]any{"addr": cfg.Addr})
	if err := http.ListenAndServe(cfg.Addr, srv.routes()); err != nil {
		slogx.Error("MimoTTS stopped", map[string]any{"error": slogx.SanitizeErr(err.Error(), 300)})
		return
	}
}
