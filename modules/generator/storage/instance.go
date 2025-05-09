package storage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/model"
	prometheus_config "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/scrape"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/storage/remote"
	"github.com/prometheus/prometheus/tsdb/agent"
	tsdb_errors "github.com/prometheus/prometheus/tsdb/errors"
)

var metricStorageRemoteWriteUpdateFailed = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "tempo",
	Name:      "metrics_generator_storage_remote_write_update_failed_total",
	Help:      "The total number of times updating the remote write configueration failed",
}, []string{"tenant"})

type Storage interface {
	storage.Appendable

	// Close closes the storage and all its underlying resources.
	Close() error
}

type storageImpl struct {
	cfg     *Config
	walDir  string
	remote  *remote.Storage
	storage storage.Storage

	tenantID string

	// Cached from the overrides
	currentHeaders       map[string]string
	sendNativeHistograms bool

	overrides Overrides
	closeCh   chan struct{}

	logger *slog.Logger
}

var _ Storage = (*storageImpl)(nil)

// New creates a metrics WAL that remote writes its data.
// TODO the passed logger does not include any other context attribute
// Should we standarize slog and deprecate go-kit/log too?
func New(cfg *Config, o Overrides, tenant string, reg prometheus.Registerer, _ log.Logger) (Storage, error) {
	// TODO move this to the generator.go
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	})).With("tenant", tenant)
	reg = prometheus.WrapRegistererWith(prometheus.Labels{"tenant": tenant}, reg)

	walDir := filepath.Join(cfg.Path, tenant)

	// clean the wal before everything
	logger.Info("clearing old WAL on start up", "dir", walDir)

	err := os.RemoveAll(walDir)
	if err != nil {
		logger.Warn(fmt.Sprintf("failed to remove wal on start up: %s", err.Error()))
	}

	logger.Info("creating WAL", "dir", walDir)

	// Create WAL directory with necessary permissions
	// This creates both <walDir>/<tenant>/ and <walDir>/<tenant>/wal/. If we don't create the wal
	// subdirectory remote storage logs a scary error.
	err = os.MkdirAll(filepath.Join(walDir, "wal"), 0o700)
	if err != nil {
		return nil, fmt.Errorf("could not create directory for metrics WAL: %w", err)
	}

	// Set up remote storage writer
	startTimeCallback := func() (int64, error) {
		return int64(model.Latest), nil
	}
	remoteStorage := remote.NewStorage(logger.With("component", "remote"), reg, startTimeCallback, walDir, cfg.RemoteWriteFlushDeadline, &noopScrapeManager{})

	headers := o.MetricsGeneratorRemoteWriteHeaders(tenant)
	generateNativeHistograms := o.MetricsGeneratorGenerateNativeHistograms(tenant)
	sendNativeHistograms := overrides.HasNativeHistograms(generateNativeHistograms)

	remoteStorageConfig := &prometheus_config.Config{
		RemoteWriteConfigs: generateTenantRemoteWriteConfigs(cfg.RemoteWrite, tenant, headers, cfg.RemoteWriteAddOrgIDHeader, logger, sendNativeHistograms),
	}

	err = remoteStorage.ApplyConfig(remoteStorageConfig)
	if err != nil {
		return nil, err
	}

	// Set up WAL
	wal, err := agent.Open(logger.With("component", "wal"), reg, remoteStorage, walDir, cfg.Wal.toPrometheusAgentOptions())
	if err != nil {
		return nil, err
	}

	s := &storageImpl{
		cfg:     cfg,
		walDir:  walDir,
		remote:  remoteStorage,
		storage: storage.NewFanout(logger, wal, remoteStorage),

		tenantID:             tenant,
		currentHeaders:       headers,
		sendNativeHistograms: sendNativeHistograms,

		overrides: o,
		closeCh:   make(chan struct{}),

		logger: logger,
	}

	go s.watchOverrides()

	return s, nil
}

func (s *storageImpl) Appender(ctx context.Context) storage.Appender {
	return s.storage.Appender(ctx)
}

func (s *storageImpl) Close() error {
	s.logger.Info("closing WAL", "dir", s.walDir)
	close(s.closeCh)

	return tsdb_errors.NewMulti(
		s.storage.Close(),
		func() error {
			// remove the WAL at shutdown since remote write starts at the end of the WAL anyways
			// https://github.com/prometheus/prometheus/issues/8809
			return os.RemoveAll(s.walDir)
		}(),
	).Err()
}

func (s *storageImpl) watchOverrides() {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			newHeaders := s.overrides.MetricsGeneratorRemoteWriteHeaders(s.tenantID)
			newGenerateNativeHistograms := s.overrides.MetricsGeneratorGenerateNativeHistograms(s.tenantID)
			newSendNativeHistograms := overrides.HasNativeHistograms(newGenerateNativeHistograms)

			if !headersEqual(s.currentHeaders, newHeaders) || s.sendNativeHistograms != newSendNativeHistograms {
				s.logger.Info("updating remote write configuration")
				s.currentHeaders = newHeaders
				s.sendNativeHistograms = newSendNativeHistograms
				err := s.remote.ApplyConfig(&prometheus_config.Config{
					RemoteWriteConfigs: generateTenantRemoteWriteConfigs(s.cfg.RemoteWrite, s.tenantID, newHeaders, s.cfg.RemoteWriteAddOrgIDHeader, s.logger, newSendNativeHistograms),
				})
				if err != nil {
					metricStorageRemoteWriteUpdateFailed.WithLabelValues(s.tenantID).Inc()
					s.logger.Info("Failed to update remote write configuration. Remote write will continue with configuration", "err", err.Error())
				}
			}
		case <-s.closeCh:
			return
		}
	}
}

func headersEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if b[k] != v {
			return false
		}
	}

	return true
}

type noopScrapeManager struct{}

func (noop *noopScrapeManager) Get() (*scrape.Manager, error) {
	return nil, errors.New("scrape manager not implemented")
}
