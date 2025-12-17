# 文件重命名方案

## 重命名原则

1. **简洁性**: 去掉冗余的前缀（如目录已经表明是 SQL 相关）
2. **一致性**: 统一使用 Go 社区推荐的命名风格（无下划线）
3. **语义化**: 文件名应该清楚表达其职责
4. **可读性**: 避免缩写，使用完整单词

## 重命名方案

### parser/ 目录

| 当前文件名 | 新文件名 | 理由 |
|-----------|---------|------|
| `ast.go` | `node.go` | 更清晰，表示 SqlNode 相关定义 |
| `antlr_sql_parser.go` | `parser.go` | 简洁，目录已表明是 parser |
| `sql_node_visitor.go` | `visitor.go` | 简洁，目录上下文已明确 |
| `sql_node_visitor_test.go` | `visitor_test.go` | 与 visitor.go 对应 |

**优点**:
- ✅ 更符合 Go 命名习惯
- ✅ 去掉了冗余的 `sql_` 前缀
- ✅ 文件名更短更清晰
- ✅ 在 IDE 中更容易识别

### analyzer/ 目录

| 当前文件名 | 新文件名 | 理由 |
|-----------|---------|------|
| `sql_analyzer.go` | `analyzer.go` | 目录名已表明是 analyzer |

### examples/ 目录

| 当前文件名 | 新文件名 | 理由 |
|-----------|---------|------|
| `antlr_example.go` | `basic.go` | 更语义化 |
| `parse_sql_example.go` | `advanced.go` | 表示高级用法 |
| `simple_example.go` | `simple.go` | 保持简洁 |

**或者更清晰的方案**:
| 当前文件名 | 新文件名 | 理由 |
|-----------|---------|------|
| `antlr_example.go` | `example_basic.go` | 基础示例 |
| `parse_sql_example.go` | `example_visitor.go` | Visitor 模式示例 |
| `simple_example.go` | `example_simple.go` | 简单示例 |

## 重命名后的项目结构

```
go-job-service/
├── analyzer/
│   └── analyzer.go              # SQL 分析器（原 sql_analyzer.go）
├── examples/
│   ├── example_basic.go         # 基础示例（原 antlr_example.go）
│   ├── example_visitor.go       # Visitor 示例（原 parse_sql_example.go）
│   └── example_simple.go        # 简单示例（原 simple_example.go）
├── grammar/
│   ├── antlr-4.13.1-complete.jar
│   ├── generate.bat
│   ├── generate.sh
│   ├── README.md
│   ├── SqlBaseLexer.g4
│   └── SqlBaseParser.g4
├── parser/
│   ├── antlr/                   # ANTLR 生成的代码（保持不变）
│   │   ├── sqlbase_lexer.go
│   │   ├── sqlbase_parser.go
│   │   ├── sqlbaseparser_base_visitor.go
│   │   └── sqlbaseparser_visitor.go
│   ├── parser.go                # 解析器入口（原 antlr_sql_parser.go）
│   ├── node.go                  # SqlNode 定义（原 ast.go）
│   ├── visitor.go               # Visitor 实现（原 sql_node_visitor.go）
│   └── visitor_test.go          # 测试文件（原 sql_node_visitor_test.go）
├── .gitignore
├── go.mod
├── go.sum
├── IMPLEMENTATION_SUMMARY.md
├── LICENSE
├── main.go
├── Makefile
├── PROJECT_CLEANUP.md
└── README.md
```

## 优势对比

### 之前（冗长、不一致）
```
parser/
├── antlr_sql_parser.go      # 17 字符，有下划线
├── ast.go                   # 3 字符，太通用
├── sql_node_visitor.go      # 19 字符，有下划线
└── sql_node_visitor_test.go # 24 字符，有下划线
```

### 之后（简洁、一致）
```
parser/
├── parser.go                # 6 字符，清晰
├── node.go                  # 4 字符，明确
├── visitor.go               # 7 字符，简洁
└── visitor_test.go          # 12 字符，一致
```

## 命名对比表

| 方面 | 之前 | 之后 | 改进 |
|-----|------|------|------|
| 平均长度 | 15.75 字符 | 7.25 字符 | ⬇️ 54% |
| 下划线使用 | 4 个文件 | 0 个文件 | ✅ 统一 |
| 冗余前缀 | 有 `sql_` | 无 | ✅ 简洁 |
| 语义清晰度 | 中等 | 高 | ✅ 提升 |
| Go 风格符合度 | 低 | 高 | ✅ 标准 |

## 参考 Go 项目的命名习惯

### 标准库示例
```
net/http/
├── client.go      # 不是 http_client.go
├── server.go      # 不是 http_server.go
├── request.go     # 不是 http_request.go
└── response.go    # 不是 http_response.go
```

### 知名项目示例
```
kubernetes/pkg/scheduler/
├── scheduler.go   # 不是 k8s_scheduler.go
├── factory.go     # 不是 scheduler_factory.go
└── queue.go       # 不是 scheduler_queue.go

docker/daemon/
├── daemon.go      # 不是 docker_daemon.go
├── container.go   # 不是 daemon_container.go
└── image.go       # 不是 daemon_image.go
```

## 实施步骤

1. **重命名文件**
   ```bash
   # parser/
   git mv parser/ast.go parser/node.go
   git mv parser/antlr_sql_parser.go parser/parser.go
   git mv parser/sql_node_visitor.go parser/visitor.go
   git mv parser/sql_node_visitor_test.go parser/visitor_test.go
   
   # analyzer/
   git mv analyzer/sql_analyzer.go analyzer/analyzer.go
   
   # examples/
   git mv examples/antlr_example.go examples/example_basic.go
   git mv examples/parse_sql_example.go examples/example_visitor.go
   git mv examples/simple_example.go examples/example_simple.go
   ```

2. **更新导入路径**
   - 文件名改变不影响包导入路径
   - 只需要更新文件内部的引用（如果有）

3. **更新文档**
   - 更新 README.md 中的文件引用
   - 更新 IMPLEMENTATION_SUMMARY.md 中的文件名

4. **验证编译**
   ```bash
   go build ./...
   go test ./...
   ```

## 建议

**推荐方案**: 采用上述重命名方案

**理由**:
1. ✅ 符合 Go 社区最佳实践
2. ✅ 代码更简洁易读
3. ✅ 在 IDE 中更容易导航
4. ✅ 减少 54% 的文件名长度
5. ✅ 统一命名风格

**注意事项**:
- ANTLR 生成的文件保持不变（因为是自动生成的）
- 文档文件（*.md）保持不变
- 配置文件保持不变

