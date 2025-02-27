package traceTools

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	traceSdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// Interface to use when setting attributes on spans
type SpanAttributeDataType interface {
	string | []string | int | bool
}

// Span kind datatype for replicated openinference spans
type OpenInferenceSpanKind string

// Replication of openinference span kinds
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

// Constants for replication of openinference traces
// Used to display traces and spans properly on phoenix
const projectName = "Zeke-Go-OpenAI-Agent"
const openInferenceProjectNameKey = "openinference.project.name"
const openInferenceSpanKindKey = "openinference.span.kind"
const openInferenceInputKey = "input.value"
const openInferenceOutputKey = "output.value"

// Global private vars for tracer provider and tracer
var tracerProvider *traceSdk.TracerProvider
var activeTracer trace.Tracer = nil

// Global variables for span context tracking accross modules
var AgentContext context.Context = nil
var LastRouterContext context.Context = nil
var HandleToolContext context.Context = nil
var LastToolContext context.Context = nil

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
		log.Panicf("Failed to initialize exporter: %s\n", err)
	}

	// Create a new tracer provider
	tracerProvider = traceSdk.NewTracerProvider(
		traceSdk.WithBatcher(exporter),
		traceSdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			attribute.String(openInferenceProjectNameKey, projectName),
		)),
	)

	log.Println("Registering tracer provider")

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

// Start a new Span configured similarly to openinference spans
// Used to displaying correctly on phoenix
func StartOpenInferenceSpan(
	spanName string,
	openInferenceSpanKind OpenInferenceSpanKind,
	parentSpanContext context.Context,
) (context.Context, trace.Span) {
	if parentSpanContext == nil {
		parentSpanContext = context.Background()
	}

	ctx, span := GetActiveTracer().Start(
		parentSpanContext,
		spanName,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String(openInferenceSpanKindKey, strings.ToUpper(string(openInferenceSpanKind))),
		),
	)

	log.Printf("Starting '%s' OpenInference Span with kind '%s'\n", spanName, openInferenceSpanKind)
	return ctx, span
}

// End an openinference replicated span
func EndOpenInferenceSpan(span trace.Span) {
	log.Printf("Ending OpenInference Span with: ID: %s\n", span.SpanContext().SpanID().String())
	span.End(
		trace.WithStackTrace(true),
		trace.WithTimestamp(time.Now()),
	)
}

func StartOpenAISpan(parentSpanContext context.Context, openaiModel string) (context.Context, trace.Span) {
	if parentSpanContext == nil {
		parentSpanContext = context.Background()
	}

	ctx, span := GetActiveTracer().Start(
		parentSpanContext,
		"ChatCompletion",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String(openInferenceSpanKindKey, strings.ToUpper(string(LLMKind))),
			attribute.String("llm.provider", "openai"),
			attribute.String("llm.invocation_parameters", fmt.Sprintf("{\"model\": \"%s\"}", openaiModel)),
			attribute.String("llm.system", "openai"),
			attribute.String("llm.model_name", openaiModel),
		),
	)

	log.Println("Starting openai chat completion LLM span")
	return ctx, span
}

/*
-----------------------------------------
Functions for setting attributes on spans
-----------------------------------------
*/

func SetSpanAttr[T SpanAttributeDataType](span trace.Span, key string, input T) {
	var attr attribute.KeyValue
	switch r := any(input).(type) {
	case string:
		attr = attribute.String(key, r)
	case []string:
		attr = attribute.StringSlice(key, r)
	case bool:
		attr = attribute.Bool(key, r)
	case int:
		attr = attribute.Int(key, r)
	}

	span.SetAttributes(attr)
}

func SetSpanInput[T SpanAttributeDataType](span trace.Span, input T) {
	SetSpanAttr(span, openInferenceInputKey, input)
}

func SetSpanOutput[T SpanAttributeDataType](span trace.Span, output T) {
	SetSpanAttr(span, openInferenceOutputKey, output)
}

func SetSpanModel(span trace.Span, model string) {
	SetSpanAttr(span, "llm.model_name", model)
}

func SetSpanAttrFromMap(span trace.Span, kvMap map[string]any) {
	for k, v := range kvMap {
		switch r := v.(type) {
		case string:
			SetSpanAttr(span, k, r)
		case int:
			SetSpanAttr(span, k, r)
		case []string:
			SetSpanAttr(span, k, r)
		case bool:
			SetSpanAttr(span, k, r)
		default:
			log.Printf("Value for key '%s' is not an expected type. Ignoring\n", k)
			continue
		}
	}
}

/*
---------------
set span status
---------------
*/

func SetSpanGenericStatus(span trace.Span, statusCode codes.Code, message string) {
	span.SetStatus(statusCode, fmt.Sprintf(
		"Span ID: '%s'. Status: %s",
		span.SpanContext().SpanID().String(),
		message,
	))
}

func SetSpanSuccessCode(span trace.Span) {
	SetSpanGenericStatus(span, codes.Ok, "Successful")
}

func SetSpanErrorCode(span trace.Span) {
	SetSpanGenericStatus(span, codes.Error, "Failed")
}
