// Code generated by github.com/atombender/go-jsonschema, DO NOT EDIT.

package otelconf

import "encoding/json"
import "fmt"
import "reflect"

type AttributeLimits struct {
	// AttributeCountLimit corresponds to the JSON schema field
	// "attribute_count_limit".
	AttributeCountLimit *int `json:"attribute_count_limit,omitempty" yaml:"attribute_count_limit,omitempty" mapstructure:"attribute_count_limit,omitempty"`

	// AttributeValueLengthLimit corresponds to the JSON schema field
	// "attribute_value_length_limit".
	AttributeValueLengthLimit *int `json:"attribute_value_length_limit,omitempty" yaml:"attribute_value_length_limit,omitempty" mapstructure:"attribute_value_length_limit,omitempty"`

	AdditionalProperties interface{}
}

type Attributes map[string]interface{}

type BatchLogRecordProcessor struct {
	// ExportTimeout corresponds to the JSON schema field "export_timeout".
	ExportTimeout *int `json:"export_timeout,omitempty" yaml:"export_timeout,omitempty" mapstructure:"export_timeout,omitempty"`

	// Exporter corresponds to the JSON schema field "exporter".
	Exporter LogRecordExporter `json:"exporter" yaml:"exporter" mapstructure:"exporter"`

	// MaxExportBatchSize corresponds to the JSON schema field
	// "max_export_batch_size".
	MaxExportBatchSize *int `json:"max_export_batch_size,omitempty" yaml:"max_export_batch_size,omitempty" mapstructure:"max_export_batch_size,omitempty"`

	// MaxQueueSize corresponds to the JSON schema field "max_queue_size".
	MaxQueueSize *int `json:"max_queue_size,omitempty" yaml:"max_queue_size,omitempty" mapstructure:"max_queue_size,omitempty"`

	// ScheduleDelay corresponds to the JSON schema field "schedule_delay".
	ScheduleDelay *int `json:"schedule_delay,omitempty" yaml:"schedule_delay,omitempty" mapstructure:"schedule_delay,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *BatchLogRecordProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return fmt.Errorf("field exporter in BatchLogRecordProcessor: required")
	}
	type Plain BatchLogRecordProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = BatchLogRecordProcessor(plain)
	return nil
}

type BatchSpanProcessor struct {
	// ExportTimeout corresponds to the JSON schema field "export_timeout".
	ExportTimeout *int `json:"export_timeout,omitempty" yaml:"export_timeout,omitempty" mapstructure:"export_timeout,omitempty"`

	// Exporter corresponds to the JSON schema field "exporter".
	Exporter SpanExporter `json:"exporter" yaml:"exporter" mapstructure:"exporter"`

	// MaxExportBatchSize corresponds to the JSON schema field
	// "max_export_batch_size".
	MaxExportBatchSize *int `json:"max_export_batch_size,omitempty" yaml:"max_export_batch_size,omitempty" mapstructure:"max_export_batch_size,omitempty"`

	// MaxQueueSize corresponds to the JSON schema field "max_queue_size".
	MaxQueueSize *int `json:"max_queue_size,omitempty" yaml:"max_queue_size,omitempty" mapstructure:"max_queue_size,omitempty"`

	// ScheduleDelay corresponds to the JSON schema field "schedule_delay".
	ScheduleDelay *int `json:"schedule_delay,omitempty" yaml:"schedule_delay,omitempty" mapstructure:"schedule_delay,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *BatchSpanProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return fmt.Errorf("field exporter in BatchSpanProcessor: required")
	}
	type Plain BatchSpanProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = BatchSpanProcessor(plain)
	return nil
}

type Common map[string]interface{}

type Console map[string]interface{}

type Detectors struct {
	// Attributes corresponds to the JSON schema field "attributes".
	Attributes *DetectorsAttributes `json:"attributes,omitempty" yaml:"attributes,omitempty" mapstructure:"attributes,omitempty"`
}

type DetectorsAttributes struct {
	// Excluded corresponds to the JSON schema field "excluded".
	Excluded []string `json:"excluded,omitempty" yaml:"excluded,omitempty" mapstructure:"excluded,omitempty"`

	// Included corresponds to the JSON schema field "included".
	Included []string `json:"included,omitempty" yaml:"included,omitempty" mapstructure:"included,omitempty"`
}

type Headers map[string]string

