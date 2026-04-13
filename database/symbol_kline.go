package database

import (
	"time"

	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SymbolKline struct {
	Guid       string    `gorm:"primaryKey;column:guid;type:text" json:"guid"`
	SymbolGuid string    `gorm:"column:symbol_guid;type:varchar(100);not null;default:''" json:"symbol_guid"`
	OpenPrice  string    `gorm:"column:open_price;type:numeric(65,18);not null;default:0" json:"open_price"`
	ClosePrice string    `gorm:"column:close_price;type:numeric(65,18);not null;default:0" json:"close_price"`
	HighPrice  string    `gorm:"column:high_price;type:numeric(65,18);not null;default:0" json:"high_price"`
	LowPrice   string    `gorm:"column:low_price;type:numeric(65,18);not null;default:0" json:"low_price"`
	Volume     string    `gorm:"column:volume;type:uint256;not null;default:0" json:"volume"`
	MarketCap  string    `gorm:"column:market_cap;type:uint256;not null;default:0" json:"market_cap"`
	IsActive   bool      `gorm:"column:is_active;type:boolean;not null;default:true" json:"is_active"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (SymbolKline) TableName() string {
	return "symbol_kline"
}

type SymbolKlineView interface {
	QuerySymbolKlineList(page, pageSize int64) ([]*SymbolKline, int64, error)
	QuerySymbolKlineListByFilter(page, pageSize int64, symbolGuid string, onlyActive bool, startAt, endAt *time.Time) ([]*SymbolKline, int64, error)
}

type SymbolKlineDB interface {
	SymbolKlineView

	StoreSymbolKlines([]SymbolKline) error
	StoreSymbolKline(*SymbolKline) error
	UpsertSymbolKlines([]SymbolKline) error
}

type symbolKlineDB struct {
	gorm *gorm.DB
}

func NewSymbolKlineDB(db *gorm.DB) SymbolKlineDB {
	return &symbolKlineDB{gorm: db}
}

func (s *symbolKlineDB) QuerySymbolKlineList(page, pageSize int64) ([]*SymbolKline, int64, error) {
	return s.QuerySymbolKlineListByFilter(page, pageSize, "", false, nil, nil)
}

func (s *symbolKlineDB) QuerySymbolKlineListByFilter(page, pageSize int64, symbolGuid string, onlyActive bool, startAt, endAt *time.Time) ([]*SymbolKline, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	var list []*SymbolKline
	query := s.gorm.Model(&SymbolKline{})
	if onlyActive {
		query = query.Where("is_active = ?", true)
	}
	if symbolGuid != "" {
		query = query.Where("symbol_guid = ?", symbolGuid)
	}
	if startAt != nil {
		query = query.Where("created_at >= ?", *startAt)
	}
	if endAt != nil {
		query = query.Where("created_at <= ?", *endAt)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Error("Failed to query symbol_kline count", "error", err)
		return nil, 0, err
	}

	if err := query.Order("created_at ASC").
		Limit(int(pageSize)).
		Offset(int(offset)).
		Find(&list).Error; err != nil {
		log.Error("Failed to query symbol_kline list", "error", err)
		return nil, 0, err
	}

	return list, total, nil
}

func (s *symbolKlineDB) StoreSymbolKlines(list []SymbolKline) error {
	if err := s.gorm.Table("symbol_kline").
		CreateInBatches(&list, len(list)).Error; err != nil {
		log.Error("Failed to store symbol_kline list", "error", err)
		return err
	}
	return nil
}

func (s *symbolKlineDB) StoreSymbolKline(data *SymbolKline) error {
	if err := s.gorm.Table("symbol_kline").
		Create(&data).Error; err != nil {
		log.Error("Failed to store symbol_kline", "error", err)
		return err
	}
	return nil
}

// UpsertSymbolKlines 按交易对和 K 线时间幂等写入聚合 K 线。
func (s *symbolKlineDB) UpsertSymbolKlines(list []SymbolKline) error {
	if len(list) == 0 {
		return nil
	}

	if err := s.gorm.Table("symbol_kline").
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "symbol_guid"},
				{Name: "created_at"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"open_price",
				"close_price",
				"high_price",
				"low_price",
				"volume",
				"market_cap",
				"is_active",
				"updated_at",
			}),
		}).
		CreateInBatches(&list, len(list)).Error; err != nil {
		log.Error("Failed to upsert symbol_kline list", "error", err)
		return err
	}
	return nil
}
