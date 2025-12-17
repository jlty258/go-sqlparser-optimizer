package analyzer

import (
	"go-job-service/parser"
	"strings"
)

// SQLAnalysis SQL 分析结果
type SQLAnalysis struct {
	Tables             []string          // 所有表名
	Columns            []string          // 所有列名
	AggregateFunctions []string          // 聚合函数列表
	JoinTypes          []string          // JOIN 类型列表
	HasSubquery        bool              // 是否包含子查询
	HasCTE             bool              // 是否包含 CTE
	HasWindowFunction  bool              // 是否包含窗口函数
	TableAliases       map[string]string // 表别名映射
	ColumnAliases      map[string]string // 列别名映射
}

// AnalyzeSQL 分析 SQL 语句（基于 SqlNode）
func AnalyzeSQL(sqlNode parser.SqlNode) *SQLAnalysis {
	if sqlNode == nil {
		return &SQLAnalysis{
			Tables:        []string{},
			Columns:       []string{},
			TableAliases:  make(map[string]string),
			ColumnAliases: make(map[string]string),
		}
	}
	
	// 创建分析 visitor
	analyzer := &SQLAnalyzer{
		Analysis: &SQLAnalysis{
			Tables:             []string{},
			Columns:            []string{},
			AggregateFunctions: []string{},
			JoinTypes:          []string{},
			TableAliases:       make(map[string]string),
			ColumnAliases:      make(map[string]string),
		},
		tableSet:  make(map[string]bool),
		columnSet: make(map[string]bool),
	}
	
	// 遍历 SqlNode 树
	sqlNode.Accept(analyzer)
	
	return analyzer.Analysis
}

// SQLAnalyzer 实现 SqlNodeVisitor 接口来分析 SQL
type SQLAnalyzer struct {
	Analysis  *SQLAnalysis
	tableSet  map[string]bool // 用于去重
	columnSet map[string]bool // 用于去重
}

// VisitIdentifier 访问标识符
func (a *SQLAnalyzer) VisitIdentifier(node *parser.SqlIdentifier) (interface{}, error) {
	// 标识符可能是列名
	if len(node.Names) > 0 {
		fullName := strings.Join(node.Names, ".")
		if !a.columnSet[fullName] && fullName != "*" {
			a.columnSet[fullName] = true
			a.Analysis.Columns = append(a.Analysis.Columns, fullName)
		}
	}
	return nil, nil
}

// VisitLiteral 访问字面量
func (a *SQLAnalyzer) VisitLiteral(node *parser.SqlLiteral) (interface{}, error) {
	// 字面量不需要特别处理
	return nil, nil
}

// VisitCall 访问函数调用/操作符
func (a *SQLAnalyzer) VisitCall(node *parser.SqlCall) (interface{}, error) {
	if node.Operator != nil {
		// 检查是否是聚合函数
		funcName := strings.ToUpper(node.Operator.Name)
		if isAggregateFunction(funcName) {
			a.Analysis.AggregateFunctions = append(a.Analysis.AggregateFunctions, funcName)
		}
		
		// 检查是否是窗口函数
		if isWindowFunction(funcName) {
			a.Analysis.HasWindowFunction = true
		}
		
		// 检查是否是 AS (别名操作符)
		if node.Operator.Kind == parser.SqlKindAs && len(node.Operands) >= 2 {
			// 第一个操作数是原始表达式，第二个是别名
			if aliasNode, ok := node.Operands[1].(*parser.SqlIdentifier); ok {
				alias := aliasNode.GetSimple()
				// 尝试判断是表别名还是列别名
				if tableNode, ok := node.Operands[0].(*parser.SqlIdentifier); ok {
					tableName := tableNode.ToString()
					if !strings.Contains(tableName, ".") {
						// 没有点号，可能是表别名
						a.Analysis.TableAliases[alias] = tableName
					}
				}
			}
		}
	}
	
	// 递归访问操作数
	for _, operand := range node.Operands {
		operand.Accept(a)
	}
	
	return nil, nil
}

// VisitSelect 访问 SELECT 语句
func (a *SQLAnalyzer) VisitSelect(node *parser.SqlSelect) (interface{}, error) {
	// 访问 SELECT 列表
	for _, selectItem := range node.SelectList {
		selectItem.Accept(a)
	}
	
	// 访问 FROM 子句（提取表名）
	if node.From != nil {
		a.extractTablesFromNode(node.From)
	}
	
	// 访问 WHERE 子句
	if node.Where != nil {
		node.Where.Accept(a)
	}
	
	// 访问 GROUP BY 子句
	for _, groupByItem := range node.GroupBy {
		groupByItem.Accept(a)
	}
	
	// 访问 HAVING 子句
	if node.Having != nil {
		node.Having.Accept(a)
	}
	
	// 访问 ORDER BY 子句
	for _, orderByItem := range node.OrderBy {
		orderByItem.Accept(a)
	}
	
	return nil, nil
}

