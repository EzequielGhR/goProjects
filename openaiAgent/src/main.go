package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/openai/openai-go"
)

const SQL_GENERATION_PROMPT = `
Generate an SQL query based on a prompt. Do not reply with anything besides the SQL query.
The prompt is:
%s

The available columns are: %s
The table name is: %s
`

const DATA_PATH = "/home/zeke/Documents/Repos/goProjects/openaiAgent/data/Store_Sales_Price_Elasticity_Promotions_Data.parquet"

var OPENAI_CLIENT = openai.NewClient()

func generateSqlQuery(prompt string, columns []string, tableName string) (string, error) {
	formattedPrompt := fmt.Sprintf(
		SQL_GENERATION_PROMPT,
		prompt,
		strings.Join(columns, ", "), tableName,
	)

	response, err := OPENAI_CLIENT.Chat.Completions.New(
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
		panic(err)
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
		panic(err)
	}

	result, err := db.Query(
		fmt.Sprintf("SELECT * FROM %s WHERE 1=2", tableName),
	)

	if err != nil {
		panic(err)
	}

	columns, err := result.Columns()
	if err != nil {
		panic(err)
	}

	sqlQuery, err := generateSqlQuery(prompt, columns, tableName)
	if err != nil {
		panic(err)
	}

	sqlQuery = strings.Trim(sqlQuery, "\n ")
	sqlQuery = strings.ReplaceAll(sqlQuery, "```sql", "")
	sqlQuery = strings.Trim(sqlQuery, "`")

	fmt.Printf("DEBUG: %s\n", sqlQuery)

	rows, err := db.Query(sqlQuery)
	if err != nil {
		panic(err)
	}

	resultData := []string{strings.Join(columns, ", ")}
	resultData = append(resultData, extractFromRows(rows, len(columns))...)

	return strings.Join(resultData, "\n")
}

func extractFromRows(rows *sql.Rows, columnsAmount int) []string {
	resultData := []string{}
	dynamicValues := make([]interface{}, columnsAmount)
	for i := range dynamicValues {
		dynamicValues[i] = new(string)
	}

	for rows.Next() {
		err := rows.Scan(dynamicValues...)
		if err != nil {
			panic(err)
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

	return resultData
}

func main() {
	result := lookupSalesData("Provide the first 10 rows of the table")
	fmt.Println(result)
}
