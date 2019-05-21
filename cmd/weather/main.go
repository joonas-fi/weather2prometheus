package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/function61/gokit/envvar"
	"github.com/joonas-fi/weather2prometheus/pkg/openweathermap"
)

type Config struct {
	WeatherZipCode       string
	WeatherCountryCode   string
	OpenWeatherMapApiKey string
	PromPipeEndpoint     string
	PromPipeAuthToken    string
}

func sendWeatherObservationToPrompipe() error {
	conf, err := getConfig()
	if err != nil {
		return err
	}

	owm := openweathermap.New(conf.OpenWeatherMapApiKey)

	ctx, cancel := context.WithTimeout(context.TODO(), openweathermap.DefaultTimeout)
	defer cancel()

	observation, err := owm.GetWeather(ctx, conf.WeatherCountryCode, conf.WeatherZipCode)
	if err != nil {
		return err
	}

	wireBytes, err := weatherAsPrometheusTextExposition(*observation, conf.WeatherZipCode)
	if err != nil {
		return err
	}

	return promPipeSend(
		conf.PromPipeEndpoint,
		wireBytes,
		conf.PromPipeAuthToken)
}

// this handler is driven by Cloudwatch scheduled event
func weatherHandler(ctx context.Context, req events.CloudWatchEvent) error {
	return sendWeatherObservationToPrompipe()
}

func main() {
	/*
		if len(os.Args) == 2 && os.Args[1] == "dev" {
			if err := sendWeatherObservationToPrompipe(); err != nil {
				panic(err)
			}
			return
		}
	*/
	lambda.Start(weatherHandler)
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
