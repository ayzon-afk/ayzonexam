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
	"ayzonexam/internal/speaking"
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

	gemini := ai.NewGemini(cfg.GeminiAPIKey, cfg.GeminiModel)
	gemini.TTSModel = cfg.GeminiTTSModel
	if cfg.GeminiDailyLimit > 0 {
		gemini.Take = (&ai.Quota{Pool: pool, Limit: cfg.GeminiDailyLimit, Now: time.Now}).Take
	}
	var checker ai.Checker = gemini
	if cfg.GeminiAPIKey == "" {
		checker = ai.Fake{}
		log.Warn("GEMINI_API_KEY не задан: проверки Writing и Speaking идут заглушкой")
	}
	store := writing.NewStore(pool)
	deps := api.Deps{
		DB:                pool,
		Users:             users.NewStore(pool),
		Writing:           store,
		AI:                checker,
		WritingDailyLimit: cfg.WritingDailyLimit,
		BotToken:          cfg.BotToken,
		DevAuth:           cfg.DevAuth,
		WebDir:            cfg.WebDir,
		Admins:            cfg.Admins,
		PayCard:           cfg.PayCard,
		PayCardName:       cfg.PayCardName,
		Log:               log,
	}
	queue := &writing.Queue{Store: store, AI: checker, Log: log, Workers: cfg.WritingWorkers, Now: time.Now}
	var b *bot.Bot
	if cfg.BotMode != config.BotOff {
		b = bot.New(cfg.BotToken, cfg.WebAppURL, log)
		b.Admins = cfg.Admins
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
		deps.Msg = b
		deps.BotUsername = b.Username()
	}
	if cfg.DevAuth {
		log.Warn("DEV_AUTH=1: вход без Telegram открыт — только для локальной разработки")
	}
	if len(cfg.Admins) == 0 {
		log.Warn("ADMIN_IDS не задан: админка закрыта для всех")
	}

	app := api.New(deps)
	// Проверка из mock готова — mock пересчитывается и, если все части сданы, закрывается.
	queue.OnDone = func(ctx context.Context, id int64) { app.AfterCheck(ctx, "writing_checks", id) }
	speak := &speaking.Queue{Store: app.Speaking(), AI: checker, Log: log, Workers: cfg.SpeakingWorkers, Now: time.Now,
		OnDone: func(ctx context.Context, id int64) { app.AfterCheck(ctx, "speaking_checks", id) }}
	if b != nil {
		speak.Notify = b
		b.OnUpdate = app.HandleUpdate
		if cfg.BotMode == config.BotPolling {
			go b.RunPolling(ctx)
		}
	}

	queueDone := make(chan struct{})
	go func() { queue.Run(ctx); close(queueDone) }()
	speakDone := make(chan struct{})
	go func() { speak.Run(ctx); close(speakDone) }()

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           app,
		ReadHeaderTimeout: 10 * time.Second,
		// Записи Speaking mock — до трёх файлов по 3 МБ с мобильного интернета.
		ReadTimeout: 2 * time.Minute,
		IdleTimeout: 120 * time.Second,
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Info("сервер запущен", "addr", cfg.Addr, "bot", cfg.BotMode, "ai", checker.Model(), "gemini_day_limit", cfg.GeminiDailyLimit)

	select {
	case err := <-errc:
		stop()
		<-queueDone
		<-speakDone
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = srv.Shutdown(shutdownCtx)
	// Воркеры возвращают незаконченные проверки в очередь — дождаться, пока база ещё открыта.
	<-queueDone
	<-speakDone
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
