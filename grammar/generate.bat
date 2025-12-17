@echo off
REM 使用 ANTLR4 生成 Go 语言解析器代码
REM 需要先安装 ANTLR4: https://www.antlr.org/download.html
REM
REM 安装方法：
REM 1. 下载 antlr-4.13.1-complete.jar
REM 2. 设置环境变量 CLASSPATH=.;C:\path\to\antlr-4.13.1-complete.jar;%CLASSPATH%
REM 3. 或者直接使用下面的命令指定jar路径

echo 正在生成 ANTLR4 Go 解析器代码...

REM 如果已经设置了CLASSPATH，使用这个命令
REM java org.antlr.v4.Tool -Dlanguage=Go -o ../parser/antlr -package antlr SqlBaseLexer.g4 SqlBaseParser.g4

REM 如果没有设置CLASSPATH，需要指定jar包路径
REM 请修改下面的路径为您的实际 antlr jar 包路径
SET ANTLR_JAR=antlr-4.13.1-complete.jar

if exist "%ANTLR_JAR%" (
    java -jar "%ANTLR_JAR%" -Dlanguage=Go -o ../parser/antlr -package antlr SqlBaseLexer.g4 SqlBaseParser.g4
    echo.
    echo 代码生成完成！生成的文件在 parser/antlr 目录中
) else (
    echo.
    echo 错误: 找不到 ANTLR jar 包
    echo 请下载 antlr-4.13.1-complete.jar 并放在当前目录，或修改脚本中的 ANTLR_JAR 路径
    echo 下载地址: https://www.antlr.org/download/antlr-4.13.1-complete.jar
    echo.
    echo 或者使用 Go 安装 ANTLR4 工具:
    echo   go install github.com/antlr4-go/antlr/v4/cmd/antlr4@latest
    echo.
    echo 然后使用以下命令生成:
    echo   antlr4 -Dlanguage=Go -o ../parser/antlr -package antlr SqlBaseLexer.g4 SqlBaseParser.g4
)

pause