// VisitJoin 访问 JOIN 节点
func (a *SQLAnalyzer) VisitJoin(node *parser.SqlJoin) (interface{}, error) {
	// 记录 JOIN 类型
	joinType := string(node.JoinType) + " JOIN"
	a.Analysis.JoinTypes = append(a.Analysis.JoinTypes, joinType)
	
	// 提取左侧表
	if node.Left != nil {
		a.extractTablesFromNode(node.Left)
	}
	
	// 提取右侧表
	if node.Right != nil {
		a.extractTablesFromNode(node.Right)
	}
	
	// 访问 JOIN 条件
	if node.Condition != nil {
		node.Condition.Accept(a)
	}
	
	return nil, nil
}

// VisitBasicCall 访问基本调用（别名等）
func (a *SQLAnalyzer) VisitBasicCall(node *parser.SqlBasicCall) (interface{}, error) {
	// 访问操作数
	if node.Operand != nil {
		node.Operand.Accept(a)
	}
	
	// 记录别名
	if node.Alias != "" {
		// 判断是表别名还是列别名
		if identifier, ok := node.Operand.(*parser.SqlIdentifier); ok {
			if len(identifier.Names) == 1 && !strings.Contains(identifier.Names[0], ".") {
				// 可能是表别名
				a.Analysis.TableAliases[node.Alias] = identifier.Names[0]
			} else {
				// 列别名
				a.Analysis.ColumnAliases[node.Alias] = identifier.ToString()
			}
		}
	}
	
	return nil, nil
}

// VisitNodeList 访问节点列表
func (a *SQLAnalyzer) VisitNodeList(node *parser.SqlNodeList) (interface{}, error) {
	for _, item := range node.List {
		item.Accept(a)
	}
	return nil, nil
}

// VisitHint 访问 Hint 节点
func (a *SQLAnalyzer) VisitHint(node *parser.SqlHint) (interface{}, error) {
	// Hint 暂时不需要分析，直接返回
	return nil, nil
}

// extractTablesFromNode 从节点中提取表名
func (a *SQLAnalyzer) extractTablesFromNode(node parser.SqlNode) {
	if node == nil {
		return
	}
	
	// 如果是标识符，可能是表名
	if identifier, ok := node.(*parser.SqlIdentifier); ok {
		if len(identifier.Names) > 0 && identifier.Names[0] != "*" {
			tableName := identifier.Names[0]
			if !a.tableSet[tableName] {
				a.tableSet[tableName] = true
				a.Analysis.Tables = append(a.Analysis.Tables, tableName)
			}
		}
		return
	}
	
	// 如果是 JOIN 节点，递归处理
	if join, ok := node.(*parser.SqlJoin); ok {
		join.Accept(a)
		return
	}
	
	// 如果是带别名的节点（SqlCall with AS）
	if call, ok := node.(*parser.SqlCall); ok {
		if call.Operator != nil && call.Operator.Kind == parser.SqlKindAs {
			// 第一个操作数是表或子查询
			if len(call.Operands) >= 1 {
				// 检查是否是子查询
				if _, isSelect := call.Operands[0].(*parser.SqlSelect); isSelect {
					a.Analysis.HasSubquery = true
				}
				a.extractTablesFromNode(call.Operands[0])
			}
			return
		}
		// 其他类型的 call，递归处理
		call.Accept(a)
		return
	}
	
	// 如果是 SELECT（子查询）
	if selectNode, ok := node.(*parser.SqlSelect); ok {
		a.Analysis.HasSubquery = true
		selectNode.Accept(a)
		return
	}
}

// isAggregateFunction 判断是否是聚合函数
func isAggregateFunction(funcName string) bool {
	aggregateFuncs := map[string]bool{
		"COUNT":   true,
		"SUM":     true,
		"AVG":     true,
		"MAX":     true,
		"MIN":     true,
		"STDDEV":  true,
		"VARIANCE": true,
		"GROUP_CONCAT": true,
		"ARRAY_AGG": true,
		"STRING_AGG": true,
	}
	return aggregateFuncs[funcName]
}

// isWindowFunction 判断是否是窗口函数
func isWindowFunction(funcName string) bool {
	windowFuncs := map[string]bool{
		"ROW_NUMBER": true,
		"RANK":       true,
		"DENSE_RANK": true,
		"NTILE":      true,
		"LAG":        true,
		"LEAD":       true,
		"FIRST_VALUE": true,
		"LAST_VALUE": true,
		"NTH_VALUE":  true,
	}
	return windowFuncs[funcName]
}

// =============================================================================
// 兼容旧 API 的函数（已弃用）
// =============================================================================

// AnalyzeSQLFromParseResult 从 ParseResult 分析 SQL（兼容旧代码）
// Deprecated: 使用 AnalyzeSQL(sqlNode) 代替
func AnalyzeSQLFromParseResult(parseResult *parser.ParseResult) *SQLAnalysis {
	if parseResult == nil || parseResult.Statement == nil {
		return &SQLAnalysis{
			Tables:        []string{},
			Columns:       []string{},
			TableAliases:  make(map[string]string),
			ColumnAliases: make(map[string]string),
		}
	}
	
	return AnalyzeSQL(parseResult.Statement)
}
