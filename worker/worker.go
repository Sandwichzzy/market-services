package worker

import (
	"context"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"

	"github.com/Sandwichzzy/market-services/config"
	"github.com/Sandwichzzy/market-services/database"
	"github.com/Sandwichzzy/market-services/redis"
)

type Worker struct {
	marketPriceHandle *MarketPriceHandle
	stopped           atomic.Bool
}

func NewWorker(db *database.DB, redisClient *redis.Client, config *config.Config, shutdown context.CancelCauseFunc) (*Worker, error) {
	marketPriceHandle, err := NewMarketPriceHandle(db, redisClient, config, shutdown)
	if err != nil {
		log.Error("Failed to create MarketPriceHandle", "error", err)
		return nil, err
	}

	return &Worker{
		marketPriceHandle: marketPriceHandle,
	}, nil
}

func (w *Worker) Start(ctx context.Context) error {
	log.Info("Starting worker")

	if err := w.marketPriceHandle.Start(); err != nil {
		log.Error("Failed to start MarketPriceHandle", "error", err)
		return err
	}

	log.Info("Worker started successfully")
	return nil
}

func (w *Worker) Stop(ctx context.Context) error {
	log.Info("Stopping worker")

	if w.marketPriceHandle != nil {
		if err := w.marketPriceHandle.Stop(); err != nil {
			log.Error("Failed to stop MarketPriceHandle", "error", err)
			w.stopped.Store(true)
			return err
		}
	}

	w.stopped.Store(true)
	log.Info("Worker stopped successfully")
	return nil
}

func (w *Worker) Stopped() bool {
	return w.stopped.Load()
}
