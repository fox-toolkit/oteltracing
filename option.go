package oteltracing

import (
	"net/http"
	"slices"
	"strings"

	"github.com/fox-toolkit/fox"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var defaultSpanNameFormatter SpanNameFormatter = func(c *fox.Context) string {
	method := strings.ToUpper(c.Request().Method)
	if !slices.Contains([]string{
		http.MethodGet, http.MethodHead,
		http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete,
		http.MethodConnect, http.MethodOptions,
		http.MethodTrace,
	}, method) {
		method = "HTTP"
	}

	if path := c.Pattern(); path != "" {
		return method + " " + path
	}

	return method
}

type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// Filter is a predicate used to determine whether a given http.request should
// be traced. A Filter must return true if the request should be traced.
type Filter func(c *fox.Context) bool

// SpanNameFormatter is a function that formats the span name given the HTTP request.
// This allows for dynamic naming of spans based on attributes of the request.
type SpanNameFormatter func(c *fox.Context) string

// MetricAttributesFunc is a function type that can be used to dynamically
// generate metric attributes for a given HTTP request. It is used in
// conjunction with the [WithMetricsAttributes] middleware option.
type MetricAttributesFunc func(c *fox.Context) []attribute.KeyValue

type config struct {
	provider   trace.TracerProvider
	propagator propagation.TextMapPropagator
	meter      metric.MeterProvider
	resolver   fox.ClientIPResolver
	carrier    func(r *http.Request) propagation.TextMapCarrier
	spanFmt    SpanNameFormatter
	attrsFn    MetricAttributesFunc
	filters    []Filter
	spanOpts   []trace.SpanStartOption
}

func defaultConfig() *config {
	return &config{
		provider:   otel.GetTracerProvider(),
		propagator: otel.GetTextMapPropagator(),
		meter:      otel.GetMeterProvider(),
		carrier: func(r *http.Request) propagation.TextMapCarrier {
			return propagation.HeaderCarrier(r.Header)
		},
		attrsFn: func(c *fox.Context) []attribute.KeyValue { return nil },
		spanFmt: defaultSpanNameFormatter,
	}
}

// WithPropagators specifies propagators to use for extracting
// information from the HTTP requests. If none are specified, global
// ones will be used.
func WithPropagators(propagators propagation.TextMapPropagator) Option {
	return optionFunc(func(c *config) {
		if propagators != nil {
			c.propagator = propagators
		}
	})
}

// WithTracerProvider specifies a tracer provider to use for tracing http request.
// If none is specified, the global tracer provider is used.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return optionFunc(func(c *config) {
		if provider != nil {
			c.provider = provider
		}
	})
}

// WithMeterProvider specifies a meter provider to use for recording http request metrics.
// If none is specified, the global meter provider is used.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return optionFunc(func(c *config) {
		if provider != nil {
			c.meter = provider
		}
	})
}

// WithTextMapCarrier specify a carrier to use for extracting information from http request.
// If none is specified, [propagation.HeaderCarrier] is used.
func WithTextMapCarrier(fn func(r *http.Request) propagation.TextMapCarrier) Option {
	return optionFunc(func(c *config) {
		if fn != nil {
			c.carrier = fn
		}
	})
}

// WithSpanNameFormatter takes a function that will be called on every request
// and the returned string will become the Span Name.
func WithSpanNameFormatter(fn SpanNameFormatter) Option {
	return optionFunc(func(c *config) {
		if fn != nil {
			c.spanFmt = fn
		}
	})
}

// WithFilter adds a filter to the list of filters used by the handler. If any filter indicates to exclude a request
// then the request will not be traced. All filters must allow a request to be traced for a Span to be created.
// If no filters are provided then all requests are traced. Filters will be invoked for each processed request,
// it is advised to make them simple and fast.
func WithFilter(f ...Filter) Option {
	return optionFunc(func(c *config) {
		c.filters = append(c.filters, f...)
	})
}

// WithSpanStartOptions configures an additional set of trace.SpanStartOptions, which are applied to each new span.
func WithSpanStartOptions(opts ...trace.SpanStartOption) Option {
	return optionFunc(func(c *config) {
		c.spanOpts = append(c.spanOpts, opts...)
	})
}

// WithMetricsAttributes specifies a function for generating metric attributes.
// The function will be invoked for each request, and its returned attributes
// will be added to the metric record.
func WithMetricsAttributes(fn MetricAttributesFunc) Option {
	return optionFunc(func(c *config) {
		if fn != nil {
			c.attrsFn = fn
		}
	})
}

// WithClientIPResolver sets a custom resolver to derive the client IP address.
// This is for advanced use cases. Most users should configure the resolver with Fox's router option using
// [fox.WithClientIPResolver] instead.
//
// Priority order for resolvers:
// 1. Resolver set with this option (highest priority)
// 2. Resolver configured at the route level in Fox
// 3. Resolver configured globally in Fox
// 4. If no resolver is configured anywhere, [DefaultClientIPResolver] is used as fallback.
//
// Only use this option when you need different IP resolution logic specifically for OpenTelemetry
// attributes than what's used by the rest of your application.
func WithClientIPResolver(resolver fox.ClientIPResolver) Option {
	return optionFunc(func(c *config) {
		if resolver != nil {
			c.resolver = resolver
		}
	})
}
