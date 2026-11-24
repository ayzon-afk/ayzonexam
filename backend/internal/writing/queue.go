package writing

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"ayzonexam/internal/ai"
)

// Notifier сообщает ученику, что проверка готова (band != nil) или не удалась (band == nil).
type Notifier interface {
	WritingChecked(ctx context.Context, telegramID int64, lang string, checkID int64, band *float64) error
}

const (
	maxAttempts = 3
	// Аренда задания: дольше самого долгого вызова AI (callTimeout). Сервер упал посреди проверки —
	// после аренды задание возьмёт другой воркер.
	lease       = 2 * time.Minute
	callTimeout = 90 * time.Second
	pollEvery   = 5 * time.Second
)

// retryDelay — пауза перед попыткой attempt+1.
func retryDelay(attempt int) time.Duration {
	if attempt <= 1 {
		return 30 * time.Second
	}
	return 2 * time.Minute
}

// Queue — воркеры AI-проверок. Задания — строки writing_checks (DECISIONS 2026-11-16: своя таблица, без River).
type Queue struct {
	Store  *Store
	AI     ai.Checker
	Notify Notifier // nil — бот выключен
	// OnDone — проверка закрыта (готова или не удалась): Writing из mock пересчитывает mock.
	OnDone  func(ctx context.Context, checkID int64)
	Log     *slog.Logger
	Workers int
	Now     func() time.Time
}

// Run держит воркеров до отмены ctx и возвращается, когда все остановились.
func (q *Queue) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for range max(q.Workers, 1) {
		wg.Go(func() {
			for ctx.Err() == nil {
				did, err := q.Step(ctx)
				if err != nil && ctx.Err() == nil {
					q.Log.Error("очередь Writing", "err", err)
				}
				if did && err == nil {
					continue
				}
				select {
				case <-ctx.Done():
				case <-q.Store.wake:
				case <-time.After(pollEvery):
				}
			}
		})
	}
	wg.Wait()
}

type job struct {
	id, telegramID int64
	task, prompt   string
	essay, lang    string
	words          int
	attempts       int
}

