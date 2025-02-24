package main

import (
	"agent"
	"context"
	"log"
	"os"
	"strings"
	"tools"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

const projectName = "Zeke-Go-OpenAI-Agent"

var tracerProvider *trace.TracerProvider

func getTracerProvider() *trace.TracerProvider {
	if tracerProvider != nil {
		return tracerProvider
	}

	log.Println("Initializing tracer provider")
	collectorEndpoint := os.Getenv("PHOENIX_COLLECTOR_ENDPOINT") + "/v1/traces"
	headers := os.Getenv("PHOENIX_CLIENT_HEADERS")

	headerMap := make(map[string]string)
	if headers != "" {
		for _, h := range strings.Split(headers, ",") {
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
	tracerProvider = trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(projectName),
		)),
	)

	log.Println("Register tracer provider")

	// Register the tracer globally
	otel.SetTracerProvider(tracerProvider)
	return tracerProvider
}

/*
----------
Entrypoint
----------
*/

func main() {
	// initialize tracer provider
	tp := getTracerProvider()
	defer tp.Shutdown(context.Background())

	tracer := otel.Tracer("my-openai-app")

	_, span := tracer.Start(context.Background(), "AgentRun")
	defer span.End()

	inputMessage := "Show me the code for a graph of the top 10 sales by store in Nov 2021, and tell me what trends you see for those top 10."
	span.SetAttributes(
		attribute.String("ai.model", tools.Model),
		attribute.String("ai.input", inputMessage),
	)

	result := agent.RunAgent(inputMessage)
	span.SetAttributes(attribute.String("ai.output", result))

	log.Println(result)
}
