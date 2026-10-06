package oteltracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fox-toolkit/fox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	b3prop "go.opentelemetry.io/contrib/propagators/b3"
)

func TestGetSpanNotInstrumented(t *testing.T) {
	f := fox.MustRouter()
	_, err := f.Add(fox.MethodGet, "/ping", func(c *fox.Context) {
		span := trace.SpanFromContext(c.Request().Context())
		ok := !span.SpanContext().IsValid()
		assert.True(t, ok)
		_ = c.String(http.StatusOK, "ok")
	})
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	f.ServeHTTP(w, r)
	response := w.Result()
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestPropagationWithGlobalPropagators(t *testing.T) {
	provider := noop.NewTracerProvider()
	otel.SetTextMapPropagator(b3prop.New())

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx := context.Background()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
	ctx, _ = provider.Tracer(ScopeName).Start(ctx, "test")
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))

	f, err := fox.NewRouter(
		fox.WithMiddleware(Middleware("foobar", WithTracerProvider(provider))),
	)
	require.NoError(t, err)
	_, err = f.Add(fox.MethodGet, "/user/{id}", func(c *fox.Context) {
		span := trace.SpanFromContext(c.Request().Context())
		assert.Equal(t, sc.TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, sc.SpanID(), span.SpanContext().SpanID())
	})

	require.NoError(t, err)
	f.ServeHTTP(w, r)
}

func TestPropagationWithCustomPropagators(t *testing.T) {
	provider := noop.NewTracerProvider()
	b3 := b3prop.New()

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx := context.Background()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
	ctx, _ = provider.Tracer(ScopeName).Start(ctx, "test")
	b3.Inject(ctx, propagation.HeaderCarrier(r.Header))

	f, err := fox.NewRouter(
		fox.WithMiddleware(Middleware("foobar", WithTracerProvider(provider), WithPropagators(b3))),
	)
	require.NoError(t, err)

	_, err = f.Add(fox.MethodGet, "/user/{id}", func(c *fox.Context) {
		span := trace.SpanFromContext(c.Request().Context())
		assert.Equal(t, sc.TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, sc.SpanID(), span.SpanContext().SpanID())
	})
	require.NoError(t, err)
	f.ServeHTTP(w, r)
}

func TestWithDefaultClientIPResolver(t *testing.T) {
	provider := noop.NewTracerProvider()
	otel.SetTextMapPropagator(b3prop.New())
	r := httptest.NewRequest("GET", "/foo", nil)
	r.Header.Set(fox.HeaderXForwardedFor, "25.13.12.11")
	w := httptest.NewRecorder()

	ctx := context.Background()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
	ctx, _ = provider.Tracer(ScopeName).Start(ctx, "test")
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))

	// Test with the default resolver
	f, err := fox.NewRouter(
		fox.WithMiddleware(Middleware("foobar", WithTracerProvider(provider))),
	)
	require.NoError(t, err)
	_, err = f.Add(fox.MethodGet, "/bar", func(c *fox.Context) {
		t.Fail()
	})
	require.NoError(t, err)
	assert.NotPanics(t, func() {
		f.ServeHTTP(w, r)
	})
}

func TestWithMetricAttributes(t *testing.T) {
	provider := noop.NewTracerProvider()
	otel.SetTextMapPropagator(b3prop.New())

	r := httptest.NewRequest("GET", "/user/123?foo=bar", nil)
	w := httptest.NewRecorder()

	ctx := context.Background()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
	ctx, _ = provider.Tracer(ScopeName).Start(ctx, "test")
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))

	f, err := fox.NewRouter(
		fox.WithMiddleware(
			Middleware("foobar", WithTracerProvider(provider), WithMetricsAttributes(func(c *fox.Context) []attribute.KeyValue {
				attrs := make([]attribute.KeyValue, 1, 2)
				attrs[0] = attribute.String("http.target", r.URL.String())
				v := c.Route().Annotation("annotation")
				attrs = append(attrs, attribute.KeyValue{
					Key:   "annotation",
					Value: attribute.StringValue(v.(string)),
				})
				return attrs
			})),
		),
	)
	require.NoError(t, err)
	_, err = f.Add(fox.MethodGet, "/user/{id}", func(c *fox.Context) {
		span := trace.SpanFromContext(c.Request().Context())
		assert.Equal(t, sc.TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, sc.SpanID(), span.SpanContext().SpanID())
	}, fox.WithAnnotation("annotation", "foobar"))
	require.NoError(t, err)

	f.ServeHTTP(w, r)
}

