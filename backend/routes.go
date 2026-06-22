package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mimotts/backend/store"

	"mimotts/backend/calllog"
	"mimotts/backend/slogx"
)

type Server struct {
	cfg     Config
	store   *store.Store
	callLog *calllog.LogStore
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("POST /api/tts", s.tts)
	mux.HandleFunc("POST /v1/audio/speech", s.openAISpeech)
	mux.HandleFunc("GET /api/voices", s.publicVoices)
	mux.HandleFunc("GET /api/admin/status", s.admin(s.status))
	mux.HandleFunc("GET /api/admin/settings", s.admin(s.getSettings))
	mux.HandleFunc("PATCH /api/admin/settings", s.admin(s.updateSettings))
	mux.HandleFunc("GET /api/admin/mimo-keys", s.admin(s.listMiMoKeys))
	mux.HandleFunc("POST /api/admin/mimo-keys", s.admin(s.addMiMoKey))
	mux.HandleFunc("DELETE /api/admin/mimo-keys/", s.admin(s.deleteMiMoKey))
	mux.HandleFunc("PATCH /api/admin/mimo-keys/", s.admin(s.updateMiMoKey))
	mux.HandleFunc("GET /api/admin/mimo-keys/", s.admin(s.revealMiMoKeySecret))
	mux.HandleFunc("GET /api/admin/ext-keys", s.admin(s.listExternalKeys))
	mux.HandleFunc("POST /api/admin/ext-keys", s.admin(s.addExternalKey))
	mux.HandleFunc("POST /api/admin/ext-keys/", s.admin(s.rotateExternalKey))
	mux.HandleFunc("DELETE /api/admin/ext-keys/", s.admin(s.deleteExternalKey))
	mux.HandleFunc("PATCH /api/admin/ext-keys/", s.admin(s.updateExternalKey))
	mux.HandleFunc("GET /api/admin/voices", s.admin(s.listVoices))
	mux.HandleFunc("POST /api/admin/voices", s.admin(s.addVoice))
	mux.HandleFunc("DELETE /api/admin/voices/", s.admin(s.deleteVoice))
	// Call log admin APIs
	mux.HandleFunc("GET /api/admin/call-logs", s.admin(s.listCallLogs))
	mux.HandleFunc("GET /api/admin/call-logs/stats", s.admin(s.callLogStats))
	mux.HandleFunc("DELETE /api/admin/call-logs", s.admin(s.purgeCallLogs))
	mux.Handle("/", spaHandler("./static"))
	return s.cors(mux)
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.CorsOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", s.cfg.CorsOrigin)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-API-Key")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) currentAdminToken() string {
	settings := s.store.GetSettings()
	if strings.TrimSpace(settings.AdminToken) != "" {
		return strings.TrimSpace(settings.AdminToken)
	}
	return s.cfg.AdminToken
}

func (s *Server) admin(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ") != s.currentAdminToken() {
			writeErr(w, http.StatusUnauthorized, "admin token 无效")
			return
		}
		fn(w, r)
	}
}

// resolveCallerKeyID extracts the external key ID from the request without storing the secret.
// Returns ("", false) if no external key is used.
func (s *Server) resolveCallerKeyID(r *http.Request) (string, bool) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		apiKey = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	if apiKey == "" {
		return "", false
	}
	// Walk through external keys to find matching prefix — we identify by prefix, not secret
	for _, ek := range s.store.ListExternalKeys() {
		if strings.HasPrefix(apiKey, ek.Prefix) {
			return ek.ID, true
		}
	}
	return "", false
}

func (s *Server) validateTTSCaller(r *http.Request) (string, bool) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		apiKey = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	if apiKey == "" {
		return "", !s.cfg.RequireExternalKey
	}
	if !s.store.ValidateExternalKey(apiKey, "tts") {
		return "", false
	}
	callerKeyID, _ := s.resolveCallerKeyID(r)
	return callerKeyID, true
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"ok": true})
}
func (s *Server) status(w http.ResponseWriter, r *http.Request) { writeJSON(w, s.store.Status()) }
func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	st := s.store.GetSettings()
	writeJSON(w, map[string]any{"adminTokenSet": strings.TrimSpace(st.AdminToken) != "", "proxyUrl": st.ProxyURL})
}
func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AdminToken *string `json:"adminToken"`
		ProxyURL   *string `json:"proxyUrl"`
	}
	if decode(w, r, &body) {
		if err := s.store.UpdateSettings(body.AdminToken, body.ProxyURL); err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	}
}
func (s *Server) publicVoices(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, sanitizeVoices(s.store.ListVoices()))
}
func (s *Server) listVoices(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, sanitizeVoices(s.store.ListVoices()))
}
func (s *Server) listMiMoKeys(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.store.ListMiMoKeys())
}
func (s *Server) listExternalKeys(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.store.ListExternalKeys())
}





