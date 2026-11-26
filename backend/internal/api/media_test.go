package api

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"testing"
)

// Файл больше куска mediaChunk: целиком, Range на стыке кусков, хвост, закрытый файл.
func TestMediaChunks(t *testing.T) {
	h := newHandler(t, Deps{Admins: []int64{adminTG}})
	data := make([]byte, 2*mediaChunk+1000)
	for i := range data {
		data[i] = byte(i * 7 % 251)
	}
	var open, closed int64
	pool.QueryRow(t.Context(), `INSERT INTO media (mime, data, private) VALUES ('audio/mpeg', $1, false) RETURNING id`, data).Scan(&open)
	pool.QueryRow(t.Context(), `INSERT INTO media (mime, data, private) VALUES ('image/jpeg', $1, true) RETURNING id`, data[:5000]).Scan(&closed)

	get := func(path, rng, auth string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", path, nil)
		if rng != "" {
			req.Header.Set("Range", rng)
		}
		if auth != "" {
			req.Header.Set("Authorization", "tma "+auth)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	path := fmt.Sprintf("/media/%d", open)

	r := get(path, "", "")
	if r.Code != 200 || !bytes.Equal(r.Body.Bytes(), data) || r.Header().Get("Content-Type") != "audio/mpeg" {
		t.Fatalf("целиком: %d, %d байт из %d", r.Code, r.Body.Len(), len(data))
	}
	cases := []struct {
		rng      string
		from, to int // включительно
	}{
		{fmt.Sprintf("bytes=%d-%d", mediaChunk-100, mediaChunk+99), mediaChunk - 100, mediaChunk + 99},
		{"bytes=-500", len(data) - 500, len(data) - 1},
		{"bytes=0-0", 0, 0},
	}
	for _, c := range cases {
		r := get(path, c.rng, "")
		if r.Code != 206 || !bytes.Equal(r.Body.Bytes(), data[c.from:c.to+1]) {
			t.Fatalf("%s: %d, %d байт", c.rng, r.Code, r.Body.Len())
		}
	}
	if r := get(fmt.Sprintf("/media/%d", closed), "", ""); r.Code != 404 {
		t.Fatalf("закрытый файл открыт всем: %d", r.Code)
	}
	if r := get("/media/999999999", "", ""); r.Code != 404 {
		t.Fatalf("нет файла: %d", r.Code)
	}
	if r := get(fmt.Sprintf("/api/admin/media/%d", closed), "", initData(adminTG, "")); r.Code != 200 || !bytes.Equal(r.Body.Bytes(), data[:5000]) {
		t.Fatalf("закрытый файл админу: %d, %d байт", r.Code, r.Body.Len())
	}
}
