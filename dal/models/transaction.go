package models

import (
	"time"

	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

type Transaction struct {
	ID                int32           `gorm:"primaryKey"` // 不是数据库自增，而是业务自增（用于交易去重）
	MinerAccount      uint            `gorm:"type:int8"`
	Payer             string          `gorm:"type:varchar(42);index"`
	Nonce             uint64          `gorm:"type:bigint"`
	ToAddress         string          `gorm:"type:varchar(42)"`
	TxHash            string          `gorm:"type:varchar(66);uniqueIndex"`
	Value             decimal.Decimal `gorm:"type:decimal(78,0)"`
	Gas               decimal.Decimal `gorm:"type:decimal(78,0)"` // 应oula需求，直接计算所需要的gas。后续涉及到统计跨链gas usage，他们去爬数据
	GasLimit          decimal.Decimal `gorm:"type:decimal(78,0)"` // 实际消耗的gas
	GasUsed           decimal.Decimal `gorm:"type:decimal(78,0)"` // 实际消耗的gas
	CumulativeGasUsed decimal.Decimal `gorm:"type:decimal(78,0)"` // 计算的gas
	GasPrice          decimal.Decimal `gorm:"type:decimal(78,0)"` // 实际消耗的gas
	Status            uint64          `gorm:"default:0"`          // 0: pending, 1: success, 2: failed
	CreatedAt         time.Time       `gorm:"index"`
	ConfirmedAt       *time.Time      `gorm:"index"`
	AggregateIds      pq.Int64Array   `gorm:"type:int8[]"`
	Tx                string          `gorm:"type:text"`
	Entry             string          `gorm:"type:text"`
}
