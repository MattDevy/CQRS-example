package tracing

import (
	"fmt"
	"io"
	"log"

	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/transport/zipkin"
	zk "github.com/uber/jaeger-client-go/zipkin"

	czipkin "contrib.go.opencensus.io/exporter/zipkin"
	"go.opencensus.io/trace"

	openzipkin "github.com/openzipkin/zipkin-go"
	zipkinHTTP "github.com/openzipkin/zipkin-go/reporter/http"
)

// NewTracer creates a new global tracer. It must be closed on service exit
// using the returned io.Closer.
func NewTracer(serviceName, zipkinURL string) (io.Closer, error) {
	// Send the tracing in Zipkin format (even if we are using Jaeger as backend).
	transport, err := zipkin.NewHTTPTransport("http://" + zipkinURL + ":9411/api/v1/spans")
	if err != nil {
		return nil, fmt.Errorf("could not init Jaeger Zipkin HTTP transport: %w", err)
	}

	// Zipkin shares span ID between client and server spans; it must be enabled via the following option.
	zipkinPropagator := zk.NewZipkinB3HTTPHeaderPropagator()

	tracer, closer := jaeger.NewTracer(
		serviceName,
		jaeger.NewConstSampler(true), // Trace everything for now.
		jaeger.NewRemoteReporter(transport),
		jaeger.TracerOptions.Injector(opentracing.HTTPHeaders, zipkinPropagator),
		jaeger.TracerOptions.Extractor(opentracing.HTTPHeaders, zipkinPropagator),
		jaeger.TracerOptions.ZipkinSharedRPCSpan(true),
		jaeger.TracerOptions.Gen128Bit(true),
	)
	opentracing.SetGlobalTracer(tracer)

	return closer, nil
}

func InitOpenCensus(zipkinURL, serviceName string) {
	// 1. Configure exporter to export traces to Zipkin.
	localEndpoint, err := openzipkin.NewEndpoint(serviceName, zipkinURL)
	if err != nil {
		log.Fatalf("Failed to create the local zipkinEndpoint: %v", err)
	}
	reporter := zipkinHTTP.NewReporter("http://" + zipkinURL + ":9411/api/v2/spans")
	ze := czipkin.NewExporter(reporter, localEndpoint)
	trace.RegisterExporter(ze)

	// 2. Configure 100% sample rate, otherwise, few traces will be sampled.
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
}
