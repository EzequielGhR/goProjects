package main

import (
	"context"
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
		case tools.LOOKUP_FUNC_NAME:
			result = tools.LookupSalesData(functionArgs.Prompt)
		case tools.ANALYZE_FUNC_NAME:
			result = tools.AnalyzeSalesData(functionArgs.Prompt, functionArgs.Data)
		case tools.VISUALIZE_FUNC_NAME:
			result = tools.GenerateVisualization(functionArgs.Data, functionArgs.VisualizationGoal)
		default:
		}

		fmt.Println()
		fmt.Printf("DEBUG TOOLCALL RESULT: %s\n", result)
		fmt.Println()

		messages = append(messages, openai.ToolMessage(toolCall.ID, result))
	}

	return messages
}

func formatAgentMessages[T string | []openai.ChatCompletionMessageParamUnion](messages T) []openai.ChatCompletionMessageParamUnion {
	fmt.Printf("Running agent with messages: %+v\n", messages)
	var openaiMessages []openai.ChatCompletionMessageParamUnion

	switch value := any(messages).(type) {
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
		tempMessages := openaiMessages
		openaiMessages = []openai.ChatCompletionMessageParamUnion{openai.SystemMessage(SYSTEM_PROMPT)}
		openaiMessages = append(openaiMessages, tempMessages...)
	}

	return openaiMessages
}

func convertToolConfigToParams(toolConfigs []ToolConfig) []openai.ChatCompletionToolParam {
	openaiToolParam := []openai.ChatCompletionToolParam{}
	for _, config := range toolConfigs {
		var propertiesMap map[string]interface{}
		switch config.Function.Name {
		case tools.LOOKUP_FUNC_NAME:
			propertiesMap = map[string]interface{}{
				"prompt": map[string]string{
					"type": config.Function.Parameters.Properties.Prompt.Type,
				},
			}
		case tools.ANALYZE_FUNC_NAME:
			propertiesMap = map[string]interface{}{
				"prompt": map[string]string{
					"type": config.Function.Parameters.Properties.Prompt.Type,
				},
				"data": map[string]string{
					"type": config.Function.Parameters.Properties.Data.Type,
				},
			}
		case tools.VISUALIZE_FUNC_NAME:
			propertiesMap = map[string]interface{}{
				"data": map[string]string{
					"type": config.Function.Parameters.Properties.Data.Type,
				},
				"visualizationGoal": map[string]string{
					"type": config.Function.Parameters.Properties.VisualizationGoal.Type,
				},
			}
		default:
			panic(fmt.Errorf("unexpected function name: %s", config.Function.Name))
		}

		openaiToolParam = append(openaiToolParam, openai.ChatCompletionToolParam{
			Type: openai.F(openai.ChatCompletionToolType(config.Type)),
			Function: openai.F(openai.FunctionDefinitionParam{
				Name:        openai.String(config.Function.Name),
				Description: openai.String(config.Function.Description),
				Parameters: openai.F(openai.FunctionParameters{
					"type":       config.Function.Parameters.Type,
					"properties": propertiesMap,
					"required":   config.Function.Parameters.Required,
				}),
			}),
		})
	}

	return openaiToolParam
}

func runAgent[T string | []openai.ChatCompletionMessageParamUnion](messages T) string {
	openaiMessages := formatAgentMessages(messages)
	openaiToolParams := convertToolConfigToParams(loadToolsJson())

	for {
		fmt.Println("Making router call for OpenAI")

		fmt.Println()
		fmt.Printf("DEBUG MESSAGES: %+v", openaiMessages)
		fmt.Println()

		response, err := tools.GetOpenaiClient().Chat.Completions.New(
			context.TODO(),
			openai.ChatCompletionNewParams{
				Model:     openai.F(tools.MODEL),
				Messages:  openai.F(openaiMessages),
				Tools:     openai.F(openaiToolParams),
				MaxTokens: openai.Int(1000),
			},
		)

		if err != nil {
			panic(err)
		}

		openaiMessages = append(openaiMessages, response.Choices[0].Message)
		toolCalls := response.Choices[0].Message.ToolCalls

		fmt.Println()
		fmt.Printf("DEBUG TOOLS: %+v\n", toolCalls)
		fmt.Println()

		if len(toolCalls) != 0 {
			fmt.Println("Processing tool calls ...")
			openaiMessages = handleToolCalls(toolCalls, openaiMessages)
		} else {
			fmt.Println("No tool calls, returning final answer")
			return response.Choices[0].Message.Content
		}
	}
}

func main() {
	result := runAgent("Show me the code for graph of the top 10 sales by store in Nov 2021, and tell me what trends you see.")
	fmt.Println(result)
}