type IncludeExclude struct {
	// Excluded corresponds to the JSON schema field "excluded".
	Excluded []string `json:"excluded,omitempty" yaml:"excluded,omitempty" mapstructure:"excluded,omitempty"`

	// Included corresponds to the JSON schema field "included".
	Included []string `json:"included,omitempty" yaml:"included,omitempty" mapstructure:"included,omitempty"`
}

type LogRecordExporter struct {
	// Console corresponds to the JSON schema field "console".
	Console Console `json:"console,omitempty" yaml:"console,omitempty" mapstructure:"console,omitempty"`

	// OTLP corresponds to the JSON schema field "otlp".
	OTLP *OTLP `json:"otlp,omitempty" yaml:"otlp,omitempty" mapstructure:"otlp,omitempty"`

	AdditionalProperties interface{}
}

type LogRecordLimits struct {
	// AttributeCountLimit corresponds to the JSON schema field
	// "attribute_count_limit".
	AttributeCountLimit *int `json:"attribute_count_limit,omitempty" yaml:"attribute_count_limit,omitempty" mapstructure:"attribute_count_limit,omitempty"`

	// AttributeValueLengthLimit corresponds to the JSON schema field
	// "attribute_value_length_limit".
	AttributeValueLengthLimit *int `json:"attribute_value_length_limit,omitempty" yaml:"attribute_value_length_limit,omitempty" mapstructure:"attribute_value_length_limit,omitempty"`
}

type LogRecordProcessor struct {
	// Batch corresponds to the JSON schema field "batch".
	Batch *BatchLogRecordProcessor `json:"batch,omitempty" yaml:"batch,omitempty" mapstructure:"batch,omitempty"`

	// Simple corresponds to the JSON schema field "simple".
	Simple *SimpleLogRecordProcessor `json:"simple,omitempty" yaml:"simple,omitempty" mapstructure:"simple,omitempty"`

	AdditionalProperties interface{}
}

type LoggerProvider struct {
	// Limits corresponds to the JSON schema field "limits".
	Limits *LogRecordLimits `json:"limits,omitempty" yaml:"limits,omitempty" mapstructure:"limits,omitempty"`

	// Processors corresponds to the JSON schema field "processors".
	Processors []LogRecordProcessor `json:"processors,omitempty" yaml:"processors,omitempty" mapstructure:"processors,omitempty"`
}

type MeterProvider struct {
	// Readers corresponds to the JSON schema field "readers".
	Readers []MetricReader `json:"readers,omitempty" yaml:"readers,omitempty" mapstructure:"readers,omitempty"`

	// Views corresponds to the JSON schema field "views".
	Views []View `json:"views,omitempty" yaml:"views,omitempty" mapstructure:"views,omitempty"`
}

type MetricExporter struct {
	// Console corresponds to the JSON schema field "console".
	Console Console `json:"console,omitempty" yaml:"console,omitempty" mapstructure:"console,omitempty"`

	// OTLP corresponds to the JSON schema field "otlp".
	OTLP *OTLPMetric `json:"otlp,omitempty" yaml:"otlp,omitempty" mapstructure:"otlp,omitempty"`

	// Prometheus corresponds to the JSON schema field "prometheus".
	Prometheus *Prometheus `json:"prometheus,omitempty" yaml:"prometheus,omitempty" mapstructure:"prometheus,omitempty"`

	AdditionalProperties interface{}
}

type MetricReader struct {
	// Periodic corresponds to the JSON schema field "periodic".
	Periodic *PeriodicMetricReader `json:"periodic,omitempty" yaml:"periodic,omitempty" mapstructure:"periodic,omitempty"`

	// Pull corresponds to the JSON schema field "pull".
	Pull *PullMetricReader `json:"pull,omitempty" yaml:"pull,omitempty" mapstructure:"pull,omitempty"`
}

