package config

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func SetupDB(env Env) (*sql.DB, *bun.DB, error) {
	// opts := &pg.Options{
	// 	Addr:            fmt.Sprintf("%s:%d", env.PostgresHost, env.PostgresPort),
	// 	User:            env.PostgresUser,
	// 	Password:        env.PostgresPassword,
	// 	Database:        env.PostgresDatabase,
	// 	ApplicationName: env.Name,
	// 	OnConnect: func(ctx context.Context, cn *pg.Conn) error {
	// 		if _, err := cn.ExecContext(ctx, "set search_path=?", env.Name); err != nil {
	// 			return err
	// 		}
	// 		return nil
	// 	},
	// }

	// if env.PostgresSecureMode {
	// 	opts.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	// }

	// db := pg.Connect(opts)
	// _, err := db.Exec("select version()")

	sslMode := "allow"
	// skipTLSVerification := true

	if env.PostgresSecureMode {
		sslMode = "require"
		// skipTLSVerification = false
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?application_name=%s&sslmode=%s&pool_max_conns=%d", env.PostgresUser, env.PostgresPassword, env.PostgresHost, env.PostgresPort, env.Name, sslMode, env.PostgresPoolSize)
	config, _ := pgxpool.ParseConfig(connStr)
	dbpool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, nil, err
	}

	// config, err := pgx.ConnConfig("postgres://postgres:@localhost:5432/test?sslmode=disable")
	// if err != nil {
	// 	panic(err)
	// }
	// config.PreferSimpleProtocol = true

	sqldb := stdlib.OpenDBFromPool(dbpool)
	db := bun.NewDB(sqldb, pgdialect.New())

	maxOpenConns := 4 * runtime.GOMAXPROCS(0)
	sqldb.SetMaxOpenConns(maxOpenConns)
	sqldb.SetMaxIdleConns(maxOpenConns)

	return sqldb, db, nil
}
