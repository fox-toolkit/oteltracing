[![Go Reference](https://pkg.go.dev/badge/github.com/fox-toolkit/oteltracing.svg)](https://pkg.go.dev/github.com/fox-toolkit/oteltracing)
[![tests](https://github.com/fox-toolkit/oteltracing/actions/workflows/tests.yaml/badge.svg)](https://github.com/fox-toolkit/oteltracing/actions?query=workflow%3Atests)
[![codecov](https://codecov.io/gh/fox-toolkit/oteltracing/graph/badge.svg?token=yDSqVOFwtN)](https://codecov.io/gh/fox-toolkit/oteltracing)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/fox-toolkit/oteltracing)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/fox-toolkit/oteltracing)
# Oteltracing

> [!NOTE]
> This repository has been transferred from `github.com/tigerwill90/otelfox` to `github.com/fox-toolkit/oteltracing`.
> Existing users should update their imports and `go.mod` accordingly.

Oteltracing is a middleware for [Fox](https://github.com/fox-toolkit/fox) that provides distributed tracing using [OpenTelemetry](https://opentelemetry.io/).

## Disclaimer
Oteltracing's API is linked to Fox router, and it will only reach v1 when the router is stabilized.
During the pre-v1 phase, breaking changes may occur and will be documented in the release notes.

## Getting started
### Installation
````shell
go get -u github.com/fox-toolkit/oteltracing
````

### Features

- Automatically creates spans for incoming HTTP requests
- Extracts and propagates trace context from incoming requests
- Annotates spans with HTTP-specific attributes, such as method, route, and status code

### Usage
````go
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/fox-toolkit/fox"
	"github.com/fox-toolkit/oteltracing"
)

func main() {
	f := fox.MustRouter(
		fox.WithMiddleware(oteltracing.Middleware("fox")),
	)

	f.MustAdd(fox.MethodGet, "/hello/{name}", func(c *fox.Context) {
		_ = c.String(http.StatusOK, fmt.Sprintf("hello %s\n", c.Param("name")))
	})

	if err := http.ListenAndServe(":8080", f); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalln(err)
	}
}
````
