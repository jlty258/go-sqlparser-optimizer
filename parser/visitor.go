package parser

import (
	"fmt"
	"strconv"
	"strings"

	antlr4 "github.com/antlr4-go/antlr/v4"
	"go-job-service/parser/antlr"
)

// SqlNodeBuilderVisitor 将 ANTLR 解析树转换为 SqlNode
// 参考 Java 版本的 SqlNodeBuilderV2
type SqlNodeBuilderVisitor struct {
	*antlr.BaseSqlBaseParserVisitor
	
	parser             *antlr.SqlBaseParser
	allDerefFields     map[string]bool       // 所有字段引用
	columnNameSet      map[string]bool       // 列名集合
	subQueryTables     map[string]SqlNode    // 子查询表
	assetMap           map[string]string     // 资产映射（别名到表名）
	currentAssetKey    string                // 当前资产键
	currentIdentifiers []string              // 当前标识符列表
	joinConditions     []*SqlCall            // JOIN 条件列表
	filterConditions   []*SqlCall            // FILTER 条件列表
	variableSet        map[string]bool       // 变量集合
}

// NewSqlNodeBuilderVisitor 创建新的 Visitor
func NewSqlNodeBuilderVisitor() *SqlNodeBuilderVisitor {
	v := &SqlNodeBuilderVisitor{
		BaseSqlBaseParserVisitor: &antlr.BaseSqlBaseParserVisitor{
			BaseParseTreeVisitor: &antlr4.BaseParseTreeVisitor{},
		},
		allDerefFields:     make(map[string]bool),
		columnNameSet:      make(map[string]bool),
		subQueryTables:     make(map[string]SqlNode),
		assetMap:           make(map[string]string),
		currentIdentifiers: []string{},
		joinConditions:     []*SqlCall{},
		filterConditions:   []*SqlCall{},
		variableSet:        make(map[string]bool),
	}
	return v
}

// =============================================================================
// Entry Point - 从这里开始解析
// =============================================================================

// VisitSingleStatement 访问单个语句（入口点）
func (v *SqlNodeBuilderVisitor) VisitSingleStatement(ctx *antlr.SingleStatementContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	stmtCtx := ctx.Statement()
	if stmtCtx == nil {
		return nil
	}
	
	// 检查是否为 StatementDefault (查询语句)
	if defaultCtx, ok := stmtCtx.(*antlr.StatementDefaultContext); ok {
		return v.VisitStatementDefault(defaultCtx)
	}
	
	return v.newError("不支持的语句类型", ctx)
}

// VisitStatementDefault 访问默认语句
func (v *SqlNodeBuilderVisitor) VisitStatementDefault(ctx *antlr.StatementDefaultContext) interface{} {
	if ctx == nil || ctx.Query() == nil {
		return nil
	}
	return v.VisitQuery(ctx.Query())
}

// VisitQuery 访问查询
func (v *SqlNodeBuilderVisitor) VisitQuery(ctx antlr.IQueryContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	queryCtx, ok := ctx.(*antlr.QueryContext)
	if !ok || queryCtx == nil {
		return nil
	}
	
	queryTermCtx := queryCtx.QueryTerm()
	if queryTermCtx == nil {
		return nil
	}
	
	// 检查是否为 QueryTermDefault
	if termDefaultCtx, ok := queryTermCtx.(*antlr.QueryTermDefaultContext); ok {
		return v.VisitQueryTermDefault(termDefaultCtx)
	}
	
	return v.newError("不支持的查询项类型", queryCtx)
}

// VisitQueryTermDefault 访问查询项默认
func (v *SqlNodeBuilderVisitor) VisitQueryTermDefault(ctx *antlr.QueryTermDefaultContext) interface{} {
	if ctx == nil || ctx.QueryPrimary() == nil {
		return nil
	}
	
	queryPrimaryCtx := ctx.QueryPrimary()
	
	// 检查是否为 QueryPrimaryDefault
	if primaryDefaultCtx, ok := queryPrimaryCtx.(*antlr.QueryPrimaryDefaultContext); ok {
		return v.VisitQueryPrimaryDefault(primaryDefaultCtx)
	}
	
	return v.newError("不支持的查询主体类型", ctx)
}

// VisitQueryPrimaryDefault 访问查询主体默认
func (v *SqlNodeBuilderVisitor) VisitQueryPrimaryDefault(ctx *antlr.QueryPrimaryDefaultContext) interface{} {
	if ctx == nil || ctx.QuerySpecification() == nil {
		return nil
	}
	
	querySpecCtx := ctx.QuerySpecification()
	
	// 检查是否为 RegularQuerySpecification
	if regularCtx, ok := querySpecCtx.(*antlr.RegularQuerySpecificationContext); ok {
		return v.VisitRegularQuerySpecification(regularCtx)
	}
	
	return v.newError("不支持的查询规范类型", ctx)
}

// =============================================================================
// SELECT 语句核心逻辑
// =============================================================================

