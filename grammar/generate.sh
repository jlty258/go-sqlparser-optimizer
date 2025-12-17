#!/bin/bash
# 使用 ANTLR4 生成 Go 语言解析器代码
# 需要先安装 ANTLR4: https://www.antlr.org/download.html

echo "正在生成 ANTLR4 Go 解析器代码..."

# 方法1: 使用 Java 运行 ANTLR4 (需要先下载 antlr jar)
# ANTLR_JAR="antlr-4.13.1-complete.jar"
# if [ -f "$ANTLR_JAR" ]; then
#     java -jar "$ANTLR_JAR" -Dlanguage=Go -o ../parser/antlr -package antlr SqlBaseLexer.g4 SqlBaseParser.g4
# fi

# 方法2: 使用 Go 安装的 ANTLR4 工具
if command -v antlr4 &> /dev/null; then
    antlr4 -Dlanguage=Go -o ../parser/antlr -package antlr SqlBaseLexer.g4 SqlBaseParser.g4
    echo ""
    echo "代码生成完成！生成的文件在 parser/antlr 目录中"
else
    echo ""
    echo "错误: 未找到 antlr4 命令"
    echo "请先安装 ANTLR4 工具:"
    echo "  go install github.com/antlr4-go/antlr/v4/cmd/antlr4@latest"
    echo ""
    echo "或者下载 antlr jar 包:"
    echo "  wget https://www.antlr.org/download/antlr-4.13.1-complete.jar"
    exit 1
fi


