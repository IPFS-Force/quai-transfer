package models

import (
	"time"

	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

type TxStatus uint64

const (
	Generated TxStatus = iota
	Confirmed
)

type Transaction struct {
	ID                int32           `gorm:"primaryKey"` // not auto increment, but business increment (for deduplication)
	MinerAccount      uint            `gorm:"type:int8"`
	Payer             string          `gorm:"type:varchar(42);index"`
	Nonce             uint64          `gorm:"type:bigint"`
	ToAddress         string          `gorm:"type:varchar(42)"`
	TxHash            string          `gorm:"type:varchar(66);uniqueIndex"`
	Value             decimal.Decimal `gorm:"type:decimal(78,0)"`
	Gas               decimal.Decimal `gorm:"type:decimal(78,0)"`
	GasLimit          decimal.Decimal `gorm:"type:decimal(78,0)"` // real gas limit
	GasUsed           decimal.Decimal `gorm:"type:decimal(78,0)"` // real gas used
	CumulativeGasUsed decimal.Decimal `gorm:"type:decimal(78,0)"` // calculated gas used
	GasPrice          decimal.Decimal `gorm:"type:decimal(78,0)"` // real gas price
	Status            TxStatus        `gorm:"default:0"`          // 0: pending, 1: success, 2: failed
	CreatedAt         time.Time       `gorm:"index"`
	ConfirmedAt       *time.Time      `gorm:"index"`
	AggregateIds      pq.Int64Array   `gorm:"type:int8[]"`
	Tx                string          `gorm:"type:jsonb"`
	Entry             string          `gorm:"type:jsonb"`
}

func (t *Transaction) TableName() string {
	return "transactions"
}
