package exam

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"slices"
	"time"
)

// Speaking mock как на экзамене (DECISIONS 2026-11-27): Part 1 — три темы по нескольку коротких вопросов
// (первая — учёба или работа, родной город или дом), Part 2 — карточка, Part 3 — вопросы на тему карточки.

// SpeakingItem — вопрос Speaking mock; Topic — тема Part 1 или карточка, к которой относится Part 3.
type SpeakingItem struct {
	Part  int    `json:"part"`
	Topic string `json:"topic,omitempty"`
	Text  string `json:"text"`
}

// IntroTopics — с них экзаменатор начинает Part 1.
var IntroTopics = map[string]bool{"Work or studies": true, "Hometown": true, "Your home": true}

// Сколько вопросов: Part 1 — по темам (4–5 минут), Part 3 — до 5 (4–5 минут).
var part1Counts = []int{4, 4, 3}

const part3Count = 5

// PickSpeaking — вопросы mock из банка: rows — активные вопросы с темой по порядку (часть, тема, pos).
// Банк без тем (старая база) — nil: тогда по одному вопросу на часть.
func PickSpeaking(rows []SpeakingItem, rnd *rand.Rand) []SpeakingItem {
	p1 := map[string][]SpeakingItem{}
	cards := map[string]SpeakingItem{}
	p3 := map[string][]SpeakingItem{}
	var p1Topics []string
	for _, q := range rows {
		switch q.Part {
		case 1:
			if len(p1[q.Topic]) == 0 {
				p1Topics = append(p1Topics, q.Topic)
			}
			p1[q.Topic] = append(p1[q.Topic], q)
		case 2:
			cards[q.Topic] = q
		case 3:
			p3[q.Topic] = append(p3[q.Topic], q)
		}
	}
	var cardTopics []string
	for t := range cards {
		if len(p3[t]) > 0 {
			cardTopics = append(cardTopics, t)
		}
	}
	if len(p1Topics) == 0 || len(cardTopics) == 0 {
		return nil
	}
	rnd.Shuffle(len(p1Topics), func(i, j int) { p1Topics[i], p1Topics[j] = p1Topics[j], p1Topics[i] })
	// Первая тема — вступительная, если такая есть.
	for i, t := range p1Topics {
		if IntroTopics[t] {
			p1Topics[0], p1Topics[i] = p1Topics[i], p1Topics[0]
			break
		}
	}
	var out []SpeakingItem
	used := 0
	for _, t := range p1Topics {
		if used == len(part1Counts) {
			break
		}
		// После вступительной — темы не вступительные: на экзамене про дом и учёбу спрашивают один раз.
		if used > 0 && IntroTopics[t] && len(p1Topics) > len(part1Counts) {
			continue
		}
		qs := p1[t]
		out = append(out, qs[:min(part1Counts[used], len(qs))]...)
		used++
	}
	// map в Go без порядка: карточки — по имени, чтобы при том же генераторе выбор повторялся (тесты).
	slices.Sort(cardTopics)
	card := cardTopics[rnd.IntN(len(cardTopics))]
	out = append(out, cards[card])
	return append(out, p3[card][:min(part3Count, len(p3[card]))]...)
}

// StartSpeaking — таймер Speaking пошёл; вопросы закрепляются за mock.
func (s *Store) StartSpeaking(ctx context.Context, m Mock, now time.Time) (Mock, error) {
	if m.SpeakingStartedAt != nil {
		return m, nil
	}
	rows, err := s.pool.Query(ctx, `SELECT part, topic, text FROM speaking_questions
		WHERE active AND topic <> '' ORDER BY part, topic, pos, id`)
	if err != nil {
		return m, err
	}
	var bank []SpeakingItem
	for rows.Next() {
		var q SpeakingItem
		if err := rows.Scan(&q.Part, &q.Topic, &q.Text); err != nil {
			rows.Close()
			return m, err
		}
		bank = append(bank, q)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return m, err
	}
	if picked := PickSpeaking(bank, rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))); picked != nil {
		js, _ := json.Marshal(picked)
		return scanMock(s.pool.QueryRow(ctx, `UPDATE mocks SET speaking_started_at = $2, speaking_questions = $3
			WHERE id = $1 RETURNING `+mockCols, m.ID, now, js))
	}
	// Старый банк без тем: по одному вопросу на часть.
	return scanMock(s.pool.QueryRow(ctx, `UPDATE mocks SET speaking_started_at = $2, speaking_questions = (
			SELECT jsonb_agg(jsonb_build_object('part', part, 'text', text) ORDER BY part) FROM (
				SELECT DISTINCT ON (part) part, text FROM speaking_questions WHERE active ORDER BY part, random()) q)
		WHERE id = $1 RETURNING `+mockCols, m.ID, now))
}
