package main

import (
	"context"
	"fmt"
	"github.com/function61/gokit/ezhttp"
	"io"
)

// TODO: all of these below should be exposed by prompipe project

const (
	promContentType = "text/plain; version=0.0.4; charset=utf-8"
)

func promPipeSend(url string, wireBytes io.Reader, bearerToken string) error {
	ctx, cancel := context.WithTimeout(context.TODO(), ezhttp.DefaultTimeout10s)
	defer cancel()

	if _, err := ezhttp.Put(
		ctx,
		url,
		ezhttp.AuthBearer(bearerToken),
		ezhttp.SendBody(wireBytes, promContentType)); err != nil {
		return fmt.Errorf("PUT failed for %s: %v", url, err)
	}

	return nil
}
