// Package config читает настройки сервера из переменных окружения.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Режимы бота.
const (
	BotOff     = "off"     // бот не запускается (тесты, локально без токена)
	BotPolling = "polling" // long polling — локально, без публичного адреса
	BotWebhook = "webhook" // webhook — на сервере
)

// devBotToken подписывает initData в режиме DEV_AUTH без настоящего бота.
const devBotToken = "dev-token"

type Config struct {
	Addr          string // адрес HTTP-сервера, по умолчанию :8080
	DatabaseURL   string
	BotToken      string // токен бота: им же проверяется подпись initData
	BotMode       string
	WebAppURL     string // HTTPS-адрес Mini App для кнопки бота
	PublicURL     string // HTTPS-адрес сервера для webhook
	WebhookSecret string
	WebDir        string // папка собранного интерфейса; пусто — сервер отдаёт только API
	DevAuth       bool   // выдаёт тестовый initData по /api/dev/init-data — только локально
}

// FromEnv собирает и проверяет настройки. Ошибка называет переменную, которую надо поправить.
func FromEnv() (Config, error) {
	c := Config{
		Addr:          env("ADDR", ":8080"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		BotToken:      os.Getenv("BOT_TOKEN"),
		BotMode:       env("BOT_MODE", BotOff),
		WebAppURL:     os.Getenv("WEBAPP_URL"),
		PublicURL:     strings.TrimRight(os.Getenv("PUBLIC_URL"), "/"),
		WebhookSecret: os.Getenv("WEBHOOK_SECRET"),
		WebDir:        os.Getenv("WEB_DIR"),
		DevAuth:       os.Getenv("DEV_AUTH") == "1",
	}
	return c, c.validate()
}

func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL не задан")
	}
	switch c.BotMode {
	case BotOff, BotPolling, BotWebhook:
	default:
		return fmt.Errorf("BOT_MODE=%q: допустимо off, polling или webhook", c.BotMode)
	}
	if c.BotToken == "" {
		if !c.DevAuth || c.BotMode != BotOff {
			return errors.New("BOT_TOKEN не задан (без него можно только DEV_AUTH=1 и BOT_MODE=off)")
		}
		c.BotToken = devBotToken
	}
	if c.BotMode != BotOff && !strings.HasPrefix(c.WebAppURL, "https://") {
		return errors.New("WEBAPP_URL должен начинаться с https:// — Telegram открывает Mini App только по HTTPS")
	}
	if c.BotMode == BotWebhook {
		if !strings.HasPrefix(c.PublicURL, "https://") {
			return errors.New("для BOT_MODE=webhook нужен PUBLIC_URL с https://")
		}
		if len(c.WebhookSecret) < 32 {
			return errors.New("для BOT_MODE=webhook нужен WEBHOOK_SECRET не короче 32 символов")
		}
		if c.DevAuth {
			return errors.New("DEV_AUTH=1 нельзя вместе с BOT_MODE=webhook: это вход без Telegram на боевом сервере")
		}
	}
	return nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// LoadDotEnv читает KEY=VALUE из файла, не перезаписывая уже заданные переменные.
// Нет файла — не ошибка: на сервере настройки приходят из окружения.
func LoadDotEnv(path string) error {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if _, set := os.LookupEnv(k); !set {
			os.Setenv(k, v)
		}
	}
	return sc.Err()
}