// VisitRegularQuerySpecification 访问常规查询规范（SELECT 语句）
// 这是核心方法，处理 SELECT, FROM, WHERE, GROUP BY, HAVING 等子句
func (v *SqlNodeBuilderVisitor) VisitRegularQuerySpecification(ctx *antlr.RegularQuerySpecificationContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	pos := v.getPosition(ctx.GetStart())
	sqlSelect := NewSqlSelect(pos)
	
	// 重置当前标识符
	v.currentIdentifiers = []string{}
	
	// 1. 处理 FROM 子句
	var fromNode SqlNode
	var fromList []SqlNode
	if fromClauseIface := ctx.FromClause(); fromClauseIface != nil {
		if fromClause, ok := fromClauseIface.(*antlr.FromClauseContext); ok {
			fromResult := v.VisitFromClause(fromClause)
			if fromResult != nil {
				if list, ok := fromResult.([]SqlNode); ok {
					fromList = list
				} else if node, ok := fromResult.(SqlNode); ok {
					fromList = []SqlNode{node}
				}
			}
			
			// 2. 处理 WHERE 子句（收集 join 和 filter 条件）
			if whereClauseIface := ctx.WhereClause(); whereClauseIface != nil {
				if whereClause, ok := whereClauseIface.(*antlr.WhereClauseContext); ok {
					v.VisitWhereClause(whereClause)
				}
			}
			
			// 3. 构建 JOIN 树
			if len(fromList) > 1 {
				fromNode = v.dealJoinList2Node(fromList)
			} else if len(fromList) == 1 {
				fromNode = fromList[0]
			}
			
			// 4. 处理 filter 条件
			whereNode := v.dealFilterConditions()
			sqlSelect.Where = whereNode
		}
	} else {
		// 没有 FROM 子句，使用 DUAL 表
		fromNode = NewSqlIdentifier([]string{"DUAL"}, pos)
	}
	
	sqlSelect.From = fromNode
	
	// 5. 处理 SELECT 列表和 HINTS
	if selectClauseIface := ctx.SelectClause(); selectClauseIface != nil {
		if selectClause, ok := selectClauseIface.(*antlr.SelectClauseContext); ok {
			selectResult := v.VisitSelectClause(selectClause)
			if selectResult != nil {
				if result, ok := selectResult.(*SelectClauseResult); ok {
					sqlSelect.Hints = result.Hints
					sqlSelect.SelectList = result.SelectList
				} else if selectList, ok := selectResult.([]SqlNode); ok {
					// 兼容旧的返回格式
					sqlSelect.SelectList = selectList
				}
			}
		}
	}
	
	// 6. 处理 GROUP BY 子句
	if aggClauseIface := ctx.AggregationClause(); aggClauseIface != nil {
		if aggClause, ok := aggClauseIface.(*antlr.AggregationClauseContext); ok {
			aggResult := v.VisitAggregationClause(aggClause)
			if aggResult != nil {
				if groupByList, ok := aggResult.([]SqlNode); ok {
					sqlSelect.GroupBy = groupByList
				}
			}
		}
	}
	
	// 7. 处理 HAVING 子句
	if havingClauseIface := ctx.HavingClause(); havingClauseIface != nil {
		if havingClause, ok := havingClauseIface.(*antlr.HavingClauseContext); ok {
			havingResult := v.VisitHavingClause(havingClause)
			if havingResult != nil {
				if havingNode, ok := havingResult.(SqlNode); ok {
					sqlSelect.Having = havingNode
				}
			}
		}
	}
	
	return sqlSelect
}

// =============================================================================
// SELECT 子句
// =============================================================================

// SelectClauseResult 保存 SELECT 子句的解析结果
type SelectClauseResult struct {
	Hints      []*SqlHint
	SelectList []SqlNode
}

// VisitSelectClause 访问 SELECT 子句
func (v *SqlNodeBuilderVisitor) VisitSelectClause(ctx *antlr.SelectClauseContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	result := &SelectClauseResult{
		Hints:      []*SqlHint{},
		SelectList: []SqlNode{},
	}
	
	// 1. 解析 HINTS（如果有）
	allHints := ctx.AllHint()
	
	for _, hintCtx := range allHints {
		if hintContext, ok := hintCtx.(*antlr.HintContext); ok {
			hints := v.VisitHint(hintContext)
			if hints != nil {
				if hintList, ok := hints.([]*SqlHint); ok {
					result.Hints = append(result.Hints, hintList...)
				}
			}
		}
	}
	
	// 2. 获取 namedExpressionSeq
	namedExprsCtx := ctx.NamedExpressionSeq()
	if namedExprsCtx == nil {
		return result
	}
	
	// 3. 检查类型
	if federatedCtx, ok := namedExprsCtx.(*antlr.FederatedQueryExpressionContext); ok {
		selectList := v.VisitFederatedQueryExpression(federatedCtx)
		if selectList != nil {
			if list, ok := selectList.([]SqlNode); ok {
				result.SelectList = list
			}
		}
	}
	
	return result
}

// VisitFederatedQueryExpression 访问联邦查询表达式
func (v *SqlNodeBuilderVisitor) VisitFederatedQueryExpression(ctx *antlr.FederatedQueryExpressionContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	result := []SqlNode{}
	
	allNamedExprs := ctx.AllNamedExpression()
	for _, namedExprIface := range allNamedExprs {
		if namedExpr, ok := namedExprIface.(*antlr.NamedExpressionContext); ok && namedExpr != nil {
			if node := v.VisitNamedExpression(namedExpr); node != nil {
				if sqlNode, ok := node.(SqlNode); ok {
					result = append(result, sqlNode)
				}
			}
		}
	}
	
	return result
}

// VisitNamedExpression 访问命名表达式（SELECT 列表中的项）
func (v *SqlNodeBuilderVisitor) VisitNamedExpression(ctx *antlr.NamedExpressionContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	// 获取表达式
	exprIface := ctx.Expression()
	if exprIface == nil {
		return nil
	}
	
	expr, ok := exprIface.(*antlr.ExpressionContext)
	if !ok {
		return nil
	}
	
	node := v.VisitExpression(expr)
	if node == nil {
		return nil
	}
	
	sqlNode, ok := node.(SqlNode)
	if !ok {
		return nil
	}
	
	// 检查是否有别名
	identCtx := ctx.ErrorCapturingIdentifier()
	if identCtx != nil && identCtx.Identifier() != nil {
		alias := identCtx.Identifier().GetText()
		if alias != "" {
			pos := v.getPosition(ctx.GetStart())
			// 使用 AS 操作符构建别名节点
			asOp := &SqlOperator{Name: "AS", Kind: SqlKindAs, Syntax: SyntaxSpecial}
			aliasNode := NewSqlIdentifier([]string{alias}, pos)
			return NewSqlCall(asOp, []SqlNode{sqlNode, aliasNode}, pos)
		}
	}
	
	return sqlNode
}

// VisitExpression 访问表达式
func (v *SqlNodeBuilderVisitor) VisitExpression(ctx *antlr.ExpressionContext) interface{} {
	if ctx == nil || ctx.BooleanExpression() == nil {
		return nil
	}
	if boolExpr, ok := ctx.BooleanExpression().(antlr.IBooleanExpressionContext); ok {
		return v.visitBooleanExpressionInternal(boolExpr)
	}
	return nil
}

// =============================================================================
// FROM 子句
// =============================================================================

// VisitFromClause 访问 FROM 子句
// 返回表的列表
func (v *SqlNodeBuilderVisitor) VisitFromClause(ctx *antlr.FromClauseContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	result := []SqlNode{}
	
	allRelations := ctx.AllRelation()
	for _, relationIface := range allRelations {
		if relation, ok := relationIface.(*antlr.RelationContext); ok {
			nodeResult := v.VisitRelation(relation)
			if nodeResult != nil {
				if node, ok := nodeResult.(SqlNode); ok {
					result = append(result, node)
				}
			}
		}
	}
	
	return result
}

