package producer

import (
	"context"
	"log"
	"os"
	"time"

	k "github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
)

// Constants and default configuration take from:
// github.com/awslabs/amazon-kinesis-producer/.../KinesisProducerConfiguration.java
const (
	maxRecordSize          = 1 << 20 // 1MiB
	maxRequestSize         = 5 << 20 // 5MiB
	maxRecordsPerRequest   = 500
	maxAggregationSize     = 1048576 // 1MiB
	maxAggregationCount    = 4294967295
	defaultAggregationSize = 51200 // 50k
	defaultMaxConnections  = 24
	defaultFlushInterval   = 5 * time.Second
	partitionKeyIndexSize  = 8
)

// Putter is the interface that wraps the KinesisAPI.PutRecords method.
type Putter interface {
	PutRecords(ctx context.Context, params *k.PutRecordsInput, optFns ...func(*k.Options)) (*k.PutRecordsOutput, error)
}

// GetShardsFunc is called to populate the shard map on initialization and during refresh
// shard interval. GetShardsFunc  will be called with the current shard list. During
// initialization, this will be nil. GetShardsFunc should return a shard list, a bool
// indicating if the shards should be updated and an error. If false bool or error is
// returned, shards will not be updated.
type GetShardsFunc func(old []types.Shard) ([]types.Shard, bool, error)

func defaultGetShardsFunc(old []types.Shard) ([]types.Shard, bool, error) { return nil, false, nil }

// Config is the Producer configuration.
type Config struct {
	// StreamName is the Kinesis stream.
	StreamName string

	// FlushInterval is a regular interval for flushing the buffer. Defaults to 5s.
	FlushInterval time.Duration

	// ShardRefreshInterval is a regular interval for refreshing the ShardMap.
	// Config.GetShards will be called at this interval. A value of 0 means no refresh
	// occurs. Default is 0
	ShardRefreshInterval time.Duration

	// GetShards is called on NewProducer to initialze the ShardMap.
	// If ShardRefreshInterval is non-zero, GetShards will be called at that interval.
	// The default function returns a nil list of shards, which results in all records being
	// aggregated to a single record.
	GetShards GetShardsFunc

	// BatchCount determine the maximum number of items to pack in batch.
	// Must not exceed length. Defaults to 500.
	BatchCount int

	// BatchSize determine the maximum number of bytes to send with a PutRecords request.
	// Must not exceed 5MiB; Default to 5MiB.
	BatchSize int

	// AggregateBatchCount determine the maximum number of items to pack into an aggregated record.
	AggregateBatchCount int

	// AggregationBatchSize determine the maximum number of bytes to pack into an aggregated record. User records larger
	// than this will bypass aggregation.
	AggregateBatchSize int

	// BacklogCount determines the channel capacity before Put() will begin blocking. Default to `BatchCount`.
	BacklogCount int

	// Number of requests to sent concurrently. Default to 24.
	// If you are using the ListShards API in your GetShards function, those connections
	// will not be counted in MaxConnections.
	MaxConnections int

	// Logger is the logger used. Default to producer.Logger.
	Logger Logger

	// Enabling verbose logging. Default to false.
	Verbose bool

	// Client is the Putter interface implementation.
	Client Putter
}

// defaults for configuration
func (c *Config) defaults() {
	if c.Logger == nil {
		c.Logger = &StdLogger{log.New(os.Stdout, "", log.LstdFlags)}
	}
	if c.BatchCount == 0 {
		c.BatchCount = maxRecordsPerRequest
	}
	falseOrPanic(c.BatchCount > maxRecordsPerRequest, "kinesis: BatchCount exceeds 500")
	if c.BatchSize == 0 {
		c.BatchSize = maxRequestSize
	}
	falseOrPanic(c.BatchSize > maxRequestSize, "kinesis: BatchSize exceeds 5MiB")
	if c.BacklogCount == 0 {
		c.BacklogCount = maxRecordsPerRequest
	}
	if c.AggregateBatchCount == 0 {
		c.AggregateBatchCount = maxAggregationCount
	}
	falseOrPanic(c.AggregateBatchCount > maxAggregationCount, "kinesis: AggregateBatchCount exceeds 4294967295")
	if c.AggregateBatchSize == 0 {
		c.AggregateBatchSize = defaultAggregationSize
	}
	falseOrPanic(c.AggregateBatchSize > maxAggregationSize, "kinesis: AggregateBatchSize exceeds 1MiB")
	if c.MaxConnections == 0 {
		c.MaxConnections = defaultMaxConnections
	}
	falseOrPanic(c.MaxConnections < 1 || c.MaxConnections > 256, "kinesis: MaxConnections must be between 1 and 256")
	if c.FlushInterval == 0 {
		c.FlushInterval = defaultFlushInterval
	}
	falseOrPanic(len(c.StreamName) == 0, "kinesis: StreamName length must be at least 1")
	if c.GetShards == nil {
		c.GetShards = defaultGetShardsFunc
	}
}

func falseOrPanic(p bool, msg string) {
	if p {
		panic(msg)
	}
}
