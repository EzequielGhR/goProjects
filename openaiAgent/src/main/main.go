package main

import (
	"agent"
	"context"
	"log"
	"os"
	"path"
	"tools"
	"traceTools"
)

var ProjectPath = "../.."

/*
Start the main span surrounding the agent run.
Receives the user prompt as `prompt`.
*/
func startMainSpan(prompt string) (string, error) {
	// Create a new span and set the agent context global var
	ctx, span := traceTools.StartOpenInferenceSpan("AgentRun", traceTools.AgentKind, nil)
	traceTools.AgentContext = ctx
	defer traceTools.EndOpenInferenceSpan(span)

	// Set span attributes
	traceTools.SetSpanInput(span, prompt)
	traceTools.SetSpanModel(span, tools.Model)

	result, err := agent.RunAgent(prompt)

	// Set span output and status code
	traceTools.SetSpanOutput(span, result)
	if err != nil {
		traceTools.SetSpanErrorCode(span)
		return "", err
	}
	traceTools.SetSpanSuccessCode(span)
	return result, nil
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s [prompt]\n", os.Args[0])
	}

	ProjectPath = path.Join(path.Dir(os.Args[0]), ProjectPath)

	tools.AssertDataPath(path.Join(ProjectPath, tools.DataPath))
	tools.AssertToolsPath(path.Join(ProjectPath, tools.ToolsJsonPath))

	result, err := startMainSpan(os.Args[1])
	shutdowunErr := traceTools.GetTracerProvider().Shutdown(context.Background())
	if shutdowunErr != nil {
		log.Panicf("ERROR: %s\n", shutdowunErr)
	}

	if err != nil {
		log.Panicf("ERROR: %s\n", err)
	}

	log.Println(result)
}
