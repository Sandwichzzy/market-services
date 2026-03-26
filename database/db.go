package database

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Sandwichzzy/market-services/common/retry"
	"github.com/Sandwichzzy/market-services/config"
)

type DB struct {
	gorm *gorm.DB
}

// NewDB 根据配置创建并返回一个数据库连接实例。
// 使用指数退避策略最多重试 10 次，适应数据库启动较慢的场景。
// DSN 按需拼接 port、user、password，未配置的字段不会出现在连接串中。
func NewDB(ctx context.Context, dbConfig config.DBConfig) (*DB, error) {
	dsn := fmt.Sprintf("host=%s dbname=%s sslmode=disable", dbConfig.Host, dbConfig.Name)
	if dbConfig.Port != 0 {
		dsn += fmt.Sprintf(" port=%d", dbConfig.Port)
	}
	if dbConfig.User != "" {
		dsn += fmt.Sprintf(" user=%s", dbConfig.User)
	}
	if dbConfig.Password != "" {
		dsn += fmt.Sprintf(" password=%s", dbConfig.Password)
	}

	// 关闭默认事务以提升批量写入性能，批量创建每批上限 3000 条
	gormConfig := gorm.Config{
		SkipDefaultTransaction: true,
		CreateBatchSize:        3_000,
	}
	// 重试间隔：1s ~ 20s 指数退避，附加最大 250ms 随机抖动
	retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}

	gorm, err := retry.Do[*gorm.DB](context.Background(), 10, retryStrategy, func() (*gorm.DB, error) {
		gorm, err := gorm.Open(postgres.Open(dsn), &gormConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}
		return gorm, nil
	})
	if err != nil {
		return nil, err
	}
	db := &DB{
		gorm: gorm,
	}
	return db, nil
}

// Close 关闭底层的 *sql.DB 连接池，释放数据库资源。
func (db *DB) Close() error {
	sql, err := db.gorm.DB()
	if err != nil {
		return err
	}
	return sql.Close()
}

// ExecuteSQLMigration 遍历指定目录下的所有 SQL 文件并按文件系统顺序依次执行。
// 跳过子目录，对每个文件读取内容后通过 gorm.Exec 执行原始 SQL。
// 任意文件读取或执行失败都会立即返回错误，不继续处理后续文件。
func (db *DB) ExecuteSQLMigration(migrationsFolder string) error {
	err := filepath.Walk(migrationsFolder, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to process migration file %s", path))
		}
		// 跳过目录，只处理文件
		if info.IsDir() {
			return nil
		}
		fileContent, readErr := os.ReadFile(path)
		if readErr != nil {
			return errors.Wrap(readErr, fmt.Sprintf("Error reading SQL file: %s", path))
		}
		execErr := db.gorm.Exec(string(fileContent)).Error
		if execErr != nil {
			return errors.Wrap(execErr, fmt.Sprintf("Error executing SQL file: %s", path))
		}
		return nil
	})
	return err
}
