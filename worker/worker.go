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
	stopped atomic.Bool
}

func NewWorker(db *database.DB, redisClient *redis.Client, config *config.Config, shutdown context.CancelCauseFunc) (*Worker, error) {
	return &Worker{}, nil
}

func (w *Worker) Start(ctx context.Context) error {
	log.Info("Starting worker")
	return nil
}

func (w *Worker) Stop(ctx context.Context) error {
	log.Info("Stopping worker")
	return nil
}

func (w *Worker) Stopped() bool {
	return w.stopped.Load()
}
