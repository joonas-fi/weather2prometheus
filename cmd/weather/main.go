package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/function61/gokit/aws/lambdautils"
	"github.com/function61/gokit/envvar"
	"github.com/function61/gokit/httputils"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/ossignal"
	"github.com/function61/gokit/promconstmetrics"
	"github.com/function61/gokit/taskrunner"
	"github.com/function61/prompipe/pkg/prompipeclient"
	"github.com/joonas-fi/weather2prometheus/pkg/openweathermap"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	promContentType = "text/plain; version=0.0.4; charset=utf-8"
)

type Config struct {
	OpenWeatherMapApiKey string
}

type OpenWeatherMapLocation struct {
	CountryCode string
	ZipCode     string
}

func main() {
	handler, err := newServerHandler()
	exitIfError(err)

	if lambdautils.InLambda() {
		lambda.Start(lambdautils.NewLambdaHttpHandlerAdapter(handler))
		return
	}

	logger := logex.StandardLogger()

	exitIfError(runStandaloneServer(
		ossignal.InterruptOrTerminateBackgroundCtx(logger),
		handler,
		logger))
}

func newServerHandler() (http.Handler, error) {
	conf, err := getConfig()
	if err != nil {
		return nil, err
	}

	routes := http.NewServeMux()

	routes.HandleFunc("/weather/{country}/{zip}/metrics", func(w http.ResponseWriter, r *http.Request) {
		loc := OpenWeatherMapLocation{r.PathValue("country"), r.PathValue("zip")}

		weatherMetricsReg, err := weather2prometheus(r.Context(), loc, conf)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		expositionOutput := &bytes.Buffer{}

		if err := prompipeclient.GatherToTextExport(weatherMetricsReg, expositionOutput); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", promContentType)

		fmt.Fprintln(w, expositionOutput.String())
	})

	return routes, nil
}

func pushObservationToPrometheusCollector(
	observation openweathermap.Observation,
	countryCode string,
	zipCode string,
	weatherMetrics *promconstmetrics.Collector,
) {
	ts := observation.GetTimestamp()

	push := func(key string, val float64) {
		weatherMetrics.Observe(weatherMetrics.Register(key, "", prometheus.Labels{
			"loc": fmt.Sprintf("%s/%s", countryCode, zipCode),
		}), val, ts)
	}

	push("weather_temperature", observation.Main.Temperature)
	push("weather_airpressure", float64(observation.Main.AirPressure))
	push("weather_relhumidity", float64(observation.Main.RelativeHumidity))
	push("weather_windspeed", observation.Wind.Speed)
	push("weather_winddirection", float64(observation.Wind.Direction))
}

func weather2prometheus(
	ctx context.Context,
	loc OpenWeatherMapLocation,
	conf *Config,
) (*prometheus.Registry, error) {
	openWeatherMap := openweathermap.New(conf.OpenWeatherMapApiKey)

	observation, err := func() (*openweathermap.Observation, error) {
		ctx, cancel := context.WithTimeout(ctx, openweathermap.DefaultTimeout)
		defer cancel()

		return openWeatherMap.GetWeather(ctx, loc.CountryCode, loc.ZipCode)
	}()
	if err != nil {
		return nil, err
	}

	weatherMetrics := promconstmetrics.NewCollector()
	weatherMetricsReg := prometheus.NewRegistry()
	if err := weatherMetricsReg.Register(weatherMetrics); err != nil {
		return nil, err
	}

	pushObservationToPrometheusCollector(
		*observation,
		loc.CountryCode,
		loc.ZipCode,
		weatherMetrics)

	return weatherMetricsReg, nil
}

func getConfig() (*Config, error) {
	var validationError error
	getRequiredEnv := func(key string) string {
		val, err := envvar.Required(key)
		if err != nil {
			validationError = err
		}

		return val
	}

	return &Config{
		OpenWeatherMapApiKey: getRequiredEnv("OPENWEATHERMAP_APIKEY"),
	}, validationError
}

func runStandaloneServer(ctx context.Context, handler http.Handler, logger *log.Logger) error {
	srv := &http.Server{
		Addr:              ":80",
		Handler:           handler,
		ReadHeaderTimeout: 60 * time.Second, // same as nginx
	}

	tasks := taskrunner.New(ctx, logger)

	tasks.Start("listener "+srv.Addr, func(_ context.Context, _ string) error {
		return httputils.RemoveGracefulServerClosedError(srv.ListenAndServe())
	})

	tasks.Start("listenershutdowner", httputils.ServerShutdownTask(srv))

	return tasks.Wait()
}

func exitIfError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
