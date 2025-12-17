package parser

import (
	"fmt"
	"strings"

	antlr4 "github.com/antlr4-go/antlr/v4"
	"go-job-service/parser/antlr"
)

// SQLParserResult 包含SQL解析结果
type SQLParserResult struct {
	Success      bool
	ErrorMessage string
	SqlNode      SqlNode     // SqlNode AST（类似 Calcite）
	AntlrTree    interface{} // 原始 ANTLR 解析树
	Errors       []string
}

// AntlrErrorListener 自定义错误监听器
type AntlrErrorListener struct {
	*antlr4.DefaultErrorListener
	Errors []string
}

// NewAntlrErrorListener 创建新的错误监听器
func NewAntlrErrorListener() *AntlrErrorListener {
	return &AntlrErrorListener{
		Errors: make([]string, 0),
	}
}

// SyntaxError 实现 ErrorListener 接口
func (l *AntlrErrorListener) SyntaxError(recognizer antlr4.Recognizer, offendingSymbol interface{}, line, column int, msg string, e antlr4.RecognitionException) {
	errorMsg := fmt.Sprintf("line %d:%d %s", line, column, msg)
	l.Errors = append(l.Errors, errorMsg)
}

// ParseSQLWithAntlr 使用ANTLR4解析SQL语句，返回 SqlNode 结构
// 使用 Visitor 模式直接在访问 AST 时构建 SqlNode
func ParseSQLWithAntlr(sql string) (*SQLParserResult, error) {
	// 清理SQL语句
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return nil, fmt.Errorf("SQL语句不能为空")
	}

	result := &SQLParserResult{
		Success: true,
		Errors:  make([]string, 0),
	}
	
	// 1. 创建输入流
	input := antlr4.NewInputStream(sql)
	
	// 2. 创建词法分析器
	lexer := antlr.NewSqlBaseLexer(input)
	
	// 3. 创建token流
	stream := antlr4.NewCommonTokenStream(lexer, antlr4.TokenDefaultChannel)
	
	// 4. 创建语法分析器
	parser := antlr.NewSqlBaseParser(stream)
	
	// 5. 添加自定义错误监听器
	errorListener := NewAntlrErrorListener()
	parser.RemoveErrorListeners()
	parser.AddErrorListener(errorListener)
	
	// 6. 解析SQL语句（从起始规则开始）
	tree := parser.SingleStatement()
	
	// 7. 检查是否有语法错误
	if len(errorListener.Errors) > 0 {
		result.Success = false
		result.Errors = errorListener.Errors
		result.ErrorMessage = strings.Join(errorListener.Errors, "; ")
		return result, fmt.Errorf("解析失败: %s", result.ErrorMessage)
	}
	
	// 8. 保存原始 ANTLR 解析树
	result.AntlrTree = tree
	
	// 9. 使用 Visitor 模式直接构建 SqlNode
	visitor := NewSqlNodeBuilderVisitor()
	
	// 类型断言并直接调用 VisitSingleStatement
	singleStmtCtx, ok := tree.(*antlr.SingleStatementContext)
	if !ok {
		return nil, fmt.Errorf("tree 类型错误: %T", tree)
	}
	
	sqlNodeResult := visitor.VisitSingleStatement(singleStmtCtx)
	
	if sqlNodeResult == nil {
		return nil, fmt.Errorf("无法构建 SqlNode")
	}
	
	sqlNode, ok := sqlNodeResult.(SqlNode)
	if !ok {
		return nil, fmt.Errorf("visitor 返回的不是 SqlNode 类型: %T", sqlNodeResult)
	}
	
	result.SqlNode = sqlNode
	
	return result, nil
}

// ExtractTableNames 从 SqlNode 中提取表名
func ExtractTableNames(sqlNode SqlNode) ([]string, error) {
	if sqlNode == nil {
		return nil, fmt.Errorf("SqlNode is nil")
	}
	
	extractor := &TableNameExtractor{
		tables: make([]string, 0),
	}
	
	_, err := sqlNode.Accept(extractor)
	return extractor.tables, err
}

