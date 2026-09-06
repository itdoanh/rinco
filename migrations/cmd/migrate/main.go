// ============================================================
// migrations/cmd/migrate/main.go
// ============================================================
// Run as a binary after `go build -o migrate ./cmd/migrate`.
//
//   ./migrate up                  # apply all
//   ./migrate down                # roll back one
//   ./migrate status              # show pending migrations
//
// Env:
//   DATABASE_URL   postgres://rinco:rinco_dev_password@postgres:5432/rinco
//
// Goose is configured against multiple SQL folders:
//   sql/    (Postgres / RLS)
//   cql/    (Scylla) — handled by a separate command, see make target
//   ch/     (ClickHouse)
//   mongo/  (MongoDB) — handled by a separate command
// ============================================================

package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	dir := flag.String("dir", "sql", "sub-directory under migrations (sql|cql|ch|mongo)")
	cmd := flag.String("cmd", "up", "goose command (up|down|status|reset|create)")
	flag.Parse()

	switch *dir {
	case "sql":
		runPg(*cmd)
	case "cql":
		runCql()
	case "ch":
		runClickhouse()
	case "mongo":
		runMongo()
	default:
		log.Fatalf("unknown dir: %s", *dir)
	}
}

func runPg(cmd string) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL required")
	}

	conn := stdlib.GetDefaultDriver()
	db, err := sql.Open(conn, dsn)
	if err != nil {
		log.Fatalf("open: %v", err)
	}
	defer db.Close()

	goose.SetBaseFS(os.DirFS("migrations/sql"))
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("dialect: %v", err)
	}

	switch cmd {
	case "up":
		if err := goose.Up(db, "."); err != nil {
			log.Fatal(err)
		}
	case "down":
		if err := goose.Down(db, "."); err != nil {
			log.Fatal(err)
		}
	case "status":
		if err := goose.Status(db, "."); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unknown command: %s", cmd)
	}
	fmt.Println("OK")
}

func runCql() {
	// delegated to a shell command in the Makefile (`make migrate-cql`)
	log.Println("use: make migrate-cql")
}

func runClickhouse() {
	// delegated to a shell command in the Makefile (`make migrate-ch`)
	log.Println("use: make migrate-ch")
}

func runMongo() {
	// delegated to a shell command in the Makefile (`make migrate-mongo`)
	log.Println("use: make migrate-mongo")
}