// VisitRelation 访问 relation
// 处理表、子查询、JOIN 等
func (v *SqlNodeBuilderVisitor) VisitRelation(ctx *antlr.RelationContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	// 获取 relationPrimary
	relPrimary := ctx.RelationPrimary()
	if relPrimary == nil {
		return nil
	}
	
	// 处理基础的 relation（表或子查询）
	result := v.visitRelationPrimaryInternal(relPrimary)
	
	// 处理 JOIN 扩展
	allExtensions := ctx.AllRelationExtension()
	if len(allExtensions) > 0 {
		// TODO: 处理显式 JOIN，这里需要根据实际的语法结构来实现
		// 目前主要依赖 WHERE 子句中的隐式 JOIN
	}
	
	return result
}

// visitRelationPrimaryInternal 内部辅助方法，处理多个子类型
func (v *SqlNodeBuilderVisitor) visitRelationPrimaryInternal(ctx antlr.IRelationPrimaryContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	// 处理表名
	if tableCtx, ok := ctx.(*antlr.TableNameContext); ok {
		return v.VisitTableName(tableCtx)
	}
	
	// 处理子查询
	if aliasedQueryCtx, ok := ctx.(*antlr.AliasedQueryContext); ok {
		return v.VisitAliasedQuery(aliasedQueryCtx)
	}
	
	// 处理带别名的 relation
	if aliasedRelCtx, ok := ctx.(*antlr.AliasedRelationContext); ok {
		return v.VisitAliasedRelation(aliasedRelCtx)
	}
	
	return nil
}

// VisitTableName 访问表名
func (v *SqlNodeBuilderVisitor) VisitTableName(ctx *antlr.TableNameContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	pos := v.getPosition(ctx.GetStart())
	
	// 获取表名
	multipartId := ctx.MultipartIdentifier()
	if multipartId == nil {
		return nil
	}
	
	tableName := multipartId.GetText()
	parts := strings.Split(tableName, ".")
	
	// 记录当前资产键
	if len(parts) > 0 {
		v.currentAssetKey = parts[len(parts)-1]
	}
	
	tableNode := NewSqlIdentifier(parts, pos)
	
	// 检查是否有别名
	tableAlias := ctx.TableAlias()
	if tableAlias != nil && tableAlias.StrictIdentifier() != nil {
		alias := tableAlias.StrictIdentifier().GetText()
		if alias != "" {
			v.currentAssetKey = alias
			v.assetMap[alias] = tableName
			// 使用 AS 操作符构建别名
			asOp := &SqlOperator{Name: "AS", Kind: SqlKindAs, Syntax: SyntaxSpecial}
			aliasNode := NewSqlIdentifier([]string{alias}, pos)
			return NewSqlCall(asOp, []SqlNode{tableNode, aliasNode}, pos)
		}
	} else {
		v.assetMap[tableName] = tableName
	}
	
	return tableNode
}

// VisitAliasedQuery 访问带别名的子查询
func (v *SqlNodeBuilderVisitor) VisitAliasedQuery(ctx *antlr.AliasedQueryContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	pos := v.getPosition(ctx.GetStart())
	
	// 访问子查询
	subqueryIface := ctx.Query()
	if subqueryIface == nil {
		return nil
	}
	
	subqueryNode := v.VisitQuery(subqueryIface)
	if subqueryNode == nil {
		return nil
	}
	
	sqlNode, ok := subqueryNode.(SqlNode)
	if !ok {
		return nil
	}
	
	// 获取别名（子查询必须有别名）
	tableAlias := ctx.TableAlias()
	if tableAlias != nil {
		aliasText := tableAlias.GetText()
		if aliasText != "" {
			v.currentAssetKey = aliasText
			// 使用 AS 操作符构建子查询别名
			asOp := &SqlOperator{Name: "AS", Kind: SqlKindAs, Syntax: SyntaxSpecial}
			aliasNode := NewSqlIdentifier([]string{aliasText}, pos)
			subQueryCall := NewSqlCall(asOp, []SqlNode{sqlNode, aliasNode}, pos)
			v.subQueryTables[aliasText] = subQueryCall
			return subQueryCall
		}
	}
	
	return sqlNode
}

// VisitAliasedRelation 访问带别名的关系
func (v *SqlNodeBuilderVisitor) VisitAliasedRelation(ctx *antlr.AliasedRelationContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	// 处理内部的 relation
	relationIface := ctx.Relation()
	if relationIface == nil {
		return nil
	}
	
	if relation, ok := relationIface.(*antlr.RelationContext); ok {
		return v.VisitRelation(relation)
	}
	
	return nil
}

// =============================================================================
// WHERE 子句
// =============================================================================

// VisitWhereClause 访问 WHERE 子句
// 这里会收集 join 条件和 filter 条件
func (v *SqlNodeBuilderVisitor) VisitWhereClause(ctx *antlr.WhereClauseContext) interface{} {
	if ctx == nil || ctx.BooleanExpression() == nil {
		return nil
	}
	if boolExpr, ok := ctx.BooleanExpression().(antlr.IBooleanExpressionContext); ok {
		return v.visitBooleanExpressionInternal(boolExpr)
	}
	return nil
}

// =============================================================================
// Boolean Expression - 布尔表达式处理
// =============================================================================

// visitBooleanExpressionInternal 内部辅助方法，访问布尔表达式
func (v *SqlNodeBuilderVisitor) visitBooleanExpressionInternal(ctx antlr.IBooleanExpressionContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	// LogicalBinary (AND, OR)
	if logicalBinaryCtx, ok := ctx.(*antlr.LogicalBinaryContext); ok {
		return v.VisitLogicalBinary(logicalBinaryCtx)
	}
	
	// LogicalNot (NOT)
	if logicalNotCtx, ok := ctx.(*antlr.LogicalNotContext); ok {
		return v.VisitLogicalNot(logicalNotCtx)
	}
	
	// Predicated
	if predicatedCtx, ok := ctx.(*antlr.PredicatedContext); ok {
		return v.VisitPredicated(predicatedCtx)
	}
	
	return nil
}

// VisitLogicalBinary 访问逻辑二元操作 (AND, OR)
func (v *SqlNodeBuilderVisitor) VisitLogicalBinary(ctx *antlr.LogicalBinaryContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	pos := v.getPosition(ctx.GetStart())
	
	left := v.visitBooleanExpressionInternal(ctx.GetLeft())
	right := v.visitBooleanExpressionInternal(ctx.GetRight())
	
	if left == nil || right == nil {
		return nil
	}
	
	leftNode, ok1 := left.(SqlNode)
	rightNode, ok2 := right.(SqlNode)
	if !ok1 || !ok2 {
		return nil
	}
	
	// 获取操作符
	opText := ctx.GetOperator().GetText()
	opName := strings.ToUpper(opText)
	
	kind := SqlKindAnd
	if opName == "OR" {
		kind = SqlKindOr
	}
	
	op := &SqlOperator{Name: opName, Kind: kind, Syntax: SyntaxBinary}
	return NewSqlCall(op, []SqlNode{leftNode, rightNode}, pos)
}

