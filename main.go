package main

import (
	"fmt"
	"go-job-service/analyzer"
	"go-job-service/parser"
	"log"
	"strings"
)

func main() {
	fmt.Println("=== Go Job Service - Presto SQL Parser ===")
	fmt.Println()

	// 示例 SQL 语句集合
	sqlExamples := []string{
		"SELECT id, name, email FROM users WHERE age > 18",
		"SELECT u.id, u.name, o.order_id FROM users u JOIN orders o ON u.id = o.user_id",
		"SELECT department, COUNT(*) as cnt FROM employees GROUP BY department HAVING COUNT(*) > 5",
		`WITH sales_data AS (
			SELECT product_id, SUM(amount) as total_sales 
			FROM orders 
			GROUP BY product_id
		)
		SELECT p.name, s.total_sales 
		FROM products p 
		JOIN sales_data s ON p.id = s.product_id 
		WHERE s.total_sales > 1000`,
		"SELECT name, salary, RANK() OVER (PARTITION BY department ORDER BY salary DESC) as rank FROM employees",
	}

	// 解析并分析每个 SQL 语句
	for i, sql := range sqlExamples {
		fmt.Printf("示例 %d:\n", i+1)
		fmt.Printf("SQL: %s\n", sql)
		fmt.Println(strings.Repeat("-", 80))

		// 解析 SQL (使用 ANTLR4)
		parseResult, err := parser.ParseSQLWithAntlr(sql)
		if err != nil {
			log.Printf("解析错误: %v\n", err)
			fmt.Println()
			continue
		}

		fmt.Printf("解析成功!\n")
		fmt.Printf("SqlNode: %s\n\n", parseResult.SqlNode.ToString())

		// 分析 SQL
		analysis := analyzer.AnalyzeSQL(parseResult.SqlNode)
		fmt.Println("分析结果:")
		fmt.Printf("  表名: %v\n", analysis.Tables)
		fmt.Printf("  列名: %v\n", analysis.Columns)
		fmt.Printf("  聚合函数: %v\n", analysis.AggregateFunctions)
		fmt.Printf("  JOIN 类型: %v\n", analysis.JoinTypes)
		fmt.Printf("  是否包含子查询: %v\n", analysis.HasSubquery)
		fmt.Printf("  是否包含 CTE: %v\n", analysis.HasCTE)
		fmt.Printf("  是否包含窗口函数: %v\n", analysis.HasWindowFunction)
		
		fmt.Println()
		fmt.Println(strings.Repeat("=", 80))
		fmt.Println()
	}
}

