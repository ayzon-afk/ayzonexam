// Сервер AYZON EXAM: API Mini App, бот и (если задан WEB_DIR) собранный интерфейс — в одном процессе.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ayzonexam/internal/ai"
	"ayzonexam/internal/api"
	"ayzonexam/internal/bot"
	"ayzonexam/internal/config"
	"ayzonexam/internal/db"
	"ayzonexam/internal/users"
	"ayzonexam/internal/writing"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("сервер остановлен", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	if err := config.LoadDotEnv(".env"); err != nil {
		return err
	}
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}

	var checker ai.Checker = ai.NewGemini(cfg.GeminiAPIKey, cfg.GeminiModel)
	if cfg.GeminiAPIKey == "" {
		checker = ai.Fake{}
		log.Warn("GEMINI_API_KEY не задан: проверки Writing идут заглушкой")
	}
	store := writing.NewStore(pool)
	deps := api.Deps{
		Users:             users.NewStore(pool),
		Writing:           store,
		AI:                checker,
		WritingDailyLimit: cfg.WritingDailyLimit,
		BotToken:          cfg.BotToken,
		DevAuth:           cfg.DevAuth,
		WebDir:            cfg.WebDir,
		Log:               log,
	}
	queue := &writing.Queue{Store: store, AI: checker, Log: log, Workers: cfg.WritingWorkers, Now: time.Now}
	if cfg.BotMode != config.BotOff {
		b := bot.New(cfg.BotToken, cfg.WebAppURL, log)
		queue.Notify = b
		webhook := ""
		if cfg.BotMode == config.BotWebhook {
			webhook = cfg.PublicURL + "/tg/webhook"
			deps.Webhook = b.WebhookHandler(cfg.WebhookSecret)
		}
		// Telegram недоступен — приложение всё равно работает; бот поднимется со следующим запуском.
		if err := b.Setup(ctx, webhook, cfg.WebhookSecret); err != nil {
			log.Warn("бот не настроен", "err", err)
		}
		if cfg.BotMode == config.BotPolling {
			go b.RunPolling(ctx)
		}
	}
	if cfg.DevAuth {
		log.Warn("DEV_AUTH=1: вход без Telegram открыт — только для локальной разработки")
	}

	queueDone := make(chan struct{})
	go func() { queue.Run(ctx); close(queueDone) }()

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.New(deps),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Info("сервер запущен", "addr", cfg.Addr, "bot", cfg.BotMode, "ai", checker.Model())

	select {
	case err := <-errc:
		stop()
		<-queueDone
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = srv.Shutdown(shutdownCtx)
	// Воркеры возвращают незаконченные проверки в очередь — дождаться, пока база ещё открыта.
	<-queueDone
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