// VisitLogicalNot 访问逻辑 NOT
func (v *SqlNodeBuilderVisitor) VisitLogicalNot(ctx *antlr.LogicalNotContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	pos := v.getPosition(ctx.GetStart())
	
	operand := v.visitBooleanExpressionInternal(ctx.BooleanExpression())
	if operand == nil {
		return nil
	}
	
	operandNode, ok := operand.(SqlNode)
	if !ok {
		return nil
	}
	
	op := &SqlOperator{Name: "NOT", Kind: SqlKindNot, Syntax: SyntaxPrefix}
	return NewSqlCall(op, []SqlNode{operandNode}, pos)
}

// VisitPredicated 访问谓词表达式
func (v *SqlNodeBuilderVisitor) VisitPredicated(ctx *antlr.PredicatedContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	// 如果没有谓词，直接返回值表达式
	if ctx.Predicate() == nil {
		return v.visitValueExpressionInternal(ctx.ValueExpression())
	}
	
	// 有谓词（IS NULL, IS NOT NULL, BETWEEN, IN, LIKE 等）
	predicate := ctx.Predicate()
	valueExpr := v.visitValueExpressionInternal(ctx.ValueExpression())
	
	if valueExpr == nil {
		return nil
	}
	
	valueNode, ok := valueExpr.(SqlNode)
	if !ok {
		return nil
	}
	
	pos := v.getPosition(ctx.GetStart())
	
	// 处理 IS NULL / IS NOT NULL
	kindToken := predicate.GetStart().GetText()
	kindTokenUpper := strings.ToUpper(kindToken)
	
	if kindTokenUpper == "NULL" {
		// IS NULL
		op := &SqlOperator{Name: "IS NULL", Kind: SqlKindOther, Syntax: SyntaxPostfix}
		callNode := NewSqlCall(op, []SqlNode{valueNode}, pos)
		v.filterConditions = append(v.filterConditions, callNode)
		return callNode
	} else if kindTokenUpper == "NOT" {
		// IS NOT NULL
		op := &SqlOperator{Name: "IS NOT NULL", Kind: SqlKindOther, Syntax: SyntaxPostfix}
		callNode := NewSqlCall(op, []SqlNode{valueNode}, pos)
		v.filterConditions = append(v.filterConditions, callNode)
		return callNode
	}
	
	// TODO: 处理其他谓词（BETWEEN, IN, LIKE 等）
	
	return valueNode
}

// =============================================================================
// Value Expression - 值表达式处理
// =============================================================================

// visitValueExpressionInternal 内部辅助方法，访问值表达式
func (v *SqlNodeBuilderVisitor) visitValueExpressionInternal(ctx antlr.IValueExpressionContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	// ValueExpressionDefault (主表达式)
	if defaultCtx, ok := ctx.(*antlr.ValueExpressionDefaultContext); ok {
		return v.VisitValueExpressionDefault(defaultCtx)
	}
	
	// ArithmeticBinary (算术二元操作)
	if binaryCtx, ok := ctx.(*antlr.ArithmeticBinaryContext); ok {
		return v.VisitArithmeticBinary(binaryCtx)
	}
	
	// ArithmeticUnary (算术一元操作)
	if unaryCtx, ok := ctx.(*antlr.ArithmeticUnaryContext); ok {
		return v.VisitArithmeticUnary(unaryCtx)
	}
	
	// Comparison (比较操作)
	if compCtx, ok := ctx.(*antlr.ComparisonContext); ok {
		return v.VisitComparison(compCtx)
	}
	
	return nil
}

// VisitValueExpressionDefault 访问值表达式默认
func (v *SqlNodeBuilderVisitor) VisitValueExpressionDefault(ctx *antlr.ValueExpressionDefaultContext) interface{} {
	if ctx == nil || ctx.PrimaryExpression() == nil {
		return nil
	}
	return v.visitPrimaryExpressionInternal(ctx.PrimaryExpression())
}

// VisitArithmeticBinary 访问算术二元操作 (+, -, *, /, %)
func (v *SqlNodeBuilderVisitor) VisitArithmeticBinary(ctx *antlr.ArithmeticBinaryContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	pos := v.getPosition(ctx.GetStart())
	
	left := v.visitValueExpressionInternal(ctx.GetLeft())
	right := v.visitValueExpressionInternal(ctx.GetRight())
	
	if left == nil || right == nil {
		return nil
	}
	
	leftNode, ok1 := left.(SqlNode)
	rightNode, ok2 := right.(SqlNode)
	if !ok1 || !ok2 {
		return nil
	}
	
	opText := ctx.GetOperator().GetText()
	kind := v.getOperatorKind(opText)
	op := &SqlOperator{Name: opText, Kind: kind, Syntax: SyntaxBinary}
	
	return NewSqlCall(op, []SqlNode{leftNode, rightNode}, pos)
}

// VisitArithmeticUnary 访问算术一元操作 (+, -)
func (v *SqlNodeBuilderVisitor) VisitArithmeticUnary(ctx *antlr.ArithmeticUnaryContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	pos := v.getPosition(ctx.GetStart())
	
	operand := v.visitValueExpressionInternal(ctx.ValueExpression())
	if operand == nil {
		return nil
	}
	
	operandNode, ok := operand.(SqlNode)
	if !ok {
		return nil
	}
	
	opText := ctx.GetOperator().GetText()
	kind := v.getOperatorKind(opText)
	op := &SqlOperator{Name: opText, Kind: kind, Syntax: SyntaxPrefix}
	
	return NewSqlCall(op, []SqlNode{operandNode}, pos)
}