type OTLP struct {
	// Certificate corresponds to the JSON schema field "certificate".
	Certificate *string `json:"certificate,omitempty" yaml:"certificate,omitempty" mapstructure:"certificate,omitempty"`

	// ClientCertificate corresponds to the JSON schema field "client_certificate".
	ClientCertificate *string `json:"client_certificate,omitempty" yaml:"client_certificate,omitempty" mapstructure:"client_certificate,omitempty"`

	// ClientKey corresponds to the JSON schema field "client_key".
	ClientKey *string `json:"client_key,omitempty" yaml:"client_key,omitempty" mapstructure:"client_key,omitempty"`

	// Compression corresponds to the JSON schema field "compression".
	Compression *string `json:"compression,omitempty" yaml:"compression,omitempty" mapstructure:"compression,omitempty"`

	// Endpoint corresponds to the JSON schema field "endpoint".
	Endpoint string `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`

	// Headers corresponds to the JSON schema field "headers".
	Headers Headers `json:"headers,omitempty" yaml:"headers,omitempty" mapstructure:"headers,omitempty"`

	// Insecure corresponds to the JSON schema field "insecure".
	Insecure *bool `json:"insecure,omitempty" yaml:"insecure,omitempty" mapstructure:"insecure,omitempty"`

	// Protocol corresponds to the JSON schema field "protocol".
	Protocol string `json:"protocol" yaml:"protocol" mapstructure:"protocol"`

	// Timeout corresponds to the JSON schema field "timeout".
	Timeout *int `json:"timeout,omitempty" yaml:"timeout,omitempty" mapstructure:"timeout,omitempty"`
}

type OTLPMetric struct {
	// Certificate corresponds to the JSON schema field "certificate".
	Certificate *string `json:"certificate,omitempty" yaml:"certificate,omitempty" mapstructure:"certificate,omitempty"`

	// ClientCertificate corresponds to the JSON schema field "client_certificate".
	ClientCertificate *string `json:"client_certificate,omitempty" yaml:"client_certificate,omitempty" mapstructure:"client_certificate,omitempty"`

	// ClientKey corresponds to the JSON schema field "client_key".
	ClientKey *string `json:"client_key,omitempty" yaml:"client_key,omitempty" mapstructure:"client_key,omitempty"`

	// Compression corresponds to the JSON schema field "compression".
	Compression *string `json:"compression,omitempty" yaml:"compression,omitempty" mapstructure:"compression,omitempty"`

	// DefaultHistogramAggregation corresponds to the JSON schema field
	// "default_histogram_aggregation".
	DefaultHistogramAggregation *OTLPMetricDefaultHistogramAggregation `json:"default_histogram_aggregation,omitempty" yaml:"default_histogram_aggregation,omitempty" mapstructure:"default_histogram_aggregation,omitempty"`

	// Endpoint corresponds to the JSON schema field "endpoint".
	Endpoint string `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`

	// Headers corresponds to the JSON schema field "headers".
	Headers Headers `json:"headers,omitempty" yaml:"headers,omitempty" mapstructure:"headers,omitempty"`

	// Insecure corresponds to the JSON schema field "insecure".
	Insecure *bool `json:"insecure,omitempty" yaml:"insecure,omitempty" mapstructure:"insecure,omitempty"`

	// Protocol corresponds to the JSON schema field "protocol".
	Protocol string `json:"protocol" yaml:"protocol" mapstructure:"protocol"`

	// TemporalityPreference corresponds to the JSON schema field
	// "temporality_preference".
	TemporalityPreference *string `json:"temporality_preference,omitempty" yaml:"temporality_preference,omitempty" mapstructure:"temporality_preference,omitempty"`

	// Timeout corresponds to the JSON schema field "timeout".
	Timeout *int `json:"timeout,omitempty" yaml:"timeout,omitempty" mapstructure:"timeout,omitempty"`
}

type OTLPMetricDefaultHistogramAggregation string

const OTLPMetricDefaultHistogramAggregationBase2ExponentialBucketHistogram OTLPMetricDefaultHistogramAggregation = "base2_exponential_bucket_histogram"
const OTLPMetricDefaultHistogramAggregationExplicitBucketHistogram OTLPMetricDefaultHistogramAggregation = "explicit_bucket_histogram"

