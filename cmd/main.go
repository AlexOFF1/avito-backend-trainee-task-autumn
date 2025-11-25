package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/AlexOFF1/avito-backend-trainee-task-autumn/internal/delivery"
	"github.com/AlexOFF1/avito-backend-trainee-task-autumn/internal/repository"
	"github.com/AlexOFF1/avito-backend-trainee-task-autumn/internal/router"
	"github.com/AlexOFF1/avito-backend-trainee-task-autumn/internal/server"
	"github.com/AlexOFF1/avito-backend-trainee-task-autumn/internal/usecase"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbConnStr := os.Getenv("DB_CONN")
	if dbConnStr == "" {
		logger.Error("DB_CONN not set")
		os.Exit(1)
	}

	sqlDB, err := sql.Open("pgx", dbConnStr)
	if err != nil {
		logger.Error("failed to open sql.DB for migrations", "err", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	goose.SetDialect("postgres")
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		logger.Error("migrations failed", "err", err)
		os.Exit(1)
	}
	logger.Info("migrations applied")

	pool, err := repository.NewPool(context.Background(), dbConnStr, logger)
	if err != nil {
		logger.Error("failed to connect to DB", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	repo := repository.NewRepository(pool, logger)
	service := usecase.NewPRService(repo, logger)
	handler := delivery.NewHandler(service, logger)

	srv := server.NewServer(":8080", router.Router(handler), logger)
	srv.Run()
}