// VisitComparison 访问比较操作 (=, !=, <, >, <=, >=)
// 这里会区分 join 条件和 filter 条件
func (v *SqlNodeBuilderVisitor) VisitComparison(ctx *antlr.ComparisonContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	pos := v.getPosition(ctx.GetStart())
	
	left := v.visitValueExpressionInternal(ctx.GetLeft())
	right := v.visitValueExpressionInternal(ctx.GetRight())
	
	if left == nil || right == nil {
		return nil
	}
	
	leftNode, ok1 := left.(SqlNode)
	rightNode, ok2 := right.(SqlNode)
	if !ok1 || !ok2 {
		return nil
	}
	
	// 获取比较操作符
	compOp := ctx.ComparisonOperator()
	if compOp == nil {
		return nil
	}
	
	opText := compOp.GetText()
	kind := v.getOperatorKind(opText)
	op := &SqlOperator{Name: opText, Kind: kind, Syntax: SyntaxBinary}
	
	basicCall := NewSqlCall(op, []SqlNode{leftNode, rightNode}, pos)
	
	// 判断是 join 条件还是 filter 条件
	// 如果左右两边都是标识符，可能是 join 条件
	leftIsIdentifier := false
	rightIsIdentifier := false
	rightIsLiteral := false
	
	if _, ok := leftNode.(*SqlIdentifier); ok {
		leftIsIdentifier = true
	}
	if _, ok := rightNode.(*SqlIdentifier); ok {
		rightIsIdentifier = true
	}
	if _, ok := rightNode.(*SqlLiteral); ok {
		rightIsLiteral = true
	}
	
	if leftIsIdentifier && rightIsIdentifier {
		// 两边都是标识符，可能是 join 条件
		// 例如: a.id = b.id
		v.joinConditions = append(v.joinConditions, basicCall)
	} else if leftIsIdentifier && rightIsLiteral {
		// 左边是标识符，右边是字面量，是 filter 条件
		// 例如: a.status = 'active'
		v.filterConditions = append(v.filterConditions, basicCall)
	} else {
		// 其他情况，默认作为 filter 条件
		v.filterConditions = append(v.filterConditions, basicCall)
	}
	
	return basicCall
}

// =============================================================================
// Primary Expression - 基础表达式处理
// =============================================================================

// visitPrimaryExpressionInternal 内部辅助方法，访问主表达式
func (v *SqlNodeBuilderVisitor) visitPrimaryExpressionInternal(ctx antlr.IPrimaryExpressionContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	// Star (*)
	if starCtx, ok := ctx.(*antlr.StarContext); ok {
		return v.VisitStar(starCtx)
	}
	
	// ColumnReference (列引用)
	if colCtx, ok := ctx.(*antlr.ColumnReferenceContext); ok {
		return v.VisitColumnReference(colCtx)
	}
	
	// Dereference (table.column)
	if derefCtx, ok := ctx.(*antlr.DereferenceContext); ok {
		return v.VisitDereference(derefCtx)
	}
	
	// ConstantDefault (常量)
	if constCtx, ok := ctx.(*antlr.ConstantDefaultContext); ok {
		return v.VisitConstantDefault(constCtx)
	}
	
	// FunctionCall (函数调用)
	if funcCtx, ok := ctx.(*antlr.FunctionCallContext); ok {
		return v.VisitFunctionCall(funcCtx)
	}
	
	// ParenthesizedExpression (括号表达式)
	if parenCtx, ok := ctx.(*antlr.ParenthesizedExpressionContext); ok {
		return v.VisitParenthesizedExpression(parenCtx)
	}
	
	return nil
}

// VisitStar 访问星号 (*)
func (v *SqlNodeBuilderVisitor) VisitStar(ctx *antlr.StarContext) interface{} {
	pos := v.getPosition(ctx.GetStart())
	
	// 检查是否是 table.*
	if ctx.QualifiedName() != nil {
		tableName := ctx.QualifiedName().GetText()
		return NewSqlIdentifier([]string{tableName, "*"}, pos)
	}
	
	return NewSqlIdentifier([]string{"*"}, pos)
}

// VisitColumnReference 访问列引用
func (v *SqlNodeBuilderVisitor) VisitColumnReference(ctx *antlr.ColumnReferenceContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	pos := v.getPosition(ctx.GetStart())
	identifier := ctx.Identifier().GetText()
	
	// 记录字段
	fullName := v.currentAssetKey + "." + identifier
	v.allDerefFields[fullName] = true
	v.columnNameSet[fullName] = true
	v.variableSet[identifier] = true
	
	return NewSqlIdentifier([]string{identifier}, pos)
}

// VisitDereference 访问字段引用 (table.column 或 schema.table.column)
func (v *SqlNodeBuilderVisitor) VisitDereference(ctx *antlr.DereferenceContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	pos := v.getPosition(ctx.GetStart())
	text := ctx.GetText()
	
	// 分割为多个部分
	parts := strings.Split(text, ".")
	
	// 记录字段
	if len(parts) >= 2 {
		v.allDerefFields[text] = true
		v.columnNameSet[text] = true
	}
	
	return NewSqlIdentifier(parts, pos)
}

// VisitConstantDefault 访问常量默认
func (v *SqlNodeBuilderVisitor) VisitConstantDefault(ctx *antlr.ConstantDefaultContext) interface{} {
	if ctx == nil || ctx.Constant() == nil {
		return nil
	}
	
	constantCtx := ctx.Constant()
	
	// StringLiteral (字符串)
	if strCtx, ok := constantCtx.(*antlr.StringLiteralContext); ok {
		return v.VisitStringLiteral(strCtx)
	}
	
	// NumericLiteral (数字)
	if numCtx, ok := constantCtx.(*antlr.NumericLiteralContext); ok {
		return v.VisitNumericLiteral(numCtx)
	}
	
	// BooleanLiteral (布尔)
	if boolCtx, ok := constantCtx.(*antlr.BooleanLiteralContext); ok {
		return v.VisitBooleanLiteral(boolCtx)
	}
	
	// NullLiteral (NULL)
	if nullCtx, ok := constantCtx.(*antlr.NullLiteralContext); ok {
		return v.VisitNullLiteral(nullCtx)
	}
	
	return nil
}

// VisitParenthesizedExpression 访问括号表达式
func (v *SqlNodeBuilderVisitor) VisitParenthesizedExpression(ctx *antlr.ParenthesizedExpressionContext) interface{} {
	if ctx == nil || ctx.Expression() == nil {
		return nil
	}
	if expr, ok := ctx.Expression().(*antlr.ExpressionContext); ok {
		return v.VisitExpression(expr)
	}
	return nil
}

// VisitFunctionCall 访问函数调用
func (v *SqlNodeBuilderVisitor) VisitFunctionCall(ctx *antlr.FunctionCallContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	pos := v.getPosition(ctx.GetStart())
	funcName := ctx.FunctionName().GetText()
	
	// 收集参数
	operands := []SqlNode{}
	allArgs := ctx.AllExpression()
	for _, argIface := range allArgs {
		if arg, ok := argIface.(*antlr.ExpressionContext); ok {
			if argNode := v.VisitExpression(arg); argNode != nil {
				if sqlNode, ok := argNode.(SqlNode); ok {
					operands = append(operands, sqlNode)
				}
			}
		}
	}
	
	op := &SqlOperator{
		Name:   strings.ToUpper(funcName),
		Kind:   SqlKindCall,
		Syntax: SyntaxFunction,
	}
	
	return NewSqlCall(op, operands, pos)
}

