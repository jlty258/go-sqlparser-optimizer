// +build simple_example

package main

import (
	"fmt"
	"go-job-service/analyzer"
	"go-job-service/parser"
	"log"
)

func main() {
	// 简单的使用示例
	sql := "SELECT u.id, u.name, o.total FROM users u JOIN orders o ON u.id = o.user_id WHERE u.age > 18"

	fmt.Println("原始 SQL:")
	fmt.Println(sql)
	fmt.Println()

	// 解析 SQL (使用 ANTLR4)
	result, err := parser.ParseSQLWithAntlr(sql)
	if err != nil {
		log.Fatalf("解析失败: %v", err)
	}

	fmt.Println("✓ SQL 解析成功!")
	fmt.Println()

	// 分析 SQL
	analysis := analyzer.AnalyzeSQL(result.SqlNode)

	fmt.Println("=== SQL 分析结果 ===")
	fmt.Printf("表名: %v\n", analysis.Tables)
	fmt.Printf("列名: %v\n", analysis.Columns)
	fmt.Printf("JOIN 类型: %v\n", analysis.JoinTypes)
	fmt.Printf("表别名: %v\n", analysis.TableAliases)
	fmt.Printf("列别名: %v\n", analysis.ColumnAliases)
	fmt.Printf("聚合函数: %v\n", analysis.AggregateFunctions)
	fmt.Printf("是否包含子查询: %v\n", analysis.HasSubquery)
	fmt.Println()
}

