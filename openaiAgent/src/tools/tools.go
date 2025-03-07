package tools

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"traceTools"

	"github.com/invopop/jsonschema"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/openai/openai-go"
)

/*
--------------------------
Important type definitions
--------------------------
*/

type visualizationConfig struct {
	ChartType string `json:"chartType" jsonschema_description:"Type of chart to generate"`
	XAxis     string `json:"xAxis" jsonschema_description:"Name of the X Axis column"`
	YAxis     string `json:"yAxis" jsonschema_description:"Name of the Y Axis column"`
	Title     string `json:"title" jsonschema_description:"Title of the chart"`
}

type visualizationConfigData struct {
	Config visualizationConfig
	Data   string
}

/*
---------------------------
Prompts and other constants
---------------------------
*/

const sqlGenerationPrompt = `
Generate an SQL query based on a prompt. Do not reply with anything besides the SQL query.
The prompt is:
%s

The available columns are: %s
The table name is: %s
`
const dataAnalysisPrompt = `
Analyze the following data: %s
Your job is to answer the following question: %s
`
const chartConfigPrompt = `
Generate a chart configuration based on this data: %s
The goal is to show: %s
`
const createChartPrompt = `
Wrtie python code to create a chart based on the following configuration.
Only return the code, no other text.
config: %+v
`
const tableName = "sales"
const Model = openai.ChatModelGPT4oMini
const LookUpFuncName = "LookUpSalesData"
const AnalyzeFuncName = "AnalyzeSalesData"
const VisualizeFuncName = "GenerateVisualization"

/*
------------------
Global definitions
------------------
*/

var openaiClient *openai.Client = nil
var visualConfigSchema = generateSchema[visualizationConfig]()
var DataPath string = "data/Store_Sales_Price_Elasticity_Promotions_Data.parquet"
var ToolsJsonPath string = "data/tools.json"

/*
-------------
Aux functions
-------------
*/

// Panic if Data doesn't exist at provided path. Redefine global var otherwise
func AssertDataPath(providedPath string) {
	if strings.HasSuffix(providedPath, ".parquet") {
		DataPath = providedPath
	}

	if _, err := os.Stat(DataPath); err != nil {
		log.Panicf("No parquet data file found at %s\n", DataPath)
	}
}

// Panic if Json doesn't exist at provided path. Redefine global var otherwise
func AssertToolsPath(providedPath string) {
	if strings.HasSuffix(providedPath, ".json") {
		ToolsJsonPath = providedPath
	}

	if _, err := os.Stat(ToolsJsonPath); err != nil {
		log.Panicf("No json file found at %s\n", ToolsJsonPath)
	}
}

// Use the same client for all calls, here and on main logic
func GetOpenaiClient() *openai.Client {
	if openaiClient == nil {
		log.Println("Creating new OpenAI client")
		openaiClient = openai.NewClient()
	}

	return openaiClient
}

// Necessary for structured outputs
func generateSchema[T any]() any {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}

	var v T
	schema := reflector.Reflect(v)
	return schema
}

// Clean up code block response from the LLM
func cleanLlmBlockResponse(llmResponse string) string {
	for _, t := range []string{"```json", "```sql", "```python"} {
		llmResponse = strings.ReplaceAll(llmResponse, t, "")
	}

	return strings.Trim(llmResponse, "`\n ")
}

// Extract rows as an array of strings
func extractFromRows(rows *sql.Rows, columnsAmount int) ([]string, error) {
	// Create two arrays of interfaces with the size being the amount of columns
	// One will be defined as pointers to the values of the other. That way providing
	// it to rows.Scan, will alter the other one's values by reference
	dynamicValues := make([]any, columnsAmount)
	pointers := make([]any, columnsAmount)

	for i := range dynamicValues {
		pointers[i] = &dynamicValues[i]
	}

	log.Println("Processing rows to strings")
	resultData := []string{}
	for rows.Next() {
		// Scan row values into previously created interface pointers
		err := rows.Scan(pointers...)
		if err != nil {
			return []string{}, err
		}

		rowValues := []string{}
		// dynamicValues' values were altered by reference, so it now contains the fields
		for _, value := range dynamicValues {
			// TODO: Find better ways to do it. For now just lazy print interface to convert to string
			content := fmt.Sprintf("%v", value)
			rowValues = append(rowValues, content)
		}

		resultData = append(resultData, strings.Join(rowValues, ", "))
	}

	return resultData, nil
}

