package wtypes

import (
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

type TransferEntry struct {
	ID             int32
	MinerAccount   string
	Value          decimal.Decimal
	ToAddress      string
	AggregateIds   pq.Int64Array
	MinerAccountID uint64
}
