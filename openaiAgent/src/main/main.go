package main

import (
	"agent"
	"context"
	"log"
	"tools"
	"traceTools"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func main() {
	// Create a new span
	_, span := traceTools.GetActiveTracer().Start(
		context.Background(),
		"AgentRun",
	)
	defer span.End()

	inputMessage := "Show me the code for a graph of only the top 10 sales by store in Nov 2021, and tell me what trends you see for those top 10."
	span.SetAttributes(
		attribute.String("ai.model", tools.Model),
		attribute.String("ai.input", inputMessage),
	)

	result := agent.RunAgent(inputMessage)
	span.SetAttributes(attribute.String("ai.output", result))
	span.SetStatus(codes.Ok, "Agent execution successful")

	log.Println(result)
}
