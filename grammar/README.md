# ANTLR4 语法文件

本目录包含用于生成 SQL 解析器的 ANTLR4 语法文件。

## 语法文件

- `SqlBaseLexer.g4` - Lexer 语法定义（词法分析器）
- `SqlBaseParser.g4` - Parser 语法定义（语法分析器）

这些语法文件基于 Presto SQL 的语法，支持标准 SQL 以及 Presto 特定的扩展。

## 生成解析器代码

### 前置要求

需要安装 ANTLR4 工具。有两种方式：

#### 方法 1: 使用 Go 安装（推荐）

```bash
go install github.com/antlr4-go/antlr/v4/cmd/antlr4@latest
```

安装后，确保 `$GOPATH/bin` 在您的 PATH 中。

#### 方法 2: 使用 Java JAR

1. 下载 ANTLR4 JAR 文件：
   ```bash
   wget https://www.antlr.org/download/antlr-4.13.1-complete.jar
   ```

2. 放在当前目录或设置 CLASSPATH

### 生成代码

#### Windows

```bash
generate.bat
```

#### Linux/Mac

```bash
chmod +x generate.sh
./generate.sh
```

#### 手动生成

使用 Go 工具：
```bash
antlr4 -Dlanguage=Go -o ../parser/antlr -package antlr SqlBaseLexer.g4 SqlBaseParser.g4
```

使用 Java：
```bash
java -jar antlr-4.13.1-complete.jar -Dlanguage=Go -o ../parser/antlr -package antlr SqlBaseLexer.g4 SqlBaseParser.g4
```

### 参数说明

- `-Dlanguage=Go`: 生成 Go 语言代码
- `-o ../parser/antlr`: 输出目录
- `-package antlr`: Go 包名

## 生成的文件

运行生成脚本后，会在 `parser/antlr` 目录下生成以下文件：

- `sqlbaselexer.go` - Lexer 实现
- `sqlbaseparser.go` - Parser 实现
- `sqlbase_listener.go` - Listener 接口（遍历语法树）
- `sqlbase_base_listener.go` - Listener 基础实现
- `SqlBaseLexer.tokens` - Token 定义文件
- `SqlBaseParser.tokens` - Token 定义文件

## 使用生成的解析器

参考 `../parser/sql_parser.go` 中的示例代码，了解如何使用生成的解析器。

## 支持的 SQL 语法

这些语法文件支持以下 SQL 功能：

### 基本查询
- SELECT 语句
- FROM 子句
- WHERE 子句
- JOIN 操作（INNER, LEFT, RIGHT, FULL, CROSS）
- GROUP BY 和 HAVING
- ORDER BY 和 LIMIT
- DISTINCT

### 高级特性
- CTE (Common Table Expressions) - WITH 子句
- 子查询
- 窗口函数
- 聚合函数
- CASE 表达式
- CAST 和类型转换

### DDL 语句
- CREATE TABLE
- ALTER TABLE
- DROP TABLE
- CREATE VIEW
- CREATE INDEX

### DML 语句
- INSERT
- UPDATE
- DELETE
- MERGE

### 其他
- EXPLAIN
- SHOW 语句
- DESCRIBE 语句
- 分区表操作
- 临时表

## 自定义语法

如果需要修改或扩展 SQL 语法：

1. 编辑 `SqlBaseLexer.g4` 或 `SqlBaseParser.g4`
2. 运行生成脚本重新生成代码
3. 更新您的解析器包装代码以使用新的语法规则

## 参考资料

- [ANTLR4 官方文档](https://github.com/antlr/antlr4/blob/master/doc/index.md)
- [ANTLR4 Go 目标](https://github.com/antlr4-go/antlr)
- [Presto SQL 文档](https://prestodb.io/docs/current/)

## 许可证

这些语法文件基于 Apache License 2.0，源自 Presto 项目。


