// Package config читает настройки сервера из переменных окружения.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Режимы бота.
const (
	BotOff     = "off"     // бот не запускается (тесты, локально без токена)
	BotPolling = "polling" // long polling — локально, без публичного адреса
	BotWebhook = "webhook" // webhook — на сервере
)

// DefaultGeminiModel — модель проверки Writing, если GEMINI_MODEL не задан. Смена модели — только после
// прогона калибровки (cmd/calibrate).
const DefaultGeminiModel = "gemini-3.5-flash"

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

	GeminiAPIKey      string // пусто — проверки заглушкой (ai.Fake), только не на боевом сервере
	GeminiModel       string
	GeminiTTSModel    string // синтез речи для Listening
	WritingDailyLimit int    // проверок Writing в день с Premium; без него — 2 в неделю (billing)
	WritingWorkers    int    // сколько проверок Writing идёт одновременно
	SpeakingWorkers   int    // сколько проверок Speaking идёт одновременно

	Admins      []int64 // ADMIN_IDS: Telegram id владельца и помощников через запятую — вход в админку
	PayCard     string  // PAY_CARD: номер карты для переводов; пусто — оплата недоступна
	PayCardName string  // PAY_CARD_NAME: получатель
}

// DefaultTTSModel — модель озвучки сценариев Listening, если GEMINI_TTS_MODEL не задан.
const DefaultTTSModel = "gemini-2.5-flash-preview-tts"

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
		GeminiAPIKey:  os.Getenv("GEMINI_API_KEY"),
		GeminiModel:   env("GEMINI_MODEL", DefaultGeminiModel),
		PayCard:       strings.TrimSpace(os.Getenv("PAY_CARD")),
		PayCardName:   strings.TrimSpace(os.Getenv("PAY_CARD_NAME")),
	}
	c.GeminiTTSModel = env("GEMINI_TTS_MODEL", DefaultTTSModel)
	for _, f := range strings.Split(os.Getenv("ADMIN_IDS"), ",") {
		if f = strings.TrimSpace(f); f == "" {
			continue
		}
		id, err := strconv.ParseInt(f, 10, 64)
		if err != nil || id <= 0 {
			return c, fmt.Errorf("ADMIN_IDS=%q: нужны Telegram id через запятую", os.Getenv("ADMIN_IDS"))
		}
		c.Admins = append(c.Admins, id)
	}
	var err error
	if c.WritingDailyLimit, err = envInt("WRITING_DAILY_LIMIT", 3, 1, 100); err != nil {
		return c, err
	}
	if c.WritingWorkers, err = envInt("WRITING_WORKERS", 3, 1, 20); err != nil {
		return c, err
	}
	if c.SpeakingWorkers, err = envInt("SPEAKING_WORKERS", 2, 1, 20); err != nil {
		return c, err
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
		if c.GeminiAPIKey == "" {
			return errors.New("для BOT_MODE=webhook нужен GEMINI_API_KEY: без него проверки шли бы заглушкой")
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

func envInt(key string, def, min, max int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < min || n > max {
		return 0, fmt.Errorf("%s=%q: нужно целое от %d до %d", key, v, min, max)
	}
	return n, nil
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
