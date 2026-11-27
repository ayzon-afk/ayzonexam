package writing

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Chart — график задания Task 1 данными (DECISIONS 2026-11-27): приложение рисует его само, а AI получает
// те же числа текстом (Text) — точнее, чем читать их с картинки. Виды: line, bar — ряды по оси x; pie —
// одна или несколько круговых диаграмм; table — таблица; process — шаги процесса по порядку.
type Chart struct {
	Type    string     `json:"type"`
	Title   string     `json:"title"`
	Unit    string     `json:"unit,omitempty"`
	X       []string   `json:"x,omitempty"`
	Series  []Series   `json:"series,omitempty"`
	Pies    []Pie      `json:"pies,omitempty"`
	Columns []string   `json:"columns,omitempty"`
	Rows    [][]string `json:"rows,omitempty"`
	Steps   []string   `json:"steps,omitempty"`
}

type Series struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
}

type Pie struct {
	Name  string  `json:"name"`
	Parts []Slice `json:"parts"`
}

type Slice struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

var ErrChart = errors.New("график задания не по формату")

// ParseChart читает и проверяет график; пусто — nil.
func ParseChart(raw []byte) (*Chart, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var c Chart
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, ErrChart
	}
	if c.Title == "" {
		return nil, fmt.Errorf("%w: нет заголовка", ErrChart)
	}
	switch c.Type {
	case "line", "bar":
		if len(c.X) < 2 || len(c.Series) == 0 || len(c.Series) > 6 {
			return nil, fmt.Errorf("%w: нужны x и от 1 до 6 рядов", ErrChart)
		}
		for _, s := range c.Series {
			if s.Name == "" || len(s.Values) != len(c.X) {
				return nil, fmt.Errorf("%w: ряд %q — значений не столько, сколько x", ErrChart, s.Name)
			}
		}
	case "pie":
		if len(c.Pies) == 0 || len(c.Pies) > 4 {
			return nil, fmt.Errorf("%w: от 1 до 4 диаграмм", ErrChart)
		}
		for _, p := range c.Pies {
			if len(p.Parts) < 2 || len(p.Parts) > 8 {
				return nil, fmt.Errorf("%w: в диаграмме %q от 2 до 8 долей", ErrChart, p.Name)
			}
		}
	case "table":
		if len(c.Columns) < 2 || len(c.Rows) == 0 {
			return nil, fmt.Errorf("%w: нужны столбцы и строки", ErrChart)
		}
		for _, r := range c.Rows {
			if len(r) != len(c.Columns) {
				return nil, fmt.Errorf("%w: в строке не столько клеток, сколько столбцов", ErrChart)
			}
		}
	case "process":
		if len(c.Steps) < 3 || len(c.Steps) > 12 {
			return nil, fmt.Errorf("%w: от 3 до 12 шагов", ErrChart)
		}
	default:
		return nil, fmt.Errorf("%w: вид %q", ErrChart, c.Type)
	}
	return &c, nil
}

func num(v float64) string {
	return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%.2f", v), "0"), ".")
}

// Text — график для AI: вид, заголовок и все числа.
func (c Chart) Text() string {
	var b strings.Builder
	kind := map[string]string{"line": "Line graph", "bar": "Bar chart", "pie": "Pie chart", "table": "Table", "process": "Process diagram"}[c.Type]
	fmt.Fprintf(&b, "%s: %s\n", kind, c.Title)
	if c.Unit != "" {
		fmt.Fprintf(&b, "Units: %s\n", c.Unit)
	}
	switch c.Type {
	case "line", "bar":
		for _, s := range c.Series {
			vals := make([]string, len(s.Values))
			for i, v := range s.Values {
				vals[i] = c.X[i] + ": " + num(v)
			}
			fmt.Fprintf(&b, "%s — %s\n", s.Name, strings.Join(vals, "; "))
		}
	case "pie":
		for _, p := range c.Pies {
			parts := make([]string, len(p.Parts))
			for i, s := range p.Parts {
				parts[i] = s.Name + " " + num(s.Value)
			}
			name := p.Name
			if name == "" {
				name = "Chart"
			}
			fmt.Fprintf(&b, "%s — %s\n", name, strings.Join(parts, "; "))
		}
	case "table":
		b.WriteString(strings.Join(c.Columns, " | ") + "\n")
		for _, r := range c.Rows {
			b.WriteString(strings.Join(r, " | ") + "\n")
		}
	case "process":
		for i, s := range c.Steps {
			fmt.Fprintf(&b, "%d. %s\n", i+1, s)
		}
	}
	return strings.TrimSpace(b.String())
}
