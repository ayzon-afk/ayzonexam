package writing

import (
	"strings"
	"testing"
)

func TestParseChart(t *testing.T) {
	bad := []string{
		`{"type":"line","title":"T","x":["a"],"series":[{"name":"s","values":[1]}]}`,
		`{"type":"bar","title":"T","x":["a","b"],"series":[{"name":"s","values":[1]}]}`,
		`{"type":"pie","title":"T","pies":[{"name":"p","parts":[{"name":"a","value":1}]}]}`,
		`{"type":"table","title":"T","columns":["a","b"],"rows":[["1"]]}`,
		`{"type":"process","title":"T","steps":["a","b"]}`,
		`{"type":"map","title":"T"}`,
		`{"type":"line","x":["a","b"],"series":[{"name":"s","values":[1,2]}]}`,
		`not json`,
	}
	for _, raw := range bad {
		if c, err := ParseChart([]byte(raw)); err == nil {
			t.Errorf("принят неверный график %s: %+v", raw, c)
		}
	}
	if c, err := ParseChart(nil); c != nil || err != nil {
		t.Fatalf("пустой график: %v %v", c, err)
	}
	c, err := ParseChart([]byte(`{"type":"line","title":"Temp","unit":"°C","x":["Jan","Feb"],"series":[{"name":"Tashkent","values":[2,4.5]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Text(); got != "Line graph: Temp\nUnits: °C\nTashkent — Jan: 2; Feb: 4.5" {
		t.Fatalf("текст для AI: %q", got)
	}
	p, _ := ParseChart([]byte(`{"type":"process","title":"Bricks","steps":["dig","shape","fire"]}`))
	if !strings.Contains(p.Text(), "3. fire") {
		t.Fatalf("процесс: %q", p.Text())
	}
}
