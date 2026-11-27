package exam

import (
	"math/rand/v2"
	"testing"
)

func TestPickSpeaking(t *testing.T) {
	var bank []SpeakingItem
	for _, topic := range []string{"Food", "Hometown", "Music", "Sport", "Your home"} {
		for k := range 4 {
			bank = append(bank, SpeakingItem{1, topic, topic + string(rune('a'+k))})
		}
	}
	bank = append(bank, SpeakingItem{2, "book", "card book"}, SpeakingItem{2, "lonely", "card without Part 3"})
	for k := range 6 {
		bank = append(bank, SpeakingItem{3, "book", "p3 " + string(rune('a'+k))})
	}
	for seed := range uint64(30) {
		got := PickSpeaking(bank, rand.New(rand.NewPCG(seed, 1)))
		if len(got) != 11+1+5 {
			t.Fatalf("вопросов %d: %+v", len(got), got)
		}
		if !IntroTopics[got[0].Topic] {
			t.Fatalf("начало не со вступительной темы: %+v", got[0])
		}
		intro := 0
		topics := map[string]bool{}
		for _, q := range got[:11] {
			if q.Part != 1 {
				t.Fatalf("Part 1 перемешан: %+v", got)
			}
			if !topics[q.Topic] && IntroTopics[q.Topic] {
				intro++
			}
			topics[q.Topic] = true
		}
		if len(topics) != 3 || intro != 1 {
			t.Fatalf("темы Part 1: %v", topics)
		}
		if got[11].Text != "card book" || got[12].Part != 3 || got[16].Text != "p3 e" {
			t.Fatalf("карточка без своих Part 3 или порядок: %+v", got[11:])
		}
	}
	if PickSpeaking([]SpeakingItem{{2, "book", "card"}}, rand.New(rand.NewPCG(1, 1))) != nil {
		t.Fatal("банк без тем Part 1 — должен быть старый путь")
	}
}
