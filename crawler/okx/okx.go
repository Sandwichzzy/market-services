package okx

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Sandwichzzy/market-services/common/tasks"
	"github.com/Sandwichzzy/market-services/database"
)

type OkxCrawler struct {
	db             *database.DB
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
}

func NewOkxCrawler(db *database.DB, shutDown context.CancelCauseFunc) (*OkxCrawler, error) {
	resourceCtx, resourceCancel := context.WithCancel(context.Background())
	return &OkxCrawler{
		db:             db,
		resourceCtx:    resourceCtx,
		resourceCancel: resourceCancel,
		tasks: tasks.Group{
			HandleCrit: func(err error) {
				shutDown(fmt.Errorf("OkxCrawler  critical Error:%v", err))
			},
		},
	}, nil
}

func (oc *OkxCrawler) Close() error {
	oc.resourceCancel()
	return oc.tasks.Wait()
}

func (oc *OkxCrawler) Start() error {
	oc.tasks.Go(func() error {
		tickerOperator := time.NewTicker(time.Second * 5)
		defer tickerOperator.Stop()
		for {
			select {
			case <-tickerOperator.C:
				log.Println("Okx fetch data start")
			case <-oc.resourceCtx.Done():
				log.Println("Okx fetch data  stopped")
				return errors.New("Okx stopped")
			}
		}
	})
	return nil
}
