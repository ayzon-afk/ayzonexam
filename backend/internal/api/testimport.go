package api

import (
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strconv"

	"github.com/jackc/pgx/v5"

	"ayzonexam/internal/exam"
	"ayzonexam/internal/users"
)

// Загрузка тестов канала «IELTS Volume» пакетом (DECISIONS 2026-11-26): база на mars.cloud видна только
// изнутри Mars, поэтому тесты приходят через админку — папка конвертера, в каждой подпапке test.json, аудио
// и картинки. Тест создаётся выключенным: владелец смотрит предпросмотр и включает сам.

const maxImportBytes = 32 << 20 // аудио Listening до 15 МБ + картинки + JSON

var sourceRe = regexp.MustCompile(`^[A-Z0-9][A-Z0-9-]{2,39}$`)

// importExtra — поля test.json сверх testBody: id теста канала и имена файлов картинок групп.
type importExtra struct {
	Source string `json:"source"`
	Groups []struct {
		ImageFile string `json:"image_file"`
	} `json:"groups"`
}

func (s *server) adminTestImport(w http.ResponseWriter, r *http.Request, a users.User) {
	if code := parseForm(w, r, maxImportBytes); code != "" {
		writeError(w, http.StatusBadRequest, code)
		return
	}
	defer r.MultipartForm.RemoveAll()
	raw := []byte(first(r.MultipartForm.Value["test"]))
	var b testBody
	var x importExtra
	if json.Unmarshal(raw, &b) != nil || json.Unmarshal(raw, &x) != nil || !sourceRe.MatchString(x.Source) {
		writeError(w, http.StatusBadRequest, "bad_test")
		return
	}
	b.Active = false
	if !validTest(&b) {
		writeError(w, http.StatusBadRequest, "bad_test")
		return
	}
	images := map[string]int{} // имя файла → номер группы с 0
	for i, g := range x.Groups {
		if g.ImageFile != "" && i < len(b.Groups) {
			images[g.ImageFile] = i
		}
	}

	ctx := r.Context()
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	defer tx.Rollback(ctx)
	var id int64
	var attempts int
	var oldMedia []int64
	err = tx.QueryRow(ctx, `SELECT t.id, (SELECT count(*) FROM attempts WHERE test_id = t.id),
			array_remove(array_append(ARRAY(SELECT (g->>'image')::bigint FROM jsonb_array_elements(t.groups) g WHERE g ? 'image'), t.audio_id), NULL)
		FROM tests t WHERE t.source = $1 FOR UPDATE`, x.Source).Scan(&id, &attempts, &oldMedia)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.fail(w, r, err)
		return
	}
	if attempts > 0 {
		writeError(w, http.StatusConflict, "has_attempts")
		return
	}

	var audioID *int64
	if fhs := r.MultipartForm.File["audio"]; len(fhs) > 0 {
		data, err := readFile(fhs[0], 15<<20)
		mime := audioMIME(data)
		if err != nil || mime == "" {
			writeError(w, http.StatusBadRequest, "bad_audio")
			return
		}
		var mid int64
		if err := tx.QueryRow(ctx, `INSERT INTO media (mime, data, private) VALUES ($1, $2, false) RETURNING id`, mime, data).Scan(&mid); err != nil {
			s.fail(w, r, err)
			return
		}
		audioID = &mid
	}
	if b.Kind == exam.Listening && audioID == nil {
		writeError(w, http.StatusBadRequest, "no_audio")
		return
	}
	for _, fh := range r.MultipartForm.File["image"] {
		g, ok := images[fh.Filename]
		if !ok {
			continue
		}
		img, code := readImage(fh)
		if code != "" {
			writeError(w, http.StatusBadRequest, code)
			return
		}
		var mid int64
		if err := tx.QueryRow(ctx, `INSERT INTO media (mime, data, private) VALUES ($1, $2, false) RETURNING id`, img.MIME, img.Data).Scan(&mid); err != nil {
			s.fail(w, r, err)
			return
		}
		b.Groups[g].Image = &mid
		delete(images, fh.Filename)
	}
	if len(images) > 0 {
		writeError(w, http.StatusBadRequest, "no_image")
		return
	}

	qs, _ := json.Marshal(b.Questions)
	gs, _ := json.Marshal(b.Groups)
	replaced := id != 0
	if replaced {
		_, err = tx.Exec(ctx, `UPDATE tests SET kind = $2, title = $3, full_size = $4, minutes = $5, passage = $6, transcript = $7,
			questions = $8, groups = $9, audio_id = $10, active = false WHERE id = $1`,
			id, b.Kind, b.Title, b.FullSize, b.Minutes, b.Passage, b.Transcript, qs, gs, audioID)
		if err == nil && len(oldMedia) > 0 {
			_, err = tx.Exec(ctx, `DELETE FROM media WHERE id = ANY($1)`, oldMedia)
		}
	} else {
		err = tx.QueryRow(ctx, `INSERT INTO tests (kind, title, full_size, minutes, passage, transcript, questions, groups, audio_id, source, active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, false) RETURNING id`,
			b.Kind, b.Title, b.FullSize, b.Minutes, b.Passage, b.Transcript, qs, gs, audioID, x.Source).Scan(&id)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.audit(ctx, a.TelegramID, "test.import", "test:"+strconv.FormatInt(id, 10), map[string]any{"source": x.Source, "replaced": replaced})
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "replaced": replaced})
}

// adminTest — задание целиком, с ключом: правка и предпросмотр (список отдаёт только строки).
func (s *server) adminTest(w http.ResponseWriter, r *http.Request, _ users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	list, err := s.rowsJSON(r.Context(), `SELECT id, kind, title, full_size, minutes, passage, transcript, audio_id, questions, groups,
			source, active FROM tests WHERE id = $1`, id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if len(list) == 0 {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"test": list[0]})
}

// readFile читает файл формы не больше limit байт.
func readFile(fh *multipart.FileHeader, limit int64) ([]byte, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err == nil && int64(len(data)) > limit {
		err = errors.New("файл больше предела")
	}
	return data, err
}