// =============================================================================
// Literals - 字面量处理
// =============================================================================

// VisitNullLiteral 访问 NULL 字面量
func (v *SqlNodeBuilderVisitor) VisitNullLiteral(ctx *antlr.NullLiteralContext) interface{} {
	return NewSqlLiteral(nil, LiteralNull, v.getPosition(ctx.GetStart()))
}

// VisitNumericLiteral 访问数字字面量
func (v *SqlNodeBuilderVisitor) VisitNumericLiteral(ctx *antlr.NumericLiteralContext) interface{} {
	if ctx == nil || ctx.Number() == nil {
		return nil
	}
	
	numberCtx := ctx.Number()
	
	// IntegerLiteral
	if intCtx, ok := numberCtx.(*antlr.IntegerLiteralContext); ok {
		return v.VisitIntegerLiteral(intCtx)
	}
	
	// DecimalLiteral
	if decCtx, ok := numberCtx.(*antlr.DecimalLiteralContext); ok {
		return v.VisitDecimalLiteral(decCtx)
	}
	
	return nil
}

// VisitIntegerLiteral 访问整数字面量
func (v *SqlNodeBuilderVisitor) VisitIntegerLiteral(ctx *antlr.IntegerLiteralContext) interface{} {
	pos := v.getPosition(ctx.GetStart())
	text := ctx.GetText()
	
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return nil
	}
	
	return NewSqlLiteral(value, LiteralInteger, pos)
}

// VisitDecimalLiteral 访问小数字面量
func (v *SqlNodeBuilderVisitor) VisitDecimalLiteral(ctx *antlr.DecimalLiteralContext) interface{} {
	pos := v.getPosition(ctx.GetStart())
	text := ctx.GetText()
	
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return nil
	}
	
	return NewSqlLiteral(value, LiteralDecimal, pos)
}

// VisitStringLiteral 访问字符串字面量
func (v *SqlNodeBuilderVisitor) VisitStringLiteral(ctx *antlr.StringLiteralContext) interface{} {
	pos := v.getPosition(ctx.GetStart())
	
	// 收集所有字符串
	allStr := ctx.AllStringLit()
	if len(allStr) == 0 {
		return nil
	}
	
	// 拼接多个字符串
	var sb strings.Builder
	for _, str := range allStr {
		text := str.GetText()
		// 去除引号
		if len(text) >= 2 {
			text = text[1 : len(text)-1]
		}
		sb.WriteString(text)
	}
	
	return NewSqlLiteral(sb.String(), LiteralString, pos)
}

// VisitBooleanLiteral 访问布尔字面量
func (v *SqlNodeBuilderVisitor) VisitBooleanLiteral(ctx *antlr.BooleanLiteralContext) interface{} {
	pos := v.getPosition(ctx.GetStart())
	value := strings.ToUpper(ctx.BooleanValue().GetText()) == "TRUE"
	return NewSqlLiteral(value, LiteralBoolean, pos)
}

// =============================================================================
// GROUP BY 和 HAVING
// =============================================================================

// VisitAggregationClause 访问聚合子句 (GROUP BY)
func (v *SqlNodeBuilderVisitor) VisitAggregationClause(ctx *antlr.AggregationClauseContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	groupByNodes := []SqlNode{}
	
	// 处理所有 GROUP BY 子句
	allGroupByClauses := ctx.AllGroupByClause()
	for _, groupByClauseIface := range allGroupByClauses {
		if groupByClause, ok := groupByClauseIface.(*antlr.GroupByClauseContext); ok {
			if expr := groupByClause.Expression(); expr != nil {
				if exprCtx, ok := expr.(*antlr.ExpressionContext); ok {
					if node := v.VisitExpression(exprCtx); node != nil {
						if sqlNode, ok := node.(SqlNode); ok {
							groupByNodes = append(groupByNodes, sqlNode)
						}
					}
				}
			}
		}
	}
	
	return groupByNodes
}

// VisitHavingClause 访问 HAVING 子句
func (v *SqlNodeBuilderVisitor) VisitHavingClause(ctx *antlr.HavingClauseContext) interface{} {
	if ctx == nil || ctx.BooleanExpression() == nil {
		return nil
	}
	if boolExpr, ok := ctx.BooleanExpression().(antlr.IBooleanExpressionContext); ok {
		return v.visitBooleanExpressionInternal(boolExpr)
	}
	return nil
}

// =============================================================================
// JOIN 处理逻辑
// =============================================================================

// dealJoinList2Node 将表列表和 join 条件转换为 JOIN 树
// 参考 Java 版本的实现逻辑
func (v *SqlNodeBuilderVisitor) dealJoinList2Node(fromList []SqlNode) SqlNode {
	if len(fromList) <= 1 {
		if len(fromList) == 1 {
			return fromList[0]
		}
		return nil
	}
	
	// 提取表名
	fromTables := []string{}
	for _, node := range fromList {
		tableName := v.extractTableName(node)
		if tableName != "" {
			fromTables = append(fromTables, tableName)
		}
	}
	
	// 如果没有 join 条件，构建笛卡尔积
	if len(v.joinConditions) == 0 {
		return v.buildCartesianProduct(fromList)
	}
	
	// 提取 join 条件中涉及的表
	joinTables := make(map[string]bool)
	for _, condition := range v.joinConditions {
		tables := v.getConditionTables(condition)
		for _, table := range tables {
			joinTables[table] = true
		}
	}
	
	// 找出不在 join 条件中的表（需要笛卡尔积连接的表）
	extraTables := []string{}
	for _, table := range fromTables {
		if !joinTables[table] {
			extraTables = append(extraTables, table)
		}
	}
	
	// 构建 JOIN 树
	var joinBuilder SqlNode
	if len(v.joinConditions) > 0 {
		joinBuilder = v.appendJoinFromConditions(v.joinConditions)
	}
	
	// 连接额外的表（笛卡尔积）
	if joinBuilder != nil {
		for _, extraTable := range extraTables {
			rightNode := NewSqlIdentifier([]string{extraTable}, &SqlParserPos{})
			// 构建 1=1 的条件（笛卡尔积）
			oneLiteral := NewSqlLiteral(int64(1), LiteralInteger, &SqlParserPos{})
			condition := NewSqlCall(
				&SqlOperator{Name: "=", Kind: SqlKindEquals, Syntax: SyntaxBinary},
				[]SqlNode{oneLiteral, oneLiteral},
				&SqlParserPos{},
			)
			joinBuilder = NewSqlJoin(joinBuilder, rightNode, JoinInner, condition, &SqlParserPos{})
		}
	} else if len(extraTables) > 0 {
		// 没有 join 条件，全部是笛卡尔积
		joinBuilder = NewSqlIdentifier([]string{extraTables[0]}, &SqlParserPos{})
		for i := 1; i < len(extraTables); i++ {
			rightNode := NewSqlIdentifier([]string{extraTables[i]}, &SqlParserPos{})
			oneLiteral := NewSqlLiteral(int64(1), LiteralInteger, &SqlParserPos{})
			condition := NewSqlCall(
				&SqlOperator{Name: "=", Kind: SqlKindEquals, Syntax: SyntaxBinary},
				[]SqlNode{oneLiteral, oneLiteral},
				&SqlParserPos{},
			)
			joinBuilder = NewSqlJoin(joinBuilder, rightNode, JoinInner, condition, &SqlParserPos{})
		}
	}
	
	return joinBuilder
}