// waitOrFail blocks on ch and fails the test instead of hanging past the
// package timeout if the client-disconnect fixture never reaches the
// expected point.
func waitOrFail(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

// disconnectClient sends a request to url, waits for the handler to start,
// then cancels the request to simulate a client disconnect.
func disconnectClient(t *testing.T, srv *httptest.Server, url string, handlerStarted <-chan struct{}) {
	t.Helper()

	reqCtx, cancel := context.WithCancel(t.Context())
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, http.NoBody)
	require.NoError(t, err)

	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		resp, doErr := srv.Client().Do(req)
		if doErr == nil {
			_ = resp.Body.Close()
		}
	}()

	waitOrFail(t, handlerStarted, "the handler to start")
	cancel()
	waitOrFail(t, requestDone, "the request to finish")
}

// findMetric collects rm until the named metric is recorded. RecordMetrics records the body-size
// histograms before the request-duration histogram, so a snapshot can contain the scope without
// yet containing the requested metric.
func findMetric(t *testing.T, reader sdkmetric.Reader, name string) metricdata.Metrics {
	t.Helper()
	var found metricdata.Metrics
	require.Eventually(t, func() bool {
		rm := metricdata.ResourceMetrics{}
		require.NoError(t, reader.Collect(t.Context(), &rm))
		if len(rm.ScopeMetrics) != 1 {
			return false
		}
		for _, m := range rm.ScopeMetrics[0].Metrics {
			if m.Name == name {
				found = m
				return true
			}
		}
		return false
	}, time.Second, 10*time.Millisecond, "expected the %s metric to be recorded", name)
	return found
}

func assertMetricErrorType(t *testing.T, m metricdata.Metrics, want string) {
	t.Helper()
	var attrs attribute.Set
	switch data := m.Data.(type) {
	case metricdata.Histogram[float64]:
		require.Len(t, data.DataPoints, 1)
		attrs = data.DataPoints[0].Attributes
	case metricdata.Histogram[int64]:
		require.Len(t, data.DataPoints, 1)
		attrs = data.DataPoints[0].Attributes
	default:
		t.Fatalf("unexpected data type %T for metric %s", m.Data, m.Name)
	}
	errorType, ok := attrs.Value(semconv.ErrorTypeKey)
	require.True(t, ok, "expected error.type attribute on the %s metric", m.Name)
	assert.Equal(t, want, errorType.AsString())
}

// TestClientDisconnect reproduces a real HTTP/1.1 client disconnect: the
// handler observes the cancelled request context and writes a 500, mirroring
// how a genuine server fault would look. The span and metrics must carry
// error.type so the disconnect is distinguishable from a real server error.
func TestClientDisconnect(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	handlerStarted := make(chan struct{})
	f, err := fox.NewRouter(
		fox.WithMiddleware(Middleware("foobar", WithTracerProvider(provider), WithMeterProvider(meterProvider))),
	)
	require.NoError(t, err)
	_, err = f.Add(fox.MethodGet, "/hello", func(c *fox.Context) {
		close(handlerStarted)
		<-c.Request().Context().Done()
		_ = c.String(http.StatusInternalServerError, "cancelled")
	})
	require.NoError(t, err)

	srv := httptest.NewServer(f)
	defer srv.Close()

	disconnectClient(t, srv, srv.URL+"/hello", handlerStarted)

	require.Eventually(t, func() bool {
		return len(sr.Ended()) == 1
	}, time.Second, 10*time.Millisecond, "handler should finish and end the span after the client disconnects")

	span := sr.Ended()[0]
	assert.Equal(t, codes.Error, span.Status().Code)
	assert.Contains(t, span.Attributes(), attribute.Int("http.response.status_code", http.StatusInternalServerError))
	assert.Contains(t, span.Attributes(), semconv.ErrorType(context.Canceled))

	want := semconv.ErrorType(context.Canceled).Value.AsString()
	for _, name := range []string{"http.server.request.duration", "http.server.request.body.size", "http.server.response.body.size"} {
		assertMetricErrorType(t, findMetric(t, reader, name), want)
	}
}