// First part of data visualization tool. Extract a chart config to create code for visualization
func extractChartConfig(data string, visualizationGoal string) visualizationConfigData {
	returnValue := visualizationConfigData{
		Config: visualizationConfig{
			ChartType: "line",
			XAxis:     "date",
			YAxis:     "value",
			Title:     visualizationGoal,
		},
		Data: data,
	}

	formattedPrompt := fmt.Sprintf(chartConfigPrompt, data, visualizationGoal)

	// Initialize span as subspan of the latest tool span. Only track context locally
	ctx, span := traceTools.StartOpenInferenceSpan("ExtractChart", traceTools.ChainKind, traceTools.LastToolContext)
	defer traceTools.EndOpenInferenceSpan(span)

	traceTools.SetSpanInput(span, formattedPrompt)

	// Start OpenAI manual tracing
	llmCtx, llmSpan := traceTools.StartOpenAISpan(ctx, Model)
	defer llmSpan.End()

	inputMessage := openai.F([]openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(formattedPrompt),
	})

	responseFormat := openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
		openai.ResponseFormatJSONSchemaParam{
			Type: openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
			JSONSchema: openai.F(openai.ResponseFormatJSONSchemaJSONSchemaParam{
				Name:        openai.F("chartConfiguration"),
				Description: openai.F("A simple configuration for a chart"),
				Schema:      openai.F(visualConfigSchema),
				Strict:      openai.Bool(true),
			}),
		},
	)

	// Add input attributes to llm span
	traceTools.SetSpanAttrFromMap(llmSpan, map[string]any{
		"llm.input_messages":  []string{inputMessage.String()},
		"llm.response_format": responseFormat.String(),
	})

	// Use structure outputs to get the chart config as expected
	// For this use ResponseFormat Param with the desired json schema
	response, err := GetOpenaiClient().Chat.Completions.New(
		llmCtx,
		openai.ChatCompletionNewParams{
			Model:          openai.F(Model),
			Messages:       inputMessage,
			ResponseFormat: responseFormat,
		},
	)

	if err != nil {
		traceTools.SetSpanErrorCode(llmSpan)
		traceTools.SetSpanErrorCode(span)
		log.Printf("WARNING: %s\n", err)
		return returnValue
	}

	responseMessage := response.Choices[0].Message
	jsonData := cleanLlmBlockResponse(responseMessage.Content)

	// Set llm span output attributes
	traceTools.SetSpanAttrFromMap(llmSpan, map[string]any{
		"llm.token_count.prompt":     int(response.Usage.PromptTokens),
		"llm.token_count.completion": int(response.Usage.CompletionTokens),
		"llm.token_count.total":      int(response.Usage.TotalTokens),
		"llm.output_messages":        []string{openai.F(responseMessage).String()},
		"llm.tools":                  []string{},
	})

	// Convert response to json
	vconf := visualizationConfig{}
	err = json.Unmarshal([]byte(jsonData), &vconf)
	if err != nil {
		traceTools.SetSpanErrorCode(llmSpan)
		traceTools.SetSpanErrorCode(span)
		log.Printf("WARNING: %s\n", err)
		return returnValue
	}

	returnValue.Config = vconf

	traceTools.SetSpanSuccessCode(llmSpan)
	traceTools.SetSpanSuccessCode(span)
	traceTools.SetSpanOutput(span, jsonData)

	return returnValue
}

// Second part of the visualization tool. Generate code from chart
func createChart(config visualizationConfigData) string {
	formattedPrompt := fmt.Sprintf(createChartPrompt, config)
	// Initialize span as subspan of the latest tool span. Only track context locally
	ctx, span := traceTools.StartOpenInferenceSpan("CreateChart", traceTools.ChainKind, traceTools.LastToolContext)
	defer traceTools.EndOpenInferenceSpan(span)

	traceTools.SetSpanInput(span, formattedPrompt)

	// Start OpenAI manual tracing
	llmCtx, llmSpan := traceTools.StartOpenAISpan(ctx, Model)
	defer llmSpan.End()

	inputMessage := openai.F([]openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(formattedPrompt),
	})

	// Add input attributes to llm span
	traceTools.SetSpanAttr(llmSpan, "llm.input_messages", []string{inputMessage.String()})

	response, err := GetOpenaiClient().Chat.Completions.New(
		llmCtx,
		openai.ChatCompletionNewParams{
			Model:    openai.F(Model),
			Messages: inputMessage,
		},
	)

	if err != nil {
		traceTools.SetSpanErrorCode(llmSpan)
		traceTools.SetSpanErrorCode(span)
		log.Printf("WARNING: Failed OpenAI interaction: %s\n", err)
		return ""
	}

	responseMessage := response.Choices[0].Message

	// Add output attributes to llm span
	traceTools.SetSpanAttrFromMap(llmSpan, map[string]any{
		"llm.token_count.prompt":     int(response.Usage.PromptTokens),
		"llm.token_count.completion": int(response.Usage.CompletionTokens),
		"llm.token_count.total":      int(response.Usage.TotalTokens),
		"llm.output_messages":        []string{openai.F(responseMessage).String()},
		"llm.tools":                  []string{},
	})
	traceTools.SetSpanSuccessCode(llmSpan)

	pythonCode := cleanLlmBlockResponse(responseMessage.Content)
	traceTools.SetSpanOutput(span, pythonCode)
	traceTools.SetSpanSuccessCode(span)

	return pythonCode
}

