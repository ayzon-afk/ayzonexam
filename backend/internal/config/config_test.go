package config

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	secret := strings.Repeat("s", 32)
	cases := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{"нет базы", Config{BotMode: BotOff, BotToken: "t"}, "DATABASE_URL"},
		{"неизвестный режим", Config{DatabaseURL: "x", BotMode: "push", BotToken: "t"}, "BOT_MODE"},
		{"нет токена без DEV_AUTH", Config{DatabaseURL: "x", BotMode: BotOff}, "BOT_TOKEN"},
		{"DEV_AUTH без токена с ботом", Config{DatabaseURL: "x", BotMode: BotPolling, DevAuth: true}, "BOT_TOKEN"},
		{"бот без https", Config{DatabaseURL: "x", BotMode: BotPolling, BotToken: "t", WebAppURL: "http://a"}, "WEBAPP_URL"},
		{"webhook без адреса", Config{DatabaseURL: "x", BotMode: BotWebhook, BotToken: "t", WebAppURL: "https://a", WebhookSecret: secret}, "PUBLIC_URL"},
		{"webhook с коротким секретом", Config{DatabaseURL: "x", BotMode: BotWebhook, BotToken: "t", WebAppURL: "https://a", PublicURL: "https://a", WebhookSecret: "s"}, "WEBHOOK_SECRET"},
		{"webhook с DEV_AUTH", Config{DatabaseURL: "x", BotMode: BotWebhook, BotToken: "t", WebAppURL: "https://a", PublicURL: "https://a", WebhookSecret: secret, DevAuth: true}, "DEV_AUTH"},
		{"webhook без ключа Gemini", Config{DatabaseURL: "x", BotMode: BotWebhook, BotToken: "t", WebAppURL: "https://a", PublicURL: "https://a", WebhookSecret: secret}, "GEMINI_API_KEY"},
		{"боевой webhook", Config{DatabaseURL: "x", BotMode: BotWebhook, BotToken: "t", WebAppURL: "https://a", PublicURL: "https://a", WebhookSecret: secret, GeminiAPIKey: "k"}, ""},
		{"локально без бота", Config{DatabaseURL: "x", BotMode: BotOff, DevAuth: true}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.cfg.validate()
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("ждали без ошибки, получили %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("ждали ошибку про %s, получили %v", c.wantErr, err)
			}
		})
	}
}

func TestDevTokenOnlyLocally(t *testing.T) {
	c := Config{DatabaseURL: "x", BotMode: BotOff, DevAuth: true}
	if err := c.validate(); err != nil {
		t.Fatal(err)
	}
	if c.BotToken != devBotToken {
		t.Fatalf("без токена в DEV_AUTH ждали тестовый токен, получили %q", c.BotToken)
	}
}

func TestWritingSettings(t *testing.T) {
	t.Setenv("DATABASE_URL", "x")
	t.Setenv("DEV_AUTH", "1")
	t.Setenv("BOT_MODE", "off")
	c, err := FromEnv()
	if err != nil || c.WritingDailyLimit != 3 || c.WritingWorkers != 3 || c.GeminiModel != DefaultGeminiModel {
		t.Fatalf("значения по умолчанию: %+v %v", c, err)
	}
	t.Setenv("WRITING_DAILY_LIMIT", "0")
	if _, err := FromEnv(); err == nil || !strings.Contains(err.Error(), "WRITING_DAILY_LIMIT") {
		t.Fatalf("лимит 0 должен быть ошибкой: %v", err)
	}
	t.Setenv("WRITING_DAILY_LIMIT", "5")
	t.Setenv("WRITING_WORKERS", "много")
	if _, err := FromEnv(); err == nil || !strings.Contains(err.Error(), "WRITING_WORKERS") {
		t.Fatalf("не число должно быть ошибкой: %v", err)
	}
}

func TestAdminsAndPayment(t *testing.T) {
	t.Setenv("DATABASE_URL", "x")
	t.Setenv("DEV_AUTH", "1")
	t.Setenv("BOT_MODE", "off")
	t.Setenv("ADMIN_IDS", " 111, 222 ")
	t.Setenv("PAY_CARD", " 8600 0000 ")
	c, err := FromEnv()
	if err != nil || len(c.Admins) != 2 || c.Admins[1] != 222 || c.PayCard != "8600 0000" || c.GeminiTTSModel != DefaultTTSModel {
		t.Fatalf("админы и карта: %+v %v", c, err)
	}
	t.Setenv("ADMIN_IDS", "111,@owner")
	if _, err := FromEnv(); err == nil || !strings.Contains(err.Error(), "ADMIN_IDS") {
		t.Fatalf("ник вместо id — ошибка: %v", err)
	}
}
