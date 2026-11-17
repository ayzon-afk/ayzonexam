package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"strings"
)

// Синтез речи для Listening: сценарий «Имя: реплика» по строкам. Два говорящих — два голоса (диалог),
// иначе один. Gemini отдаёт сырой PCM 16 бит 24 кГц моно — оборачиваем в WAV. WAV тяжёлый (~2.8 МБ в
// минуту), поэтому админка пережимает его в браузере перед сохранением (DECISIONS).

const ttsRate = 24000

var voices = []string{"Kore", "Puck"}

// speakers — имена говорящих по порядку появления.
func speakers(script string) []string {
	var names []string
	seen := map[string]bool{}
	for _, line := range strings.Split(script, "\n") {
		name, _, ok := strings.Cut(line, ":")
		name = strings.TrimSpace(name)
		if !ok || name == "" || len(name) > 30 || strings.Contains(name, " ") && len(strings.Fields(name)) > 3 {
			continue
		}
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	return names
}

func (g *Gemini) Speak(ctx context.Context, script string) ([]byte, error) {
	script = strings.TrimSpace(script)
	if script == "" {
		return nil, errors.New("пустой сценарий")
	}
	speech := map[string]any{"voiceConfig": map[string]any{"prebuiltVoiceConfig": map[string]string{"voiceName": voices[0]}}}
	prompt := "Read this aloud clearly at a natural pace for an English listening exam:\n\n" + script
	if names := speakers(script); len(names) == 2 {
		var cfg []map[string]any
		for i, n := range names {
			cfg = append(cfg, map[string]any{"speaker": n, "voiceConfig": map[string]any{"prebuiltVoiceConfig": map[string]string{"voiceName": voices[i]}}})
		}
		speech = map[string]any{"multiSpeakerVoiceConfig": map[string]any{"speakerVoiceConfigs": cfg}}
		prompt = "TTS the following conversation between " + names[0] + " and " + names[1] + " for an English listening exam, at a natural pace:\n\n" + script
	}
	c, err := g.call(ctx, g.TTSModel, generateRequest{
		Contents: []content{{Role: "user", Parts: []part{{Text: prompt}}}},
		GenerationConfig: map[string]any{
			"responseModalities": []string{"AUDIO"},
			"speechConfig":       speech,
		},
	})
	if err != nil {
		return nil, err
	}
	var pcm []byte
	for _, p := range c.Parts {
		if p.InlineData == nil {
			continue
		}
		d, err := base64.StdEncoding.DecodeString(p.InlineData.Data)
		if err != nil {
			return nil, errors.New("gemini: аудио не base64")
		}
		pcm = append(pcm, d...)
	}
	if len(pcm) == 0 {
		return nil, errors.New("gemini: в ответе нет аудио")
	}
	return WAV(pcm, ttsRate), nil
}

// WAV оборачивает PCM 16 бит моно в заголовок WAV.
func WAV(pcm []byte, rate int) []byte {
	var b bytes.Buffer
	w := func(v any) { binary.Write(&b, binary.LittleEndian, v) }
	b.WriteString("RIFF")
	w(uint32(36 + len(pcm)))
	b.WriteString("WAVEfmt ")
	w(uint32(16))       // размер блока fmt
	w(uint16(1))        // PCM
	w(uint16(1))        // моно
	w(uint32(rate))     // частота
	w(uint32(rate * 2)) // байт в секунду
	w(uint16(2))        // байт на отсчёт
	w(uint16(16))       // бит на отсчёт
	b.WriteString("data")
	w(uint32(len(pcm)))
	b.Write(pcm)
	return b.Bytes()
}

// Fake — тишина нужной длины: секунда на строку сценария.
func (Fake) Speak(_ context.Context, script string) ([]byte, error) {
	lines := len(strings.Split(strings.TrimSpace(script), "\n"))
	return WAV(make([]byte, ttsRate*2*lines), ttsRate), nil
}
