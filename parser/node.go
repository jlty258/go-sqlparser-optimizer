package parser

import (
	"fmt"
	"strings"
)

// =============================================================================
// SqlNode - 基础接口（类似 Calcite 的 SqlNode）
// =============================================================================

// SqlNode 是所有 SQL 语法树节点的基础接口
// 参考 Apache Calcite 的 SqlNode 设计
type SqlNode interface {
	// Accept 访问者模式
	Accept(visitor SqlNodeVisitor) (interface{}, error)
	
	// GetKind 获取节点类型
	GetKind() SqlKind
	
	// GetPos 获取位置信息
	GetPos() *SqlParserPos
	
	// ToString 转换为 SQL 字符串
	ToString() string
	
	// Clone 克隆节点
	Clone() SqlNode
}

// SqlKind SQL 节点类型枚举（类似 Calcite 的 SqlKind）
type SqlKind string

const (
	// Queries
	SqlKindSelect      SqlKind = "SELECT"
	SqlKindInsert      SqlKind = "INSERT"
	SqlKindUpdate      SqlKind = "UPDATE"
	SqlKindDelete      SqlKind = "DELETE"
	SqlKindMerge       SqlKind = "MERGE"
	
	// DDL
	SqlKindCreateTable SqlKind = "CREATE_TABLE"
	SqlKindAlterTable  SqlKind = "ALTER_TABLE"
	SqlKindDropTable   SqlKind = "DROP_TABLE"
	SqlKindCreateView  SqlKind = "CREATE_VIEW"
	SqlKindDropView    SqlKind = "DROP_VIEW"
	
	// Expressions
	SqlKindIdentifier  SqlKind = "IDENTIFIER"
	SqlKindLiteral     SqlKind = "LITERAL"
	SqlKindCall        SqlKind = "CALL"
	
	// Operators
	SqlKindPlus        SqlKind = "PLUS"
	SqlKindMinus       SqlKind = "MINUS"
	SqlKindTimes       SqlKind = "TIMES"
	SqlKindDivide      SqlKind = "DIVIDE"
	SqlKindEquals      SqlKind = "EQUALS"
	SqlKindNotEquals   SqlKind = "NOT_EQUALS"
	SqlKindGreaterThan SqlKind = "GREATER_THAN"
	SqlKindLessThan    SqlKind = "LESS_THAN"
	SqlKindAnd         SqlKind = "AND"
	SqlKindOr          SqlKind = "OR"
	SqlKindNot         SqlKind = "NOT"
	
	// Other
	SqlKindJoin        SqlKind = "JOIN"
	SqlKindOrderBy     SqlKind = "ORDER_BY"
	SqlKindAs          SqlKind = "AS"
	SqlKindOther       SqlKind = "OTHER"
)

// SqlParserPos 解析位置信息（类似 Calcite 的 SqlParserPos）
type SqlParserPos struct {
	LineNumber   int
	ColumnNumber int
	EndLine      int
	EndColumn    int
}

// =============================================================================
// BaseSqlNode - 基础实现
// =============================================================================

// BaseSqlNode 提供 SqlNode 的基础实现
type BaseSqlNode struct {
	Kind SqlKind
	Pos  *SqlParserPos
}

func (n *BaseSqlNode) GetKind() SqlKind {
	return n.Kind
}

func (n *BaseSqlNode) GetPos() *SqlParserPos {
	return n.Pos
}

// =============================================================================
// SqlIdentifier - 标识符节点
// =============================================================================

// SqlIdentifier 表示标识符（表名、列名等）
// 类似 Calcite 的 SqlIdentifier
type SqlIdentifier struct {
	BaseSqlNode
	Names []string // 支持多部分标识符，如 schema.table.column
}

func NewSqlIdentifier(names []string, pos *SqlParserPos) *SqlIdentifier {
	return &SqlIdentifier{
		BaseSqlNode: BaseSqlNode{Kind: SqlKindIdentifier, Pos: pos},
		Names:       names,
	}
}

func (n *SqlIdentifier) Accept(visitor SqlNodeVisitor) (interface{}, error) {
	return visitor.VisitIdentifier(n)
}

func (n *SqlIdentifier) ToString() string {
	return strings.Join(n.Names, ".")
}

func (n *SqlIdentifier) Clone() SqlNode {
	names := make([]string, len(n.Names))
	copy(names, n.Names)
	return NewSqlIdentifier(names, n.Pos)
}

func (n *SqlIdentifier) GetSimple() string {
	if len(n.Names) > 0 {
		return n.Names[len(n.Names)-1]
	}
	return ""
}

// =============================================================================
// SqlLiteral - 字面量节点
// =============================================================================

