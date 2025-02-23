package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/invopop/jsonschema"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/openai/openai-go"
)


type VisualizationConfig struct {
	ChartType string `json:"chartType" jsonschema_description:"Type of chart to generate"`
	XAxis string `json:"xAxis" jsonschema_description:"Name of the X Axis column"`
	YAxis string `json:"yAxis" jsonschema_description:"Name of the Y Axis column"`
	Title string `json:"title" jsonschema_description:"Title of the chart"`
}

type VisualizationConfigResponse struct {
	ChartType string `json:"chartType" jsonschema_description:"Type of chart to generate"`
	XAxis string `json:"xAxis" jsonschema_description:"Name of the X Axis column"`
	YAxis string `json:"yAxis" jsonschema_description:"Name of the Y Axis column"`
	Title string `json:"title" jsonschema_description:"Title of the chart"`
	Data string
}


const SQL_GENERATION_PROMPT = `
Generate an SQL query based on a prompt. Do not reply with anything besides the SQL query.
The prompt is:
%s

The available columns are: %s
The table name is: %s
`
const DATA_ANALYSIS_PROMPT = `
Analyze the following data: %s
Your job is to answer the following question: {prompt}
`

const CHART_CONFIG_PROMPT = `
Generate a chart configuration based on this data: %s
The goal is to show: %s
`
const DATA_PATH = "/home/zeke/Documents/Repos/goProjects/openaiAgent/data/Store_Sales_Price_Elasticity_Promotions_Data.parquet"
const MODEL = openai.ChatModelGPT4oMini


var openaiClient = openai.NewClient()

func GenerateSchema[T any]() interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference: true,
	}

	var _v T
	schema := reflector.Reflect(_v)
	return schema
}

var visualConfigSchema = GenerateSchema[VisualizationConfig]()

func generateSqlQuery(prompt string, columns []string, tableName string) (string, error) {
	formattedPrompt := fmt.Sprintf(
		SQL_GENERATION_PROMPT,
		prompt,
		strings.Join(columns, ", "), tableName,
	)

	response, err := openaiClient.Chat.Completions.New(
		context.TODO(),
		openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(formattedPrompt),
			}),
			Model: openai.F(openai.ChatModelGPT4oMini),
		},
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed OpenAI interaction: %s\n", err)
		return "", err
	}

	return response.Choices[0].Message.Content, nil
}

func lookupSalesData(prompt string) string {
	tableName := "sales"

	db, err := sql.Open("duckdb", "data.db")
	if err != nil {
		return fmt.Sprintf("Failed to open Database: %s\n", err)
	}
	defer db.Close()

	_, err = db.Exec(
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s AS
			SELECT * FROM read_parquet('%s')`,
			tableName,
			DATA_PATH,
		),
	)

	if err != nil {
		return fmt.Sprintf("Failed to execute table creation SQL: %s\n", err)
	}

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

func extractFromRows(rows *sql.Rows, columnsAmount int) ([]string, error) {
	resultData := []string{}
	dynamicValues := make([]interface{}, columnsAmount)
	for i := range dynamicValues {
		dynamicValues[i] = new(string)
	}

	for rows.Next() {
		err := rows.Scan(dynamicValues...)
		if err != nil {
			return []string{}, err
		}

		rowValues := []string{}
		for _, value := range dynamicValues {
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

func analyzeSalesData(prompt string, data string) string {
	var finalAnalysis string
	formatedPrompt := fmt.Sprintf(DATA_ANALYSIS_PROMPT, data, prompt)
	response, err := openaiClient.Chat.Completions.New(
		context.TODO(),
		openai.ChatCompletionNewParams{
			Model: openai.F(MODEL),
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(formatedPrompt),
			}),
		},
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "There was an issue with the OpenAI interaction: %s\n", err)
		finalAnalysis = ""
	} else {
		finalAnalysis = strings.Trim(response.Choices[0].Message.Content, "\n ")
	}

	if finalAnalysis == "" {
		return "No analysis could be generated"
	}

	return finalAnalysis
}

func extractChartConfig(data string, visualizationGoal string) VisualizationConfigResponse {
	returnValue := VisualizationConfigResponse{
		ChartType: "line",
		XAxis: "date",
		YAxis: "value",
		Title: visualizationGoal,
		Data: data,
	}

	formattedPrompt := fmt.Sprintf(CHART_CONFIG_PROMPT, data, visualizationGoal)

	response, err := openaiClient.Chat.Completions.New(
		context.TODO(),
		openai.ChatCompletionNewParams{
			Model: openai.F(MODEL),
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(formattedPrompt),
			}),
			ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
				openai.ResponseFormatJSONSchemaParam{
					Type: openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
					JSONSchema: openai.F(openai.ResponseFormatJSONSchemaJSONSchemaParam{
						Schema: openai.F(visualConfigSchema),
						Strict: openai.Bool(true),
					}),
				},
			),
		},
	)

	if err != nil {
		panic(err)
		return returnValue
	}

	vconf := VisualizationConfig{}
	err = json.Unmarshal([]byte(response.Choices[0].Message.Content), &vconf)
	if err != nil {
		return returnValue
	}

	returnValue.ChartType = vconf.ChartType
	returnValue.XAxis = vconf.XAxis
	returnValue.YAxis = vconf.YAxis
	returnValue.Title = vconf.Title

	return returnValue
}

func main() {
	exampleData := lookupSalesData("Show me all the sales from store 1320 on November 1st, 2021")
	//fmt.Println(exampleData)

	// analysis := analyzeSalesData("What trends do you see in this data", exampleData)
	// fmt.Println(analysis)
	fmt.Printf("Schema: %+v\n", visualConfigSchema)

	config := extractChartConfig(exampleData, "A bat chart of sales by product SKU. Put the product SKU on the x-axis and the sales on the y-axis")
	fmt.Printf("%+v\n", config)
}
