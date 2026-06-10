package main

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/function61/gokit/app/aws/lambdautils"
	"github.com/function61/gokit/app/cli"
	"github.com/function61/gokit/app/promconstmetrics"
	"github.com/function61/gokit/net/http/httputils"
	"github.com/function61/gokit/os/osutil"
	"github.com/joonas-fi/weather2prometheus/pkg/openweathermap"
	"github.com/joonas-fi/weather2prometheus/pkg/prompipeclient"
	"github.com/joonas-fi/weather2prometheus/pkg/weathermodel"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
)

const (
	promContentType = "text/plain; version=0.0.4; charset=utf-8"
)

type Config struct {
	OpenWeatherMapAPIKey string
}

type OpenWeatherMapLocation struct {
	CountryCode string
	ZipCode     string
}

func main() {
	handler, err := newServerHandler()
	osutil.ExitIfError(err)

	if lambdautils.InLambda() {
		lambda.Start(lambdautils.NewLambdaHttpHandlerAdapter(handler))
		return
	}

	cli.Execute(&cobra.Command{
		Short: "Serve weather data",
		RunE: func(cmd *cobra.Command, _ []string) error {
			srv := &http.Server{
				Addr:              cmp.Or(os.Getenv("PORT"), ":80"),
				Handler:           handler,
				ReadHeaderTimeout: httputils.DefaultReadHeaderTimeout,
			}

			return httputils.CancelableServer(cmd.Context(), srv, srv.ListenAndServe)
		},
	})
}

func newServerHandler() (http.Handler, error) {
	conf, err := getConfig()
	if err != nil {
		return nil, err
	}

	openWeatherMap := openweathermap.New(conf.OpenWeatherMapAPIKey)

	routes := http.NewServeMux()

	routes.HandleFunc("/weather/api/{country}/{zip}", func(w http.ResponseWriter, r *http.Request) {
		loc := OpenWeatherMapLocation{r.PathValue("country"), r.PathValue("zip")}

		observation, err := func() (*openweathermap.Observation, error) {
			ctx, cancel := context.WithTimeout(r.Context(), openweathermap.DefaultTimeout)
			defer cancel()

			return openWeatherMap.GetWeather(ctx, loc.CountryCode, loc.ZipCode)
		}()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		httputils.RespondJSON(w, weathermodel.Observation{
			Timestamp:        observation.GetTimestamp(),
			Temperature:      observation.Main.Temperature,
			AirPressure:      observation.Main.AirPressure,
			RelativeHumidity: observation.Main.RelativeHumidity,
			Wind: weathermodel.WindSpec{
				Speed:     observation.Wind.Speed,
				Direction: observation.Wind.Direction,
			},
		})
	})

	metricsHandler := func(w http.ResponseWriter, r *http.Request, loc OpenWeatherMapLocation) {
		weatherMetricsReg, err := weather2prometheus(r.Context(), loc, openWeatherMap)
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
	}

	routes.HandleFunc("/weather/api/{country}/{zip}/metrics", func(w http.ResponseWriter, r *http.Request) {
		metricsHandler(w, r, OpenWeatherMapLocation{r.PathValue("country"), r.PathValue("zip")})
	})

	routes.HandleFunc("/weather/fi/{zip}/metrics", func(w http.ResponseWriter, r *http.Request) { // backwards compat
		metricsHandler(w, r, OpenWeatherMapLocation{"fi", r.PathValue("zip")})
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

func weather2prometheus(ctx context.Context, loc OpenWeatherMapLocation, openWeatherMap *openweathermap.Client) (*prometheus.Registry, error) {
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

	pushObservationToPrometheusCollector(*observation, loc.CountryCode, loc.ZipCode, weatherMetrics)

	return weatherMetricsReg, nil
}

func getConfig() (*Config, error) {
	var validationError error
	getenvRequired := func(key string) string {
		val, err := osutil.GetenvRequired(key)
		if err != nil {
			validationError = err
		}

		return val
	}

	return &Config{
		OpenWeatherMapAPIKey: getenvRequired("OPENWEATHERMAP_APIKEY"),
	}, validationError
}
