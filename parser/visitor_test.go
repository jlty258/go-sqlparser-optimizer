package parser

import (
	"encoding/json"
	"testing"
)

func TestSqlNodeVisitor_SimpleSelect(t *testing.T) {
	sql := "SELECT id, name FROM users WHERE age > 18"
	
	result, err := ParseSQLWithAntlr(sql)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	
	if !result.Success {
		t.Fatalf("解析不成功: %s", result.ErrorMessage)
	}
	
	if result.SqlNode == nil {
		t.Fatal("SqlNode 为空")
	}
	
	// 检查是否为 SELECT 语句
	sqlSelect, ok := result.SqlNode.(*SqlSelect)
	if !ok {
		t.Fatalf("期望 SqlSelect，实际得到: %T", result.SqlNode)
	}
	
	// 检查 SELECT 列表
	if len(sqlSelect.SelectList) != 2 {
		t.Errorf("期望 2 个 SELECT 项，实际得到: %d", len(sqlSelect.SelectList))
	}
	
	// 检查 FROM 子句
	if sqlSelect.From == nil {
		t.Error("FROM 子句为空")
	}
	
	// 检查 WHERE 子句
	if sqlSelect.Where == nil {
		t.Error("WHERE 子句为空")
	}
	
	t.Logf("解析成功: %+v", sqlSelect)
}

func TestSqlNodeVisitor_SelectWithJoin(t *testing.T) {
	sql := "SELECT u.id, u.name, o.order_id FROM users u JOIN orders o ON u.id = o.user_id WHERE u.age > 18"
	
	result, err := ParseSQLWithAntlr(sql)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	
	if !result.Success {
		t.Fatalf("解析不成功: %s", result.ErrorMessage)
	}
	
	if result.SqlNode == nil {
		t.Fatal("SqlNode 为空")
	}
	
	sqlSelect, ok := result.SqlNode.(*SqlSelect)
	if !ok {
		t.Fatalf("期望 SqlSelect，实际得到: %T", result.SqlNode)
	}
	
	t.Logf("解析成功: %+v", sqlSelect)
}

func TestSqlNodeVisitor_SelectWithGroupBy(t *testing.T) {
	sql := "SELECT department, COUNT(*) FROM employees GROUP BY department"
	
	result, err := ParseSQLWithAntlr(sql)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	
	if !result.Success {
		t.Fatalf("解析不成功: %s", result.ErrorMessage)
	}
	
	if result.SqlNode == nil {
		t.Fatal("SqlNode 为空")
	}
	
	sqlSelect, ok := result.SqlNode.(*SqlSelect)
	if !ok {
		t.Fatalf("期望 SqlSelect，实际得到: %T", result.SqlNode)
	}
	
	// 检查 GROUP BY
	if len(sqlSelect.GroupBy) == 0 {
		t.Error("GROUP BY 子句为空")
	}
	
	t.Logf("解析成功: %+v", sqlSelect)
}

func TestSqlNodeVisitor_ComplexQuery(t *testing.T) {
	sql := `
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
	
	result, err := ParseSQLWithAntlr(sql)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	
	if !result.Success {
		t.Fatalf("解析不成功: %s", result.ErrorMessage)
	}
	
	if result.SqlNode == nil {
		t.Fatal("SqlNode 为空")
	}
	
	sqlSelect, ok := result.SqlNode.(*SqlSelect)
	if !ok {
		t.Fatalf("期望 SqlSelect，实际得到: %T", result.SqlNode)
	}
	
	// 将 SqlNode 转换为 JSON 以便查看
	jsonBytes, err := json.MarshalIndent(sqlSelect, "", "  ")
	if err != nil {
		t.Logf("无法转换为 JSON: %v", err)
	} else {
		t.Logf("SqlNode JSON:\n%s", string(jsonBytes))
	}
	
	t.Logf("解析成功！")
}

func TestSqlNodeVisitor_ExtractTableNames(t *testing.T) {
	sql := "SELECT u.id, o.order_id FROM users u JOIN orders o ON u.id = o.user_id"
	
	result, err := ParseSQLWithAntlr(sql)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	
	if !result.Success {
		t.Fatalf("解析不成功: %s", result.ErrorMessage)
	}
	
	// 提取表名
	tables, err := ExtractTableNames(result.SqlNode)
	if err != nil {
		t.Fatalf("提取表名失败: %v", err)
	}
	
	t.Logf("提取的表名: %v", tables)
	
	// 应该包含 users 和 orders
	if len(tables) < 1 {
		t.Error("期望至少提取到 1 个表名")
	}
}

func TestSqlNodeVisitor_ExtractColumns(t *testing.T) {
	sql := "SELECT id, name, email FROM users WHERE age > 18"
	
	result, err := ParseSQLWithAntlr(sql)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	
	if !result.Success {
		t.Fatalf("解析不成功: %s", result.ErrorMessage)
	}
	
	// 提取列名
	columns, err := ExtractColumns(result.SqlNode)
	if err != nil {
		t.Fatalf("提取列名失败: %v", err)
	}
	
	t.Logf("提取的列名: %v", columns)
	
	if len(columns) != 3 {
		t.Errorf("期望提取到 3 个列名，实际得到: %d", len(columns))
	}
}

