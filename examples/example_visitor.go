// +build parse_sql_example

package main

import (
	"fmt"
	"go-job-service/parser"
)

func main() {
	// 示例 1: 简单的 SELECT 语句
	fmt.Println("=== 示例 1: 简单的 SELECT ===")
	sql1 := "SELECT id, name, email FROM users WHERE age > 18"
	parseAndPrint(sql1)

	// 示例 2: 带 JOIN 的查询
	fmt.Println("\n=== 示例 2: 带 JOIN 的查询 ===")
	sql2 := "SELECT u.id, u.name, o.order_id FROM users u JOIN orders o ON u.id = o.user_id WHERE u.age > 18"
	parseAndPrint(sql2)

	// 示例 3: 带 GROUP BY 的查询
	fmt.Println("\n=== 示例 3: 带 GROUP BY 的查询 ===")
	sql3 := "SELECT department, COUNT(*) as cnt FROM employees WHERE salary > 50000 GROUP BY department"
	parseAndPrint(sql3)

	// 示例 4: 复杂查询
	fmt.Println("\n=== 示例 4: 复杂查询 ===")
	sql4 := `
		SELECT 
			u.id,
			u.name,
			COUNT(o.order_id) as order_count
		FROM users u
		LEFT JOIN orders o ON u.id = o.user_id
		WHERE u.age > 18 AND u.status = 'active'
		GROUP BY u.id, u.name
		HAVING COUNT(o.order_id) > 5
	`
	parseAndPrint(sql4)
}

func parseAndPrint(sql string) {
	fmt.Printf("SQL: %s\n\n", sql)

	// 解析 SQL
	result, err := parser.ParseSQLWithAntlr(sql)
	if err != nil {
		fmt.Printf("❌ 解析失败: %v\n", err)
		return
	}

	if !result.Success {
		fmt.Printf("❌ 解析不成功: %s\n", result.ErrorMessage)
		return
	}

	fmt.Printf("✅ 解析成功！\n\n")

	// 检查 SqlNode 类型
	if sqlSelect, ok := result.SqlNode.(*parser.SqlSelect); ok {
		fmt.Printf("SQL 语句类型: SELECT\n")
		fmt.Printf("SELECT 列数: %d\n", len(sqlSelect.SelectList))
		
		// 打印 SELECT 列
		fmt.Println("SELECT 列:")
		for i, col := range sqlSelect.SelectList {
			fmt.Printf("  %d. %T\n", i+1, col)
		}
		
		// 打印 FROM 表
		if sqlSelect.From != nil {
			fmt.Printf("FROM: %T\n", sqlSelect.From)
		}
		
		// 打印 WHERE 条件
		if sqlSelect.Where != nil {
			fmt.Printf("WHERE: 存在\n")
		}
		
		// 打印 GROUP BY
		if len(sqlSelect.GroupBy) > 0 {
			fmt.Printf("GROUP BY: %d 个表达式\n", len(sqlSelect.GroupBy))
		}
		
		// 打印 HAVING
		if sqlSelect.Having != nil {
			fmt.Printf("HAVING: 存在\n")
		}
	}

	// 提取表名
	tables, err := parser.ExtractTableNames(result.SqlNode)
	if err == nil && len(tables) > 0 {
		fmt.Printf("\n涉及的表: %v\n", tables)
	}

	// 提取列名
	columns, err := parser.ExtractColumns(result.SqlNode)
	if err == nil && len(columns) > 0 {
		fmt.Printf("涉及的列: %v\n", columns)
	}

	// 输出 JSON（可选）
	// jsonBytes, err := json.MarshalIndent(result.SqlNode, "", "  ")
	// if err == nil {
	// 	fmt.Printf("\nSqlNode JSON:\n%s\n", string(jsonBytes))
	// }
}

// 高级示例：遍历 SqlNode 树
func demonstrateVisitorPattern() {
	sql := "SELECT id, name FROM users WHERE age > 18"
	result, _ := parser.ParseSQLWithAntlr(sql)
	
	// 使用自定义 Visitor 遍历 AST
	visitor := &CustomVisitor{}
	result.SqlNode.Accept(visitor)
}

// CustomVisitor 是一个自定义的 visitor 实现
type CustomVisitor struct{}

func (v *CustomVisitor) VisitIdentifier(node *parser.SqlIdentifier) (interface{}, error) {
	fmt.Printf("发现标识符: %s\n", node.ToString())
	return nil, nil
}

func (v *CustomVisitor) VisitLiteral(node *parser.SqlLiteral) (interface{}, error) {
	fmt.Printf("发现字面量: %v\n", node.Value)
	return nil, nil
}

func (v *CustomVisitor) VisitCall(node *parser.SqlCall) (interface{}, error) {
	fmt.Printf("发现调用: %s\n", node.Operator.Name)
	for _, operand := range node.Operands {
		operand.Accept(v)
	}
	return nil, nil
}

func (v *CustomVisitor) VisitSelect(node *parser.SqlSelect) (interface{}, error) {
	fmt.Println("发现 SELECT 语句")
	for _, item := range node.SelectList {
		item.Accept(v)
	}
	if node.From != nil {
		node.From.Accept(v)
	}
	if node.Where != nil {
		node.Where.Accept(v)
	}
	return nil, nil
}

func (v *CustomVisitor) VisitJoin(node *parser.SqlJoin) (interface{}, error) {
	fmt.Println("发现 JOIN")
	return nil, nil
}

func (v *CustomVisitor) VisitBasicCall(node *parser.SqlBasicCall) (interface{}, error) {
	if node.Operand != nil {
		node.Operand.Accept(v)
	}
	return nil, nil
}

func (v *CustomVisitor) VisitNodeList(node *parser.SqlNodeList) (interface{}, error) {
	for _, item := range node.List {
		item.Accept(v)
	}
	return nil, nil
}

