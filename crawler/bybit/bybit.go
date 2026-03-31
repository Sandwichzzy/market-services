package bybit

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Sandwichzzy/market-services/common/tasks"
	"github.com/Sandwichzzy/market-services/database"
)

type BybitCrawler struct {
	db             *database.DB
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
}

func NewBybitCrawler(db *database.DB, shutDown context.CancelCauseFunc) (*BybitCrawler, error) {
	resourceCtx, resourceCancel := context.WithCancel(context.Background())
	return &BybitCrawler{
		db:             db,
		resourceCtx:    resourceCtx,
		resourceCancel: resourceCancel,
		tasks: tasks.Group{
			HandleCrit: func(err error) {
				shutDown(fmt.Errorf("BybitCrawler  critical Error:%v", err))
			},
		},
	}, nil
}

func (bc *BybitCrawler) Close() error {
	bc.resourceCancel()
	return bc.tasks.Wait()
}

func (bc *BybitCrawler) Start() error {
	bc.tasks.Go(func() error {
		tickerOperator := time.NewTicker(time.Second * 5)
		defer tickerOperator.Stop()
		for {
			select {
			case <-tickerOperator.C:
				log.Println("Bybit fetch data start")
			case <-bc.resourceCtx.Done():
				log.Println("Bybit fetch data  stopped")
				return errors.New("Bybit stopped")
			}
		}
	})
	return nil
}