// Step берёт одно задание и доводит его до результата, повтора или отказа. false — заданий нет.
func (q *Queue) Step(ctx context.Context) (bool, error) {
	now := q.Now()
	var j job
	err := q.Store.pool.QueryRow(ctx, `UPDATE writing_checks c
		SET status = 'processing', attempts = c.attempts + 1, locked_until = $2
		FROM users u
		WHERE u.id = c.user_id AND c.id = (
			SELECT id FROM writing_checks
			WHERE (status = 'queued' AND run_after <= $1) OR (status = 'processing' AND locked_until < $1)
			ORDER BY run_after LIMIT 1
			FOR UPDATE SKIP LOCKED)
		RETURNING c.id, u.telegram_id, c.task, c.prompt, c.essay, c.lang, c.words, c.attempts`,
		now, now.Add(lease)).Scan(&j.id, &j.telegramID, &j.task, &j.prompt, &j.essay, &j.lang, &j.words, &j.attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// Попытки кончились, пока воркеры падали посреди проверки.
	if j.attempts > maxAttempts {
		return true, q.fail(ctx, j, "попытки исчерпаны")
	}

	req := ai.WritingRequest{Task: j.task, Prompt: j.prompt, Essay: j.essay, Words: j.words, Lang: j.lang}
	if j.task == Task1 {
		var img ai.Image
		err := q.Store.pool.QueryRow(ctx, `SELECT mime, data FROM writing_images WHERE check_id = $1`, j.id).Scan(&img.MIME, &img.Data)
		if err != nil {
			return true, q.fail(ctx, j, "нет фото задания: "+err.Error())
		}
		req.Image = &img
	}

	callCtx, cancel := context.WithTimeout(ctx, callTimeout)
	res, err := q.AI.GradeWriting(callCtx, req)
	cancel()
	if ctx.Err() != nil {
		// Сервер останавливается: вернуть задание в очередь, попытка не считается.
		_, err := q.Store.pool.Exec(context.WithoutCancel(ctx), `UPDATE writing_checks
			SET status = 'queued', attempts = attempts - 1, locked_until = NULL
			WHERE id = $1 AND status = 'processing' AND attempts = $2`, j.id, j.attempts)
		return true, err
	}
	if errors.Is(err, ai.ErrQuota) {
		// Запросы к Gemini на сегодня кончились — ждать новых суток Gemini, попытка не считается.
		_, err := q.Store.pool.Exec(ctx, `UPDATE writing_checks
			SET status = 'queued', attempts = attempts - 1, run_after = $3, locked_until = NULL, last_error = $4
			WHERE id = $1 AND status = 'processing' AND attempts = $2`, j.id, j.attempts, ai.QuotaReset(now), err.Error())
		return true, err
	}
	if err != nil {
		q.Log.Warn("проверка Writing не удалась", "check", j.id, "attempt", j.attempts, "err", err)
		if j.attempts >= maxAttempts {
			return true, q.fail(ctx, j, err.Error())
		}
		_, err := q.Store.pool.Exec(ctx, `UPDATE writing_checks
			SET status = 'queued', run_after = $3, locked_until = NULL, last_error = $4
			WHERE id = $1 AND status = 'processing' AND attempts = $2`,
			j.id, j.attempts, now.Add(retryDelay(j.attempts)), clipErr(err.Error()))
		return true, err
	}
	return true, q.finish(ctx, j, res)
}

func (q *Queue) finish(ctx context.Context, j job, r ai.WritingResult) error {
	band := Band(r.TA.Score, r.CC.Score, r.LR.Score, r.GRA.Score)
	fb, err := json.Marshal(Feedback{
		Summary:   r.Summary,
		Criteria:  Comments{TA: r.TA.Comment, CC: r.CC.Comment, LR: r.LR.Comment, GRA: r.GRA.Comment},
		Mistakes:  r.Mistakes,
		Corrected: r.Corrected,
		NextBand:  r.NextBand,
	})
	if err != nil {
		return err
	}
	done, err := q.close(ctx, j, `UPDATE writing_checks SET status = 'done', band = $3, ta = $4, cc = $5, lr = $6, gra = $7,
			feedback = $8, model = $9, locked_until = NULL, last_error = '', finished_at = $10
		WHERE id = $1 AND status = 'processing' AND attempts = $2`,
		band, r.TA.Score, r.CC.Score, r.LR.Score, r.GRA.Score, fb, q.AI.Model(), q.Now())
	if err != nil || !done {
		return err
	}
	q.notify(ctx, j, &band)
	return nil
}

func (q *Queue) fail(ctx context.Context, j job, reason string) error {
	done, err := q.close(ctx, j, `UPDATE writing_checks SET status = 'failed', locked_until = NULL, last_error = $3, finished_at = $4
		WHERE id = $1 AND status = 'processing' AND attempts = $2`, clipErr(reason), q.Now())
	if err != nil || !done {
		return err
	}
	q.notify(ctx, j, nil)
	return nil
}

// close записывает итог и удаляет фото задания. false — задание уже забрал другой воркер
// (аренда истекла), итог не записан, уведомлять не нужно.
func (q *Queue) close(ctx context.Context, j job, sql string, args ...any) (bool, error) {
	tx, err := q.Store.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, sql, append([]any{j.id, j.attempts}, args...)...)
	if err != nil || tag.RowsAffected() == 0 {
		return false, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM writing_images WHERE check_id = $1`, j.id); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

func (q *Queue) notify(ctx context.Context, j job, band *float64) {
	if q.OnDone != nil {
		q.OnDone(ctx, j.id)
	}
	if q.Notify == nil {
		return
	}
	if err := q.Notify.WritingChecked(ctx, j.telegramID, j.lang, j.id, band); err != nil {
		q.Log.Warn("уведомление о проверке не отправлено", "check", j.id, "err", err)
	}
}

// clipErr обрезает текст ошибки для last_error, не разрывая UTF-8 (иначе PostgreSQL отвергнет строку).
func clipErr(s string) string {
	if len(s) > 500 {
		return strings.ToValidUTF8(s[:500], "")
	}
	return s
}
