package main

import (
	"bytes"
	"github.com/function61/hautomo/pkg/constmetrics"
	"github.com/joonas-fi/weather2prometheus/pkg/openweathermap"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"io"
)

func weatherAsPrometheusTextExposition(observation openweathermap.Observation, zipCode string) (*bytes.Buffer, error) {
	ts := observation.GetTimestamp()

	metrics := constmetrics.NewCollector()

	push := func(key string, val float64) {
		metrics.Observe(metrics.Register(key, "", "loc", zipCode), val, ts)
	}

	push("weather_temperature", observation.Main.Temperature)
	push("weather_airpressure", float64(observation.Main.AirPressure))
	push("weather_relhumidity", float64(observation.Main.RelativeHumidity))
	push("weather_windspeed", observation.Wind.Speed)
	push("weather_winddirection", float64(observation.Wind.Direction))

	expositionOutput := &bytes.Buffer{}

	if err := prometheusCollectorToTextExposition(metrics, expositionOutput); err != nil {
		return nil, err
	}

	return expositionOutput, nil
}

// I'll be the first one to admit that I don't actually understand why the registry is
// required, but it seemed necessary to plumb all the datatype conversions together..
func prometheusCollectorToTextExposition(collector prometheus.Collector, output io.Writer) error {
	reg := prometheus.NewRegistry()
	if err := reg.Register(collector); err != nil {
		return err
	}

	wireEncoder := expfmt.NewEncoder(output, expfmt.FmtText)

	metricFamilies, err := reg.Gather()
	if err != nil {
		return err
	}

	for _, metricFamily := range metricFamilies {
		if err := wireEncoder.Encode(metricFamily); err != nil {
			return err
		}
	}

	return nil
}