// Create a query from a user prompt
func generateSqlQuery(prompt string, columns []string, tableName string) (string, error) {
	formattedPrompt := fmt.Sprintf(
		sqlGenerationPrompt,
		prompt,
		strings.Join(columns, ", "), tableName,
	)

	// Initialize span as subspan of the latest tool span. Only track context locally
	ctx, span := traceTools.StartOpenInferenceSpan("SqlGeneration", traceTools.ChainKind, traceTools.LastToolContext)
	defer traceTools.EndOpenInferenceSpan(span)

	traceTools.SetSpanInput(span, formattedPrompt)

	// Manually trace OpenAI calls
	llmCtx, llmSpan := traceTools.StartOpenAISpan(ctx, Model)
	defer llmSpan.End()

	inputMessage := openai.F([]openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(formattedPrompt),
	})

	// Add input attributes to llm span
	traceTools.SetSpanAttr(llmSpan, "llm.input_messages", []string{inputMessage.String()})

	response, err := GetOpenaiClient().Chat.Completions.New(
		llmCtx,
		openai.ChatCompletionNewParams{
			Messages: inputMessage,
			Model:    openai.F(openai.ChatModelGPT4oMini),
		},
	)

	if err != nil {
		traceTools.SetSpanErrorCode(llmSpan)
		traceTools.SetSpanErrorCode(span)
		log.Printf("WARNING: Failed OpenAI interaction: %s\n", err)
		return "", err
	}

	answer := response.Choices[0].Message

	// Add output attributes to llm span
	traceTools.SetSpanAttrFromMap(llmSpan, map[string]any{
		"llm.token_count.prompt":     int(response.Usage.PromptTokens),
		"llm.token_count.completion": int(response.Usage.CompletionTokens),
		"llm.token_count.total":      int(response.Usage.TotalTokens),
		"llm.output_messages":        []string{openai.F(answer).String()},
		"llm.tools":                  []string{},
	})
	traceTools.SetSpanSuccessCode(llmSpan)
	traceTools.SetSpanOutput(span, answer.Content)
	traceTools.SetSpanSuccessCode(span)

	return answer.Content, nil
}

/*
-----------
Agent tools
-----------
*/

