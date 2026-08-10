package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"orderflow/pkg/xerrors"
	"orderflow/services/inventory/internal/api"
	"orderflow/services/inventory/internal/core"
	"orderflow/services/inventory/internal/store"
)

const shutdownGrace = 10 * time.Second

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(log); err != nil {
		log.Error("inventory service stopped", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	addr := flag.String("addr", ":8081", "address to listen on")
	snapshot := flag.String("snapshot", "", "warehouse export to boot from, empty starts with an empty shelf")
	flag.Parse()

	levels := store.NewMemory()
	if *snapshot != "" {
		if err := levels.LoadSnapshot(*snapshot); err != nil {
			return xerrors.Wrapf(err, "boot from snapshot %s", *snapshot)
		}
	}

	stock, err := core.NewService(levels)
	if err != nil {
		return xerrors.Wrap(err, "build stock service")
	}
	router, err := api.NewRouter(stock, log)
	if err != nil {
		return xerrors.Wrap(err, "build router")
	}

	server := &http.Server{
		Addr:              *addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Info("inventory service listening",
		slog.String("addr", *addr),
		slog.String("snapshot", *snapshot),
	)
	return serve(server, log)
}

func serve(server *http.Server, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	failed := make(chan error, 1)
	go func() {
		failed <- server.ListenAndServe()
	}()

	select {
	case err := <-failed:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return xerrors.Wrap(err, "listen")
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return xerrors.Wrap(err, "shut down")
		}
		return nil
	}
}
