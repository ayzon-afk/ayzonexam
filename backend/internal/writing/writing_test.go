package writing

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ayzonexam/internal/ai"
	"ayzonexam/internal/testpg"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	os.Exit(testpg.Run(25432, func(p *pgxpool.Pool) int {
		pool = p
		return m.Run()
	}))
}

func TestCountWords(t *testing.T) {
	// Те же примеры — в frontend/src/lib/writing.test.ts: правило подсчёта одно.
	cases := map[string]int{
		"":                                  0,
		"   \n\t ":                          0,
		"One two  three":                    3,
		"Well-known facts — and 2 numbers.": 5,
		"line\nbreak\n\nparagraph":          3,
		"don't stop , ok":                   3,
		"Ўзбекча матн va lotin":             4,
		"- — – ...":                         0,
	}
	for in, want := range cases {
		if got := CountWords(in); got != want {
			t.Errorf("CountWords(%q) = %d, ждали %d", in, got, want)
		}
	}
}

func TestBand(t *testing.T) {
	cases := []struct {
		ta, cc, lr, gra int
		want            float64
	}{
		{6, 6, 6, 6, 6.0},
		{7, 6, 6, 6, 6.5}, // 6.25 → 6.5
		{7, 7, 6, 6, 6.5}, // 6.5
		{7, 7, 7, 6, 7.0}, // 6.75 → 7.0
		{9, 9, 9, 9, 9.0},
		{0, 0, 0, 1, 0.5},
		{5, 6, 6, 6, 6.0}, // 5.75 → 6.0
	}
	for _, c := range cases {
		if got := Band(c.ta, c.cc, c.lr, c.gra); got != c.want {
			t.Errorf("Band(%d,%d,%d,%d) = %.2f, ждали %.1f", c.ta, c.cc, c.lr, c.gra, got, c.want)
		}
	}
}

// newUser создаёт ученика напрямую в базе.
func newUser(t *testing.T, tgID int64) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(t.Context(), `INSERT INTO users (telegram_id, nickname) VALUES ($1, $2) RETURNING id`, tgID, fmt.Sprint("u", tgID)).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// Полдень 15.11.2026 по Ташкенту.
var (
	tashkent = time.FixedZone("UZT", 5*3600)
	now      = time.Date(2026, 11, 15, 12, 0, 0, 0, tashkent)
	dayStart = time.Date(2026, 11, 15, 0, 0, 0, 0, tashkent)
)

