package openweathermap

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/function61/gokit/ezhttp"
)

const (
	DefaultTimeout = 10 * time.Second
)

type Observation struct {
	Main struct {
		Temperature      float64 `json:"temp"`     // [°C]
		AirPressure      int     `json:"pressure"` // [hPa]
		RelativeHumidity int     `json:"humidity"` // [%]
	} `json:"main"`
	Wind struct {
		Speed     float64 `json:"speed"` // [m/s]
		Direction int     `json:"deg"`   // [°]
	}
	Code      int   `json:"cod"` // HTTP code, in application level (genius)
	Timestamp int64 `json:"dt"`  // unix timestamp
}

func (o *Observation) GetTimestamp() time.Time {
	return time.Unix(o.Timestamp, 0)
}

type Client struct {
	apiKey string
}

func New(apiKey string) *Client {
	return &Client{apiKey}
}

func (c *Client) GetWeather(ctx context.Context, countryCode string, zipCode string) (*Observation, error) {
	url := fmt.Sprintf(
		"https://api.openweathermap.org/data/2.5/weather?zip=%s,%s&APPID=%s&units=metric",
		zipCode,
		countryCode,
		c.apiKey)

	result := &Observation{}
	if _, err := ezhttp.Get(ctx, url, ezhttp.RespondsJson(result, true)); err != nil {
		return nil, err
	}

	// HTTP code within JSON - brilliant 🙄
	if result.Code != http.StatusOK {
		return nil, fmt.Errorf("result code in JSON not OK; was %d", result.Code)
	}

	return result, nil
}
