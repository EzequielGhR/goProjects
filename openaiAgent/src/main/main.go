package main

import (
	"agent"
	"log"
	"os"
	"tools"
	"traceTools"
)

func startMainSpan() string {
	// Create a new span
	span := traceTools.StartOpenInferenceSpan("AgentRun", traceTools.AgentKind)
	defer span.End()
	inputMessage := "Show me the code for a graph of only the top 10 sales by store in Nov 2021, and tell me what trends you see for those top 10."

	traceTools.SetSpanInput(span, inputMessage)
	traceTools.SetSpanModel(span, tools.Model)

	result := agent.RunAgent(inputMessage)
	traceTools.SetSpanSuccessCode(span)
	return result
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s [parquet-sales-data-path]\n", os.Args[0])
	}

	tools.AssertDataPath(os.Args[1])
	result := startMainSpan()
	log.Println(result)
}