var enumValues_OTLPMetricDefaultHistogramAggregation = []interface{}{
	"explicit_bucket_histogram",
	"base2_exponential_bucket_histogram",
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OTLPMetricDefaultHistogramAggregation) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	var ok bool
	for _, expected := range enumValues_OTLPMetricDefaultHistogramAggregation {
		if reflect.DeepEqual(v, expected) {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("invalid value (expected one of %#v): %#v", enumValues_OTLPMetricDefaultHistogramAggregation, v)
	}
	*j = OTLPMetricDefaultHistogramAggregation(v)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OTLPMetric) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["endpoint"]; raw != nil && !ok {
		return fmt.Errorf("field endpoint in OTLPMetric: required")
	}
	if _, ok := raw["protocol"]; raw != nil && !ok {
		return fmt.Errorf("field protocol in OTLPMetric: required")
	}
	type Plain OTLPMetric
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = OTLPMetric(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OTLP) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["endpoint"]; raw != nil && !ok {
		return fmt.Errorf("field endpoint in OTLP: required")
	}
	if _, ok := raw["protocol"]; raw != nil && !ok {
		return fmt.Errorf("field protocol in OTLP: required")
	}
	type Plain OTLP
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = OTLP(plain)
	return nil
}

type OpenTelemetryConfiguration struct {
	// AttributeLimits corresponds to the JSON schema field "attribute_limits".
	AttributeLimits *AttributeLimits `json:"attribute_limits,omitempty" yaml:"attribute_limits,omitempty" mapstructure:"attribute_limits,omitempty"`

	// Disabled corresponds to the JSON schema field "disabled".
	Disabled *bool `json:"disabled,omitempty" yaml:"disabled,omitempty" mapstructure:"disabled,omitempty"`

	// FileFormat corresponds to the JSON schema field "file_format".
	FileFormat string `json:"file_format" yaml:"file_format" mapstructure:"file_format"`

	// LoggerProvider corresponds to the JSON schema field "logger_provider".
	LoggerProvider *LoggerProvider `json:"logger_provider,omitempty" yaml:"logger_provider,omitempty" mapstructure:"logger_provider,omitempty"`

	// MeterProvider corresponds to the JSON schema field "meter_provider".
	MeterProvider *MeterProvider `json:"meter_provider,omitempty" yaml:"meter_provider,omitempty" mapstructure:"meter_provider,omitempty"`

	// Propagator corresponds to the JSON schema field "propagator".
	Propagator *Propagator `json:"propagator,omitempty" yaml:"propagator,omitempty" mapstructure:"propagator,omitempty"`

	// Resource corresponds to the JSON schema field "resource".
	Resource *Resource `json:"resource,omitempty" yaml:"resource,omitempty" mapstructure:"resource,omitempty"`

	// TracerProvider corresponds to the JSON schema field "tracer_provider".
	TracerProvider *TracerProvider `json:"tracer_provider,omitempty" yaml:"tracer_provider,omitempty" mapstructure:"tracer_provider,omitempty"`

	AdditionalProperties interface{}
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OpenTelemetryConfiguration) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["file_format"]; raw != nil && !ok {
		return fmt.Errorf("field file_format in OpenTelemetryConfiguration: required")
	}
	type Plain OpenTelemetryConfiguration
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = OpenTelemetryConfiguration(plain)
	return nil
}

type PeriodicMetricReader struct {
	// Exporter corresponds to the JSON schema field "exporter".
	Exporter MetricExporter `json:"exporter" yaml:"exporter" mapstructure:"exporter"`

	// Interval corresponds to the JSON schema field "interval".
	Interval *int `json:"interval,omitempty" yaml:"interval,omitempty" mapstructure:"interval,omitempty"`

	// Timeout corresponds to the JSON schema field "timeout".
	Timeout *int `json:"timeout,omitempty" yaml:"timeout,omitempty" mapstructure:"timeout,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *PeriodicMetricReader) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return fmt.Errorf("field exporter in PeriodicMetricReader: required")
	}
	type Plain PeriodicMetricReader
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = PeriodicMetricReader(plain)
	return nil
}

type Prometheus struct {
	// Host corresponds to the JSON schema field "host".
	Host *string `json:"host,omitempty" yaml:"host,omitempty" mapstructure:"host,omitempty"`

	// Port corresponds to the JSON schema field "port".
	Port *int `json:"port,omitempty" yaml:"port,omitempty" mapstructure:"port,omitempty"`

	// WithResourceConstantLabels corresponds to the JSON schema field
	// "with_resource_constant_labels".
	WithResourceConstantLabels *IncludeExclude `json:"with_resource_constant_labels,omitempty" yaml:"with_resource_constant_labels,omitempty" mapstructure:"with_resource_constant_labels,omitempty"`

	// WithoutScopeInfo corresponds to the JSON schema field "without_scope_info".
	WithoutScopeInfo *bool `json:"without_scope_info,omitempty" yaml:"without_scope_info,omitempty" mapstructure:"without_scope_info,omitempty"`

	// WithoutTypeSuffix corresponds to the JSON schema field "without_type_suffix".
	WithoutTypeSuffix *bool `json:"without_type_suffix,omitempty" yaml:"without_type_suffix,omitempty" mapstructure:"without_type_suffix,omitempty"`

	// WithoutUnits corresponds to the JSON schema field "without_units".
	WithoutUnits *bool `json:"without_units,omitempty" yaml:"without_units,omitempty" mapstructure:"without_units,omitempty"`
}

type Propagator struct {
	// Composite corresponds to the JSON schema field "composite".
	Composite []string `json:"composite,omitempty" yaml:"composite,omitempty" mapstructure:"composite,omitempty"`

	AdditionalProperties interface{}
}

type PullMetricReader struct {
	// Exporter corresponds to the JSON schema field "exporter".
	Exporter MetricExporter `json:"exporter" yaml:"exporter" mapstructure:"exporter"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *PullMetricReader) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return fmt.Errorf("field exporter in PullMetricReader: required")
	}
	type Plain PullMetricReader
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = PullMetricReader(plain)
	return nil
}

