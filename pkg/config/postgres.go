package config

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/extra/bundebug"
)

func getPackagePath() string {
	_, sourceCode, _, _ := runtime.Caller(0)
	for dir, last := filepath.Split(sourceCode); dir != ""; dir, last = filepath.Split(filepath.Clean(dir)) {
		if last == "pkg" {
			return dir
		}
	}

	return ""
}

func migrateDB(dir string, db *sql.DB) error {
	var mig *migrate.Migrate
	var driver database.Driver
	var err error

	if driver, err = postgres.WithInstance(db, &postgres.Config{}); err != nil {
		return err
	}

	//
	uri := fmt.Sprintf("file:///%s", dir)
	if mig, err = migrate.NewWithDatabaseInstance(uri, "postgres", driver); err != nil {
		return err
	}

	if err = mig.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

func SetupDB(env Env) (*sql.DB, *bun.DB, error) {
	sslMode := "allow"
	// skipTLSVerification := true

	if env.PostgresSecureMode {
		sslMode = "require"
		// skipTLSVerification = false
	}

	formatStr := "postgres://%s:%s@%s:%d/%s?application_name=%s&sslmode=%s&pool_max_conns=%d"
	connStr := fmt.Sprintf(formatStr,
		env.PostgresUser, env.PostgresPassword, env.PostgresHost,
		env.PostgresPort, env.PostgresDatabase, env.Name, sslMode, env.PostgresPoolSize,
	)
	config, _ := pgxpool.ParseConfig(connStr)
	dbpool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, nil, err
	}

	sqldb := stdlib.OpenDBFromPool(dbpool)
	db := bun.NewDB(sqldb, pgdialect.New())

	if env.PostgresDebug {
		db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))
	}

	maxOpenConns := 4 * runtime.GOMAXPROCS(0)
	sqldb.SetMaxOpenConns(maxOpenConns)
	sqldb.SetMaxIdleConns(maxOpenConns)

	if err := migrateDB(filepath.Join(getPackagePath(), "sql"), sqldb); err != nil {
		return sqldb, db, err
	}

	return sqldb, db, nil
}
