package postgres

import (
	"context"

	"github.com/RyseUp/ChatterGo/config"
	"github.com/RyseUp/ChatterGo/internal/repositories"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Queries struct {
	db     *gorm.DB
	cfg    *config.Config
	logger logger.Interface
}

func NewQueries(db *gorm.DB, cfg *config.Config) *Queries {
	return &Queries{
		db:     db,
		cfg:    cfg,
		logger: logger.Default.LogMode(logger.Info),
	}
}

func (q *Queries) Ping() error {
	sqlDB, err := q.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func (q *Queries) WithTxn(txn *gorm.DB) *Queries {
	nr := *q
	nr.db = txn
	return &nr
}

func (q *Queries) Transaction(ctx context.Context, txFunc func(repositories.Repository) error) (err error) {
	tx := q.db.Begin()
	defer func() {
		p := recover()
		switch {
		case p != nil:
			if rollbackErr := tx.Rollback().Error; rollbackErr != nil {
				q.logger.Error(ctx, "failed to rollback transaction: %v", rollbackErr)
			}
			panic(p)
		case err != nil:
			if rollbackErr := tx.Rollback().Error; rollbackErr != nil {
				q.logger.Error(ctx, "error during rollback to savepoint: %v", rollbackErr)
			}
		default:
			err = tx.Commit().Error
		}
	}()
	return txFunc(q.WithTxn(tx))
}
