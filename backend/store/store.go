package store

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type MiMoKey struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	Key       string    `json:"key"`
	Weight    int       `json:"weight"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
}

type ExternalKey struct {
	ID          string     `json:"id"`
	Label       string     `json:"label"`
	Hash        string     `json:"hash"`
	Prefix      string     `json:"prefix"`
	Masked      string     `json:"masked"`
	Secret      string     `json:"secret,omitempty"`
	Permissions []string   `json:"permissions"`
	Enabled     bool       `json:"enabled"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt,omitempty"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
}

// Voice 是本项目音色库的核心模型，对外暴露 voiceId 即可被外部调用。
// 内部按 kind 选择不同的回放策略，从而屏蔽上游 key 差异。
type Voice struct {
	ID           string    `json:"id"`       // 项目侧 voiceId，供外部 API 使用
	Label        string    `json:"label"`    // 名称
	Kind         string    `json:"kind"`     // preset | design | clone
	Preset       string    `json:"preset"`   // kind=preset：预置音色名（冰糖/茉莉/...）
	Design       string    `json:"design"`   // kind=design：音色描述
	CloneDataURL string    `json:"clone"`    // kind=clone：data:audio/... base64
	Language     string    `json:"language"` // zh / en
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"createdAt"`
	// 哪些上游 key 曾经为这个 voice 生成过内容；用于在“按 key 轮询”时快速选路。
	// 如果为空，表示本项目已用任意 key 合成过，轮询任意 key 都视为可用。
	AvailableKeys []string `json:"availableKeys"`
}

type UpstreamChannel struct {
	ID           string            `json:"id"`
	Label        string            `json:"label"`
	BaseURL      string            `json:"baseUrl"`
	APIKey       string            `json:"apiKey"`
	Weight       int               `json:"weight"`
	Enabled      bool              `json:"enabled"`
	Models       []string          `json:"models"`       // 本地标准模型支持范围
	ModelAliases map[string]string `json:"modelAliases"` // 本地标准模型 -> 上游实际模型
	CreatedAt    time.Time         `json:"createdAt"`
}

type Settings struct {
	AdminToken string `json:"adminToken,omitempty"`
	ProxyURL   string `json:"proxyUrl,omitempty"`
}

type Stats struct {
	TotalCalls     int       `json:"totalCalls"`
	TotalKeys      int       `json:"totalKeys"`
	LastCallAt     time.Time `json:"lastCallAt,omitempty"`
	FailedCalls    int       `json:"failedCalls"`
	LastFailedAt   time.Time `json:"lastFailedAt,omitempty"`
	KeyRoundsTotal int       `json:"keyRoundsTotal"`
}

type State struct {
	MiMoKeys     []MiMoKey         `json:"mimoKeys"`
	ExternalKeys []ExternalKey     `json:"externalKeys"`
	Channels     []UpstreamChannel `json:"channels"`
	Voices       []Voice           `json:"voices"`
	Stats        Stats             `json:"stats"`
	Settings     Settings          `json:"settings"`
}

type Store struct {
	mu   sync.RWMutex
	path string
	data State
}

func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "mimotts.json")}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		s.data = State{MiMoKeys: []MiMoKey{}, ExternalKeys: []ExternalKey{}, Channels: []UpstreamChannel{}, Voices: []Voice{}}
		return s.saveLocked()
	}
	if err != nil {
		return err
	}
	if len(b) == 0 {
		s.data = State{MiMoKeys: []MiMoKey{}, ExternalKeys: []ExternalKey{}, Channels: []UpstreamChannel{}, Voices: []Voice{}}
		return nil
	}
	if err := json.Unmarshal(b, &s.data); err != nil {
		return err
	}
	if s.data.Voices == nil {
		s.data.Voices = []Voice{}
	}
	if s.data.MiMoKeys == nil {
		s.data.MiMoKeys = []MiMoKey{}
	}
	if s.data.ExternalKeys == nil {
		s.data.ExternalKeys = []ExternalKey{}
	}
	if s.data.Channels == nil {
		s.data.Channels = []UpstreamChannel{}
	}
	for i := range s.data.Channels {
		if s.data.Channels[i].ModelAliases == nil {
			s.data.Channels[i].ModelAliases = map[string]string{}
		}
	}
	return nil
}

func (s *Store) saveLocked() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0600)
}

// ====================== 系统设置 ======================

func (s *Store) GetSettings() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Settings
}

func (s *Store) UpdateSettings(adminToken *string, proxyURL *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if adminToken != nil {
		next := strings.TrimSpace(*adminToken)
		if next != "" {
			s.data.Settings.AdminToken = next
		}
	}
	if proxyURL != nil {
		s.data.Settings.ProxyURL = strings.TrimSpace(*proxyURL)
	}
	return s.saveLocked()
}

// ====================== MiMo Key 管理 ======================

func (s *Store) Status() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	usable := 0
	for _, key := range s.data.MiMoKeys {
		if key.Enabled && strings.TrimSpace(key.Key) != "" {
			usable++
		}
	}
	return map[string]any{
		"mimoKeyCount":     len(s.data.MiMoKeys),
		"mimoUsableCount":  usable,
		"externalKeyCount": len(s.data.ExternalKeys),
		"voiceCount":       len(s.data.Voices),
		"channelCount":     len(s.data.Channels),
		"stats":            s.data.Stats,
	}
}

// EnabledMiMoKeys 返回按 (weight desc, createdAt asc) 排序后的可用 key。
// 同一调用链内，调用方使用 stable 顺序 + 起点轮询，天然带负载均衡。
func (s *Store) EnabledMiMoKeys() []MiMoKey {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]MiMoKey, 0, len(s.data.MiMoKeys))
	for _, key := range s.data.MiMoKeys {
		if key.Enabled && strings.TrimSpace(key.Key) != "" {
			w := key.Weight
			if w <= 0 {
				w = 1
			}
			key.Weight = w
			out = append(out, key)
		}
	}
	sortByWeightThenCreated(out)
	return out
}

func (s *Store) ListMiMoKeys() []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]map[string]any, 0, len(s.data.MiMoKeys))
	for _, key := range s.data.MiMoKeys {
		out = append(out, map[string]any{
			"id":        key.ID,
			"label":     key.Label,
			"masked":    maskSecret(key.Key),
			"weight":    weightOrOne(key.Weight),
			"enabled":   key.Enabled,
			"createdAt": key.CreatedAt,
		})
	}
	return out
}

func (s *Store) AddMiMoKey(label, secret string, weight int) (MiMoKey, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return MiMoKey{}, errors.New("key required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := MiMoKey{
		ID:        id("mk"),
		Label:     fallback(label, "MiMo Key"),
		Key:       secret,
		Weight:    weightOrOne(weight),
		Enabled:   true,
		CreatedAt: time.Now(),
	}
	s.data.MiMoKeys = append(s.data.MiMoKeys, key)
	return key, s.saveLocked()
}

func (s *Store) DeleteMiMoKey(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := s.data.MiMoKeys[:0]
	found := false
	for _, key := range s.data.MiMoKeys {
		if key.ID == id {
			found = true
			continue
		}
		keys = append(keys, key)
	}
	if !found {
		return errors.New("key not found")
	}
	s.data.MiMoKeys = keys
	return s.saveLocked()
}

func (s *Store) GetMiMoKeySecret(id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, key := range s.data.MiMoKeys {
		if key.ID == id {
			return key.Key, nil
		}
	}
	return "", errors.New("key not found")
}

func (s *Store) UpdateMiMoKey(id string, weight *int, enabled *bool, label *string, secret *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.MiMoKeys {
		if s.data.MiMoKeys[i].ID == id {
			if label != nil {
				s.data.MiMoKeys[i].Label = fallback(*label, "MiMo Key")
			}
			if secret != nil {
				next := strings.TrimSpace(*secret)
				if next == "" {
					return errors.New("key required")
				}
				s.data.MiMoKeys[i].Key = next
			}
			if weight != nil {
				s.data.MiMoKeys[i].Weight = weightOrOne(*weight)
			}
			if enabled != nil {
				s.data.MiMoKeys[i].Enabled = *enabled
			}
			return s.saveLocked()
		}
	}
	return errors.New("key not found")
}

// ====================== Voice 管理 ======================

func (s *Store) ListVoices() []Voice {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Voice, len(s.data.Voices))
	copy(out, s.data.Voices)
	return out
}

func (s *Store) GetVoice(id string) (Voice, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.data.Voices {
		if v.ID == id {
			return v, true
		}
	}
	return Voice{}, false
}

func (s *Store) AddPresetVoice(label, preset, language, description string) (Voice, error) {
	if _, ok := PresetVoices[preset]; !ok {
		return Voice{}, errors.New("不支持的预置音色")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v := Voice{
		ID:          id("vp"),
		Label:       fallback(label, preset),
		Kind:        "preset",
		Preset:      preset,
		Language:    language,
		Description: description,
		CreatedAt:   time.Now(),
	}
	s.data.Voices = append(s.data.Voices, v)
	return v, s.saveLocked()
}

func (s *Store) AddDesignVoice(label, design, language, description string) (Voice, error) {
	if strings.TrimSpace(design) == "" {
		return Voice{}, errors.New("音色描述不能为空")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v := Voice{
		ID:          id("vd"),
		Label:       fallback(label, "自定义音色"),
		Kind:        "design",
		Design:      design,
		Language:    language,
		Description: description,
		CreatedAt:   time.Now(),
	}
	s.data.Voices = append(s.data.Voices, v)
	return v, s.saveLocked()
}

func (s *Store) AddCloneVoice(label, dataURL, language, description string) (Voice, error) {
	if !strings.HasPrefix(dataURL, "data:audio/") {
		return Voice{}, errors.New("音色克隆需要 mp3/wav 音频")
	}
	if len(dataURL) > 14*1024*1024 {
		return Voice{}, errors.New("音色样本过大（>10MB）")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v := Voice{
		ID:           id("vc"),
		Label:        fallback(label, "克隆音色"),
		Kind:         "clone",
		CloneDataURL: dataURL,
		Language:     language,
		Description:  description,
		CreatedAt:    time.Now(),
	}
	s.data.Voices = append(s.data.Voices, v)
	return v, s.saveLocked()
}

func (s *Store) DeleteVoice(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	voices := s.data.Voices[:0]
	found := false
	for _, v := range s.data.Voices {
		if v.ID == id {
			found = true
			continue
		}
		voices = append(voices, v)
	}
	if !found {
		return errors.New("voice not found")
	}
	s.data.Voices = voices
	return s.saveLocked()
}

func (s *Store) MarkVoiceKeyUsed(voiceID, keyID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Voices {
		if s.data.Voices[i].ID == voiceID {
			for _, k := range s.data.Voices[i].AvailableKeys {
				if k == keyID {
					return
				}
			}
			s.data.Voices[i].AvailableKeys = append(s.data.Voices[i].AvailableKeys, keyID)
			_ = s.saveLocked()
			return
		}
	}
}

// ====================== 上游渠道管理 ======================

func (s *Store) ListChannels() []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]map[string]any, 0, len(s.data.Channels))
	for _, ch := range s.data.Channels {
		out = append(out, map[string]any{
			"id": ch.ID, "label": ch.Label, "baseUrl": ch.BaseURL,
			"masked": maskSecret(ch.APIKey), "weight": weightOrOne(ch.Weight),
			"enabled": ch.Enabled, "models": ch.Models, "modelAliases": ch.ModelAliases, "createdAt": ch.CreatedAt,
		})
	}
	return out
}

func (s *Store) EnabledChannels() []UpstreamChannel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]UpstreamChannel, 0, len(s.data.Channels))
	for _, ch := range s.data.Channels {
		if ch.Enabled && strings.TrimSpace(ch.BaseURL) != "" {
			ch.Weight = weightOrOne(ch.Weight)
			out = append(out, ch)
		}
	}
	return out
}

func (s *Store) AddChannel(label, baseURL, apiKey string, weight int, models []string, modelAliases map[string]string) (UpstreamChannel, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return UpstreamChannel{}, errors.New("base_url required")
	}
	if len(models) == 0 {
		models = []string{"mimo-v2.5-tts", "mimo-v2.5-tts-voicedesign"}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := UpstreamChannel{ID: id("up"), Label: fallback(label, "上游渠道"), BaseURL: baseURL, APIKey: strings.TrimSpace(apiKey), Weight: weightOrOne(weight), Enabled: true, Models: models, ModelAliases: cleanModelAliases(modelAliases), CreatedAt: time.Now()}
	s.data.Channels = append(s.data.Channels, ch)
	return ch, s.saveLocked()
}

func (s *Store) DeleteChannel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	chs := s.data.Channels[:0]
	found := false
	for _, ch := range s.data.Channels {
		if ch.ID == id {
			found = true
			continue
		}
		chs = append(chs, ch)
	}
	if !found {
		return errors.New("channel not found")
	}
	s.data.Channels = chs
	return s.saveLocked()
}

func (s *Store) GetChannelConfig(id string) (UpstreamChannel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ch := range s.data.Channels {
		if ch.ID == id {
			return ch, nil
		}
	}
	return UpstreamChannel{}, errors.New("channel not found")
}

func (s *Store) GetChannelSecret(id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ch := range s.data.Channels {
		if ch.ID == id {
			return ch.APIKey, nil
		}
	}
	return "", errors.New("channel not found")
}

func (s *Store) UpdateChannel(id string, weight *int, enabled *bool, label *string, baseURL *string, apiKey *string, models []string, modelAliases map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Channels {
		if s.data.Channels[i].ID == id {
			if weight != nil {
				s.data.Channels[i].Weight = weightOrOne(*weight)
			}
			if enabled != nil {
				s.data.Channels[i].Enabled = *enabled
			}
			if label != nil && strings.TrimSpace(*label) != "" {
				s.data.Channels[i].Label = strings.TrimSpace(*label)
			}
			if baseURL != nil {
				newURL := strings.TrimRight(strings.TrimSpace(*baseURL), "/")
				if newURL != "" {
					s.data.Channels[i].BaseURL = newURL
				}
			}
			if apiKey != nil {
				// Non-empty replacement; empty string means "keep existing"
				if strings.TrimSpace(*apiKey) != "" {
					s.data.Channels[i].APIKey = strings.TrimSpace(*apiKey)
				}
			}
			if models != nil {
				s.data.Channels[i].Models = models
			}
			if modelAliases != nil {
				s.data.Channels[i].ModelAliases = cleanModelAliases(modelAliases)
			}
			return s.saveLocked()
		}
	}
	return errors.New("channel not found")
}

// ====================== 外部 Key ======================

func (s *Store) ListExternalKeys() []ExternalKey {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ExternalKey, len(s.data.ExternalKeys))
	copy(out, s.data.ExternalKeys)
	return out
}

func (s *Store) AddExternalKey(label string, permissions []string) (ExternalKey, string, error) {
	secret := "mtts_" + randomHex(24)
	now := time.Now()
	ext := ExternalKey{ID: id("ek"), Label: fallback(label, "外部应用"), Hash: hash(secret), Prefix: secret[:10], Masked: maskSecret(secret), Secret: secret, Permissions: normalizePermissions(permissions), Enabled: true, CreatedAt: now, UpdatedAt: now}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.ExternalKeys = append(s.data.ExternalKeys, ext)
	return ext, secret, s.saveLocked()
}

func (s *Store) DeleteExternalKey(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := s.data.ExternalKeys[:0]
	found := false
	for _, key := range s.data.ExternalKeys {
		if key.ID == id {
			found = true
			continue
		}
		keys = append(keys, key)
	}
	if !found {
		return errors.New("key not found")
	}
	s.data.ExternalKeys = keys
	return s.saveLocked()
}

func (s *Store) UpdateExternalKey(id string, enabled *bool, permissions []string, label *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.ExternalKeys {
		if s.data.ExternalKeys[i].ID == id {
			if enabled != nil {
				s.data.ExternalKeys[i].Enabled = *enabled
			}
			if permissions != nil {
				s.data.ExternalKeys[i].Permissions = normalizePermissions(permissions)
			}
			if label != nil {
				s.data.ExternalKeys[i].Label = fallback(*label, "外部应用")
			}
			s.data.ExternalKeys[i].UpdatedAt = time.Now()
			return s.saveLocked()
		}
	}
	return errors.New("key not found")
}

func (s *Store) RotateExternalKey(id string) (string, error) {
	secret := "mtts_" + randomHex(24)
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.ExternalKeys {
		if s.data.ExternalKeys[i].ID == id {
			s.data.ExternalKeys[i].Hash = hash(secret)
			s.data.ExternalKeys[i].Prefix = secret[:10]
			s.data.ExternalKeys[i].Masked = maskSecret(secret)
			s.data.ExternalKeys[i].Secret = secret
			s.data.ExternalKeys[i].UpdatedAt = time.Now()
			return secret, s.saveLocked()
		}
	}
	return "", errors.New("key not found")
}

func (s *Store) ValidateExternalKey(secret, permission string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	secretHash := hash(secret)
	for i := range s.data.ExternalKeys {
		key := &s.data.ExternalKeys[i]
		if key.Enabled && (key.Hash == secretHash || (key.Secret != "" && key.Secret == secret)) && hasPermission(key.Permissions, permission) {
			now := time.Now()
			key.LastUsedAt = &now
			s.data.Stats.TotalCalls++
			s.data.Stats.LastCallAt = now
			_ = s.saveLocked()
			return true
		}
	}
	return false
}

// ====================== 统计 ======================

func (s *Store) RecordSuccess() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Stats.TotalCalls++
	s.data.Stats.LastCallAt = time.Now()
	_ = s.saveLocked()
}

func (s *Store) RecordFailure() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Stats.FailedCalls++
	s.data.Stats.LastFailedAt = time.Now()
	_ = s.saveLocked()
}

func (s *Store) RecordKeyRound(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Stats.KeyRoundsTotal += n
	_ = s.saveLocked()
}

// ====================== helpers ======================

func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return "****"
	}
	return secret[:min(4, len(secret))] + "****" + secret[len(secret)-4:]
}

func hash(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func id(prefix string) string {
	return fmt.Sprintf("%s_%d_%s", prefix, time.Now().UnixNano(), randomHex(4))
}
func fallback(v, fb string) string {
	if strings.TrimSpace(v) == "" {
		return fb
	}
	return strings.TrimSpace(v)
}
func cleanModelAliases(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k != "" && v != "" {
			out[k] = v
		}
	}
	return out
}

func normalizePermissions(p []string) []string {
	if len(p) == 0 {
		return []string{"tts"}
	}
	return p
}
func hasPermission(list []string, p string) bool {
	for _, item := range list {
		if item == p || item == "*" {
			return true
		}
	}
	return false
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func weightOrOne(w int) int {
	if w <= 0 {
		return 1
	}
	return w
}
func sortByWeightThenCreated(keys []MiMoKey) {
	// 简单插入排序，key 数量通常 < 100，开销可忽略
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && (keys[j].Weight > keys[j-1].Weight || (keys[j].Weight == keys[j-1].Weight && keys[j].CreatedAt.Before(keys[j-1].CreatedAt))); j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
}
