package main

import (
	"fmt"
	"tools"
)

func main() {
	exampleData := tools.LookupSalesData("Show me all the sales for store 1320 on November 1st, 2021")
	analysis := tools.AnalyzeSalesData("What trends do you see in this data?", exampleData)

	fmt.Println("**Analysis:**")
	fmt.Println(analysis)
	fmt.Println()

	pythonCode := tools.GenerateVisualization(exampleData, "A bar chart of sales by product SKU. Put the product SKU on the x-axis and the sales on the y-axis.")

	fmt.Println("**Python Code for Visualization:**")
	fmt.Println(pythonCode)
}
