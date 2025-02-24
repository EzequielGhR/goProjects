package main

import (
	"agent"
	"context"
	"log"
	"tools"
	"traceTools"
)

func main() {
	// Create a new span
	_, span := traceTools.GetActiveTracer().Start(
		context.Background(),
		"AgentRun",
	)
	defer span.End()

	inputMessage := "Show me the code for a graph of only the top 10 sales by store in Nov 2021, and tell me what trends you see for those top 10."

	traceTools.SetSpanInput(span, inputMessage)
	traceTools.SetSpanModel(span, tools.Model)

	result := agent.RunAgent(inputMessage)
	traceTools.SetSpanSuccessCode(span)

	log.Println(result)
}
