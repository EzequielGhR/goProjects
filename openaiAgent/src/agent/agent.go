package agent

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"tools"
	"traceTools"

	"github.com/openai/openai-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

/*
---------------------------------
Tools Json structure with structs
---------------------------------
*/

// Property information for tool function parameter
type toolFunctionParameterPropertyInfo struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// Properties for tool function
type toolFunctionParameterProperties struct {
	Data              toolFunctionParameterPropertyInfo `json:"data"`
	Prompt            toolFunctionParameterPropertyInfo `json:"prompt"`
	VisualizationGoal toolFunctionParameterPropertyInfo `json:"visualizationGoal"`
}

// Parameters information fot tool function
type toolFunctionParametersInfo struct {
	Type       string                          `json:"type"`
	Properties toolFunctionParameterProperties `json:"properties"`
	Required   []string                        `json:"required"`
}

// Main level of configuration for tool function
type toolFunctionConfig struct {
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Parameters  toolFunctionParametersInfo `json:"parameters"`
}

// Tool config base level
type toolConfig struct {
	Type     string             `json:"type"`
	Function toolFunctionConfig `json:"function"`
}

type toolFunctionArgs struct {
	Data              string `json:"data"`
	Prompt            string `json:"prompt"`
	VisualizationGoal string `json:"visualizationGoal"`
}

type AgentInput interface {
	string | []openai.ChatCompletionMessageParamUnion
}

/*
---------
Constants
---------
*/

const toolsJsonPath = "/home/zeke/Documents/Repos/goProjects/openaiAgent/src/tools/tools.json"
const systemPrompt = "You are a helpful assistant that can answer questions about the Store Sales Price Elasticity Promotions dataset."

/*
-------------
Aux functions
-------------
*/

func loadToolsJson() []toolConfig {
	log.Println("Loading tools json ...")
	jsonFile, err := os.Open(toolsJsonPath)
	if err != nil {
		panic(err)
	}

	config := []toolConfig{}
	json.NewDecoder(jsonFile).Decode(&config)
	return config
}

// Handle the different tool calls and append result messages to ongoing conversation
func handleToolCalls(
	toolCalls []openai.ChatCompletionMessageToolCall,
	messages []openai.ChatCompletionMessageParamUnion,
) []openai.ChatCompletionMessageParamUnion {
	tracer := traceTools.GetActiveTracer()
	_, span := tracer.Start(context.Background(), "handleToolCalls")
	defer span.End()

	span.SetAttributes(attribute.String("ai.model", tools.Model))

	inputAttr := []string{}
	outputAttr := []string{}
	for _, toolCall := range toolCalls {
		inputAttr = append(inputAttr, toolCall.JSON.RawJSON())

		functionName := toolCall.Function.Name
		functionArgs := toolFunctionArgs{}

		log.Printf("Processing Tool Call '%s' for function '%s'\n", toolCall.ID, functionName)

		err := json.Unmarshal([]byte(toolCall.Function.Arguments), &functionArgs)
		if err != nil {
			span.SetStatus(codes.Error, "There was an issue unwraping function args")
			log.Fatal(err)
		}

		result := ""
		switch functionName {
		case tools.LookUpFuncName:
			result = tools.LookUpSalesData(functionArgs.Prompt)
		case tools.AnalyzeFuncName:
			result = tools.AnalyzeSalesData(functionArgs.Prompt, functionArgs.Data)
		case tools.VisualizeFuncName:
			result = tools.GenerateVisualization(functionArgs.Data, functionArgs.VisualizationGoal)
		default:
			log.Fatal("Invalid function name")
		}

		response := openai.ToolMessage(toolCall.ID, result)
		messages = append(messages, response)
		outputAttr = append(outputAttr, response.Content.String())
	}

	span.SetAttributes(
		attribute.String("ai.input", "["+strings.Join(inputAttr, ",\n")+"["),
		attribute.String("ai.output", strings.Join(outputAttr, "\n")),
	)

	span.SetStatus(codes.Ok, "Finished tools processing")
	return messages
}

