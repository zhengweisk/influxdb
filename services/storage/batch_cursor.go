package storage

import (
	"fmt"

	"github.com/influxdata/influxdb/tsdb"
)

func newAggregateBatchCursor(agg *Aggregate, cursor tsdb.Cursor) tsdb.Cursor {
	if cursor == nil {
		return nil
	}

	switch agg.Type {
	case AggregateTypeSum:
		return newSumBatchCursor(cursor)
	case AggregateTypeCount:
		return newCountBatchCursor(cursor)

	default:
		// TODO(sgc): should be validated higher up
		panic("invalid aggregate")
	}
}

func newSumBatchCursor(cur tsdb.Cursor) tsdb.Cursor {
	switch cur := cur.(type) {
	case tsdb.FloatBatchCursor:
		return &floatSumBatchCursor{FloatBatchCursor: cur}

	case tsdb.IntegerBatchCursor:
		return &integerSumBatchCursor{IntegerBatchCursor: cur}

	case tsdb.UnsignedBatchCursor:
		return &unsignedSumBatchCursor{UnsignedBatchCursor: cur}

	default:
		panic("unreachable")
	}
}

func newCountBatchCursor(cur tsdb.Cursor) tsdb.Cursor {
	switch cur := cur.(type) {
	case tsdb.FloatBatchCursor:
		return &integerFloatCountBatchCursor{FloatBatchCursor: cur}

	case tsdb.IntegerBatchCursor:
		return &integerIntegerCountBatchCursor{IntegerBatchCursor: cur}

	case tsdb.UnsignedBatchCursor:
		return &integerUnsignedCountBatchCursor{UnsignedBatchCursor: cur}

	case tsdb.StringBatchCursor:
		return &integerStringCountBatchCursor{StringBatchCursor: cur}

	case tsdb.BooleanBatchCursor:
		return &integerBooleanCountBatchCursor{BooleanBatchCursor: cur}

	default:
		panic("unreachable")
	}
}

func newMultiShardBatchCursor(row plannerRow, asc bool, start, end int64) tsdb.Cursor {
	req := &tsdb.CursorRequest{
		Measurement: row.measurement,
		Series:      row.key,
		Field:       row.field,
		Ascending:   asc,
		StartTime:   start,
		EndTime:     end,
	}

	var cond expression
	if row.valueCond != nil {
		cond = &astExpr{row.valueCond}
	}

	var shard *tsdb.Shard
	var cur tsdb.Cursor
	for cur == nil && len(row.shards) > 0 {
		shard, row.shards = row.shards[0], row.shards[1:]
		cur, _ = shard.CreateCursor(req)
	}

	switch c := cur.(type) {
	case tsdb.IntegerBatchCursor:
		return newIntegerMultiShardBatchCursor(c, req, row.shards, cond)

	case tsdb.FloatBatchCursor:
		return newFloatMultiShardBatchCursor(c, req, row.shards, cond)

	case tsdb.UnsignedBatchCursor:
		return newUnsignedMultiShardBatchCursor(c, req, row.shards, cond)

	case tsdb.StringBatchCursor:
		return newStringMultiShardBatchCursor(c, req, row.shards, cond)

	case tsdb.BooleanBatchCursor:
		return newBooleanMultiShardBatchCursor(c, req, row.shards, cond)

	case nil:
		return nil

	default:
		panic("unreachable: " + fmt.Sprintf("%T", cur))
	}
}
