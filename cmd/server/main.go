package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/makarmolochaev/subscribe-service/api"
	"github.com/makarmolochaev/subscribe-service/internal/config"
	"github.com/makarmolochaev/subscribe-service/internal/service"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info("Starting...")

	grpcServer := grpc.NewServer(
		grpc.ConnectionTimeout(cfg.GRPC.Timeout),
	)

	pubSubService := service.NewPubSubService(log)
	api.RegisterPubSubServer(grpcServer, pubSubService)

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.GRPC.Host, cfg.GRPC.Port))
	if err != nil {
		log.Error("Failed to listen", slog.Any("error", err))
	}

	go func() {
		log.Info("Starting gRPC server",
			slog.String("host", cfg.GRPC.Host),
			slog.Int("port", cfg.GRPC.Port))
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("Failed to serve", slog.Any("error", err))
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop

	log.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	grpcServer.GracefulStop()
	if err := pubSubService.Shutdown(ctx); err != nil {
		log.Error("Failed to shutdown pubsub service", slog.Any("error", err))
	}

	log.Info("App stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case "local":
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case "dev":
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case "prod":
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}
