package traceTools

import (
	"context"
	"log"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	traceSdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

const projectName = "Zeke-Go-OpenAI-Agent"

var tracerProvider *traceSdk.TracerProvider
var activeTracer trace.Tracer = nil

func GetTracerProvider() *traceSdk.TracerProvider {
	if tracerProvider != nil {
		return tracerProvider
	}

	log.Println("Initializing tracer provider")
	collectorEndpoint := os.Getenv("PHOENIX_COLLECTOR_ENDPOINT") + "/v1/traces"
	headers := os.Getenv("PHOENIX_CLIENT_HEADERS")

	headerMap := make(map[string]string)
	if headers != "" {
		for h := range strings.SplitSeq(headers, ",") {
			parts := strings.Split(h, "=")
			if len(parts) == 2 {
				headerMap[parts[0]] = parts[1]
			}
		}
	}

	// Create an OpenTelemetry HTTP exporter to send traces to Phoenix AI
	exporter, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpointURL(collectorEndpoint),
		otlptracehttp.WithHeaders(headerMap),
	)

	if err != nil {
		log.Fatalf("Failed to initialize exporter: %s\n", err)
	}

	// Create a new tracer provider
	tracerProvider = traceSdk.NewTracerProvider(
		traceSdk.WithBatcher(exporter),
		traceSdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			attribute.String("service.name", projectName),
		)),
	)

	log.Println("Register tracer provider")

	// Register the tracer globally
	otel.SetTracerProvider(tracerProvider)
	return tracerProvider
}

func GetActiveTracer() trace.Tracer {
	GetTracerProvider()

	if activeTracer != nil {
		return activeTracer
	}

	activeTracer = otel.Tracer("my-openai-agent")
	return activeTracer
}