// appendJoinFromConditions 根据 join 条件构建 JOIN 树
func (v *SqlNodeBuilderVisitor) appendJoinFromConditions(joinConditions []*SqlCall) SqlNode {
	if len(joinConditions) == 0 {
		return nil
	}
	
	var joinBuilder SqlNode
	usedTables := make(map[string]bool)
	
	for _, condition := range joinConditions {
		tables := v.getConditionTables(condition)
		if len(tables) < 2 {
			continue
		}
		
		leftTable := tables[0]
		rightTable := tables[1]
		
		// 获取表节点（可能是子查询）
		var leftNode SqlNode = NewSqlIdentifier([]string{leftTable}, &SqlParserPos{})
		var rightNode SqlNode = NewSqlIdentifier([]string{rightTable}, &SqlParserPos{})
		
		if subQuery, exists := v.subQueryTables[leftTable]; exists {
			leftNode = subQuery
		}
		if subQuery, exists := v.subQueryTables[rightTable]; exists {
			rightNode = subQuery
		}
		
		if joinBuilder == nil {
			joinBuilder = leftNode
			usedTables[leftTable] = true
		}
		
		// 选择连接节点（优先使用右侧，但如果右侧已使用则使用左侧）
		connectNode := rightNode
		if usedTables[rightTable] {
			connectNode = leftNode
		}
		
		// 构建 JOIN
		joinBuilder = NewSqlJoin(
			joinBuilder,
			connectNode,
			JoinInner,
			condition,
			&SqlParserPos{},
		)
		
		usedTables[leftTable] = true
		usedTables[rightTable] = true
	}
	
	return joinBuilder
}

// buildCartesianProduct 构建笛卡尔积
func (v *SqlNodeBuilderVisitor) buildCartesianProduct(fromList []SqlNode) SqlNode {
	if len(fromList) == 0 {
		return nil
	}
	
	result := fromList[0]
	for i := 1; i < len(fromList); i++ {
		// 构建 1=1 条件（笛卡尔积）
		oneLiteral := NewSqlLiteral(int64(1), LiteralInteger, &SqlParserPos{})
		condition := NewSqlCall(
			&SqlOperator{Name: "=", Kind: SqlKindEquals, Syntax: SyntaxBinary},
			[]SqlNode{oneLiteral, oneLiteral},
			&SqlParserPos{},
		)
		result = NewSqlJoin(result, fromList[i], JoinInner, condition, &SqlParserPos{})
	}
	
	return result
}

// dealFilterConditions 处理 filter 条件，合并为单个 WHERE 表达式
func (v *SqlNodeBuilderVisitor) dealFilterConditions() SqlNode {
	if len(v.filterConditions) == 0 {
		return nil
	}
	
	if len(v.filterConditions) == 1 {
		return v.filterConditions[0]
	}
	
	// 用 AND 连接所有条件
	var result SqlNode = v.filterConditions[0]
	for i := 1; i < len(v.filterConditions); i++ {
		op := &SqlOperator{Name: "AND", Kind: SqlKindAnd, Syntax: SyntaxBinary}
		result = NewSqlCall(op, []SqlNode{result, v.filterConditions[i]}, &SqlParserPos{})
	}
	
	return result
}

// =============================================================================
// Helper Methods - 辅助方法
// =============================================================================

// getPosition 从 Token 获取位置信息
func (v *SqlNodeBuilderVisitor) getPosition(token antlr4.Token) *SqlParserPos {
	if token == nil {
		return &SqlParserPos{LineNumber: 0, ColumnNumber: 0}
	}
	
	return &SqlParserPos{
		LineNumber:   token.GetLine(),
		ColumnNumber: token.GetColumn(),
		EndLine:      token.GetLine(),
		EndColumn:    token.GetColumn() + len(token.GetText()),
	}
}

// getOperatorKind 根据操作符文本获取 SqlKind
func (v *SqlNodeBuilderVisitor) getOperatorKind(op string) SqlKind {
	switch op {
	case "+":
		return SqlKindPlus
	case "-":
		return SqlKindMinus
	case "*":
		return SqlKindTimes
	case "/":
		return SqlKindDivide
	case "=", "==":
		return SqlKindEquals
	case "!=", "<>":
		return SqlKindNotEquals
	case ">":
		return SqlKindGreaterThan
	case "<":
		return SqlKindLessThan
	case ">=":
		return SqlKindOther // TODO: 添加 GreaterThanOrEqual
	case "<=":
		return SqlKindOther // TODO: 添加 LessThanOrEqual
	default:
		return SqlKindOther
	}
}

// extractTableName 从节点中提取表名
func (v *SqlNodeBuilderVisitor) extractTableName(node SqlNode) string {
	if node == nil {
		return ""
	}
	
	// 如果是标识符
	if identifier, ok := node.(*SqlIdentifier); ok {
		if len(identifier.Names) > 0 {
			return identifier.Names[0]
		}
	}
	
	// 如果是带别名的表（AS 操作符）
	if call, ok := node.(*SqlCall); ok {
		if call.Operator != nil && call.Operator.Kind == SqlKindAs {
			if len(call.Operands) >= 2 {
				// 第二个操作数是别名
				if aliasNode, ok := call.Operands[1].(*SqlIdentifier); ok {
					return aliasNode.GetSimple()
				}
			}
		}
	}
	
	return ""
}

