// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package semconv

// Generate semconv package:
//go:generate go tool -modfile=../../go.tool.mod gotmpl --body=../shared/semconv/bench_test.go.tmpl "--data={}" --out=bench_test.go
//go:generate go tool -modfile=../../go.tool.mod gotmpl --body=../shared/semconv/common_test.go.tmpl "--data={ \"pkg\": \"github.com/fox-toolkit/oteltracing\" }" --out=common_test.go
//go:generate go tool -modfile=../../go.tool.mod gotmpl --body=../shared/semconv/server.go.tmpl "--data={ \"pkg\": \"github.com/fox-toolkit/oteltracing\" }" --out=server.go
//go:generate go tool -modfile=../../go.tool.mod gotmpl --body=../shared/semconv/server_test.go.tmpl "--data={ \"pkg\": \"github.com/fox-toolkit/oteltracing\" }" --out=server_test.go
//go:generate go tool -modfile=../../go.tool.mod gotmpl --body=../shared/semconv/client.go.tmpl "--data={ \"pkg\": \"github.com/fox-toolkit/oteltracing\" }" --out=client.go
//go:generate go tool -modfile=../../go.tool.mod gotmpl --body=../shared/semconv/client_test.go.tmpl "--data={ \"pkg\": \"github.com/fox-toolkit/oteltracing\" }" --out=client_test.go
//go:generate go tool -modfile=../../go.tool.mod gotmpl --body=../shared/semconv/httpconvtest_test.go.tmpl "--data={ \"pkg\": \"github.com/fox-toolkit/oteltracing\" }" --out=httpconvtest_test.go
//go:generate go tool -modfile=../../go.tool.mod gotmpl --body=../shared/semconv/util.go.tmpl "--data={ \"pkg\": \"github.com/fox-toolkit/oteltracing\" }" --out=util.go
//go:generate go tool -modfile=../../go.tool.mod gotmpl --body=../shared/semconv/util_test.go.tmpl "--data={}" --out=util_test.go
