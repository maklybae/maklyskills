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
	"orderflow/services/orders/internal/api"
	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/inventory"
	"orderflow/services/orders/internal/store"
)

const shutdownGrace = 10 * time.Second

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(log); err != nil {
		log.Error("orders service stopped", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	addr := flag.String("addr", ":8080", "address to listen on")
	inventoryURL := flag.String("inventory", "", "base url of the inventory service, empty to run without stock reservations")
	flag.Parse()

	stock, err := openInventory(*inventoryURL, log)
	if err != nil {
		return err
	}

	orders, err := core.NewService(store.NewMemory(), stock)
	if err != nil {
		return xerrors.Wrap(err, "build order service")
	}
	router, err := api.NewRouter(orders, log)
	if err != nil {
		return xerrors.Wrap(err, "build router")
	}

	server := &http.Server{
		Addr:              *addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Info("orders service listening", slog.String("addr", *addr))
	return serve(server, log)
}

func openInventory(baseURL string, log *slog.Logger) (core.Inventory, error) {
	if baseURL == "" {
		log.Warn("no inventory endpoint configured, stock is not reserved")
		return core.NopInventory{}, nil
	}

	client, err := inventory.NewClient(baseURL)
	if err != nil {
		return nil, xerrors.Wrap(err, "build inventory client")
	}
	return client, nil
}

// serve blocks until the process is asked to stop or the listener gives up.
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
