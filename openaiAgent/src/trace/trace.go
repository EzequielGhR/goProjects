package traceTools

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	traceSdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

type SpanDataType interface {
	string | []string | int | bool
}

type OpenInferenceSpanKind string

const (
	AgentKind     OpenInferenceSpanKind = "agent"
	ChainKind     OpenInferenceSpanKind = "chain"
	EmbeddingKind OpenInferenceSpanKind = "embedding"
	EvaluatorKind OpenInferenceSpanKind = "evaluator"
	GuardrailKind OpenInferenceSpanKind = "guardrail"
	LLMKind       OpenInferenceSpanKind = "llm"
	RerankerKind  OpenInferenceSpanKind = "reranker"
	RetrieverKind OpenInferenceSpanKind = "retriever"
	ToolKind      OpenInferenceSpanKind = "tool"
	UnknownKind   OpenInferenceSpanKind = "unknown"
)

const projectName = "Zeke-Go-OpenAI-Agent"
const openInferenceProjectNameKey = "openinference.project.name"
const openInferenceSpanKindKey = "openinference.span.kind"
const openInferenceInputKey = "input.value"
const openInferenceOutputKey = "output.value"

var tracerProvider *traceSdk.TracerProvider
var activeTracer trace.Tracer = nil

// Get or initialize tracer provider
func GetTracerProvider() *traceSdk.TracerProvider {
	if tracerProvider != nil {
		return tracerProvider
	}

	log.Println("Initializing tracer provider")
	collectorEndpoint := os.Getenv("PHOENIX_COLLECTOR_ENDPOINT") + "/v1/traces"
	headers := os.Getenv("PHOENIX_CLIENT_HEADERS")
	if collectorEndpoint == "" || headers == "" {
		log.Fatalln("'PHOENIX_COLLECTOR_ENDPOINT' or 'PHOENIX_CLIENT_HEADERS' environment variables are not defined")
	}

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
			attribute.String(openInferenceProjectNameKey, projectName),
		)),
	)

	log.Println("Register tracer provider")

	// Register the tracer globally
	otel.SetTracerProvider(tracerProvider)
	return tracerProvider
}

// Get or initialize tracer
func GetActiveTracer() trace.Tracer {
	GetTracerProvider()

	if activeTracer != nil {
		return activeTracer
	}

	activeTracer = otel.Tracer(projectName)
	return activeTracer
}

// Start a new Span
func StartOpenInferenceSpan(spanName string, openInferenceSpanKind OpenInferenceSpanKind) trace.Span {
	_, span := GetActiveTracer().Start(
		context.Background(),
		spanName,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String(openInferenceSpanKindKey, strings.ToUpper(string(openInferenceSpanKind))),
		),
	)

	return span
}

func SetSpanAttr[T SpanDataType](span trace.Span, key string, input T) {
	var attr attribute.KeyValue
	switch any(input).(type) {
	case string:
		attr = attribute.String(key, any(input).(string))
	case []string:
		attr = attribute.StringSlice(key, any(input).([]string))
	case bool:
		attr = attribute.Bool(key, any(input).(bool))
	}

	span.SetAttributes(attr)
}

func SetSpanInput[T SpanDataType](span trace.Span, input T) {
	SetSpanAttr(span, openInferenceInputKey, input)
}

func SetSpanOutput[T SpanDataType](span trace.Span, output T) {
	SetSpanAttr(span, openInferenceOutputKey, output)
}

func SetSpanModel(span trace.Span, model string) {
	SetSpanAttr(span, "model", model)
}

func SetSpanAttrFromMap[T SpanDataType](span trace.Span, kvMap map[string]T) {
	for k, v := range kvMap {
		SetSpanAttr(span, k, v)
	}
}

func SetSpanGenericStatus(span trace.Span, statusCode codes.Code) {
	span.SetStatus(statusCode, fmt.Sprintf("%s Successful", span.SpanContext().SpanID().String()))
}

func SetSpanSuccessCode(span trace.Span) {
	SetSpanGenericStatus(span, codes.Ok)
}

func SetSpanErrorCode(span trace.Span) {
	SetSpanGenericStatus(span, codes.Error)
}