// ExtractColumns 从 SqlNode 中提取列名
func ExtractColumns(sqlNode SqlNode) ([]string, error) {
	if sqlNode == nil {
		return nil, fmt.Errorf("SqlNode is nil")
	}
	
	extractor := &ColumnNameExtractor{
		columns: make([]string, 0),
	}
	
	_, err := sqlNode.Accept(extractor)
	return extractor.columns, err
}

// =============================================================================
// 信息提取 Visitor
// =============================================================================

// TableNameExtractor 提取表名的 Visitor
type TableNameExtractor struct {
	tables []string
}

func (v *TableNameExtractor) VisitIdentifier(node *SqlIdentifier) (interface{}, error) {
	return nil, nil
}

func (v *TableNameExtractor) VisitLiteral(node *SqlLiteral) (interface{}, error) {
	return nil, nil
}

func (v *TableNameExtractor) VisitCall(node *SqlCall) (interface{}, error) {
	for _, operand := range node.Operands {
		operand.Accept(v)
	}
	return nil, nil
}

func (v *TableNameExtractor) VisitSelect(node *SqlSelect) (interface{}, error) {
	// 提取 FROM 子句中的表名
	if node.From != nil {
		node.From.Accept(v)
	}
	return nil, nil
}

func (v *TableNameExtractor) VisitJoin(node *SqlJoin) (interface{}, error) {
	if node.Left != nil {
		node.Left.Accept(v)
	}
	if node.Right != nil {
		node.Right.Accept(v)
	}
	return nil, nil
}

func (v *TableNameExtractor) VisitBasicCall(node *SqlBasicCall) (interface{}, error) {
	// 检查操作数是否为标识符（表名）
	if identifier, ok := node.Operand.(*SqlIdentifier); ok {
		// 排除列名（单个部分）和星号
		if len(identifier.Names) > 0 && identifier.Names[0] != "*" {
			tableName := identifier.ToString()
			v.tables = append(v.tables, tableName)
		}
	} else {
		// 递归处理子节点
		node.Operand.Accept(v)
	}
	return nil, nil
}

func (v *TableNameExtractor) VisitNodeList(node *SqlNodeList) (interface{}, error) {
	for _, item := range node.List {
		item.Accept(v)
	}
	return nil, nil
}

func (v *TableNameExtractor) VisitHint(node *SqlHint) (interface{}, error) {
	// Hint 不包含表名，直接返回
	return nil, nil
}

// ColumnNameExtractor 提取列名的 Visitor
type ColumnNameExtractor struct {
	columns []string
}

func (v *ColumnNameExtractor) VisitIdentifier(node *SqlIdentifier) (interface{}, error) {
	return nil, nil
}

func (v *ColumnNameExtractor) VisitLiteral(node *SqlLiteral) (interface{}, error) {
	return nil, nil
}

func (v *ColumnNameExtractor) VisitCall(node *SqlCall) (interface{}, error) {
	for _, operand := range node.Operands {
		operand.Accept(v)
	}
	return nil, nil
}

func (v *ColumnNameExtractor) VisitSelect(node *SqlSelect) (interface{}, error) {
	// 提取 SELECT 列表中的列名
	for _, selectItem := range node.SelectList {
		if basicCall, ok := selectItem.(*SqlBasicCall); ok {
			// 带别名的列
			if identifier, ok := basicCall.Operand.(*SqlIdentifier); ok {
				v.columns = append(v.columns, identifier.ToString())
			}
		} else if identifier, ok := selectItem.(*SqlIdentifier); ok {
			// 直接的标识符
			if identifier.ToString() != "*" {
				v.columns = append(v.columns, identifier.ToString())
			}
		}
	}
	return nil, nil
}

func (v *ColumnNameExtractor) VisitJoin(node *SqlJoin) (interface{}, error) {
	return nil, nil
}

func (v *ColumnNameExtractor) VisitBasicCall(node *SqlBasicCall) (interface{}, error) {
	return nil, nil
}

func (v *ColumnNameExtractor) VisitNodeList(node *SqlNodeList) (interface{}, error) {
	for _, item := range node.List {
		item.Accept(v)
	}
	return nil, nil
}

func (v *ColumnNameExtractor) VisitHint(node *SqlHint) (interface{}, error) {
	// Hint 不包含列名，直接返回
	return nil, nil
}