func (s *Server) addMiMoKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Label, Key string
		Weight     int
	}
	if decode(w, r, &body) {
		key, err := s.store.AddMiMoKey(body.Label, body.Key, body.Weight)
		if err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		writeJSON(w, map[string]any{"id": key.ID})
	}
}
func (s *Server) deleteMiMoKey(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/admin/mimo-keys/")
	if err := s.store.DeleteMiMoKey(id); err != nil {
		writeErr(w, 404, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}
func (s *Server) revealMiMoKeySecret(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/mimo-keys/")
	if !strings.HasSuffix(path, "/secret") {
		writeErr(w, 404, "not found")
		return
	}
	id := strings.TrimSuffix(path, "/secret")
	if strings.TrimSpace(id) == "" {
		writeErr(w, 404, "not found")
		return
	}
	secret, err := s.store.GetMiMoKeySecret(id)
	if err != nil {
		writeErr(w, 404, err.Error())
		return
	}
	writeJSON(w, map[string]string{"id": id, "secret": secret})
}

func (s *Server) updateMiMoKey(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/admin/mimo-keys/")
	var body struct {
		Label   *string `json:"label"`
		Key     *string `json:"key"`
		Weight  *int    `json:"weight"`
		Enabled *bool   `json:"enabled"`
	}
	if decode(w, r, &body) {
		if err := s.store.UpdateMiMoKey(id, body.Weight, body.Enabled, body.Label, body.Key); err != nil {
			writeErr(w, 404, err.Error())
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	}
}

func (s *Server) addExternalKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Label       string
		Permissions []string
	}
	if decode(w, r, &body) {
		ext, secret, err := s.store.AddExternalKey(body.Label, body.Permissions)
		if err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		writeJSON(w, map[string]any{"key": ext, "secret": secret})
	}
}
func (s *Server) deleteExternalKey(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/admin/ext-keys/")
	if err := s.store.DeleteExternalKey(id); err != nil {
		writeErr(w, 404, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}
func (s *Server) rotateExternalKey(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/ext-keys/")
	if !strings.HasSuffix(path, "/rotate") {
		writeErr(w, 404, "not found")
		return
	}
	id := strings.TrimSuffix(path, "/rotate")
	if strings.TrimSpace(id) == "" {
		writeErr(w, 404, "not found")
		return
	}
	secret, err := s.store.RotateExternalKey(id)
	if err != nil {
		writeErr(w, 404, err.Error())
		return
	}
	writeJSON(w, map[string]any{"id": id, "secret": secret})
}

func (s *Server) updateExternalKey(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/admin/ext-keys/")
	var body struct {
		Enabled     *bool    `json:"enabled"`
		Permissions []string `json:"permissions"`
		Label       *string  `json:"label"`
	}
	if decode(w, r, &body) {
		if err := s.store.UpdateExternalKey(id, body.Enabled, body.Permissions, body.Label); err != nil {
			writeErr(w, 404, err.Error())
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	}
}

func (s *Server) addVoice(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Kind         string `json:"kind"`
		Label        string `json:"label"`
		Preset       string `json:"preset"`
		Design       string `json:"design"`
		CloneDataURL string `json:"cloneDataUrl"`
		Language     string `json:"language"`
		Description  string `json:"description"`
	}
	if !decode(w, r, &body) {
		return
	}
	var v store.Voice
	var err error
	switch body.Kind {
	case "preset":
		v, err = s.store.AddPresetVoice(body.Label, body.Preset, body.Language, body.Description)
	case "design":
		v, err = s.store.AddDesignVoice(body.Label, body.Design, body.Language, body.Description)
	case "clone":
		v, err = s.store.AddCloneVoice(body.Label, body.CloneDataURL, body.Language, body.Description)
	default:
		err = fmt.Errorf("未知音色类型: %s", body.Kind)
	}
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, sanitizeVoice(v))
}
func (s *Server) deleteVoice(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/admin/voices/")
	if err := s.store.DeleteVoice(id); err != nil {
		writeErr(w, 404, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

// ====================== TTS endpoints (with call logging) ======================

func (s *Server) tts(w http.ResponseWriter, r *http.Request) {
	requestID := fmt.Sprintf("tts_%d_%s", time.Now().UnixNano(), randomHex(4))
	start := time.Now()

	callerKeyID, ok := s.validateTTSCaller(r)
	if !ok {
		s.recordCall(calllog.CallRecord{
			RequestID: requestID, Timestamp: start, Endpoint: "/api/tts",
			Success: false, HTTPStatus: http.StatusUnauthorized,
			DurationMs:  time.Since(start).Milliseconds(),
			CallerKeyID: callerKeyID,
			Error:       "外部调用 key 无效或缺失",
		})
		writeErr(w, http.StatusUnauthorized, "外部调用 key 无效或缺失")
		return
	}

	var req TTSRequest
	if !decode(w, r, &req) {
		return
	}
	if req.Model == "" {
		req.Model = "mimo-v2.5-tts"
	}
	inputChars := len(req.Text)

	requestedModel, requestedVoice := req.Model, req.Voice
	audio, key, resolvedReq, resolvedVoice, err := synthesizeWithPool(s.store, req)
	durationMs := time.Since(start).Milliseconds()
	httpStatus := 200
	errMsg := ""

	if err != nil {
		s.store.RecordFailure()
		httpStatus = 400
		errMsg = slogx.SanitizeErr(err.Error(), 200)
		s.recordCall(calllog.CallRecord{
			RequestID: requestID, Timestamp: start, Endpoint: "/api/tts",
			Model: resolvedReq.Model, Voice: func() string { v, _, _ := resolvedVoiceDisplay(resolvedReq, resolvedVoice, requestedVoice); return v }(),
			VoiceKind: func() string { _, k, _ := resolvedVoiceDisplay(resolvedReq, resolvedVoice, requestedVoice); return k }(), ProjectVoiceID: func() string { _, _, id := resolvedVoiceDisplay(resolvedReq, resolvedVoice, requestedVoice); return id }(),
			RequestedModel: requestedModel, RequestedVoice: requestedVoice,
			UpstreamID: key.ID, UpstreamLabel: key.Label, UpstreamBaseURL: slogx.MaskURL(key.BaseURL),
			CallerKeyID: callerKeyID,
			Success:     false, HTTPStatus: httpStatus, DurationMs: durationMs,
			InputChars: inputChars, Error: errMsg,
		})
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	s.store.RecordSuccess()
	s.recordCall(calllog.CallRecord{
		RequestID: requestID, Timestamp: start, Endpoint: "/api/tts",
		Model: resolvedReq.Model, Voice: func() string { v, _, _ := resolvedVoiceDisplay(resolvedReq, resolvedVoice, requestedVoice); return v }(),
		VoiceKind: func() string { _, k, _ := resolvedVoiceDisplay(resolvedReq, resolvedVoice, requestedVoice); return k }(), ProjectVoiceID: func() string { _, _, id := resolvedVoiceDisplay(resolvedReq, resolvedVoice, requestedVoice); return id }(),
		RequestedModel: requestedModel, RequestedVoice: requestedVoice,
		UpstreamID: key.ID, UpstreamLabel: key.Label, UpstreamBaseURL: slogx.MaskURL(key.BaseURL),
		CallerKeyID: callerKeyID,
		Success:     true, HTTPStatus: httpStatus, DurationMs: durationMs,
		AudioBytes: len(audio), InputChars: inputChars,
	})

	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Content-Disposition", "inline; filename=mimotts.wav")
	w.Header().Set("X-MiMo-Key-ID", key.ID)
	_, _ = w.Write(audio)
}

// openAISpeech provides an OpenAI /v1/audio/speech compatible endpoint.
func (s *Server) openAISpeech(w http.ResponseWriter, r *http.Request) {
	requestID := fmt.Sprintf("oai_%d_%s", time.Now().UnixNano(), randomHex(4))
	start := time.Now()

	callerKeyID, ok := s.validateTTSCaller(r)
	if !ok {
		s.recordCall(calllog.CallRecord{
			RequestID: requestID, Timestamp: start, Endpoint: "/v1/audio/speech",
			Success: false, HTTPStatus: http.StatusUnauthorized,
			DurationMs:  time.Since(start).Milliseconds(),
			CallerKeyID: callerKeyID,
			Error:       "外部调用 key 无效或缺失",
		})
		writeErr(w, http.StatusUnauthorized, "外部调用 key 无效或缺失")
		return
	}

	var body struct {
		Model          string `json:"model"`
		Input          string `json:"input"`
		Voice          string `json:"voice"`
		ResponseFormat string `json:"response_format"`
		Instructions   string `json:"instructions"`
	}
	if !decode(w, r, &body) {
		return
	}
	req := TTSRequest{
		Text:    body.Input,
		Model:   body.Model,
		Voice:   body.Voice,
		Context: body.Instructions,
	}
	if req.Model == "" {
		req.Model = "mimo-v2.5-tts"
	}
	inputChars := len(req.Text)

	requestedModel, requestedVoice := req.Model, req.Voice
	audio, key, resolvedReq, resolvedVoice, err := synthesizeWithPool(s.store, req)
	durationMs := time.Since(start).Milliseconds()
	httpStatus := 200
	errMsg := ""

	if err != nil {
		s.store.RecordFailure()
		httpStatus = 400
		errMsg = slogx.SanitizeErr(err.Error(), 200)
		s.recordCall(calllog.CallRecord{
			RequestID: requestID, Timestamp: start, Endpoint: "/v1/audio/speech",
			Model: resolvedReq.Model, Voice: func() string { v, _, _ := resolvedVoiceDisplay(resolvedReq, resolvedVoice, requestedVoice); return v }(),
			VoiceKind: func() string { _, k, _ := resolvedVoiceDisplay(resolvedReq, resolvedVoice, requestedVoice); return k }(), ProjectVoiceID: func() string { _, _, id := resolvedVoiceDisplay(resolvedReq, resolvedVoice, requestedVoice); return id }(),
			RequestedModel: requestedModel, RequestedVoice: requestedVoice,
			UpstreamID: key.ID, UpstreamLabel: key.Label, UpstreamBaseURL: slogx.MaskURL(key.BaseURL),
			CallerKeyID: callerKeyID,
			Success:     false, HTTPStatus: httpStatus, DurationMs: durationMs,
			InputChars: inputChars, Error: errMsg,
		})
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	s.store.RecordSuccess()
	s.recordCall(calllog.CallRecord{
		RequestID: requestID, Timestamp: start, Endpoint: "/v1/audio/speech",
		Model: resolvedReq.Model, Voice: func() string { v, _, _ := resolvedVoiceDisplay(resolvedReq, resolvedVoice, requestedVoice); return v }(),
		VoiceKind: func() string { _, k, _ := resolvedVoiceDisplay(resolvedReq, resolvedVoice, requestedVoice); return k }(), ProjectVoiceID: func() string { _, _, id := resolvedVoiceDisplay(resolvedReq, resolvedVoice, requestedVoice); return id }(),
		RequestedModel: requestedModel, RequestedVoice: requestedVoice,
		UpstreamID: key.ID, UpstreamLabel: key.Label, UpstreamBaseURL: slogx.MaskURL(key.BaseURL),
		CallerKeyID: callerKeyID,
		Success:     true, HTTPStatus: httpStatus, DurationMs: durationMs,
		AudioBytes: len(audio), InputChars: inputChars,
	})

	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Content-Disposition", "inline; filename=mimotts.wav")
	w.Header().Set("X-MiMo-Key-ID", key.ID)
	_, _ = w.Write(audio)
}

func resolvedVoiceDisplay(req TTSRequest, voice store.Voice, requestedVoice string) (string, string, string) {
	projectVoiceID := ""
	voiceKind := voice.Kind
	displayVoice := req.Voice
	if voice.ID != "" {
		projectVoiceID = voice.ID
	}
	if displayVoice == "" {
		switch voice.Kind {
		case "clone":
			displayVoice = requestedVoice + " → voiceDataURL"
		case "design":
			displayVoice = requestedVoice + " → text design"
		case "preset":
			displayVoice = voice.Preset
		}
	}
	if displayVoice == "" {
		displayVoice = requestedVoice
	}
	return displayVoice, voiceKind, projectVoiceID
}

// recordCall is a helper that logs the call record both to the structured logger and the JSONL store.
func (s *Server) recordCall(r calllog.CallRecord) {
	fields := map[string]any{
		"requestId":       r.RequestID,
		"endpoint":        r.Endpoint,
		"model":           r.Model,
		"voice":           r.Voice,
		"upstreamId":      r.UpstreamID,
		"upstreamLabel":   r.UpstreamLabel,
		"upstreamBaseUrl": r.UpstreamBaseURL,
		"callerKeyId":     r.CallerKeyID,
		"success":         r.Success,
		"httpStatus":      r.HTTPStatus,
		"durationMs":      r.DurationMs,
		"audioBytes":      r.AudioBytes,
		"inputChars":      r.InputChars,
	}
	if r.Error != "" {
		fields["error"] = r.Error
	}
	if r.Success {
		slogx.Info("tts call completed", fields)
	} else {
		slogx.Warn("tts call failed", fields)
	}
	// Persist to JSONL (best-effort — don't fail the request on log errors)
	if s.callLog != nil {
		_ = s.callLog.Record(r)
	}
}

// ====================== Call Log Admin APIs ======================

func (s *Server) listCallLogs(w http.ResponseWriter, r *http.Request) {
	if s.callLog == nil {
		writeJSON(w, calllog.QueryResult{Logs: []calllog.CallRecord{}, Total: 0})
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	endpoint := q.Get("endpoint")

	var success *bool
	if v := q.Get("success"); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			success = &b
		}
	}

	result, err := s.callLog.Query(calllog.QueryOpts{
		Limit:    limit,
		Offset:   offset,
		Endpoint: endpoint,
		Success:  success,
	})
	if err != nil {
		writeErr(w, 500, "查询调用记录失败: "+err.Error())
		return
	}
	writeJSON(w, result)
}

func (s *Server) callLogStats(w http.ResponseWriter, r *http.Request) {
	if s.callLog == nil {
		writeJSON(w, calllog.CallLogStats{})
		return
	}
	stats, err := s.callLog.Stats()
	if err != nil {
		writeErr(w, 500, "统计调用记录失败: "+err.Error())
		return
	}
	writeJSON(w, stats)
}

func (s *Server) purgeCallLogs(w http.ResponseWriter, r *http.Request) {
	if s.callLog == nil {
		writeJSON(w, map[string]any{"deleted": 0})
		return
	}
	q := r.URL.Query()
	daysStr := q.Get("olderThanDays")
	days := 30 // default: purge logs older than 30 days
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}
	before := time.Now().AddDate(0, 0, -days)
	deleted, err := s.callLog.PurgeBefore(before)
	if err != nil {
		writeErr(w, 500, "清理调用记录失败: "+err.Error())
		return
	}
	slogx.Info("call logs purged", map[string]any{"deleted": deleted, "olderThanDays": days})
	writeJSON(w, map[string]any{"deleted": deleted, "olderThanDays": days})
}

// ====================== helpers ======================

func sanitizeVoices(voices []store.Voice) []map[string]any {
	out := make([]map[string]any, 0, len(voices))
	for _, v := range voices {
		out = append(out, sanitizeVoice(v))
	}
	return out
}
func sanitizeVoice(v store.Voice) map[string]any {
	return map[string]any{
		"id": v.ID, "label": v.Label, "kind": v.Kind, "preset": v.Preset,
		"language": v.Language, "description": v.Description, "createdAt": v.CreatedAt,
		"availableKeyCount": len(v.AvailableKeys), "design": truncate(v.Design, 80),
	}
}
func truncate(v string, n int) string {
	runes := []rune(v)
	if len(runes) <= n {
		return v
	}
	return string(runes[:n]) + "..."
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	defer r.Body.Close()
	const maxBody = 16 << 20
	limited := io.LimitReader(r.Body, maxBody+1)
	b, err := io.ReadAll(limited)
	if err != nil {
		writeErr(w, 400, "读取请求体失败")
		return false
	}
	if len(b) > maxBody {
		writeErr(w, http.StatusRequestEntityTooLarge, "请求体过大（最大 16MB）")
		return false
	}
	if err := json.Unmarshal(b, v); err != nil {
		writeErr(w, 400, "JSON 格式错误")
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