type Resource struct {
	// Attributes corresponds to the JSON schema field "attributes".
	Attributes Attributes `json:"attributes,omitempty" yaml:"attributes,omitempty" mapstructure:"attributes,omitempty"`

	// Detectors corresponds to the JSON schema field "detectors".
	Detectors *Detectors `json:"detectors,omitempty" yaml:"detectors,omitempty" mapstructure:"detectors,omitempty"`

	// SchemaUrl corresponds to the JSON schema field "schema_url".
	SchemaUrl *string `json:"schema_url,omitempty" yaml:"schema_url,omitempty" mapstructure:"schema_url,omitempty"`
}

type Sampler struct {
	// AlwaysOff corresponds to the JSON schema field "always_off".
	AlwaysOff SamplerAlwaysOff `json:"always_off,omitempty" yaml:"always_off,omitempty" mapstructure:"always_off,omitempty"`

	// AlwaysOn corresponds to the JSON schema field "always_on".
	AlwaysOn SamplerAlwaysOn `json:"always_on,omitempty" yaml:"always_on,omitempty" mapstructure:"always_on,omitempty"`

	// JaegerRemote corresponds to the JSON schema field "jaeger_remote".
	JaegerRemote *SamplerJaegerRemote `json:"jaeger_remote,omitempty" yaml:"jaeger_remote,omitempty" mapstructure:"jaeger_remote,omitempty"`

	// ParentBased corresponds to the JSON schema field "parent_based".
	ParentBased *SamplerParentBased `json:"parent_based,omitempty" yaml:"parent_based,omitempty" mapstructure:"parent_based,omitempty"`

	// TraceIDRatioBased corresponds to the JSON schema field "trace_id_ratio_based".
	TraceIDRatioBased *SamplerTraceIDRatioBased `json:"trace_id_ratio_based,omitempty" yaml:"trace_id_ratio_based,omitempty" mapstructure:"trace_id_ratio_based,omitempty"`

	AdditionalProperties interface{}
}

type SamplerAlwaysOff map[string]interface{}

type SamplerAlwaysOn map[string]interface{}

type SamplerJaegerRemote struct {
	// Endpoint corresponds to the JSON schema field "endpoint".
	Endpoint *string `json:"endpoint,omitempty" yaml:"endpoint,omitempty" mapstructure:"endpoint,omitempty"`

	// InitialSampler corresponds to the JSON schema field "initial_sampler".
	InitialSampler *Sampler `json:"initial_sampler,omitempty" yaml:"initial_sampler,omitempty" mapstructure:"initial_sampler,omitempty"`

	// Interval corresponds to the JSON schema field "interval".
	Interval *int `json:"interval,omitempty" yaml:"interval,omitempty" mapstructure:"interval,omitempty"`
}

