package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"mimotts/backend/store"
)

var presetVoices = store.PresetVoices
var rrCounter uint64

type TTSRequest struct {
	Text          string `json:"text"`
	Model         string `json:"model"`
	Voice         string `json:"voice"`
	Context       string `json:"context"`
	VoiceDataURL  string `json:"voiceDataUrl"`
}

type mimoResponse struct {
	Choices []struct {
		Message struct {
			Audio *struct {
				Data string `json:"data"`
			} `json:"audio"`
		} `json:"message"`
	} `json:"choices"`
	Error any `json:"error,omitempty"`
}

type selectedKey struct {
	ID      string
	Label   string
	BaseURL string
	Secret  string
}

func synthesizeWithPool(st *store.Store, req TTSRequest) ([]byte, selectedKey, TTSRequest, store.Voice, error) {
	keys := st.EnabledMiMoKeys()
	if len(keys) == 0 {
		return nil, selectedKey{}, req, store.Voice{}, errors.New("MiMo API Key 未配置/全部禁用")
	}
	voice, err := resolveProjectVoice(st, &req)
	if err != nil {
		return nil, selectedKey{}, req, voice, err
	}
	ordered := orderUpstreams(keys, voice)
	var lastErr error
	for _, key := range ordered {
		chosen := key
		settings := st.GetSettings()
		audio, err := callMiMo(chosen.BaseURL, chosen.Secret, req, settings.ProxyURL)
		if err == nil {
			if voice.ID != "" {
				st.MarkVoiceKeyUsed(voice.ID, chosen.ID)
			}
			st.RecordKeyRound(1)
			return audio, chosen, req, voice, nil
		}
		lastErr = err
	}
	st.RecordKeyRound(len(ordered))
	return nil, selectedKey{}, req, voice, fmt.Errorf("全部 MiMo Key 调用失败，最后错误：%w", lastErr)
}

func resolveProjectVoice(st *store.Store, req *TTSRequest) (store.Voice, error) {
	if strings.TrimSpace(req.Text) == "" {
		return store.Voice{}, errors.New("合成文本不能为空")
	}
	if req.Model == "" {
		req.Model = "mimo-v2.5-tts"
	}
	if req.Voice == "" {
		return store.Voice{}, nil
	}
	if _, ok := presetVoices[req.Voice]; ok {
		return store.Voice{Kind: "preset", Preset: req.Voice}, nil
	}
	v, ok := st.GetVoice(req.Voice)
	if !ok {
		return store.Voice{}, fmt.Errorf("项目音色不存在：%s", req.Voice)
	}
	switch v.Kind {
	case "preset":
		req.Model = "mimo-v2.5-tts"
		req.Voice = v.Preset
	case "design":
		req.Model = "mimo-v2.5-tts-voicedesign"
		req.Voice = ""
		if strings.TrimSpace(req.Context) == "" {
			req.Context = v.Design
		} else {
			req.Context = v.Design + "\n\n本次风格控制：" + req.Context
		}
	case "clone":
		req.Model = "mimo-v2.5-tts-voiceclone"
		req.Voice = ""
		req.VoiceDataURL = v.CloneDataURL
	default:
		return store.Voice{}, errors.New("未知音色类型")
	}
	return v, nil
}

func orderUpstreams(keys []store.MiMoKey, voice store.Voice) []selectedKey {
	expanded := make([]selectedKey, 0, len(keys))
	for _, key := range keys {
		weight := key.Weight
		if weight <= 0 {
			weight = 1
		}
		for i := 0; i < weight; i++ {
			expanded = append(expanded, selectedKey{ID: key.ID, Label: key.Label, BaseURL: "https://api.xiaomimimo.com/v1", Secret: key.Key})
		}
	}
	if len(expanded) == 0 {
		return []selectedKey{}
	}
	// Filter by availableKeys if voice has them, then round-robin.
	var filtered []selectedKey
	if voice.ID != "" && len(voice.AvailableKeys) > 0 {
		for _, k := range expanded {
			if contains(voice.AvailableKeys, k.ID) {
				filtered = append(filtered, k)
			}
		}
		if len(filtered) == 0 {
			filtered = expanded
		}
	} else {
		filtered = expanded
	}
	return makeRoundRobinList(filtered)
}

func makeRoundRobinList(expanded []selectedKey) []selectedKey {
	if len(expanded) <= 1 {
		return expanded
	}
	start := int(atomic.AddUint64(&rrCounter, 1) % uint64(len(expanded)))
	seen := map[string]bool{}
	out := make([]selectedKey, 0, len(expanded))
	for i := 0; i < len(expanded); i++ {
		k := expanded[(start+i)%len(expanded)]
		if seen[k.ID] {
			continue
		}
		seen[k.ID] = true
		out = append(out, k)
	}
	return out
}



func containerReachableBaseURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	baseURL = strings.Replace(baseURL, "http://127.0.0.1:", "http://host.docker.internal:", 1)
	baseURL = strings.Replace(baseURL, "http://localhost:", "http://host.docker.internal:", 1)
	return baseURL
}

func callMiMo(baseURL, apiKey string, req TTSRequest, proxyURL string) ([]byte, error) {
	baseURL = containerReachableBaseURL(baseURL)
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.xiaomimimo.com/v1"
	}
	if strings.TrimSpace(req.Text) == "" {
		return nil, errors.New("合成文本不能为空")
	}
	if req.Model == "" {
		req.Model = "mimo-v2.5-tts"
	}
	messages := []map[string]any{}
	if strings.TrimSpace(req.Context) != "" {
		messages = append(messages, map[string]any{"role": "user", "content": req.Context})
	}
	messages = append(messages, map[string]any{"role": "assistant", "content": req.Text})
	audio := map[string]string{"format": "wav"}
	switch req.Model {
	case "mimo-v2.5-tts":
		if _, ok := presetVoices[req.Voice]; !ok {
			return nil, errors.New("请选择有效的预置音色或项目音色 ID")
		}
		audio["voice"] = req.Voice
	case "mimo-v2.5-tts-voicedesign":
		if strings.TrimSpace(req.Context) == "" {
			return nil, errors.New("音色设计模式需要填写音色描述")
		}
	case "mimo-v2.5-tts-voiceclone":
		if !strings.HasPrefix(req.VoiceDataURL, "data:audio/") {
			return nil, errors.New("音色克隆模式需要上传 mp3/wav 音频")
		}
		audio["voice"] = req.VoiceDataURL
	default:
		return nil, errors.New("不支持的模型")
	}
	payload := map[string]any{"model": req.Model, "messages": messages, "audio": audio}
	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(apiKey) != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("MiMo API 返回 %d: %s", resp.StatusCode, string(respBody))
	}
	var parsed mimoResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Choices) == 0 || parsed.Choices[0].Message.Audio == nil || parsed.Choices[0].Message.Audio.Data == "" {
		return nil, errors.New("MiMo API 未返回音频数据")
	}
	return base64.StdEncoding.DecodeString(parsed.Choices[0].Message.Audio.Data)
}

// randomHex generates n random bytes and returns them as hex string.
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
