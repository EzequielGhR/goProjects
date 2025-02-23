package main

import (
	"encoding/json"
	"fmt"
	"os"
	"tools"

	"github.com/openai/openai-go"
)

type ToolFunctionParameterPropertyInfo struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ToolFunctionParameterProperties struct {
	Data              ToolFunctionParameterPropertyInfo `json:"data"`
	Prompt            ToolFunctionParameterPropertyInfo `json:"prompt"`
	VisualizationGoal ToolFunctionParameterPropertyInfo `json:"visualizationGoal"`
}

type ToolFunctionParametersInfo struct {
	Type       string                          `json:"type"`
	Properties ToolFunctionParameterProperties `json:"properties"`
	Required   []string                        `json:"required"`
}

type ToolFunctionConfig struct {
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Parameters  ToolFunctionParametersInfo `json:"parameters"`
}

type ToolConfig struct {
	Type     string             `json:"type"`
	Function ToolFunctionConfig `json:"function"`
}

type ToolFunctionArgs struct {
	Data              string `json:"data"`
	Prompt            string `json:"prompt"`
	VisualizationGoal string `json:"visualizationGoal"`
}

const TOOLS_JSON_PATH = "/home/zeke/Documents/Repos/goProjects/openaiAgent/src/tools/tools.json"
const SYSTEM_PROMPT = "You are a helpful assistant that can answer questions about the Store Sales Price Elasticity Promotions dataset."

func loadToolsJson() []ToolConfig {
	jsonFile, err := os.Open(TOOLS_JSON_PATH)
	if err != nil {
		panic(err)
	}

	config := []ToolConfig{}
	json.NewDecoder(jsonFile).Decode(&config)
	return config
}

func handleToolCalls(
	toolCalls []openai.ChatCompletionMessageToolCall,
	messages []openai.ChatCompletionMessageParamUnion,
) []openai.ChatCompletionMessageParamUnion {
	for _, toolCall := range toolCalls {
		functionName := toolCall.Function.Name
		functionArgs := ToolFunctionArgs{}
		err := json.Unmarshal([]byte(toolCall.Function.Arguments), &functionArgs)
		if err != nil {
			panic(err)
		}

		result := ""
		switch functionName {
		case "LookUpSalesData":
			result = tools.LookupSalesData(functionArgs.Prompt)
		case "AnalyzeSalesData":
			result = tools.AnalyzeSalesData(functionArgs.Prompt, functionArgs.Data)
		case "GenerateVisualization":
			result = tools.GenerateVisualization(functionArgs.Data, functionArgs.VisualizationGoal)
		default:
		}

		messages = append(messages, openai.ToolMessage(toolCall.ID, result))
	}

	return messages
}

func formatAgentMessages(messages interface{}) []openai.ChatCompletionMessageParamUnion {
	fmt.Printf("Running agent with messages: %+v\n", messages)
	var openaiMessages []openai.ChatCompletionMessageParamUnion

	switch value := messages.(type) {
	case string:
		openaiMessages = []openai.ChatCompletionMessageParamUnion{openai.UserMessage(value)}
	case []openai.ChatCompletionMessageParamUnion:
		openaiMessages = value
	default:
		panic(fmt.Errorf("messages are not on expected types"))
	}

	hasSystemMessage := false
	for _, message := range openaiMessages {
		if _, ok := message.(openai.ChatCompletionSystemMessageParam); ok {
			hasSystemMessage = true
			break
		}
	}

	if !hasSystemMessage {
		fmt.Println("doesn't. Adding")
		openaiMessages = append(openaiMessages, openai.SystemMessage(SYSTEM_PROMPT))
	}

	return openaiMessages
}

func main() {
	//openaiMessages := []openai.ChatCompletionMessageParamUnion{openai.SystemMessage("You are helpful")}
	openaiMessages := "Hello"
	fmt.Printf("%+v\n", formatAgentMessages(openaiMessages))
}