type SamplerParentBased struct {
	// LocalParentNotSampled corresponds to the JSON schema field
	// "local_parent_not_sampled".
	LocalParentNotSampled *Sampler `json:"local_parent_not_sampled,omitempty" yaml:"local_parent_not_sampled,omitempty" mapstructure:"local_parent_not_sampled,omitempty"`

	// LocalParentSampled corresponds to the JSON schema field "local_parent_sampled".
	LocalParentSampled *Sampler `json:"local_parent_sampled,omitempty" yaml:"local_parent_sampled,omitempty" mapstructure:"local_parent_sampled,omitempty"`

	// RemoteParentNotSampled corresponds to the JSON schema field
	// "remote_parent_not_sampled".
	RemoteParentNotSampled *Sampler `json:"remote_parent_not_sampled,omitempty" yaml:"remote_parent_not_sampled,omitempty" mapstructure:"remote_parent_not_sampled,omitempty"`

	// RemoteParentSampled corresponds to the JSON schema field
	// "remote_parent_sampled".
	RemoteParentSampled *Sampler `json:"remote_parent_sampled,omitempty" yaml:"remote_parent_sampled,omitempty" mapstructure:"remote_parent_sampled,omitempty"`

	// Root corresponds to the JSON schema field "root".
	Root *Sampler `json:"root,omitempty" yaml:"root,omitempty" mapstructure:"root,omitempty"`
}

type SamplerTraceIDRatioBased struct {
	// Ratio corresponds to the JSON schema field "ratio".
	Ratio *float64 `json:"ratio,omitempty" yaml:"ratio,omitempty" mapstructure:"ratio,omitempty"`
}

type SimpleLogRecordProcessor struct {
	// Exporter corresponds to the JSON schema field "exporter".
	Exporter LogRecordExporter `json:"exporter" yaml:"exporter" mapstructure:"exporter"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SimpleLogRecordProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return fmt.Errorf("field exporter in SimpleLogRecordProcessor: required")
	}
	type Plain SimpleLogRecordProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = SimpleLogRecordProcessor(plain)
	return nil
}

type SimpleSpanProcessor struct {
	// Exporter corresponds to the JSON schema field "exporter".
	Exporter SpanExporter `json:"exporter" yaml:"exporter" mapstructure:"exporter"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SimpleSpanProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return fmt.Errorf("field exporter in SimpleSpanProcessor: required")
	}
	type Plain SimpleSpanProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = SimpleSpanProcessor(plain)
	return nil
}

type SpanExporter struct {
	// Console corresponds to the JSON schema field "console".
	Console Console `json:"console,omitempty" yaml:"console,omitempty" mapstructure:"console,omitempty"`

	// OTLP corresponds to the JSON schema field "otlp".
	OTLP *OTLP `json:"otlp,omitempty" yaml:"otlp,omitempty" mapstructure:"otlp,omitempty"`

	// Zipkin corresponds to the JSON schema field "zipkin".
	Zipkin *Zipkin `json:"zipkin,omitempty" yaml:"zipkin,omitempty" mapstructure:"zipkin,omitempty"`

	AdditionalProperties interface{}
}

type SpanLimits struct {
	// AttributeCountLimit corresponds to the JSON schema field
	// "attribute_count_limit".
	AttributeCountLimit *int `json:"attribute_count_limit,omitempty" yaml:"attribute_count_limit,omitempty" mapstructure:"attribute_count_limit,omitempty"`

	// AttributeValueLengthLimit corresponds to the JSON schema field
	// "attribute_value_length_limit".
	AttributeValueLengthLimit *int `json:"attribute_value_length_limit,omitempty" yaml:"attribute_value_length_limit,omitempty" mapstructure:"attribute_value_length_limit,omitempty"`

	// EventAttributeCountLimit corresponds to the JSON schema field
	// "event_attribute_count_limit".
	EventAttributeCountLimit *int `json:"event_attribute_count_limit,omitempty" yaml:"event_attribute_count_limit,omitempty" mapstructure:"event_attribute_count_limit,omitempty"`

	// EventCountLimit corresponds to the JSON schema field "event_count_limit".
	EventCountLimit *int `json:"event_count_limit,omitempty" yaml:"event_count_limit,omitempty" mapstructure:"event_count_limit,omitempty"`

	// LinkAttributeCountLimit corresponds to the JSON schema field
	// "link_attribute_count_limit".
	LinkAttributeCountLimit *int `json:"link_attribute_count_limit,omitempty" yaml:"link_attribute_count_limit,omitempty" mapstructure:"link_attribute_count_limit,omitempty"`

	// LinkCountLimit corresponds to the JSON schema field "link_count_limit".
	LinkCountLimit *int `json:"link_count_limit,omitempty" yaml:"link_count_limit,omitempty" mapstructure:"link_count_limit,omitempty"`
}

