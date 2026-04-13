package database

import (
	"time"

	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ExchangeSymbolKline struct {
	Guid         string    `gorm:"primaryKey;column:guid;type:text" json:"guid"`
	ExchangeGuid string    `gorm:"column:exchange_guid;type:varchar(100);not null;default:''" json:"exchange_guid"`
	SymbolGuid   string    `gorm:"column:symbol_guid;type:varchar(100);not null;default:''" json:"symbol_guid"`
	OpenPrice    string    `gorm:"column:open_price;type:numeric(65,18);not null;default:0" json:"open_price"`
	ClosePrice   string    `gorm:"column:close_price;type:numeric(65,18);not null;default:0" json:"close_price"`
	HighPrice    string    `gorm:"column:high_price;type:numeric(65,18);not null;default:0" json:"high_price"`
	LowPrice     string    `gorm:"column:low_price;type:numeric(65,18);not null;default:0" json:"low_price"`
	Volume       string    `gorm:"column:volume;type:uint256;not null;default:0" json:"volume"`
	MarketCap    string    `gorm:"column:market_cap;type:uint256;not null;default:0" json:"market_cap"`
	IsActive     bool      `gorm:"column:is_active;type:boolean;not null;default:true" json:"is_active"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ExchangeSymbolKline) TableName() string {
	return "exchange_symbol_kline"
}

type ExchangeSymbolKlineView interface {
	QueryExchangeSymbolKlineList(page, pageSize int64) ([]*ExchangeSymbolKline, int64, error)
	QueryExchangeSymbolKlineListByFilter(page, pageSize int64, exchangeGuid, symbolGuid string, onlyActive bool, startAt, endAt *time.Time) ([]*ExchangeSymbolKline, int64, error)
	QueryExchangeSymbolKlinesSince(since time.Time) ([]*ExchangeSymbolKline, error)
}

type ExchangeSymbolKlineDB interface {
	ExchangeSymbolKlineView

	StoreExchangeSymbolKlines([]ExchangeSymbolKline) error
	StoreExchangeSymbolKline(*ExchangeSymbolKline) error
	UpsertExchangeSymbolKlines([]ExchangeSymbolKline) error
}

type exchangeSymbolKlineDB struct {
	gorm *gorm.DB
}

func NewExchangeSymbolKlineDB(db *gorm.DB) ExchangeSymbolKlineDB {
	return &exchangeSymbolKlineDB{gorm: db}
}

func (e *exchangeSymbolKlineDB) QueryExchangeSymbolKlineList(page, pageSize int64) ([]*ExchangeSymbolKline, int64, error) {
	return e.QueryExchangeSymbolKlineListByFilter(page, pageSize, "", "", false, nil, nil)
}

func (e *exchangeSymbolKlineDB) QueryExchangeSymbolKlineListByFilter(page, pageSize int64, exchangeGuid, symbolGuid string, onlyActive bool, startAt, endAt *time.Time) ([]*ExchangeSymbolKline, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	var list []*ExchangeSymbolKline
	query := e.gorm.Model(&ExchangeSymbolKline{})
	if onlyActive {
		query = query.Where("is_active = ?", true)
	}
	if exchangeGuid != "" {
		query = query.Where("exchange_guid = ?", exchangeGuid)
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
		log.Error("Failed to query exchange_symbol_kline count", "error", err)
		return nil, 0, err
	}

	if err := query.Order("created_at ASC").
		Limit(int(pageSize)).
		Offset(int(offset)).
		Find(&list).Error; err != nil {
		log.Error("Failed to query exchange_symbol_kline list", "error", err)
		return nil, 0, err
	}

	return list, total, nil
}

// QueryExchangeSymbolKlinesSince 查询指定时间之后的交易所 K 线快照。
func (e *exchangeSymbolKlineDB) QueryExchangeSymbolKlinesSince(since time.Time) ([]*ExchangeSymbolKline, error) {
	var list []*ExchangeSymbolKline
	if err := e.gorm.Table("exchange_symbol_kline").
		Where("created_at >= ? AND is_active = ?", since, true).
		Order("created_at ASC").
		Find(&list).Error; err != nil {
		log.Error("Failed to query exchange_symbol_kline by since", "since", since, "error", err)
		return nil, err
	}
	return list, nil
}

func (e *exchangeSymbolKlineDB) StoreExchangeSymbolKlines(list []ExchangeSymbolKline) error {
	if err := e.gorm.Table("exchange_symbol_kline").
		CreateInBatches(&list, len(list)).Error; err != nil {
		log.Error("Failed to store exchange_symbol_kline list", "error", err)
		return err
	}
	return nil
}

// UpsertExchangeSymbolKlines 按交易所、交易对和 K 线时间幂等写入数据。
// 不存在的行插入，已存在的行更新
func (e *exchangeSymbolKlineDB) UpsertExchangeSymbolKlines(list []ExchangeSymbolKline) error {
	if len(list) == 0 {
		return nil
	}

	//Clauses 是 GORM 里用来给 SQL 附加额外子句 的方法
	if err := e.gorm.Table("exchange_symbol_kline").
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "exchange_guid"},
				{Name: "symbol_guid"},
				{Name: "created_at"},
			}, //这里定义了“什么叫冲突”
			DoUpdates: clause.AssignmentColumns([]string{
				"open_price",
				"close_price",
				"high_price",
				"low_price",
				"volume",
				"market_cap",
				"is_active",
				"updated_at",
			}), //冲突就更新这些字段
		}).
		CreateInBatches(&list, len(list)).Error; err != nil {
		log.Error("Failed to upsert exchange_symbol_kline list", "error", err)
		return err
	}
	return nil
}

func (e *exchangeSymbolKlineDB) StoreExchangeSymbolKline(data *ExchangeSymbolKline) error {
	if err := e.gorm.Table("exchange_symbol_kline").
		Create(&data).Error; err != nil {
		log.Error("Failed to store exchange_symbol_kline", "error", err)
		return err
	}
	return nil
}
