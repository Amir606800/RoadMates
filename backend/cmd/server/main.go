package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"roadmates/backend/internal/auth"
	"roadmates/backend/internal/buildinfo"
	"roadmates/backend/internal/cache"
	"roadmates/backend/internal/config"
	"roadmates/backend/internal/operational"
	"roadmates/backend/internal/pii"
	"roadmates/backend/internal/ratelimit"
	"roadmates/backend/internal/repository"
	"roadmates/backend/internal/router"
	"roadmates/backend/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err.Error())
		os.Exit(1)
	}
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	if err := run(cfg, logger); err != nil {
		logger.Error("server stopped", "error", err.Error())
		os.Exit(1)
	}
}

func run(cfg config.Config, logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Never log connection URLs/errors: they can contain credentials.
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return errors.New("invalid DATABASE_URL")
	}
	poolCfg.ConnConfig.ConnectTimeout = 2 * time.Second
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return errors.New("could not initialize PostgreSQL pool")
	}
	defer pool.Close()
	redisCfg, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return errors.New("invalid REDIS_URL")
	}
	redisCfg.DialTimeout, redisCfg.ReadTimeout, redisCfg.WriteTimeout = time.Second, time.Second, time.Second
	redisCfg.MaxRetries = -1
	redisCfg.ContextTimeoutEnabled = true
	redis.SetLogger(redisLogger{logger})
	redisClient := redis.NewClient(redisCfg)
	defer redisClient.Close()
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	if pool.Ping(checkCtx) != nil {
		logger.Warn("PostgreSQL unavailable; readiness will return 503")
	}
	if redisClient.Ping(checkCtx).Err() != nil {
		logger.Warn("Redis unavailable; running degraded")
	}
	cancel()
	tokens := auth.NewTokens(cfg.JWTSecret, cfg.AccessTokenTTL)
	piiCipher, err := pii.New(cfg.PIIEncryptionKey)
	if err != nil {
		return err
	}
	applications := service.NewDriverApplications(repository.NewDriverApplications(pool), piiCipher)
	publicCache := cache.NewRedis(redisClient)
	limiter := ratelimit.NewRedis(ratelimit.NewRedisCounter(redisClient), logger)
	districts := service.NewDistricts(repository.NewDistricts(pool), publicCache, logger)
	rides := service.NewRides(repository.NewRides(pool), publicCache, logger)
	authService, err := service.NewAuth(repository.NewAuth(pool), tokens, cfg.RefreshTokenTTL, logger)
	if err != nil {
		return errors.New("could not initialize authentication")
	}
	bootstrapCtx, bootstrapCancel := context.WithTimeout(ctx, 10*time.Second)
	err = authService.Bootstrap(bootstrapCtx, cfg.AdminEmail, cfg.AdminPassword)
	bootstrapCancel()
	if err != nil {
		return err
	}
	cleanupCtx, cleanupCancel := context.WithCancel(ctx)
	cleanupDone := make(chan struct{})
	go func() { defer close(cleanupDone); authService.Cleanup(cleanupCtx) }()
	defer func() { cleanupCancel(); <-cleanupDone }()
	info := buildinfo.Current(cfg.MinSupportedAppVersion)
	logger.Info("starting RoadMates", "app_version", info.AppVersion, "commit", info.Commit, "build_time", info.BuildTime, "go_version", info.GoVersion)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router.New(cfg, logger, operational.Handlers{Postgres: pool.Ping, Redis: func(ctx context.Context) error { return redisClient.Ping(ctx).Err() }, Version: info}, authService, tokens, applications, districts, rides, limiter),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second,
		ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server listening", "port", cfg.Port)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		stop()
		logger.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return err
		}
	}
	return nil
}

type redisLogger struct{ logger *slog.Logger }

func (l redisLogger) Printf(context.Context, string, ...interface{}) {
	l.logger.Warn("Redis connection warning")
}