// TestClientDisconnectWithoutErrorStatus reproduces a client disconnect
// where the handler observes the cancelled request context and returns
// without writing an error response, leaving fox's default 200 status in
// place. Per the HTTP semantic conventions, span status must still be
// Error because a detected error (the disconnect) exists, even though
// http.response.status_code itself stays 200.
func TestClientDisconnectWithoutErrorStatus(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	handlerStarted := make(chan struct{})
	f, err := fox.NewRouter(
		fox.WithMiddleware(Middleware("foobar", WithTracerProvider(provider), WithMeterProvider(meterProvider))),
	)
	require.NoError(t, err)
	_, err = f.Add(fox.MethodGet, "/hello", func(c *fox.Context) {
		close(handlerStarted)
		<-c.Request().Context().Done()
	})
	require.NoError(t, err)

	srv := httptest.NewServer(f)
	defer srv.Close()

	disconnectClient(t, srv, srv.URL+"/hello", handlerStarted)

	require.Eventually(t, func() bool {
		return len(sr.Ended()) == 1
	}, time.Second, 10*time.Millisecond, "handler should finish and end the span after the client disconnects")

	span := sr.Ended()[0]
	assert.Equal(t, codes.Error, span.Status().Code, "span status must be Error when the request context carries a detected error")
	assert.Contains(t, span.Attributes(), attribute.Int("http.response.status_code", http.StatusOK), "the selected response status code should be preserved")
	assert.Contains(t, span.Attributes(), semconv.ErrorType(context.Canceled))

	assertMetricErrorType(t, findMetric(t, reader, "http.server.request.duration"), semconv.ErrorType(context.Canceled).Value.AsString())
}

// TestClientDisconnectHandlerReplacesContext ensures that a handler
// replacing the request with one wrapping its own (non-client) cancelable
// context does not cause a successful request to be misclassified as a
// client disconnect: only the original request context's cancellation
// should be treated as a disconnect signal.
func TestClientDisconnectHandlerReplacesContext(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	f, err := fox.NewRouter(fox.WithMiddleware(Middleware("foobar", WithTracerProvider(provider))))
	require.NoError(t, err)
	_, err = f.Add(fox.MethodGet, "/hello", func(c *fox.Context) {
		ctx, cancel := context.WithCancel(c.Request().Context())
		defer cancel()
		c.SetRequest(c.Request().WithContext(ctx))
		_ = c.String(http.StatusOK, "ok")
	})
	require.NoError(t, err)

	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/hello", http.NoBody)
	w := httptest.NewRecorder()
	f.ServeHTTP(w, r)
	require.Equal(t, http.StatusOK, w.Code)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, codes.Unset, span.Status().Code, "a handler-local canceled context must not be mistaken for a client disconnect")
	for _, attr := range span.Attributes() {
		assert.NotEqual(t, semconv.ErrorTypeKey, attr.Key, "error.type must not be set when the client did not disconnect")
	}
}

// customErrorTypeError has a distinct ErrorType() value from context.Canceled
// so tests can tell whether a caller-supplied error.type survived.
type customErrorTypeError struct{}

func (customErrorTypeError) Error() string     { return "custom" }
func (customErrorTypeError) ErrorType() string { return "custom_error_type" }

// TestClientDisconnectCustomErrorTypeAttribute ensures that a caller-supplied
// error.type from WithMetricsAttributes takes precedence over the error.type
// derived from a client disconnect.
func TestClientDisconnectCustomErrorTypeAttribute(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	handlerStarted := make(chan struct{})
	f, err := fox.NewRouter(
		fox.WithMiddleware(Middleware(
			"foobar",
			WithMeterProvider(meterProvider),
			WithMetricsAttributes(func(*fox.Context) []attribute.KeyValue {
				return []attribute.KeyValue{semconv.ErrorType(customErrorTypeError{})}
			}),
		)),
	)
	require.NoError(t, err)
	_, err = f.Add(fox.MethodGet, "/hello", func(c *fox.Context) {
		close(handlerStarted)
		<-c.Request().Context().Done()
		_ = c.String(http.StatusInternalServerError, "cancelled")
	})
	require.NoError(t, err)

	srv := httptest.NewServer(f)
	defer srv.Close()

	disconnectClient(t, srv, srv.URL+"/hello", handlerStarted)

	assertMetricErrorType(t, findMetric(t, reader, "http.server.request.duration"), "custom_error_type")
}
