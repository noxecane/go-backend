package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/noxecane/anansi"
	"github.com/noxecane/anansi/sessions"
	"github.com/noxecane/anansi/tokens"
	"github.com/noxecane/anansi/webpack"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
	"tsaron.com/godview-starter/pkg/config"
	"tsaron.com/godview-starter/pkg/notification"
	"tsaron.com/godview-starter/pkg/rest"
	"tsaron.com/godview-starter/pkg/workspaces"
)

func main() {
	var err error

	var env config.Env
	if err = anansi.LoadEnv(&env); err != nil {
		panic(err)
	}

	log := anansi.NewLogger(env.Name)

	ctx, cancel := anansi.WithCancel(context.Background())
	defer cancel()

	// connect to postgresql
	var db *bun.DB
	if _, db, err = config.SetupDB(env); err != nil {
		panic(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Err(err).Msg("failed to disconnect from postgres cleanly")
		}
	}()
	log.Info().Msg("successfully connected to postgres")

	// setup redis connection
	var redisClient *redis.Client
	startupCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	if redisClient, err = config.SetupRedis(startupCtx, env); err != nil {
		panic(err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Err(err).Msg("failed to disconnect from redis cleanly")
		}
	}()
	log.Info().Msg("successfully connected to redis")

	var sessionTimeout time.Duration
	if sessionTimeout, err = time.ParseDuration(env.SessionTimeout); err != nil {
		panic(err)
	}
	app := &config.App{
		DB:     db,
		Env:    &env,
		Redis:  redisClient,
		Auth:   sessions.NewStore(redisClient, env.Secret),
		Tokens: tokens.NewStore(redisClient, env.Secret),
	}
	app.Auth = anansi.NewSessionStore(env.Secret, env.Scheme, sessionTimeout, app.Tokens)

	// API router
	router := chi.NewRouter()

	webpack.Webpack(router, log, webpack.WebpackOps{
		Environment: env.AppEnv,
		CORSOrigins: []string{
			"https://*.tsaron.com",
			"http://localhost:8080",
		},
	})

	router.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Whoops!! This route doesn't exist", http.StatusNotFound)
	})

	// dependency factory
	sStore := sessions.NewStore(app.Tokens, workspaces.NewRepo(db))
	noty := notification.New(notification.MailOpts{
		Key:             env.SendgridKey,
		Sender:          env.MailSender,
		NotifyEmail:     env.NotifyEmail,
		PostmasterEmail: env.PostmasterEmail,
		TemplatePath:    env.TemplateDir,
	})

	// setup routes
	rest.Invitations(router, app, sStore, noty)

	// mount API on app router
	appRouter := chi.NewRouter()
	appRouter.Mount("/api/v1", router)
	appRouter.Get("/", config.HealthChecker(app))
	appRouter.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Whoops!! This route doesn't exist", http.StatusNotFound)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", env.Port),
		Handler: appRouter,
	}

	go func() {
		l := log.With().Logger()
		<-ctx.Done()

		// shutdown server in 5s
		shutCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		if err := server.Shutdown(shutCtx); err != nil {
			l.Err(err).Msg("could not shut down server cleanly...")
		}
	}()

	log.Info().Msgf("serving api at http://127.0.0.1:%d", env.Port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Err(err).Msg("could not start the server")
	}

	<-ctx.Done()
}