// Tool for sales lookup
func LookUpSalesData(prompt string) string {
	// Start span as sub span of the handleToolCalls span and update the latest tool context global variable
	ctx, span := traceTools.StartOpenInferenceSpan("LookUpTool", traceTools.ToolKind, traceTools.HandleToolContext)
	defer traceTools.EndOpenInferenceSpan(span)
	traceTools.LastToolContext = ctx

	traceTools.SetSpanInput(span, prompt)

	// Open or Create DB
	db, err := sql.Open("duckdb", "data.db")
	if err != nil {
		log.Printf("WARNING: %s\n", err)
		traceTools.SetSpanErrorCode(span)
		return fmt.Sprintf("Failed to open Database: %s\n", err)
	}
	defer db.Close()

	// Create table
	_, err = db.Exec(
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s AS
			SELECT * FROM read_parquet('%s')`,
			tableName,
			DataPath,
		),
	)

	if err != nil {
		log.Printf("WARNING: %s\n", err)
		traceTools.SetSpanErrorCode(span)
		return fmt.Sprintf("Failed to execute table creation SQL: %s\n", err)
	}

	// Do a simple non-match query to return table columns
	result, err := db.Query(
		fmt.Sprintf("SELECT * FROM %s WHERE 1=2", tableName),
	)

	if err != nil {
		log.Printf("WARNING: %s\n", err)
		traceTools.SetSpanErrorCode(span)
		return fmt.Sprintf("Failed to fetch database columns: %s\n", err)
	}

	columns, err := result.Columns()
	if err != nil {
		log.Printf("WARNING: %s\n", err)
		traceTools.SetSpanErrorCode(span)
		return fmt.Sprintf("Failed to fetch database columns: %s\n", err)
	}

	sqlQuery, err := generateSqlQuery(prompt, columns, tableName)
	if err != nil {
		log.Printf("WARNING: %s\n", err)
		traceTools.SetSpanErrorCode(span)
		return fmt.Sprintf("Failed to generate SQL query: %s\n", err)
	}

	sqlQuery = cleanLlmBlockResponse(sqlQuery)
	log.Printf("Query to be used: %s\n", sqlQuery)

	rows, err := db.Query(sqlQuery)
	if err != nil {
		log.Printf("WARNING: %s\n", err)
		traceTools.SetSpanErrorCode(span)
		return fmt.Sprintf("Failed to select data from database: %s\n", err)
	}

	columns, err = rows.Columns()
	if err != nil {
		log.Printf("WARNING: %s\n", err)
		traceTools.SetSpanErrorCode(span)
		return fmt.Sprintf("Failed to fetch query result columns: %s\n", err)
	}

	resultData := []string{strings.Join(columns, ", ")}
	extractedRows, err := extractFromRows(rows, len(columns))
	if err != nil {
		log.Printf("WARNING: %s\n", err)
		traceTools.SetSpanErrorCode(span)
		return fmt.Sprintf("Failed to extract data from columns: %s\n", err)
	}

	resultData = append(resultData, extractedRows...)
	returnValue := strings.Join(resultData, "\n")

	traceTools.SetSpanOutput(span, returnValue)
	traceTools.SetSpanSuccessCode(span)

	return returnValue
}

// Tool for data analysis
func AnalyzeSalesData(prompt string, data string) string {
	formatedPrompt := fmt.Sprintf(dataAnalysisPrompt, data, prompt)

	// Start span as sub span of the handleToolCalls span and update the latest tool context global variable
	ctx, span := traceTools.StartOpenInferenceSpan("AnalyzeTool", traceTools.ToolKind, traceTools.HandleToolContext)
	defer traceTools.EndOpenInferenceSpan(span)
	traceTools.LastToolContext = ctx

	traceTools.SetSpanInput(span, formatedPrompt)
	inputMessage := openai.F([]openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(formatedPrompt),
	})

	// Start OpenAI manual tracing
	llmCtx, llmSpan := traceTools.StartOpenAISpan(ctx, Model)
	defer llmSpan.End()

	// Add input attributes to llm span
	traceTools.SetSpanAttr(llmSpan, "llm.input_messages", []string{inputMessage.String()})

	response, err := GetOpenaiClient().Chat.Completions.New(
		llmCtx,
		openai.ChatCompletionNewParams{
			Model:    openai.F(Model),
			Messages: inputMessage,
		},
	)

	finalAnalysis := ""
	responseMessages := []string{}
	promptTokens, completionTokens, totalTokens := 0, 0, 0
	if err == nil {
		responseMessage := response.Choices[0].Message
		finalAnalysis = strings.Trim(responseMessage.Content, "\n ")
		responseMessages = []string{openai.F(responseMessages).String()}

		promptTokens = int(response.Usage.PromptTokens)
		completionTokens = int(response.Usage.CompletionTokens)
		totalTokens = int(response.Usage.TotalTokens)
	} else {
		log.Printf("WARNING: There was an issue with the OpenAI interaction: %s\n", err)
	}

	// Set output attributes to llm span
	traceTools.SetSpanAttrFromMap(llmSpan, map[string]any{
		"llm.token_count.prompt":     promptTokens,
		"llm.token_count.completion": completionTokens,
		"llm.token_count.total":      totalTokens,
		"llm.output_messages":        responseMessages,
		"llm.tools":                  []string{},
	})

	if finalAnalysis == "" {
		traceTools.SetSpanErrorCode(span)
		traceTools.SetSpanErrorCode(llmSpan)
		return "No analysis could be generated"
	}

	traceTools.SetSpanOutput(span, finalAnalysis)
	traceTools.SetSpanSuccessCode(llmSpan)
	traceTools.SetSpanSuccessCode(span)
	return finalAnalysis
}

// Tool for data visualization
func GenerateVisualization(data string, visualizationGoal string) string {
	// Start span as sub span of the handleToolCalls span and update the latest tool context global variable
	ctx, span := traceTools.StartOpenInferenceSpan("VisualizationTool", traceTools.ToolKind, traceTools.HandleToolContext)
	defer traceTools.EndOpenInferenceSpan(span)
	traceTools.LastToolContext = ctx

	traceTools.SetSpanInput(span, []string{data, visualizationGoal})

	config := extractChartConfig(data, visualizationGoal)
	code := createChart(config)

	traceTools.SetSpanOutput(span, code)
	traceTools.SetSpanSuccessCode(span)

	return code
}