// SqlLiteral 表示字面量值
// 类似 Calcite 的 SqlLiteral
type SqlLiteral struct {
	BaseSqlNode
	Value     interface{} // 实际值
	ValueType SqlLiteralType
	TypeName  string // 类型名称，如 "INTEGER", "VARCHAR"
}

type SqlLiteralType int

const (
	LiteralNull SqlLiteralType = iota
	LiteralBoolean
	LiteralInteger
	LiteralDecimal
	LiteralString
	LiteralDate
	LiteralTime
	LiteralTimestamp
)

func NewSqlLiteral(value interface{}, valueType SqlLiteralType, pos *SqlParserPos) *SqlLiteral {
	return &SqlLiteral{
		BaseSqlNode: BaseSqlNode{Kind: SqlKindLiteral, Pos: pos},
		Value:       value,
		ValueType:   valueType,
	}
}

func (n *SqlLiteral) Accept(visitor SqlNodeVisitor) (interface{}, error) {
	return visitor.VisitLiteral(n)
}

func (n *SqlLiteral) ToString() string {
	if n.Value == nil {
		return "NULL"
	}
	return fmt.Sprintf("%v", n.Value)
}

func (n *SqlLiteral) Clone() SqlNode {
	return NewSqlLiteral(n.Value, n.ValueType, n.Pos)
}

// =============================================================================
// SqlCall - 函数调用/操作符节点
// =============================================================================

// SqlCall 表示函数调用或操作符
// 类似 Calcite 的 SqlCall
type SqlCall struct {
	BaseSqlNode
	Operator *SqlOperator // 操作符信息
	Operands []SqlNode    // 操作数
}

func NewSqlCall(operator *SqlOperator, operands []SqlNode, pos *SqlParserPos) *SqlCall {
	kind := SqlKindCall
	if operator != nil && operator.Kind != "" {
		kind = operator.Kind
	}
	return &SqlCall{
		BaseSqlNode: BaseSqlNode{Kind: kind, Pos: pos},
		Operator:    operator,
		Operands:    operands,
	}
}

func (n *SqlCall) Accept(visitor SqlNodeVisitor) (interface{}, error) {
	return visitor.VisitCall(n)
}

func (n *SqlCall) ToString() string {
	if n.Operator == nil {
		return "UNKNOWN"
	}
	return n.Operator.Format(n.Operands)
}

func (n *SqlCall) Clone() SqlNode {
	operands := make([]SqlNode, len(n.Operands))
	for i, op := range n.Operands {
		operands[i] = op.Clone()
	}
	return NewSqlCall(n.Operator, operands, n.Pos)
}

// SqlOperator 操作符信息
type SqlOperator struct {
	Name     string
	Kind     SqlKind
	Syntax   SqlSyntax
	LeftPrec int // 左结合优先级
	RightPrec int // 右结合优先级
}

type SqlSyntax int

const (
	SyntaxFunction      SqlSyntax = iota // 函数调用: f(x, y)
	SyntaxPrefix                         // 前缀: -x, NOT x
	SyntaxPostfix                        // 后缀: x!
	SyntaxBinary                         // 二元: x + y
	SyntaxSpecial                        // 特殊语法
)

func (op *SqlOperator) Format(operands []SqlNode) string {
	switch op.Syntax {
	case SyntaxFunction:
		args := make([]string, len(operands))
		for i, arg := range operands {
			args[i] = arg.ToString()
		}
		return fmt.Sprintf("%s(%s)", op.Name, strings.Join(args, ", "))
	case SyntaxBinary:
		if len(operands) == 2 {
			return fmt.Sprintf("%s %s %s", operands[0].ToString(), op.Name, operands[1].ToString())
		}
	case SyntaxPrefix:
		if len(operands) == 1 {
			return fmt.Sprintf("%s %s", op.Name, operands[0].ToString())
		}
	}
	return op.Name
}

// =============================================================================
// SqlHint - HINT 节点
// =============================================================================

// SqlHint 表示 SQL HINT（优化器提示）
// 格式: /*+ HINT_NAME(param1, param2, ...) */
type SqlHint struct {
	BaseSqlNode
	Name       string    // Hint 名称，如 "JOIN", "FUNC", "LOCAL"
	Parameters []SqlNode // Hint 参数列表
}

func NewSqlHint(name string, parameters []SqlNode, pos *SqlParserPos) *SqlHint {
	return &SqlHint{
		BaseSqlNode: BaseSqlNode{Kind: SqlKindOther, Pos: pos},
		Name:        name,
		Parameters:  parameters,
	}
}

func (n *SqlHint) Accept(visitor SqlNodeVisitor) (interface{}, error) {
	return visitor.VisitHint(n)
}

