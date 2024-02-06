package metric

import (
	// "time"
	"bytes"

	"github.com/prometheus/common/expfmt"
	dto "github.com/prometheus/client_model/go"
)

func GenerateGaugeMetric(metricName string, description string, values []float64, labels []map[string]string) string {
	metrics := []*dto.Metric{}
	// timestampMs := nowUTC.UnixNano() / int64(time.Millisecond)

	for i := range values {
		labelPairs := []*dto.LabelPair{}
		for k, v := range labels[i] {
			k := k
			v := v
			labelPairs = append(labelPairs, &dto.LabelPair{Name: &k, Value: &v})
		}

		metric := &dto.Metric{
			Gauge:       &dto.Gauge{Value: &values[i]},
			Label:       labelPairs,
			// TimestampMs: timestampMs,
		}
		metrics = append(metrics, metric)
	}

	metricFamily := &dto.MetricFamily{
		Name: &metricName,
		Help: &description,
		Type: dto.MetricType_GAUGE.Enum(),
		Metric: metrics,
	}

	var buf bytes.Buffer
	expfmt.MetricFamilyToOpenMetrics(&buf, metricFamily)
	return buf.String()
}
