package stats

import (
	"github.com/honeycombio/opencensus-exporter/honeycomb"
)

type HoneycombConfig struct {
	APIKey           string
	Dataset          string
	TraceProbability float64
}

func NewHoneycombExporter(conf HoneycombConfig) *honeycomb.Exporter {
	e := honeycomb.NewExporter(conf.APIKey, conf.Dataset)
	e.SampleFraction = conf.TraceProbability

	return e
}