// Correctly format messages for agent handling
func formatAgentMessages[T AgentInput](messages T) []openai.ChatCompletionMessageParamUnion {
	var openaiMessages []openai.ChatCompletionMessageParamUnion

	// Convert the message to expected format if necessary
	switch value := any(messages).(type) {
	case string:
		log.Println("Converting string message")
		openaiMessages = []openai.ChatCompletionMessageParamUnion{openai.UserMessage(value)}
	case []openai.ChatCompletionMessageParamUnion:
		openaiMessages = value
	default:
		log.Fatal("messages are not on expected types")
	}

	// Add a system message if none present
	hasSystemMessage := false
	for _, message := range openaiMessages {
		if _, ok := message.(openai.ChatCompletionSystemMessageParam); ok {
			hasSystemMessage = true
			break
		}
	}

	if !hasSystemMessage {
		log.Println("Adding system message")
		tempMessages := openaiMessages
		openaiMessages = []openai.ChatCompletionMessageParamUnion{openai.SystemMessage(systemPrompt)}
		openaiMessages = append(openaiMessages, tempMessages...)
	}

	return openaiMessages
}

// Convert an array of tool configs to openai expected tool param
func convertToolConfigToParams(toolConfigs []toolConfig) []openai.ChatCompletionToolParam {
	openaiToolParam := []openai.ChatCompletionToolParam{}
	for _, config := range toolConfigs {
		log.Printf("Converting tool config for function '%s' to param\n", config.Function.Name)

		// Each config has its own properties, map them using the function name
		var propertiesMap map[string]interface{}
		switch config.Function.Name {
		case tools.LookUpFuncName:
			propertiesMap = map[string]interface{}{
				"prompt": map[string]string{
					"type": config.Function.Parameters.Properties.Prompt.Type,
				},
			}
		case tools.AnalyzeFuncName:
			propertiesMap = map[string]interface{}{
				"prompt": map[string]string{
					"type": config.Function.Parameters.Properties.Prompt.Type,
				},
				"data": map[string]string{
					"type": config.Function.Parameters.Properties.Data.Type,
				},
			}
		case tools.VisualizeFuncName:
			propertiesMap = map[string]interface{}{
				"data": map[string]string{
					"type": config.Function.Parameters.Properties.Data.Type,
				},
				"visualizationGoal": map[string]string{
					"type": config.Function.Parameters.Properties.VisualizationGoal.Type,
				},
			}
		default:
			log.Fatal("Unexpected function name")
		}

		// Add each config as a param
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

/*
-------------------
Main Agent function
-------------------
*/

func RunAgent[T AgentInput](messages T) string {
	tracer := traceTools.GetActiveTracer()
	openaiMessages := formatAgentMessages(messages)
	openaiToolParams := convertToolConfigToParams(loadToolsJson())

	for {
		log.Println("Making router call for OpenAI")
		_, span := tracer.Start(context.Background(), "RouterCall", trace.WithSpanKind(trace.SpanKindInternal))

		inputMessage := openai.F(openaiMessages[len(openaiMessages)-1]).String()
		span.SetAttributes(
			attribute.String("ai.model", tools.Model),
			attribute.String("ai.input", inputMessage),
		)

		response, err := tools.GetOpenaiClient().Chat.Completions.New(
			context.TODO(),
			openai.ChatCompletionNewParams{
				Model:     openai.F(tools.Model),
				Messages:  openai.F(openaiMessages),
				Tools:     openai.F(openaiToolParams),
				MaxTokens: openai.Int(1000),
			},
		)

		if err != nil {
			log.Println("Here")
			span.SetStatus(codes.Error, "There was an issue with the openai interaction")
			span.End()
			log.Fatal(err)
		}

		// Add the response to a tool call message, needed for next steps
		responseMessage := response.Choices[0].Message
		openaiMessages = append(openaiMessages, responseMessage)
		toolCalls := response.Choices[0].Message.ToolCalls

		rawJsonToolCalls := []string{}
		for _, toolCall := range toolCalls {
			rawJsonToolCalls = append(rawJsonToolCalls, toolCall.JSON.RawJSON())
		}

		span.SetStatus(codes.Ok, "Successful tool call iteration")

		if len(toolCalls) != 0 {
			log.Println("Processing tool calls ...")
			span.SetAttributes(attribute.String("ai.output", "["+strings.Join(rawJsonToolCalls, ",\n")+"]"))
			openaiMessages = handleToolCalls(toolCalls, openaiMessages)
		} else {
			span.SetAttributes(attribute.String("ai.output", responseMessage.Content))
			log.Println("No tool calls, returning final answer")
			return response.Choices[0].Message.Content
		}
	}
}
