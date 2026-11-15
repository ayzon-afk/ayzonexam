// devdb — PostgreSQL для локальной разработки без Docker и без установки базы.
// Данные живут в backend/.devdb между запусками; Ctrl+C останавливает базу.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

func main() {
	dir, err := filepath.Abs(".devdb")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cache, _ := os.UserCacheDir()
	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Version(embeddedpostgres.V17).
		// UTF-8 явно: на русской Windows initdb иначе берёт WIN1251, и узбекская кириллица не сохранится.
		Encoding("UTF8").
		Locale("C").
		Port(5433).
		Username("ayzonexam").
		Password("ayzonexam").
		Database("ayzonexam").
		CachePath(filepath.Join(cache, "ayzonexam-embedded-pg")).
		RuntimePath(filepath.Join(dir, "run")).
		DataPath(filepath.Join(dir, "data")).
		Logger(nil))
	if err := pg.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "база не запустилась:", err)
		os.Exit(1)
	}
	fmt.Println("PostgreSQL слушает localhost:5433 — DATABASE_URL из backend/.env.example. Ctrl+C — остановить.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	if err := pg.Stop(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
