// Калибровка проверки Writing: прогоняет эссе с баллами учителей через Gemini и показывает расхождение.
// Запуск из корня: npm run calibrate -- -dir <папка> [-model ...] [-out результаты.json] [-list-models]
//
// Файл эссе — <имя>.txt в UTF-8:
//
//	band: 6.5
//	task: task2            (task1 — нужен image: <файл картинки рядом>)
//	prompt: текст задания  (можно пусто)
//	---
//	текст эссе
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"ayzonexam/internal/ai"
	"ayzonexam/internal/config"
	"ayzonexam/internal/writing"
)

type essay struct {
	name   string
	band   float64
	req    ai.WritingRequest
	result ai.WritingResult
	aiBand float64
	err    error
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "калибровка:", err)
		os.Exit(1)
	}
}

func run() error {
	dir := flag.String("dir", "", "папка с эссе (*.txt)")
	model := flag.String("model", "", "модель Gemini; по умолчанию GEMINI_MODEL или "+config.DefaultGeminiModel)
	lang := flag.String("lang", "ru", "язык пояснений: uz, ru, en")
	workers := flag.Int("n", 2, "сколько эссе проверять одновременно")
	out := flag.String("out", "", "куда сохранить полные ответы (JSON)")
	listModels := flag.Bool("list-models", false, "показать модели, доступные ключу, и выйти")
	flag.Parse()

	if err := config.LoadDotEnv(".env"); err != nil {
		return err
	}
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		return errors.New("GEMINI_API_KEY не задан (backend/.env)")
	}
	if *model == "" {
		*model = os.Getenv("GEMINI_MODEL")
	}
	if *model == "" {
		*model = config.DefaultGeminiModel
	}
	g := ai.NewGemini(key, *model)

	if *listModels {
		names, err := g.ListModels(context.Background())
		if err != nil {
			return err
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Println(n)
		}
		return nil
	}
	if *dir == "" {
		return errors.New("укажите -dir")
	}
	essays, err := load(*dir, *lang)
	if err != nil {
		return err
	}
	fmt.Printf("модель %s, эссе: %d\n\n", *model, len(essays))

	start := time.Now()
	sem := make(chan struct{}, max(*workers, 1))
	var wg sync.WaitGroup
	for _, e := range essays {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			// Бесплатный тариф Gemini — несколько запросов в минуту: на 429 ждём и повторяем.
			for try := 0; try < 5; try++ {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
				e.result, e.err = g.GradeWriting(ctx, e.req)
				cancel()
				if e.err == nil || !strings.Contains(e.err.Error(), "429") {
					break
				}
				// Суть — в хвосте: какой лимит (в минуту или в день) и сколько ждать.
				msg := e.err.Error()
				if i := strings.Index(msg, "Quota exceeded"); i >= 0 {
					msg = msg[i:]
				}
				fmt.Fprintf(os.Stderr, "%s: лимит Gemini, жду минуту: %.300s\n", e.name, msg)
				time.Sleep(65 * time.Second)
			}
			if e.err == nil {
				r := e.result
				e.aiBand = writing.Band(r.TA.Score, r.CC.Score, r.LR.Score, r.GRA.Score)
			}
		})
	}
	wg.Wait()
	report(essays, time.Since(start))

	if *out != "" {
		type row struct {
			Name    string           `json:"name"`
			Teacher float64          `json:"teacher"`
			AI      float64          `json:"ai"`
			Error   string           `json:"error,omitempty"`
			Result  ai.WritingResult `json:"result"`
		}
		var rows []row
		for _, e := range essays {
			r := row{Name: e.name, Teacher: e.band, AI: e.aiBand, Result: e.result}
			if e.err != nil {
				r.Error = e.err.Error()
			}
			rows = append(rows, r)
		}
		b, _ := json.MarshalIndent(rows, "", "  ")
		if err := os.WriteFile(*out, b, 0o644); err != nil {
			return err
		}
		fmt.Println("\nполные ответы:", *out)
	}
	return nil
}

func load(dir, lang string) ([]*essay, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.txt"))
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("в %s нет .txt", dir)
	}
	sort.Strings(files)
	var essays []*essay
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		head, body, ok := strings.Cut(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n---\n")
		if !ok {
			return nil, fmt.Errorf("%s: нет строки --- между заголовком и эссе", f)
		}
		e := &essay{name: strings.TrimSuffix(filepath.Base(f), ".txt")}
		e.req = ai.WritingRequest{Task: writing.Task2, Essay: strings.TrimSpace(body), Lang: lang}
		e.req.Words = writing.CountWords(e.req.Essay)
		for _, line := range strings.Split(head, "\n") {
			k, v, _ := strings.Cut(line, ":")
			v = strings.TrimSpace(v)
			switch strings.TrimSpace(k) {
			case "band":
				if e.band, err = strconv.ParseFloat(v, 64); err != nil {
					return nil, fmt.Errorf("%s: band: %w", f, err)
				}
			case "task":
				e.req.Task = v
			case "prompt":
				e.req.Prompt = v
			case "image":
				data, err := os.ReadFile(filepath.Join(dir, v))
				if err != nil {
					return nil, err
				}
				e.req.Image = &ai.Image{MIME: mime.TypeByExtension(filepath.Ext(v)), Data: data}
			}
		}
		if e.req.Task == writing.Task1 && e.req.Image == nil {
			return nil, fmt.Errorf("%s: для task1 нужна строка image:", f)
		}
		essays = append(essays, e)
	}
	return essays, nil
}

func report(essays []*essay, took time.Duration) {
	fmt.Printf("%-10s %7s %5s %7s   %s\n", "эссе", "учитель", "AI", "разница", "TA CC LR GRA")
	var n, within05, within1 int
	var sumAbs, sum float64
	for _, e := range essays {
		if e.err != nil {
			fmt.Printf("%-10s %7.1f  ошибка: %v\n", e.name, e.band, e.err)
			continue
		}
		d := e.aiBand - e.band
		r := e.result
		fmt.Printf("%-10s %7.1f %5.1f %+7.1f   %d  %d  %d  %d\n", e.name, e.band, e.aiBand, d, r.TA.Score, r.CC.Score, r.LR.Score, r.GRA.Score)
		n++
		sum += d
		sumAbs += math.Abs(d)
		if math.Abs(d) <= 0.5 {
			within05++
		}
		if math.Abs(d) <= 1 {
			within1++
		}
	}
	if n == 0 {
		return
	}
	fmt.Printf("\nпроверено %d из %d за %s\n", n, len(essays), took.Round(time.Second))
	fmt.Printf("среднее расхождение: %.2f балла (цель ≤ 0.5)\n", sumAbs/float64(n))
	fmt.Printf("сдвиг: %+.2f (плюс — AI завышает, минус — занижает)\n", sum/float64(n))
	fmt.Printf("в пределах 0.5: %d из %d, в пределах 1.0: %d из %d\n", within05, n, within1, n)
}