func (n *SqlHint) ToString() string {
	var sb strings.Builder
	sb.WriteString("/*+ ")
	sb.WriteString(n.Name)
	
	if len(n.Parameters) > 0 {
		sb.WriteString("(")
		for i, param := range n.Parameters {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(param.ToString())
		}
		sb.WriteString(")")
	}
	
	sb.WriteString(" */")
	return sb.String()
}

func (n *SqlHint) Clone() SqlNode {
	params := make([]SqlNode, len(n.Parameters))
	for i, p := range n.Parameters {
		params[i] = p.Clone()
	}
	return NewSqlHint(n.Name, params, n.Pos)
}

// =============================================================================
// SqlSelect - SELECT 语句节点
// =============================================================================

// SqlSelect 表示 SELECT 语句
// 类似 Calcite 的 SqlSelect
type SqlSelect struct {
	BaseSqlNode
	Hints        []*SqlHint  // HINT 列表，如 /*+ JOIN(TEE) */
	KeywordList  []string    // 关键字列表，如 DISTINCT
	SelectList   []SqlNode   // SELECT 列表
	From         SqlNode     // FROM 子句
	Where        SqlNode     // WHERE 子句
	GroupBy      []SqlNode   // GROUP BY 列表
	Having       SqlNode     // HAVING 子句
	WindowDecls  []SqlNode   // WINDOW 声明
	OrderBy      []SqlNode   // ORDER BY 列表
	Offset       SqlNode     // OFFSET
	Fetch        SqlNode     // FETCH/LIMIT
}

func NewSqlSelect(pos *SqlParserPos) *SqlSelect {
	return &SqlSelect{
		BaseSqlNode: BaseSqlNode{Kind: SqlKindSelect, Pos: pos},
		Hints:       []*SqlHint{},
		KeywordList: []string{},
		SelectList:  []SqlNode{},
		GroupBy:     []SqlNode{},
		WindowDecls: []SqlNode{},
		OrderBy:     []SqlNode{},
	}
}

func (n *SqlSelect) Accept(visitor SqlNodeVisitor) (interface{}, error) {
	return visitor.VisitSelect(n)
}

func (n *SqlSelect) ToString() string {
	var sb strings.Builder
	sb.WriteString("SELECT ")
	
	// 添加 HINT
	if len(n.Hints) > 0 {
		for _, hint := range n.Hints {
			sb.WriteString(hint.ToString())
			sb.WriteString(" ")
		}
	}
	
	if len(n.KeywordList) > 0 {
		sb.WriteString(strings.Join(n.KeywordList, " "))
		sb.WriteString(" ")
	}
	
	for i, item := range n.SelectList {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(item.ToString())
	}
	
	if n.From != nil {
		sb.WriteString(" FROM ")
		sb.WriteString(n.From.ToString())
	}
	
	if n.Where != nil {
		sb.WriteString(" WHERE ")
		sb.WriteString(n.Where.ToString())
	}
	
	if len(n.GroupBy) > 0 {
		sb.WriteString(" GROUP BY ")
		for i, g := range n.GroupBy {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(g.ToString())
		}
	}
	
	if n.Having != nil {
		sb.WriteString(" HAVING ")
		sb.WriteString(n.Having.ToString())
	}
	
	if len(n.OrderBy) > 0 {
		sb.WriteString(" ORDER BY ")
		for i, o := range n.OrderBy {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(o.ToString())
		}
	}
	
	if n.Fetch != nil {
		sb.WriteString(" LIMIT ")
		sb.WriteString(n.Fetch.ToString())
	}
	
	return sb.String()
}

func (n *SqlSelect) Clone() SqlNode {
	// 实现克隆逻辑
	return n
}

// =============================================================================
// SqlJoin - JOIN 节点
// =============================================================================

// SqlJoin 表示 JOIN 操作
type SqlJoin struct {
	BaseSqlNode
	Left      SqlNode
	Right     SqlNode
	JoinType  JoinType
	Condition SqlNode // ON 条件
	Using     []SqlNode // USING 列表
}

type JoinType string

const (
	JoinInner JoinType = "INNER"
	JoinLeft  JoinType = "LEFT"
	JoinRight JoinType = "RIGHT"
	JoinFull  JoinType = "FULL"
	JoinCross JoinType = "CROSS"
)

func NewSqlJoin(left, right SqlNode, joinType JoinType, condition SqlNode, pos *SqlParserPos) *SqlJoin {
	return &SqlJoin{
		BaseSqlNode: BaseSqlNode{Kind: SqlKindJoin, Pos: pos},
		Left:        left,
		Right:       right,
		JoinType:    joinType,
		Condition:   condition,
	}
}

func (n *SqlJoin) Accept(visitor SqlNodeVisitor) (interface{}, error) {
	return visitor.VisitJoin(n)
}

