package config

import (
	"net/http"

	"github.com/noxecane/anansi/sessions"
	"github.com/noxecane/anansi/tokens"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
)

type App struct {
	Env    *Env
	DB     *bun.DB
	Redis  *redis.Client
	Auth   *sessions.Store
	Tokens *tokens.Store
}

func HealthChecker(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		if err := app.DB.Ping(); err != nil {
			http.Error(w, "Could not reach postgres", http.StatusInternalServerError)
			return
		}

		if _, err := app.Redis.Ping(r.Context()).Result(); err != nil {
			http.Error(w, "Could not reach redis", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		// we don't have a plan for when writes fail
		_, _ = w.Write([]byte("Up and Running!"))
	}
}
