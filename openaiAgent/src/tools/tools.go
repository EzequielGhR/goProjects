package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

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
const dataPath = "/home/zeke/Documents/Repos/goProjects/openaiAgent/data/Store_Sales_Price_Elasticity_Promotions_Data.parquet"
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

/*
-------------
Aux functions
-------------
*/

// Use the same client for all calls, here and on main logic
func GetOpenaiClient() *openai.Client {
	if openaiClient == nil {
		log.Println("Creating new OpenAI client")
		openaiClient = openai.NewClient()
	}

	return openaiClient
}

// Necessary for structured outputs
func generateSchema[T any]() interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}

	var v T
	schema := reflector.Reflect(v)
	return schema
}

// Extract rows as an array of strings
func extractFromRows(rows *sql.Rows, columnsAmount int) ([]string, error) {
	// Initialize return value
	resultData := []string{}

	// Create an array of interfaces with the size being the amount of columns
	// Each interface will have a string pointer for later populatin with rows.Scan
	dynamicValues := make([]interface{}, columnsAmount)
	for i := range dynamicValues {
		dynamicValues[i] = new(string)
	}

	log.Println("Processing rows to strings")
	for rows.Next() {
		// Scan row values into previously created string pointers
		err := rows.Scan(dynamicValues...)
		if err != nil {
			return []string{}, err
		}

		rowValues := []string{}
		for _, value := range dynamicValues {
			// Type assert to string pointers, derreference and append to row values
			content, ok := value.(*string)
			if ok {
				rowValues = append(rowValues, *content)
			} else {
				rowValues = append(rowValues, "")
			}
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

	// Use structure outputs to get the chart config as expected
	response, err := GetOpenaiClient().Chat.Completions.New(
		context.TODO(),
		openai.ChatCompletionNewParams{
			Model: openai.F(Model),
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(formattedPrompt),
			}),
			ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
				openai.ResponseFormatJSONSchemaParam{
					Type: openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
					JSONSchema: openai.F(openai.ResponseFormatJSONSchemaJSONSchemaParam{
						Name:        openai.F("chartConfiguration"),
						Description: openai.F("A simple configuration for a chart"),
						Schema:      openai.F(visualConfigSchema),
						Strict:      openai.Bool(true),
					}),
				},
			),
		},
	)

	if err != nil {
		log.Printf("WARNING: %s\n", err)
		return returnValue
	}

	jsonData := strings.Trim(strings.ReplaceAll(response.Choices[0].Message.Content, "```json", ""), "`\n ")

	// Convert response to json
	vconf := visualizationConfig{}
	err = json.Unmarshal([]byte(jsonData), &vconf)
	if err != nil {
		log.Printf("WARNING: %s\n", err)
		return returnValue
	}

	returnValue.Config = vconf

	return returnValue
}

// Second part of the visualization tool. Generate code from chart
func createChart(config visualizationConfigData) string {
	formattedPrompt := fmt.Sprintf(createChartPrompt, config)

	response, err := GetOpenaiClient().Chat.Completions.New(
		context.TODO(),
		openai.ChatCompletionNewParams{
			Model: openai.F(Model),
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(formattedPrompt),
			}),
		},
	)

	if err != nil {
		log.Printf("WARNING: Failed OpenAI interaction: %s\n", err)
		return ""
	}

	return strings.Trim(strings.ReplaceAll(response.Choices[0].Message.Content, "```python", ""), "`\n ")
}

// Create a query from a user prompt
func generateSqlQuery(prompt string, columns []string, tableName string) (string, error) {
	formattedPrompt := fmt.Sprintf(
		sqlGenerationPrompt,
		prompt,
		strings.Join(columns, ", "), tableName,
	)

	response, err := GetOpenaiClient().Chat.Completions.New(
		context.TODO(),
		openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(formattedPrompt),
			}),
			Model: openai.F(openai.ChatModelGPT4oMini),
		},
	)

	if err != nil {
		log.Printf("WARNING: Failed OpenAI interaction: %s\n", err)
		return "", err
	}

	return response.Choices[0].Message.Content, nil
}

/*
-----------
Agent tools
-----------
*/

// Tool for sales lookup
func LookUpSalesData(prompt string) string {
	tableName := "sales"

	// Open or Creae DB
	db, err := sql.Open("duckdb", "data.db")
	if err != nil {
		return fmt.Sprintf("Failed to open Database: %s\n", err)
	}
	defer db.Close()

	// Create table
	_, err = db.Exec(
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s AS
			SELECT * FROM read_parquet('%s')`,
			tableName,
			dataPath,
		),
	)

	if err != nil {
		return fmt.Sprintf("Failed to execute table creation SQL: %s\n", err)
	}

	// Do a simple non-match query to return table columns
	result, err := db.Query(
		fmt.Sprintf("SELECT * FROM %s WHERE 1=2", tableName),
	)

	if err != nil {
		return fmt.Sprintf("Failed to fetch database columns: %s\n", err)
	}

	columns, err := result.Columns()
	if err != nil {
		return fmt.Sprintf("Failed to fetch database columns: %s\n", err)
	}

	sqlQuery, err := generateSqlQuery(prompt, columns, tableName)
	if err != nil {
		return fmt.Sprintf("Failed to generate SQL query: %s\n", err)
	}

	sqlQuery = strings.Trim(sqlQuery, "\n ")
	sqlQuery = strings.ReplaceAll(sqlQuery, "```sql", "")
	sqlQuery = strings.Trim(sqlQuery, "`")

	log.Printf("Query to be used: %s\n", sqlQuery)

	rows, err := db.Query(sqlQuery)
	if err != nil {
		return fmt.Sprintf("Failed to select data from database: %s\n", err)
	}

	resultData := []string{strings.Join(columns, ", ")}
	extractedRows, err := extractFromRows(rows, len(columns))
	if err != nil {
		return fmt.Sprintf("Failed to extract data from columns: %s\n", err)
	}

	resultData = append(resultData, extractedRows...)
	return strings.Join(resultData, "\n")
}

// Tool for data analysis
func AnalyzeSalesData(prompt string, data string) string {
	var finalAnalysis string
	formatedPrompt := fmt.Sprintf(dataAnalysisPrompt, data, prompt)
	response, err := GetOpenaiClient().Chat.Completions.New(
		context.TODO(),
		openai.ChatCompletionNewParams{
			Model: openai.F(Model),
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(formatedPrompt),
			}),
		},
	)

	if err != nil {
		log.Printf("There was an issue with the OpenAI interaction: %s\n", err)
		finalAnalysis = ""
	} else {
		finalAnalysis = strings.Trim(response.Choices[0].Message.Content, "\n ")
	}

	if finalAnalysis == "" {
		return "No analysis could be generated"
	}

	return finalAnalysis
}

// Tool for data visualization
func GenerateVisualization(data string, visualizationGoal string) string {
	config := extractChartConfig(data, visualizationGoal)
	code := createChart(config)
	return code
}