func create(t *testing.T, s *Store, userID int64, task string, img *ai.Image) int64 {
	t.Helper()
	id, err := s.Create(t.Context(), NewCheck{UserID: userID, Task: task, Prompt: "p", Essay: "Some essay text", Words: 3,
		Source: "text", Lang: "ru", Image: img}, 3, dayStart, now)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestDailyLimit(t *testing.T) {
	s := NewStore(pool)
	u := newUser(t, 7001)
	for range 3 {
		create(t, s, u, Task2, nil)
	}
	_, err := s.Create(t.Context(), NewCheck{UserID: u, Task: Task2, Essay: "e", Words: 1, Source: "text", Lang: "ru"}, 3, dayStart, now)
	if !errors.Is(err, ErrDailyLimit) {
		t.Fatalf("4-я проверка за день: ждали ErrDailyLimit, получили %v", err)
	}

	// Неудавшаяся проверка не съедает лимит.
	pool.Exec(t.Context(), `UPDATE writing_checks SET status = 'failed' WHERE id = (SELECT max(id) FROM writing_checks WHERE user_id = $1)`, u)
	if _, err := s.Create(t.Context(), NewCheck{UserID: u, Task: Task2, Essay: "e", Words: 1, Source: "text", Lang: "ru"}, 3, dayStart, now); err != nil {
		t.Fatalf("после failed лимит должен освободиться: %v", err)
	}
	// Новый день по Ташкенту — снова можно.
	tomorrow := dayStart.AddDate(0, 0, 1)
	if _, err := s.Create(t.Context(), NewCheck{UserID: u, Task: Task2, Essay: "e", Words: 1, Source: "text", Lang: "ru"}, 3, tomorrow, tomorrow.Add(time.Hour)); err != nil {
		t.Fatalf("на следующий день: %v", err)
	}
}

func TestDailyLimitConcurrent(t *testing.T) {
	s := NewStore(pool)
	u := newUser(t, 7002)
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for range 10 {
		wg.Go(func() {
			_, err := s.Create(context.Background(), NewCheck{UserID: u, Task: Task2, Essay: "e", Words: 1, Source: "text", Lang: "ru"}, 3, dayStart, now)
			if err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if ok != 3 {
		t.Fatalf("10 одновременных отправок: прошло %d, ждали ровно 3", ok)
	}
}

func TestOwnChecksOnly(t *testing.T) {
	s := NewStore(pool)
	a, b := newUser(t, 7003), newUser(t, 7004)
	id := create(t, s, a, Task2, nil)
	if _, err := s.Get(t.Context(), b, id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("чужая проверка: ждали ErrNotFound, получили %v", err)
	}
	if list, _ := s.List(t.Context(), b, 30); len(list) != 0 {
		t.Fatalf("в истории чужие проверки: %d", len(list))
	}
	list, err := s.List(t.Context(), a, 30)
	if err != nil || len(list) != 1 || list[0].Essay != "" {
		t.Fatalf("история без текста эссе: %+v %v", list, err)
	}
}

func TestUsage(t *testing.T) {
	s := NewStore(pool)
	u := newUser(t, 7005)
	for i := range 3 {
		ok, err := s.TakeUsage(t.Context(), u, "recognize", 2, dayStart, now)
		if err != nil || ok != (i < 2) {
			t.Fatalf("вызов %d: ok=%v err=%v", i+1, ok, err)
		}
	}
	s.ReturnUsage(t.Context(), u, "recognize")
	if ok, _ := s.TakeUsage(t.Context(), u, "recognize", 2, dayStart, now); !ok {
		t.Fatal("возвращённый вызов должен освободить лимит")
	}
}

// stubAI отвечает заданными ошибками по очереди, потом — результатом.
type stubAI struct {
	ai.Fake // Speaking и озвучка — заглушкой: тест про Writing
	mu      sync.Mutex
	errs    []error
	calls   int
	last    ai.WritingRequest
}

func (s *stubAI) Model() string { return "stub" }
func (s *stubAI) Recognize(context.Context, []ai.Image) (string, error) {
	return "", nil
}
func (s *stubAI) GradeWriting(_ context.Context, req ai.WritingRequest) (ai.WritingResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.last = req
	if len(s.errs) > 0 {
		err := s.errs[0]
		s.errs = s.errs[1:]
		return ai.WritingResult{}, err
	}
	return ai.WritingResult{TA: ai.Criterion{Score: 7, Comment: "ta"}, CC: ai.Criterion{Score: 6, Comment: "cc"}, LR: ai.Criterion{Score: 6, Comment: "lr"}, GRA: ai.Criterion{Score: 6, Comment: "gra"},
		Summary: "s", Mistakes: []ai.Mistake{{Quote: "a", Fix: "b", Why: "c", Criterion: "GRA"}}, Corrected: "fixed", NextBand: []string{"tip"}}, nil
}

type sent struct {
	tgID, check int64
	lang        string
	band        *float64
}

type stubNotify struct {
	mu   sync.Mutex
	sent []sent
}

func (n *stubNotify) WritingChecked(_ context.Context, tgID int64, lang string, check int64, band *float64) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, sent{tgID, check, lang, band})
	return nil
}

// clearQueue снимает задания прошлых тестов: тесты пакета делят одну базу.
func clearQueue(t *testing.T) {
	t.Helper()
	if _, err := pool.Exec(t.Context(), `UPDATE writing_checks SET status = 'failed' WHERE status IN ('queued', 'processing')`); err != nil {
		t.Fatal(err)
	}
}

// newQueue — очередь на часах clock.
func newQueue(s *Store, a ai.Checker, n Notifier, clock *time.Time) *Queue {
	return &Queue{Store: s, AI: a, Notify: n, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Workers: 1,
		Now: func() time.Time { return *clock }}
}

// drain обрабатывает очередь до пустой. Тесты пакета идут последовательно, так что чужих заданий нет.
func drain(t *testing.T, q *Queue) {
	t.Helper()
	for {
		did, err := q.Step(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if !did {
			return
		}
	}
}

func TestQueueDone(t *testing.T) {
	clearQueue(t)
	s := NewStore(pool)
	u := newUser(t, 7006)
	img := &ai.Image{MIME: "image/jpeg", Data: []byte{1, 2, 3}}
	id := create(t, s, u, Task1, img)
	stub, notify := &stubAI{}, &stubNotify{}
	clock := now
	drain(t, newQueue(s, stub, notify, &clock))

	c, err := s.Get(t.Context(), u, id)
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != StatusDone || c.Band == nil || *c.Band != 6.5 || *c.TA != 7 || c.Feedback == nil || c.Feedback.Criteria.TA != "ta" || c.Feedback.Corrected != "fixed" {
		t.Fatalf("итог проверки: %+v", c)
	}
	if stub.last.Image == nil || stub.last.Image.MIME != "image/jpeg" || stub.last.Lang != "ru" {
		t.Fatalf("AI не получил фото задания или язык: %+v", stub.last)
	}
	if len(notify.sent) != 1 || notify.sent[0].tgID != 7006 || notify.sent[0].check != id || *notify.sent[0].band != 6.5 {
		t.Fatalf("уведомление: %+v", notify.sent)
	}
	var images int
	pool.QueryRow(t.Context(), `SELECT count(*) FROM writing_images WHERE check_id = $1`, id).Scan(&images)
	if images != 0 {
		t.Fatal("фото задания должно удаляться после проверки")
	}
}

func TestQueueRetryThenFail(t *testing.T) {
	clearQueue(t)
	s := NewStore(pool)
	u := newUser(t, 7007)
	id := create(t, s, u, Task2, nil)
	boom := errors.New("gemini: 503")
	stub, notify := &stubAI{errs: []error{boom, boom, boom}}, &stubNotify{}
	clock := now
	q := newQueue(s, stub, notify, &clock)

	drain(t, q)
	if stub.calls != 1 {
		t.Fatalf("повтор не должен идти раньше паузы: вызовов %d", stub.calls)
	}
	clock = clock.Add(31 * time.Second)
	drain(t, q)
	clock = clock.Add(2*time.Minute + time.Second)
	drain(t, q)
	if stub.calls != 3 {
		t.Fatalf("ждали 3 попытки, было %d", stub.calls)
	}
	c, _ := s.Get(t.Context(), u, id)
	if c.Status != StatusFailed {
		t.Fatalf("после 3 ошибок — failed, а не %s", c.Status)
	}
	if len(notify.sent) != 1 || notify.sent[0].band != nil {
		t.Fatalf("ученику сообщают о неудаче один раз: %+v", notify.sent)
	}
}

func TestQueueLeaseExpired(t *testing.T) {
	clearQueue(t)
	s := NewStore(pool)
	u := newUser(t, 7008)
	id := create(t, s, u, Task2, nil)
	// Воркер взял задание и «упал»: строка осталась processing с арендой.
	pool.Exec(t.Context(), `UPDATE writing_checks SET status = 'processing', attempts = 1, locked_until = $2 WHERE id = $1`, id, now.Add(lease))

	stub := &stubAI{}
	clock := now
	q := newQueue(s, stub, nil, &clock)
	drain(t, q)
	if stub.calls != 0 {
		t.Fatal("пока аренда не истекла, задание не трогают")
	}
	clock = clock.Add(lease + time.Second)
	drain(t, q)
	if c, _ := s.Get(t.Context(), u, id); c.Status != StatusDone {
		t.Fatalf("после аренды задание должно доделаться: %s", c.Status)
	}

	// Попытки кончились на падениях — failed без вызова AI.
	id2 := create(t, s, u, Task2, nil)
	pool.Exec(t.Context(), `UPDATE writing_checks SET status = 'processing', attempts = $2, locked_until = $3 WHERE id = $1`, id2, maxAttempts, now)
	calls := stub.calls
	drain(t, q)
	if c, _ := s.Get(t.Context(), u, id2); c.Status != StatusFailed || stub.calls != calls {
		t.Fatalf("исчерпанные попытки: %s, вызовов AI %d", c.Status, stub.calls-calls)
	}
}

func TestQueueShutdownReturnsJob(t *testing.T) {
	clearQueue(t)
	s := NewStore(pool)
	u := newUser(t, 7009)
	id := create(t, s, u, Task2, nil)
	ctx, cancel := context.WithCancel(t.Context())
	clock := now
	q := newQueue(s, cancelAI{cancel: cancel}, nil, &clock)
	if _, err := q.Step(ctx); err != nil {
		t.Fatal(err)
	}
	var status string
	var attempts int
	pool.QueryRow(t.Context(), `SELECT status, attempts FROM writing_checks WHERE id = $1`, id).Scan(&status, &attempts)
	if status != StatusQueued || attempts != 0 {
		t.Fatalf("при остановке задание возвращается без потери попытки: %s, попыток %d", status, attempts)
	}
}

// cancelAI имитирует остановку сервера посреди вызова AI.
type cancelAI struct {
	ai.Fake
	cancel context.CancelFunc
}

func (cancelAI) Model() string                                         { return "cancel" }
func (cancelAI) Recognize(context.Context, []ai.Image) (string, error) { return "", nil }
func (c cancelAI) GradeWriting(ctx context.Context, _ ai.WritingRequest) (ai.WritingResult, error) {
	c.cancel()
	return ai.WritingResult{}, ctx.Err()
}

func TestRunWakesOnCreate(t *testing.T) {
	clearQueue(t)
	s := NewStore(pool)
	u := newUser(t, 7010)
	ctx, cancel := context.WithCancel(t.Context())
	q := newQueue(s, &stubAI{}, nil, &now)
	q.Now = time.Now
	q.Workers = 2
	stopped := make(chan struct{})
	go func() { q.Run(ctx); close(stopped) }()

	id, err := s.Create(t.Context(), NewCheck{UserID: u, Task: Task2, Essay: "e", Words: 1, Source: "text", Lang: "uz"}, 3, dayStart, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second) // меньше опроса раз в 5 с: будит именно сигнал
	for {
		if c, _ := s.Get(t.Context(), u, id); c.Status == StatusDone {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("новая проверка не разбудила воркеров")
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	<-stopped
}
