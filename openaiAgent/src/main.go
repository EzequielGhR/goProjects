package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/openai/openai-go"
)

const SQL_GENERATION_PROMPT = `
Generate an SQL query based on a prompt. Do not reply with anything besides the SQL query.
The prompt is:
%s

The available columns are: %s
The table name is: %s
`

var OPENAI_CLIENT = openai.NewClient()

var DATA_PATH = path.Join(
	path.Dir(os.Args[0]),
	"../data/Store_Sales_Price_Elasticity_Promotions_Data.parquet",
)

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
			SELECT * FROM read_parquet(%s)`,
			tableName,
			DATA_PATH,
		),
	)

	if err != nil {
		panic(err)
	}

	// TODO: Finish up
	return "success"
}

func main() {

}