type SpanProcessor struct {
	// Batch corresponds to the JSON schema field "batch".
	Batch *BatchSpanProcessor `json:"batch,omitempty" yaml:"batch,omitempty" mapstructure:"batch,omitempty"`

	// Simple corresponds to the JSON schema field "simple".
	Simple *SimpleSpanProcessor `json:"simple,omitempty" yaml:"simple,omitempty" mapstructure:"simple,omitempty"`

	AdditionalProperties interface{}
}

type TracerProvider struct {
	// Limits corresponds to the JSON schema field "limits".
	Limits *SpanLimits `json:"limits,omitempty" yaml:"limits,omitempty" mapstructure:"limits,omitempty"`

	// Processors corresponds to the JSON schema field "processors".
	Processors []SpanProcessor `json:"processors,omitempty" yaml:"processors,omitempty" mapstructure:"processors,omitempty"`

	// Sampler corresponds to the JSON schema field "sampler".
	Sampler *Sampler `json:"sampler,omitempty" yaml:"sampler,omitempty" mapstructure:"sampler,omitempty"`
}

type View struct {
	// Selector corresponds to the JSON schema field "selector".
	Selector *ViewSelector `json:"selector,omitempty" yaml:"selector,omitempty" mapstructure:"selector,omitempty"`

	// Stream corresponds to the JSON schema field "stream".
	Stream *ViewStream `json:"stream,omitempty" yaml:"stream,omitempty" mapstructure:"stream,omitempty"`
}

type ViewSelector struct {
	// InstrumentName corresponds to the JSON schema field "instrument_name".
	InstrumentName *string `json:"instrument_name,omitempty" yaml:"instrument_name,omitempty" mapstructure:"instrument_name,omitempty"`

	// InstrumentType corresponds to the JSON schema field "instrument_type".
	InstrumentType *ViewSelectorInstrumentType `json:"instrument_type,omitempty" yaml:"instrument_type,omitempty" mapstructure:"instrument_type,omitempty"`

	// MeterName corresponds to the JSON schema field "meter_name".
	MeterName *string `json:"meter_name,omitempty" yaml:"meter_name,omitempty" mapstructure:"meter_name,omitempty"`

	// MeterSchemaUrl corresponds to the JSON schema field "meter_schema_url".
	MeterSchemaUrl *string `json:"meter_schema_url,omitempty" yaml:"meter_schema_url,omitempty" mapstructure:"meter_schema_url,omitempty"`

	// MeterVersion corresponds to the JSON schema field "meter_version".
	MeterVersion *string `json:"meter_version,omitempty" yaml:"meter_version,omitempty" mapstructure:"meter_version,omitempty"`

	// Unit corresponds to the JSON schema field "unit".
	Unit *string `json:"unit,omitempty" yaml:"unit,omitempty" mapstructure:"unit,omitempty"`
}

type ViewSelectorInstrumentType string

const ViewSelectorInstrumentTypeCounter ViewSelectorInstrumentType = "counter"
const ViewSelectorInstrumentTypeHistogram ViewSelectorInstrumentType = "histogram"
const ViewSelectorInstrumentTypeObservableCounter ViewSelectorInstrumentType = "observable_counter"
const ViewSelectorInstrumentTypeObservableGauge ViewSelectorInstrumentType = "observable_gauge"
const ViewSelectorInstrumentTypeObservableUpDownCounter ViewSelectorInstrumentType = "observable_up_down_counter"
const ViewSelectorInstrumentTypeUpDownCounter ViewSelectorInstrumentType = "up_down_counter"

var enumValues_ViewSelectorInstrumentType = []interface{}{
	"counter",
	"histogram",
	"observable_counter",
	"observable_gauge",
	"observable_up_down_counter",
	"up_down_counter",
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ViewSelectorInstrumentType) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	var ok bool
	for _, expected := range enumValues_ViewSelectorInstrumentType {
		if reflect.DeepEqual(v, expected) {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("invalid value (expected one of %#v): %#v", enumValues_ViewSelectorInstrumentType, v)
	}
	*j = ViewSelectorInstrumentType(v)
	return nil
}

