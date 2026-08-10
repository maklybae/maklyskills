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

type config struct {
	addr         string
	backend      string
	dataPath     string
	inventoryURL string
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(log); err != nil {
		log.Error("orders service stopped", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	var cfg config
	flag.StringVar(&cfg.addr, "addr", ":8080", "address to listen on")
	flag.StringVar(&cfg.backend, "store", "memory", "order backend: memory or jsonfile")
	flag.StringVar(&cfg.dataPath, "data", "orders.json", "document used by the jsonfile backend")
	flag.StringVar(&cfg.inventoryURL, "inventory", "", "base url of the inventory service, empty to run without stock reservations")
	flag.Parse()

	orderStore, err := openStore(cfg)
	if err != nil {
		return err
	}
	stock, err := openInventory(cfg, log)
	if err != nil {
		return err
	}

	orders, err := core.NewService(orderStore, stock)
	if err != nil {
		return xerrors.Wrap(err, "build order service")
	}
	router, err := api.NewRouter(orders, log)
	if err != nil {
		return xerrors.Wrap(err, "build router")
	}

	server := &http.Server{
		Addr:              cfg.addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Info("orders service listening",
		slog.String("addr", cfg.addr),
		slog.String("store", cfg.backend),
	)
	return serve(server, log)
}

func openStore(cfg config) (core.Store, error) {
	switch cfg.backend {
	case "memory":
		return store.NewMemory(), nil
	case "jsonfile":
		file, err := store.NewJSONFile(cfg.dataPath)
		if err != nil {
			return nil, xerrors.Wrapf(err, "open order document %s", cfg.dataPath)
		}
		return file, nil
	default:
		return nil, errors.New("unknown -store value " + cfg.backend + ", use memory or jsonfile")
	}
}

func openInventory(cfg config, log *slog.Logger) (core.Inventory, error) {
	if cfg.inventoryURL == "" {
		log.Warn("no inventory endpoint configured, stock is not reserved")
		return core.NopInventory{}, nil
	}

	client, err := inventory.NewClient(cfg.inventoryURL)
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
