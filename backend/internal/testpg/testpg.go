// Package testpg поднимает встроенный PostgreSQL для тестов пакета: без Docker и без установленной базы.
package testpg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5/pgxpool"

	"ayzonexam/internal/db"
)

// Run запускает PostgreSQL на port, накатывает миграции, выполняет тесты и останавливает базу.
// Порт у каждого тестового пакета свой: go test гоняет пакеты параллельно.
func Run(port uint32, tests func(pool *pgxpool.Pool) int) int {
	dir, err := os.MkdirTemp("", "ayzonexam-pg")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer os.RemoveAll(dir)

	cache, _ := os.UserCacheDir()
	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Version(embeddedpostgres.V17).
		// UTF-8 явно: на русской Windows initdb иначе берёт WIN1251, и узбекская кириллица не сохранится.
		Encoding("UTF8").
		Locale("C").
		Port(port).
		Database("ayzonexam").
		CachePath(filepath.Join(cache, "ayzonexam-embedded-pg")).
		RuntimePath(filepath.Join(dir, "run")).
		DataPath(filepath.Join(dir, "data")).
		Logger(nil))
	if err := pg.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "встроенный PostgreSQL не запустился:", err)
		return 1
	}
	defer pg.Stop()

	ctx := context.Background()
	pool, err := db.Open(ctx, fmt.Sprintf("postgres://postgres:postgres@localhost:%d/ayzonexam?sslmode=disable", port))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	// Повторный прогон обязан ничего не делать.
	if err := db.Migrate(ctx, pool); err != nil {
		fmt.Fprintln(os.Stderr, "повторная миграция:", err)
		return 1
	}
	return tests(pool)
}
