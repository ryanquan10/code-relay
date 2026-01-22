@echo off
echo ========================================
echo 测试 Account API 是否返回 used_balance
echo ========================================
echo.

echo 1. 检查后端是否运行...
curl -s http://localhost:8087/health >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] 后端服务未运行，请先启动 codex-relay.exe
    pause
    exit /b 1
)
echo [OK] 后端服务正在运行
echo.

echo 2. 测试 /api/accounts 接口...
curl -s http://localhost:8087/api/accounts > test_response.json
echo [OK] API 响应已保存到 test_response.json
echo.

echo 3. 检查响应中是否包含 used_balance 字段...
findstr /C:"used_balance" test_response.json >nul
if %errorlevel% equ 0 (
    echo [成功] 响应中包含 used_balance 字段！
    echo.
    echo 响应内容（前500字符）:
    type test_response.json | findstr /R "." | head -10
) else (
    echo [失败] 响应中没有 used_balance 字段！
    echo.
    echo 完整响应:
    type test_response.json
    echo.
    echo 可能的原因:
    echo - 数据库表没有 used_balance 列
    echo - 后端代码没有重新编译
    echo - Entity 结构体映射错误
)
echo.

echo 4. 清理临时文件...
del test_response.json

echo.
echo 测试完成！
pause
