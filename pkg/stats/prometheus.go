package stats

import (
	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/gsmcwhirter/go-util/v7/errors"
)

type PrometheusConfig struct {
	Namespace string
}

func NewPrometheusExporter(conf PrometheusConfig) (*prometheus.Exporter, error) {
	var e *prometheus.Exporter
	var err error
	if e, err = prometheus.NewExporter(prometheus.Options{Namespace: conf.Namespace}); err != nil {
		return nil, errors.Wrap(err, "could not create prometheus stats exporter")
	}

	return e, nil
}
