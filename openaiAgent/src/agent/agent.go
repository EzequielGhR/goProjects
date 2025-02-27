package agent

import (
	"encoding/json"
	"log"
	"os"
	"tools"
	"traceTools"

	"github.com/openai/openai-go"
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

// Agent input interface
type AgentInput interface {
	string | []openai.ChatCompletionMessageParamUnion
}

/*
---------
Constants
---------
*/

const systemPrompt = "You are a helpful assistant that can answer questions about the Store Sales Price Elasticity Promotions dataset."

/*
-------------
Aux functions
-------------
*/

// Load tools config from a json file
func loadToolsJson() []toolConfig {
	log.Println("Loading tools json ...")
	jsonFile, err := os.Open(tools.ToolsJsonPath)
	if err != nil {
		log.Panicln(err)
	}
	defer jsonFile.Close()

	config := []toolConfig{}
	json.NewDecoder(jsonFile).Decode(&config)
	return config
}

// Handle the different tool calls and append result messages to ongoing conversation.
// Receives an array of tool calls and an array of current conversation messages.
func handleToolCalls(
	toolCalls []openai.ChatCompletionMessageToolCall,
	messages []openai.ChatCompletionMessageParamUnion,
) []openai.ChatCompletionMessageParamUnion {
	// Start Span as sub span of the last router call's. Set the global variable for the handleToolCalls span context
	ctx, span := traceTools.StartOpenInferenceSpan("HandleToolCalls", traceTools.ChainKind, traceTools.LastRouterContext)
	defer traceTools.EndOpenInferenceSpan(span)
	traceTools.HandleToolContext = ctx

	// Track input and output for span's attribute set up
	inputAttr := []string{}
	outputAttr := []string{}
	for _, toolCall := range toolCalls {
		// Update the input attribute
		inputAttr = append(inputAttr, toolCall.JSON.RawJSON())

		functionName := toolCall.Function.Name
		functionArgs := toolFunctionArgs{}

		log.Printf("Processing Tool Call '%s' for function '%s'\n", toolCall.ID, functionName)

		err := json.Unmarshal([]byte(toolCall.Function.Arguments), &functionArgs)
		if err != nil {
			traceTools.SetSpanErrorCode(span)
			log.Panic(err)
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
			traceTools.SetSpanErrorCode(span)
			log.Panic("Invalid function name")
		}

		response := openai.ToolMessage(toolCall.ID, result)
		messages = append(messages, response)

		// Update output attribute
		outputAttr = append(outputAttr, response.Content.String())
	}

	traceTools.SetSpanInput(span, inputAttr)
	traceTools.SetSpanOutput(span, outputAttr)
	traceTools.SetSpanModel(span, tools.Model)

	// Mark the execution as a success
	traceTools.SetSpanSuccessCode(span)
	return messages
}

// Correctly format messages for agent handling. Expects a type of AgentInput which can be
// a string or an array of ChatcompletionMessageParamUnion
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
		log.Panic("messages are not on expected types")
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
		var propertiesMap map[string]any
		switch config.Function.Name {
		case tools.LookUpFuncName:
			propertiesMap = map[string]any{
				"prompt": map[string]string{
					"type": config.Function.Parameters.Properties.Prompt.Type,
				},
			}
		case tools.AnalyzeFuncName:
			propertiesMap = map[string]any{
				"prompt": map[string]string{
					"type": config.Function.Parameters.Properties.Prompt.Type,
				},
				"data": map[string]string{
					"type": config.Function.Parameters.Properties.Data.Type,
				},
			}
		case tools.VisualizeFuncName:
			propertiesMap = map[string]any{
				"data": map[string]string{
					"type": config.Function.Parameters.Properties.Data.Type,
				},
				"visualizationGoal": map[string]string{
					"type": config.Function.Parameters.Properties.VisualizationGoal.Type,
				},
			}
		default:
			log.Panic("Unexpected function name")
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

func RunAgent[T AgentInput](messages T) (string, error) {
	openaiMessages := formatAgentMessages(messages)
	openaiToolParams := convertToolConfigToParams(loadToolsJson())

	for {
		log.Println("Making router call for OpenAI and starting new span")

		// Manually start span and set the las router call context global var
		// The span starts with the Agent span's context to work as child span
		ctx, span := traceTools.StartOpenInferenceSpan("RouterCall", traceTools.ChainKind, traceTools.AgentContext)
		defer traceTools.EndOpenInferenceSpan(span)
		traceTools.LastRouterContext = ctx

		inputMessage := openai.F(openaiMessages[len(openaiMessages)-1]).String()
		traceTools.SetSpanInput(span, inputMessage)
		traceTools.SetSpanModel(span, tools.Model)

		response, err := tools.GetOpenaiClient().Chat.Completions.New(
			ctx,
			openai.ChatCompletionNewParams{
				Model:     openai.F(tools.Model),
				Messages:  openai.F(openaiMessages),
				Tools:     openai.F(openaiToolParams),
				MaxTokens: openai.Int(1000),
			},
		)

		if err != nil {
			traceTools.SetSpanErrorCode(span)
			return "", err
		}

		// Add the response to a tool call message, needed for next steps
		responseMessage := response.Choices[0].Message
		openaiMessages = append(openaiMessages, responseMessage)
		toolCalls := response.Choices[0].Message.ToolCalls

		rawJsonToolCalls := []string{}
		for _, toolCall := range toolCalls {
			rawJsonToolCalls = append(rawJsonToolCalls, toolCall.JSON.RawJSON())
		}

		// Set span as successful
		traceTools.SetSpanSuccessCode(span)

		if len(toolCalls) != 0 {
			log.Println("Processing tool calls ...")
			traceTools.SetSpanOutput(span, rawJsonToolCalls)
			openaiMessages = handleToolCalls(toolCalls, openaiMessages)
		} else {
			log.Println("No tool calls, returning final answer")
			traceTools.SetSpanOutput(span, responseMessage.Content)
			return response.Choices[0].Message.Content, nil
		}
	}
}
