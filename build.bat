@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

echo ============================================
echo  Xiuno BBS - 全栈单文件二进制构建脚本
echo ============================================
echo.

echo [1/3] 开始构建 Vue 现代前端...
cd /d "%~dp0..\xiuno-ui"
call npm run build
if %errorlevel% neq 0 (
    echo [失败] Vue 构建出错，终止编译
    exit /b %errorlevel%
)
echo [成功] Vue 构建完成
echo.

echo [2/3] 将前端产物注入 Go 内嵌目录...
cd /d "%~dp0"
if exist ui\dist rmdir /s /q ui\dist
xcopy ..\xiuno-ui\dist ui\dist\ /e /i /h /y >nul
echo [成功] 前端产物已复制到 ui/dist/
echo.

echo [3/3] 编译终极单文件二进制引擎 (压缩符号表)...
set CGO_ENABLED=0
go build -ldflags="-s -w" -o xiuno-server.exe cmd\xiuno\main.go
if %errorlevel% neq 0 (
    echo [失败] Go 编译出错
    exit /b %errorlevel%
)
echo [成功] 编译完成！
echo.

echo ============================================
echo  产物：xiuno-server.exe
for %%I in (xiuno-server.exe) do echo  大小：%%~zI 字节
echo ============================================
echo.
echo 部署步骤：
echo  1. 将 xiuno-server.exe 放到空目录
echo  2. 复制 schema.sql 到同目录
echo  3. 创建 config.json（参考 conf/config.json）
echo  4. 创建 upload/ 目录（含 attach/ avatar/ forum/ 子目录）
echo  5. 双击 xiuno-server.exe 启动
echo.
pause
