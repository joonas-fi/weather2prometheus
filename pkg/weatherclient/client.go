package weatherclient

import (
	"context"

	"github.com/function61/gokit/net/http/ezhttp"
	"github.com/joonas-fi/weather2prometheus/pkg/weathermodel"
)

const (
	Function61 = "https://function61.com"
)

type Client struct {
	baseURL     string
	bearerToken string
}

func New(endpoint string, bearerToken string) *Client {
	return &Client{
		baseURL:     endpoint,
		bearerToken: bearerToken,
	}
}

func (c *Client) Weather(ctx context.Context, country string, zip string) (*weathermodel.Observation, error) {
	obs := &weathermodel.Observation{}
	_, err := ezhttp.Get(ctx, c.baseURL+"/weather/api/"+country+"/"+zip, ezhttp.RespondsJSONAllowUnknownFields(obs))
	return obs, err
}
