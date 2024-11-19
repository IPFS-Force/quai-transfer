package dal

import (
	"context"
	"quai-transfer/dal/models"
	"time"

	"github.com/dominant-strategies/go-quai/core/types"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type TransactionDAL struct {
	db *gorm.DB
}

func NewTransactionDAL(db *gorm.DB) *TransactionDAL {
	return &TransactionDAL{db: db}
}

func (d *TransactionDAL) CreateTransaction(ctx context.Context, tx *models.Transaction) error {
	return d.db.WithContext(ctx).Create(tx).Error
}

func (d *TransactionDAL) UpdateTransactionStatus(ctx context.Context, txHash string, gasUsedAmount decimal.Decimal, receipt *types.Receipt) error {
	gasUsedCalculated := decimal.NewFromInt(int64(receipt.GasUsed))
	cumulativeGasUsed := decimal.NewFromInt(int64(receipt.CumulativeGasUsed))

	return d.db.WithContext(ctx).Model(&models.Transaction{}).
		Where("tx_hash = ?", txHash).
		Updates(map[string]interface{}{
			"status":              receipt.Status,
			"gas":                 gasUsedAmount,
			"gas_used":            gasUsedCalculated,
			"cumulative_gas_used": cumulativeGasUsed,
			"confirmed_at":        time.Now(),
		}).Error
}

// 根据id判断数据库中是否已经存在该交易
func (d *TransactionDAL) IsTransactionExist(ctx context.Context, id int32) (bool, error) {
	var tx models.Transaction
	tmp := d.db.WithContext(ctx).Model(&models.Transaction{}).Where("id = ?", id).First(&tx)
	if err := tmp.Error; err != nil {
		return false, err
	}
	return tmp.RowsAffected > 0, nil
}
