package database

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"
)

type SymbolMarket struct {
	Guid       string    `gorm:"primaryKey;column:guid;type:text" json:"guid"`
	SymbolGuid string    `gorm:"column:symbol_guid;type:varchar(100);not null;default:''" json:"symbol_guid"`
	Price      string    `gorm:"column:price;type:numeric(65,18);not null;default:0" json:"price"`
	AskPrice   string    `gorm:"column:ask_price;type:numeric(65,18);not null;default:0" json:"ask_price"`
	BidPrice   string    `gorm:"column:bid_price;type:numeric(65,18);not null;default:0" json:"bid_price"`
	Volume     string    `gorm:"column:volume;type:uint256;not null;default:0" json:"volume"`
	MarketCap  string    `gorm:"column:market_cap;type:uint256;not null;default:0" json:"market_cap"`
	Radio      string    `gorm:"column:radio;type:numeric(65,18);not null;default:0" json:"radio"`
	IsActive   bool      `gorm:"column:is_active;type:boolean;not null;default:true" json:"is_active"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (SymbolMarket) TableName() string {
	return "symbol_market"
}

type SymbolMarketView interface {
	QuerySymbolMarketList(page, pageSize int64) ([]*SymbolMarket, int64, error)
	QuerySymbolMarketTodayFirstDataBySymbol(symbolGuid string) (*SymbolMarket, error)
}

type SymbolMarketDB interface {
	SymbolMarketView

	StoreSymbolMarkets([]SymbolMarket) error
	StoreSymbolMarket(*SymbolMarket) error
}

type symbolMarketDB struct {
	gorm *gorm.DB
}

func NewSymbolMarketDB(db *gorm.DB) SymbolMarketDB {
	return &symbolMarketDB{gorm: db}
}

func (s *symbolMarketDB) QuerySymbolMarketList(page, pageSize int64) ([]*SymbolMarket, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	var list []*SymbolMarket
	query := s.gorm.Model(&SymbolMarket{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Error("Failed to query symbol_market count", "error", err)
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").
		Limit(int(pageSize)).
		Offset(int(offset)).
		Find(&list).Error; err != nil {
		log.Error("Failed to query symbol_market list", "error", err)
		return nil, 0, err
	}

	return list, total, nil
}

// QuerySymbolMarketTodayFirstDataBySymbol 查询指定交易对当日最早的一条行情快照。
// 若当日尚无数据，返回 (nil, nil)，由上层按“首条记录”场景处理。
func (s *symbolMarketDB) QuerySymbolMarketTodayFirstDataBySymbol(symbolGuid string) (*SymbolMarket, error) {
	var symbolMarket *SymbolMarket
	now := time.Now().UTC()
	utcStartOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if err := s.gorm.Table("symbol_market").
		Where("symbol_guid = ? AND created_at > ?", symbolGuid, utcStartOfDay).
		Order("created_at ASC").
		Order("guid ASC").
		First(&symbolMarket).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Error("QuerySymbolMarketTodayFirstDataBySymbol error", "symbol_guid", symbolGuid, "error", err)
		return nil, err
	}
	return symbolMarket, nil
}

func (s *symbolMarketDB) StoreSymbolMarkets(list []SymbolMarket) error {
	if err := s.gorm.Table("symbol_market").
		CreateInBatches(&list, len(list)).Error; err != nil {
		log.Error("Failed to store symbol_market list", "error", err)
		return err
	}
	return nil
}

func (s *symbolMarketDB) StoreSymbolMarket(data *SymbolMarket) error {
	if err := s.gorm.Table("symbol_market").
		Create(&data).Error; err != nil {
		log.Error("Failed to store symbol_market", "error", err)
		return err
	}
	return nil
}
