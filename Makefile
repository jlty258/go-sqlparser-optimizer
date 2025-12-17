.PHONY: build run test clean help install gen-antlr install-antlr

# 默认目标
all: build

# 构建项目
build:
	@echo "构建项目..."
	go build -o bin/go-job-service main.go

# 运行主程序
run:
	@echo "运行主程序..."
	go run main.go

# 运行简单示例
run-example:
	@echo "运行示例程序..."
	go run examples/simple_example.go

# 运行测试
test:
	@echo "运行测试..."
	go test -v ./...

# 运行测试并显示覆盖率
test-coverage:
	@echo "运行测试并生成覆盖率报告..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# 格式化代码
fmt:
	@echo "格式化代码..."
	go fmt ./...

# 静态检查
vet:
	@echo "运行静态检查..."
	go vet ./...

# 安装依赖
install:
	@echo "安装依赖..."
	go mod download
	go mod tidy

# 安装 ANTLR4 工具
install-antlr:
	@echo "安装 ANTLR4 工具..."
	go install github.com/antlr4-go/antlr/v4/cmd/antlr4@latest
	@echo "ANTLR4 工具安装完成"

# 生成 ANTLR4 解析器代码
gen-antlr:
	@echo "生成 ANTLR4 解析器代码..."
	@cd grammar && antlr4 -Dlanguage=Go -o ../parser/antlr -package antlr SqlBaseLexer.g4 SqlBaseParser.g4
	@echo "ANTLR4 代码生成完成！文件位于 parser/antlr/ 目录"

# 清理构建文件
clean:
	@echo "清理构建文件..."
	rm -rf bin/
	rm -rf parser/antlr/
	rm -f coverage.out coverage.html
	go clean

# 显示帮助信息
help:
	@echo "Go Job Service - Presto SQL Parser"
	@echo ""
	@echo "可用命令:"
	@echo "  make build          - 构建项目"
	@echo "  make run            - 运行主程序"
	@echo "  make run-example    - 运行示例程序"
	@echo "  make test           - 运行测试"
	@echo "  make test-coverage  - 运行测试并生成覆盖率报告"
	@echo "  make fmt            - 格式化代码"
	@echo "  make vet            - 运行静态检查"
	@echo "  make install        - 安装依赖"
	@echo "  make install-antlr  - 安装 ANTLR4 工具"
	@echo "  make gen-antlr      - 生成 ANTLR4 解析器代码"
	@echo "  make clean          - 清理构建文件"
	@echo "  make help           - 显示此帮助信息"

