package querier

import (
	"flag"
	"time"

	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/grpcclient"

	"github.com/grafana/tempo/modules/querier/worker"
)

// Config for a querier.
type Config struct {
	Search    SearchConfig    `yaml:"search"`
	TraceByID TraceByIDConfig `yaml:"trace_by_id"`
	Metrics   MetricsConfig   `yaml:"metrics"`

	ExtraQueryDelay                        time.Duration `yaml:"extra_query_delay,omitempty"`
	MaxConcurrentQueries                   int           `yaml:"max_concurrent_queries"`
	Worker                                 worker.Config `yaml:"frontend_worker"`
	ShuffleShardingIngestersEnabled        bool          `yaml:"shuffle_sharding_ingesters_enabled"`
	ShuffleShardingIngestersLookbackPeriod time.Duration `yaml:"shuffle_sharding_ingesters_lookback_period"`
	QueryRelevantIngesters                 bool          `yaml:"query_relevant_ingesters"`
	SecondaryIngesterRing                  string        `yaml:"secondary_ingester_ring,omitempty"`
}

type SearchConfig struct {
	QueryTimeout time.Duration `yaml:"query_timeout"`
}

type TraceByIDConfig struct {
	QueryTimeout time.Duration `yaml:"query_timeout"`
}

type MetricsConfig struct {
	ConcurrentBlocks int `yaml:"concurrent_blocks,omitempty"`

	// TimeOverlapCutoff is a tuning factor that controls whether the trace-level
	// timestamp columns are used in a metrics query.  Loading these columns has a cost,
	// so in some cases it faster to skip these columns entirely, reducing I/O but
	// increasing the number of spans evalulated and thrown away. The value is a ratio
	// between 0.0 and 1.0.  If a block overlaps the time window by less than this value,
	// then we skip the columns. A value of 1.0 will always load the columns, and 0.0 never.
	TimeOverlapCutoff float64 `yaml:"time_overlap_cutoff,omitempty"`
}

// RegisterFlagsAndApplyDefaults register flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.TraceByID.QueryTimeout = 10 * time.Second
	cfg.QueryRelevantIngesters = false
	cfg.ExtraQueryDelay = 0
	cfg.MaxConcurrentQueries = 20
	cfg.Search.QueryTimeout = 30 * time.Second
	cfg.Metrics.ConcurrentBlocks = 2
	cfg.Metrics.TimeOverlapCutoff = 0.2
	cfg.Worker = worker.Config{
		MatchMaxConcurrency:   true,
		MaxConcurrentRequests: cfg.MaxConcurrentQueries,
		Parallelism:           2,
		GRPCClientConfig: grpcclient.Config{
			MaxRecvMsgSize:  100 << 20,
			MaxSendMsgSize:  16 << 20,
			GRPCCompression: "snappy",
			BackoffConfig: backoff.Config{ // the max possible backoff should be lesser than QueryTimeout, with room for actual query response time
				MinBackoff: 100 * time.Millisecond,
				MaxBackoff: 1 * time.Second,
				MaxRetries: 5,
			},
		},
		DNSLookupPeriod: 10 * time.Second,
	}
	cfg.ShuffleShardingIngestersLookbackPeriod = 1 * time.Hour

	f.StringVar(&cfg.Worker.FrontendAddress, prefix+".frontend-address", "", "Address of query frontend service, in host:port format.")
}
