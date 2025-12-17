// +build antlr_example

package main

import (
	"fmt"
	"strings"
	"go-job-service/parser"
)

func main() {
	fmt.Println("=== ANTLR4 SQL 解析器示例 ===")
	fmt.Println()

	// 示例 SQL 语句
	sqls := []string{
		"SELECT id, name FROM users WHERE age > 18",
		"SELECT u.id, u.name, o.order_id FROM users u JOIN orders o ON u.id = o.user_id",
		"WITH cte AS (SELECT * FROM users) SELECT * FROM cte",
		"INSERT INTO users (id, name) VALUES (1, 'Alice')",
		"UPDATE users SET name = 'Bob' WHERE id = 1",
		"DELETE FROM users WHERE id = 1",
	}

	fmt.Println("注意：此示例需要先生成 ANTLR4 解析器代码")
	fmt.Println("请运行以下命令：")
	fmt.Println("  1. make install-antlr  # 安装 ANTLR4 工具（首次）")
	fmt.Println("  2. make gen-antlr      # 生成解析器代码")
	fmt.Println()

	// 遍历所有SQL语句进行解析
	for i, sql := range sqls {
		fmt.Printf("示例 %d: %s\n", i+1, sql)
		fmt.Println(strings.Repeat("-", 80))

		// 使用 ANTLR4 解析器
		result, err := parser.ParseSQLWithAntlr(sql)
		if err != nil {
			fmt.Printf("❌ 解析失败: %v\n", err)
		} else {
			fmt.Printf("✓ 解析成功\n")
			if result.SqlNode != nil {
				fmt.Printf("  SqlNode: %s\n", result.SqlNode.ToString())
			}
		}

		fmt.Println()
	}

	fmt.Println("=== 完整示例流程 ===")
	fmt.Println()
	fmt.Println("1. 安装 ANTLR4 工具:")
	fmt.Println("   make install-antlr")
	fmt.Println()
	fmt.Println("2. 生成解析器代码:")
	fmt.Println("   make gen-antlr")
	fmt.Println()
	fmt.Println("3. 查看生成的文件:")
	fmt.Println("   ls parser/antlr/")
	fmt.Println()
	fmt.Println("4. 更新 antlr_sql_parser.go 中的代码，取消注释ANTLR相关代码")
	fmt.Println()
	fmt.Println("5. 重新运行此示例:")
	fmt.Println("   go run examples/antlr_example.go")
	fmt.Println()
	fmt.Println("详细文档请参考:")
	fmt.Println("  - grammar/README.md  - ANTLR4 语法文件说明")
	fmt.Println("  - README.md          - 项目总体说明")
}