type ViewStream struct {
	// Aggregation corresponds to the JSON schema field "aggregation".
	Aggregation *ViewStreamAggregation `json:"aggregation,omitempty" yaml:"aggregation,omitempty" mapstructure:"aggregation,omitempty"`

	// AttributeKeys corresponds to the JSON schema field "attribute_keys".
	AttributeKeys []string `json:"attribute_keys,omitempty" yaml:"attribute_keys,omitempty" mapstructure:"attribute_keys,omitempty"`

	// Description corresponds to the JSON schema field "description".
	Description *string `json:"description,omitempty" yaml:"description,omitempty" mapstructure:"description,omitempty"`

	// Name corresponds to the JSON schema field "name".
	Name *string `json:"name,omitempty" yaml:"name,omitempty" mapstructure:"name,omitempty"`
}

type ViewStreamAggregation struct {
	// Base2ExponentialBucketHistogram corresponds to the JSON schema field
	// "base2_exponential_bucket_histogram".
	Base2ExponentialBucketHistogram *ViewStreamAggregationBase2ExponentialBucketHistogram `json:"base2_exponential_bucket_histogram,omitempty" yaml:"base2_exponential_bucket_histogram,omitempty" mapstructure:"base2_exponential_bucket_histogram,omitempty"`

	// Default corresponds to the JSON schema field "default".
	Default ViewStreamAggregationDefault `json:"default,omitempty" yaml:"default,omitempty" mapstructure:"default,omitempty"`

	// Drop corresponds to the JSON schema field "drop".
	Drop ViewStreamAggregationDrop `json:"drop,omitempty" yaml:"drop,omitempty" mapstructure:"drop,omitempty"`

	// ExplicitBucketHistogram corresponds to the JSON schema field
	// "explicit_bucket_histogram".
	ExplicitBucketHistogram *ViewStreamAggregationExplicitBucketHistogram `json:"explicit_bucket_histogram,omitempty" yaml:"explicit_bucket_histogram,omitempty" mapstructure:"explicit_bucket_histogram,omitempty"`

	// LastValue corresponds to the JSON schema field "last_value".
	LastValue ViewStreamAggregationLastValue `json:"last_value,omitempty" yaml:"last_value,omitempty" mapstructure:"last_value,omitempty"`

	// Sum corresponds to the JSON schema field "sum".
	Sum ViewStreamAggregationSum `json:"sum,omitempty" yaml:"sum,omitempty" mapstructure:"sum,omitempty"`
}

type ViewStreamAggregationBase2ExponentialBucketHistogram struct {
	// MaxScale corresponds to the JSON schema field "max_scale".
	MaxScale *int `json:"max_scale,omitempty" yaml:"max_scale,omitempty" mapstructure:"max_scale,omitempty"`

	// MaxSize corresponds to the JSON schema field "max_size".
	MaxSize *int `json:"max_size,omitempty" yaml:"max_size,omitempty" mapstructure:"max_size,omitempty"`

	// RecordMinMax corresponds to the JSON schema field "record_min_max".
	RecordMinMax *bool `json:"record_min_max,omitempty" yaml:"record_min_max,omitempty" mapstructure:"record_min_max,omitempty"`
}

type ViewStreamAggregationDefault map[string]interface{}

type ViewStreamAggregationDrop map[string]interface{}

type ViewStreamAggregationExplicitBucketHistogram struct {
	// Boundaries corresponds to the JSON schema field "boundaries".
	Boundaries []float64 `json:"boundaries,omitempty" yaml:"boundaries,omitempty" mapstructure:"boundaries,omitempty"`

	// RecordMinMax corresponds to the JSON schema field "record_min_max".
	RecordMinMax *bool `json:"record_min_max,omitempty" yaml:"record_min_max,omitempty" mapstructure:"record_min_max,omitempty"`
}

type ViewStreamAggregationLastValue map[string]interface{}

type ViewStreamAggregationSum map[string]interface{}

type Zipkin struct {
	// Endpoint corresponds to the JSON schema field "endpoint".
	Endpoint string `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`

	// Timeout corresponds to the JSON schema field "timeout".
	Timeout *int `json:"timeout,omitempty" yaml:"timeout,omitempty" mapstructure:"timeout,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *Zipkin) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["endpoint"]; raw != nil && !ok {
		return fmt.Errorf("field endpoint in Zipkin: required")
	}
	type Plain Zipkin
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = Zipkin(plain)
	return nil
}
