package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/function61/gokit/envvar"
	"github.com/function61/gokit/promconstmetrics"
	"github.com/function61/prompipe/pkg/prompipeclient"
	"github.com/joonas-fi/weather2prometheus/pkg/openweathermap"
	"github.com/prometheus/client_golang/prometheus"
	"os"
)

type Config struct {
	WeatherZipCode       string
	WeatherCountryCode   string
	OpenWeatherMapApiKey string
	PromPipeEndpoint     string
	PromPipeAuthToken    string
}

func pushObservationToPrometheusCollector(
	observation openweathermap.Observation,
	zipCode string,
	weatherMetrics *promconstmetrics.Collector,
) {
	ts := observation.GetTimestamp()

	push := func(key string, val float64) {
		weatherMetrics.Observe(weatherMetrics.Register(key, "", prometheus.Labels{
			"loc": zipCode,
		}), val, ts)
	}

	push("weather_temperature", observation.Main.Temperature)
	push("weather_airpressure", float64(observation.Main.AirPressure))
	push("weather_relhumidity", float64(observation.Main.RelativeHumidity))
	push("weather_windspeed", observation.Wind.Speed)
	push("weather_winddirection", float64(observation.Wind.Direction))
}

func pushToPromPipe(ctx context.Context, allMetrics *prometheus.Registry, conf Config) error {
	return prompipeclient.New(conf.PromPipeEndpoint, conf.PromPipeAuthToken).Send(ctx, allMetrics)
}

func printMetrics(ctx context.Context, allMetrics *prometheus.Registry, conf Config) error {
	expositionOutput := &bytes.Buffer{}

	if err := prompipeclient.GatherToTextExport(allMetrics, expositionOutput); err != nil {
		return err
	}

	_, err := fmt.Println(expositionOutput.String())
	return err
}

func weather2prometheus(
	ctx context.Context,
	processor func(context.Context, *prometheus.Registry, Config) error,
) error {
	conf, err := getConfig()
	if err != nil {
		return err
	}

	openWeatherMap := openweathermap.New(conf.OpenWeatherMapApiKey)

	observation, err := func() (*openweathermap.Observation, error) {
		ctx, cancel := context.WithTimeout(ctx, openweathermap.DefaultTimeout)
		defer cancel()

		return openWeatherMap.GetWeather(ctx, conf.WeatherCountryCode, conf.WeatherZipCode)
	}()
	if err != nil {
		return err
	}

	weatherMetrics := promconstmetrics.NewCollector()
	allMetrics := prometheus.NewRegistry()
	if err := allMetrics.Register(weatherMetrics); err != nil {
		return err
	}

	pushObservationToPrometheusCollector(*observation, conf.WeatherZipCode, weatherMetrics)

	return processor(ctx, allMetrics, *conf)
}

// this handler is driven by Cloudwatch scheduled event
func lambdaHandler(ctx context.Context, req events.CloudWatchEvent) error {
	return weather2prometheus(ctx, pushToPromPipe)
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "dev" {
		if err := weather2prometheus(context.Background(), printMetrics); err != nil {
			panic(err)
		}
		return
	}

	lambda.Start(lambdaHandler)
}

func getConfig() (*Config, error) {
	var validationError error
	getRequiredEnv := func(key string) string {
		val, err := envvar.Get(key)
		if err != nil {
			validationError = err
		}

		return val
	}

	return &Config{
		WeatherZipCode:       getRequiredEnv("WEATHER_ZIPCODE"),
		WeatherCountryCode:   getRequiredEnv("WEATHER_COUNTRYCODE"),
		OpenWeatherMapApiKey: getRequiredEnv("OPENWEATHERMAP_APIKEY"),
		PromPipeEndpoint:     getRequiredEnv("PROMPIPE_ENDPOINT"),
		PromPipeAuthToken:    getRequiredEnv("PROMPIPE_AUTHTOKEN"),
	}, validationError
}