func (n *SqlJoin) ToString() string {
	var sb strings.Builder
	sb.WriteString(n.Left.ToString())
	sb.WriteString(" ")
	sb.WriteString(string(n.JoinType))
	sb.WriteString(" JOIN ")
	sb.WriteString(n.Right.ToString())
	if n.Condition != nil {
		sb.WriteString(" ON ")
		sb.WriteString(n.Condition.ToString())
	}
	return sb.String()
}

func (n *SqlJoin) Clone() SqlNode {
	return NewSqlJoin(n.Left.Clone(), n.Right.Clone(), n.JoinType, n.Condition.Clone(), n.Pos)
}

// =============================================================================
// SqlBasicCall - 简单表引用（带别名）
// =============================================================================

// SqlBasicCall 表示基本的表引用，可以包含别名
type SqlBasicCall struct {
	BaseSqlNode
	Operand SqlNode
	Alias   string
}

func NewSqlBasicCall(operand SqlNode, alias string, pos *SqlParserPos) *SqlBasicCall {
	return &SqlBasicCall{
		BaseSqlNode: BaseSqlNode{Kind: SqlKindAs, Pos: pos},
		Operand:     operand,
		Alias:       alias,
	}
}

func (n *SqlBasicCall) Accept(visitor SqlNodeVisitor) (interface{}, error) {
	return visitor.VisitBasicCall(n)
}

func (n *SqlBasicCall) ToString() string {
	if n.Alias != "" {
		return fmt.Sprintf("%s AS %s", n.Operand.ToString(), n.Alias)
	}
	return n.Operand.ToString()
}

func (n *SqlBasicCall) Clone() SqlNode {
	return NewSqlBasicCall(n.Operand.Clone(), n.Alias, n.Pos)
}

// =============================================================================
// SqlOrderBy - ORDER BY 节点
// =============================================================================

// SqlOrderBy ORDER BY 排序项
type SqlOrderBy struct {
	BaseSqlNode
	Query     SqlNode
	OrderList []SqlNode
}

// =============================================================================
// SqlNodeList - 节点列表
// =============================================================================

// SqlNodeList 表示节点列表
type SqlNodeList struct {
	BaseSqlNode
	List []SqlNode
}

func NewSqlNodeList(list []SqlNode, pos *SqlParserPos) *SqlNodeList {
	return &SqlNodeList{
		BaseSqlNode: BaseSqlNode{Kind: SqlKindOther, Pos: pos},
		List:        list,
	}
}

func (n *SqlNodeList) Accept(visitor SqlNodeVisitor) (interface{}, error) {
	return visitor.VisitNodeList(n)
}

func (n *SqlNodeList) ToString() string {
	items := make([]string, len(n.List))
	for i, item := range n.List {
		items[i] = item.ToString()
	}
	return strings.Join(items, ", ")
}

func (n *SqlNodeList) Clone() SqlNode {
	list := make([]SqlNode, len(n.List))
	for i, item := range n.List {
		list[i] = item.Clone()
	}
	return NewSqlNodeList(list, n.Pos)
}

// =============================================================================
// Visitor 接口
// =============================================================================

// SqlNodeVisitor 访问者接口
type SqlNodeVisitor interface {
	VisitIdentifier(node *SqlIdentifier) (interface{}, error)
	VisitLiteral(node *SqlLiteral) (interface{}, error)
	VisitCall(node *SqlCall) (interface{}, error)
	VisitSelect(node *SqlSelect) (interface{}, error)
	VisitJoin(node *SqlJoin) (interface{}, error)
	VisitBasicCall(node *SqlBasicCall) (interface{}, error)
	VisitNodeList(node *SqlNodeList) (interface{}, error)
	VisitHint(node *SqlHint) (interface{}, error)
}

// =============================================================================
// 兼容性类型（保持向后兼容）
// =============================================================================

// ParseResult 解析结果
type ParseResult struct {
	SQL       string
	Statement SqlNode // 改为 SqlNode 接口
	Error     error
}

// 为了兼容 analyzer，保留旧的结构体
type Statement struct {
	SelectStmt *SelectStatement
}

type SelectStatement struct {
	CTEs        []*CTE
	Columns     []*Column
	From        *FromClause
	FromClause  *FromClause
	Where       string
	GroupBy     []string
	Having      string
	OrderBy     []string
	Limit       string
}

type CTE struct {
	Name  string
	Query string
}

type Column struct {
	Expression string
	Alias      string
	IsFunction bool
	Function   string
}

type FromClause struct {
	Tables []*TableReference
	Joins  []*Join
}

type TableReference struct {
	Name  string
	Alias string
}

type Join struct {
	Type      string
	Table     *TableReference
	Condition string
}

