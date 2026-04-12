package oteltracing

import (
	"fmt"
	"time"

	"github.com/fox-toolkit/fox"
	"github.com/fox-toolkit/oteltracing/internal/clientip"
	"github.com/fox-toolkit/oteltracing/internal/semconv"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	// ScopeName is the instrumentation scope name.
	ScopeName = "github.com/fox-toolkit/oteltracing"
)

var (
	// DefaultClientIPResolver attempts to resolve client IP addresses in the following order:
	// 1. Leftmost non-private IP in X-Forwarded-For header
	// 2. Leftmost non-private IP in Forwarded header
	// 3. X-Real-IP header
	// 4. CF-Connecting-IP header
	// 5. True-Client-IP header
	// 6. Fastly-Client-IP header
	// 7. X-Azure-ClientIP header
	// 8. X-Azure-SocketIP header
	// 9. X-Appengine-Remote-Addr header
	// 10. Fly-Client-IP header
	// 11. RemoteAddr from the request
	//
	// The DefaultClientIPResolver uses resolvers (particularly Leftmost in X-Forwarded-For/Forwarded headers,
	// and X-Azure-ClientIP) that are trivially spoofable by clients. For security-critical applications
	// where IP addresses must be trusted, consider using a Rightmost resolver or implementing
	// your own strategy tailored to your infrastructure.
	DefaultClientIPResolver = clientip.DefaultResolver
)

// Middleware returns middleware that will trace incoming requests.
// The service parameter should describe the name of the (virtual)
// server handling the request.
func Middleware(service string, opts ...Option) fox.MiddlewareFunc {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt.apply(cfg)
	}

	tracer := cfg.provider.Tracer(ScopeName, oteltrace.WithInstrumentationVersion(Version))
	meter := cfg.meter.Meter(ScopeName, metric.WithInstrumentationVersion(Version))

	sc := semconv.NewHTTPServer(meter)

	return func(next fox.HandlerFunc) fox.HandlerFunc {
		return func(c *fox.Context) {
			requestStartTime := time.Now()

			req := c.Request()

			for _, f := range cfg.filters {
				if !f(c) {
					next(c)
					return
				}
			}

			defer func() {
				// rollback to the original request
				c.SetRequest(req)
			}()

			ctx := cfg.propagator.Extract(req.Context(), cfg.carrier(req))
			requestTraceAttrOpts := semconv.RequestTraceAttrsOpts{
				HTTPClientIP: serverClientIP(c, cfg.resolver),
			}

			opts := []oteltrace.SpanStartOption{
				oteltrace.WithAttributes(sc.RequestTraceAttrs(service, req, requestTraceAttrOpts)...),
				oteltrace.WithAttributes(sc.Route(c.Pattern())),
				oteltrace.WithSpanKind(oteltrace.SpanKindServer),
			}

			opts = append(opts, cfg.spanOpts...)

			spanName := cfg.spanFmt(c)
			if spanName == "" {
				spanName = fmt.Sprintf("HTTP %s route not found", req.Method)
			}

			ctx, span := tracer.Start(ctx, spanName, opts...)
			defer span.End()

			// pass the span through the request context
			c.SetRequest(req.WithContext(ctx))

			next(c)

			status := c.Writer().Status()
			span.SetStatus(sc.Status(status))
			span.SetAttributes(sc.ResponseTraceAttrs(semconv.ResponseTelemetry{
				StatusCode: status,
				WriteBytes: int64(c.Writer().Size()),
			})...)

			// Record the server-side attributes.
			var additionalAttributes []attribute.KeyValue
			if cfg.attrsFn != nil {
				additionalAttributes = append(additionalAttributes, cfg.attrsFn(c)...)
			}
			sc.RecordMetrics(ctx, semconv.ServerMetricData{
				ServerName:   service,
				ResponseSize: int64(c.Writer().Size()),
				MetricAttributes: semconv.MetricAttributes{
					Req:                  c.Request(),
					StatusCode:           status,
					Route:                c.Pattern(),
					AdditionalAttributes: additionalAttributes,
				},
				MetricData: semconv.MetricData{
					RequestSize:     c.Request().ContentLength,
					RequestDuration: time.Since(requestStartTime),
				},
			})
		}
	}
}

func serverClientIP(c *fox.Context, resolver fox.ClientIPResolver) string {
	// Try custom resolver first if provided
	if resolver != nil {
		if ipAddr, err := resolver.ClientIP(c); err == nil {
			return ipAddr.String()
		}
	} else {
		// Try router's configured resolver
		if ipAddr, err := c.ClientIP(); err == nil {
			return ipAddr.String()
		}
	}

	// Fall back to DefaultResolver which is safer than relying on semconv's
	// leftmost XFF extraction.
	if ipAddr, err := clientip.DefaultResolver.ClientIP(c); err == nil {
		return ipAddr.String()
	}
	return ""
}
