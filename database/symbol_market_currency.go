package database

import (
	"time"

	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"
)

type SymbolMarketCurrency struct {
	Guid         string    `gorm:"primaryKey;column:guid;type:text" json:"guid"`
	SymbolGuid   string    `gorm:"column:symbol_guid;type:varchar(100);not null;default:''" json:"symbol_guid"`
	CurrencyGuid string    `gorm:"column:currency_guid;type:varchar(100);not null;default:''" json:"currency_guid"`
	Price        string    `gorm:"column:price;type:numeric(65,18);not null;default:0" json:"price"`
	AskPrice     string    `gorm:"column:ask_price;type:numeric(65,18);not null;default:0" json:"ask_price"`
	BidPrice     string    `gorm:"column:bid_price;type:numeric(65,18);not null;default:0" json:"bid_price"`
	IsActive     bool      `gorm:"column:is_active;type:boolean;not null;default:true" json:"is_active"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (SymbolMarketCurrency) TableName() string {
	return "symbol_market_currency"
}

type SymbolMarketCurrencyView interface {
	QuerySymbolMarketCurrencyList(page, pageSize int64) ([]*SymbolMarketCurrency, int64, error)
}

type SymbolMarketCurrencyDB interface {
	SymbolMarketCurrencyView

	StoreSymbolMarketCurrencies([]SymbolMarketCurrency) error
	StoreSymbolMarketCurrency(*SymbolMarketCurrency) error
}

type symbolMarketCurrencyDB struct {
	gorm *gorm.DB
}

// NewSymbolMarketCurrencyDB 创建 symbol_market_currency 表对应的仓储实现。
func NewSymbolMarketCurrencyDB(db *gorm.DB) SymbolMarketCurrencyDB {
	return &symbolMarketCurrencyDB{gorm: db}
}

// QuerySymbolMarketCurrencyList 按分页查询法币行情快照列表，并返回总数。
func (s *symbolMarketCurrencyDB) QuerySymbolMarketCurrencyList(page, pageSize int64) ([]*SymbolMarketCurrency, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	var list []*SymbolMarketCurrency
	query := s.gorm.Model(&SymbolMarketCurrency{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Error("Failed to query symbol_market_currency count", "error", err)
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").
		Limit(int(pageSize)).
		Offset(int(offset)).
		Find(&list).Error; err != nil {
		log.Error("Failed to query symbol_market_currency list", "error", err)
		return nil, 0, err
	}

	return list, total, nil
}

// StoreSymbolMarketCurrencies 批量写入法币行情快照。
func (s *symbolMarketCurrencyDB) StoreSymbolMarketCurrencies(list []SymbolMarketCurrency) error {
	if err := s.gorm.Table("symbol_market_currency").
		CreateInBatches(&list, len(list)).Error; err != nil {
		log.Error("Failed to store symbol_market_currency list", "error", err)
		return err
	}
	return nil
}

// StoreSymbolMarketCurrency 写入单条法币行情快照。
func (s *symbolMarketCurrencyDB) StoreSymbolMarketCurrency(data *SymbolMarketCurrency) error {
	if err := s.gorm.Table("symbol_market_currency").
		Create(&data).Error; err != nil {
		log.Error("Failed to store symbol_market_currency", "error", err)
		return err
	}
	return nil
}