// getConditionTables 从 join 条件中提取表名
func (v *SqlNodeBuilderVisitor) getConditionTables(condition *SqlCall) []string {
	tables := []string{}
	
	if condition == nil || len(condition.Operands) < 2 {
		return tables
	}
	
	// 从左右操作数中提取表名
	for _, operand := range condition.Operands {
		if identifier, ok := operand.(*SqlIdentifier); ok {
			if len(identifier.Names) >= 2 {
				// 假设格式为 table.column
				tables = append(tables, identifier.Names[0])
			}
		}
	}
	
	// 去重
	uniqueTables := make(map[string]bool)
	result := []string{}
	for _, table := range tables {
		if !uniqueTables[table] {
			uniqueTables[table] = true
			result = append(result, table)
		}
	}
	
	return result
}

// newError 创建错误
func (v *SqlNodeBuilderVisitor) newError(msg string, ctx interface{}) error {
	return fmt.Errorf("%s: %v", msg, ctx)
}

// =============================================================================
// Public Methods - 对外提供的方法
// =============================================================================

// GetAllDerefFields 获取所有字段引用
func (v *SqlNodeBuilderVisitor) GetAllDerefFields() []string {
	result := []string{}
	for field := range v.allDerefFields {
		result = append(result, field)
	}
	return result
}

// GetColumnNames 获取所有列名
func (v *SqlNodeBuilderVisitor) GetColumnNames() []string {
	result := []string{}
	for col := range v.columnNameSet {
		result = append(result, col)
	}
	return result
}

// GetSubQueryTables 获取子查询表
func (v *SqlNodeBuilderVisitor) GetSubQueryTables() map[string]SqlNode {
	return v.subQueryTables
}

// GetAssetMap 获取资产映射
func (v *SqlNodeBuilderVisitor) GetAssetMap() map[string]string {
	return v.assetMap
}

// =============================================================================
// Hint 解析方法
// =============================================================================

// VisitHint 访问 hint 节点
// hint: HENT_START hintStatements+=hintStatement (COMMA? hintStatements+=hintStatement)* HENT_END
func (v *SqlNodeBuilderVisitor) VisitHint(ctx *antlr.HintContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	hints := []*SqlHint{}
	
	// 获取所有 hintStatement
	allHintStmts := ctx.AllHintStatement()
	for _, hintStmtIface := range allHintStmts {
		if hintStmt, ok := hintStmtIface.(*antlr.HintStatementContext); ok {
			hint := v.VisitHintStatement(hintStmt)
			if hint != nil {
				if sqlHint, ok := hint.(*SqlHint); ok {
					hints = append(hints, sqlHint)
				}
			}
		}
	}
	
	return hints
}

// VisitHintStatement 访问 hintStatement 节点
// hintStatement: hintName=identifier | hintName=identifier LEFT_PAREN parameters+=primaryExpression (COMMA parameters+=primaryExpression)* RIGHT_PAREN
func (v *SqlNodeBuilderVisitor) VisitHintStatement(ctx *antlr.HintStatementContext) interface{} {
	if ctx == nil {
		return nil
	}
	
	pos := v.getPositionFromContext(ctx)
	
	// 获取 hint 名称
	hintNameIdentifier := ctx.GetHintName()
	if hintNameIdentifier == nil {
		// 尝试使用 Identifier() 方法
		hintNameIdentifier = ctx.Identifier()
	}
	
	if hintNameIdentifier == nil {
		return nil
	}
	
	var hintName string
	if identCtx, ok := hintNameIdentifier.(*antlr.IdentifierContext); ok {
		hintName = v.getIdentifierText(identCtx)
	}
	
	if hintName == "" {
		return nil
	}
	
	// 获取参数列表
	parameters := []SqlNode{}
	allPrimaryExprs := ctx.AllPrimaryExpression()
	for _, paramIface := range allPrimaryExprs {
		// 访问 PrimaryExpression（直接使用接口类型）
		paramNode := v.visitPrimaryExpressionAsNode(paramIface)
		if paramNode != nil {
			parameters = append(parameters, paramNode)
		}
	}
	
	return NewSqlHint(hintName, parameters, pos)
}

// getIdentifierText 从 IdentifierContext 获取标识符文本
func (v *SqlNodeBuilderVisitor) getIdentifierText(ctx *antlr.IdentifierContext) string {
	if ctx == nil {
		return ""
	}
	
	if ctx.StrictIdentifier() != nil {
		return ctx.GetText()
	}
	
	if ctx.StrictNonReserved() != nil {
		return ctx.GetText()
	}
	
	return ctx.GetText()
}

// getPositionFromContext 从上下文获取位置信息
func (v *SqlNodeBuilderVisitor) getPositionFromContext(ctx antlr4.ParserRuleContext) *SqlParserPos {
	if ctx == nil || ctx.GetStart() == nil {
		return &SqlParserPos{}
	}
	
	start := ctx.GetStart()
	stop := ctx.GetStop()
	
	pos := &SqlParserPos{
		LineNumber:   start.GetLine(),
		ColumnNumber: start.GetColumn(),
	}
	
	if stop != nil {
		pos.EndLine = stop.GetLine()
		pos.EndColumn = stop.GetColumn() + len(stop.GetText())
	} else {
		pos.EndLine = pos.LineNumber
		pos.EndColumn = pos.ColumnNumber + len(start.GetText())
	}
	
	return pos
}

// visitPrimaryExpressionAsNode 将 PrimaryExpression 转换为 SqlNode
func (v *SqlNodeBuilderVisitor) visitPrimaryExpressionAsNode(ctx antlr.IPrimaryExpressionContext) SqlNode {
	if ctx == nil {
		return nil
	}
	
	pos := v.getPositionFromContext(ctx)
	text := ctx.GetText()
	
	// 去掉引号（如果有）
	text = strings.Trim(text, "'\"")
	
	// 尝试解析为整数
	if intVal, err := strconv.ParseInt(text, 10, 64); err == nil {
		return NewSqlLiteral(intVal, LiteralInteger, pos)
	}
	
	// 尝试解析为浮点数
	if floatVal, err := strconv.ParseFloat(text, 64); err == nil {
		return NewSqlLiteral(floatVal, LiteralDecimal, pos)
	}
	
	// 检查是否为 true/false
	lowerText := strings.ToLower(text)
	if lowerText == "true" || lowerText == "false" {
		return NewSqlLiteral(lowerText == "true", LiteralBoolean, pos)
	}
	
	// 检查是否为 NULL
	if strings.ToUpper(text) == "NULL" {
		return NewSqlLiteral(nil, LiteralNull, pos)
	}
	
	// 默认作为标识符
	return NewSqlIdentifier([]string{text}, pos)
}
