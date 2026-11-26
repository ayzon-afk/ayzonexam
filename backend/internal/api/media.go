package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Файлы из таблицы media (аудио Listening до 15 МБ, скрины, фото) отдаются из базы кусками: в памяти сервера —
// один кусок на запрос, а не весь файл. Телефон слушает аудио запросами Range по частям; если на каждый читать
// файл целиком, десяток учеников на Listening разом упирается в память контейнера (256 МБ).

const mediaChunk = 256 << 10

// serveMedia отдаёт файл id; onlyPublic — только открытые (аудио Listening), иначе любой (админка).
// false — файла нет: ответ 404 пишет вызывающий, у ученика и у админки он разный.
func (s *server) serveMedia(w http.ResponseWriter, r *http.Request, id int64, onlyPublic bool, cache string) bool {
	var mime string
	var size int64
	err := s.DB.QueryRow(r.Context(), `SELECT mime, octet_length(data) FROM media WHERE id = $1 AND (NOT $2 OR NOT private)`,
		id, onlyPublic).Scan(&mime, &size)
	if err != nil {
		return false
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", cache)
	http.ServeContent(w, r, "", time.Time{}, &mediaReader{ctx: r.Context(), db: s.DB, id: id, size: size})
	return true
}

// mediaReader читает media.data кусками по mediaChunk через substring: PostgreSQL достаёт из TOAST только
// нужные части (data хранится без сжатия — миграция 0021).
type mediaReader struct {
	ctx    context.Context
	db     *pgxpool.Pool
	id     int64
	size   int64
	off    int64
	buf    []byte
	bufOff int64
}

func (m *mediaReader) Read(p []byte) (int, error) {
	if m.off >= m.size {
		return 0, io.EOF
	}
	if m.off < m.bufOff || m.off >= m.bufOff+int64(len(m.buf)) {
		var chunk []byte
		err := m.db.QueryRow(m.ctx, `SELECT substring(data FROM $2 FOR $3) FROM media WHERE id = $1`,
			m.id, m.off+1, mediaChunk).Scan(&chunk)
		if err != nil {
			return 0, err
		}
		if len(chunk) == 0 {
			return 0, io.ErrUnexpectedEOF
		}
		m.buf, m.bufOff = chunk, m.off
	}
	n := copy(p, m.buf[m.off-m.bufOff:])
	m.off += int64(n)
	return n, nil
}

func (m *mediaReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		offset += m.off
	case io.SeekEnd:
		offset += m.size
	default:
		return 0, errors.New("media: неверный whence")
	}
	if offset < 0 {
		return 0, errors.New("media: отрицательное смещение")
	}
	m.off = offset
	return offset, nil
}
