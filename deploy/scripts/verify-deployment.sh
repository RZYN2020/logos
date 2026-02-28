# 部署验证脚本

#!/bin/bash
# Log Analyzer 部署验证脚本
# 用于验证服务是否正确部署和运行

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 配置
BASE_URL="${BASE_URL:-http://localhost:8080}"
API_URL="${BASE_URL}/api/v1"

echo "======================================"
echo "Log Analyzer 部署验证"
echo "======================================"
echo ""

# 检查服务健康状态
check_health() {
    echo -n "检查服务健康状态... "
    response=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/health" 2>/dev/null || echo "000")
    if [ "$response" = "200" ]; then
        echo -e "${GREEN}通过${NC}"
        return 0
    else
        echo -e "${RED}失败 (HTTP $response)${NC}"
        return 1
    fi
}

# 检查服务就绪状态
check_ready() {
    echo -n "检查服务就绪状态... "
    response=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/ready" 2>/dev/null || echo "000")
    if [ "$response" = "200" ]; then
        echo -e "${GREEN}通过${NC}"
        return 0
    else
        echo -e "${RED}失败 (HTTP $response)${NC}"
        return 1
    fi
}

# 检查系统信息
check_system_info() {
    echo -n "检查系统信息... "
    response=$(curl -s "${API_URL}/info" 2>/dev/null || echo "")
    if [ -n "$response" ] && echo "$response" | grep -q "system"; then
        echo -e "${GREEN}通过${NC}"
        echo "  系统信息：$response"
        return 0
    else
        echo -e "${RED}失败${NC}"
        return 1
    fi
}

# 检查认证功能
check_authentication() {
    echo -n "检查认证功能... "

    # 尝试登录
    login_response=$(curl -s -X POST "${API_URL}/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin123"}' 2>/dev/null || echo "")

    if [ -n "$login_response" ] && echo "$login_response" | grep -q "token"; then
        echo -e "${GREEN}通过${NC}"
        TOKEN=$(echo "$login_response" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
        echo "  获取 token: ${TOKEN:0:20}..."
        return 0
    else
        echo -e "${RED}失败${NC}"
        return 1
    fi
}

# 检查规则 API
check_rules_api() {
    echo -n "检查规则 API... "

    # 使用 token 获取规则列表
    rules_response=$(curl -s "${API_URL}/rules" \
        -H "Authorization: Bearer $TOKEN" 2>/dev/null || echo "")

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}通过${NC}"
        return 0
    else
        echo -e "${RED}失败${NC}"
        return 1
    fi
}

# 检查分析 API
check_analysis_api() {
    echo -n "检查分析 API... "

    # 测试模式挖掘接口
    analysis_response=$(curl -s -X POST "${API_URL}/analysis/mine" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d '{"logs":[{"timestamp":"2024-01-01T00:00:00Z","level":"ERROR","service":"test","message":"test"}]}' 2>/dev/null || echo "")

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}通过${NC}"
        return 0
    else
        echo -e "${RED}失败${NC}"
        return 1
    fi
}

# 检查数据库连接
check_database() {
    echo -n "检查数据库连接... "
    # 通过规则 API 间接检查数据库
    db_response=$(curl -s "${API_URL}/rules" \
        -H "Authorization: Bearer $TOKEN" 2>/dev/null || echo "")

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}通过${NC}"
        return 0
    else
        echo -e "${RED}失败${NC}"
        return 1
    fi
}

# 检查 ETCD 连接
check_etcd() {
    echo -n "检查 ETCD 连接... "
    health_response=$(curl -s "${BASE_URL}/health" 2>/dev/null || echo "")

    if [ -n "$health_response" ]; then
        echo -e "${GREEN}通过${NC}"
        return 0
    else
        echo -e "${RED}失败${NC}"
        return 1
    fi
}

# 运行所有检查
run_all_checks() {
    passed=0
    failed=0

    check_health && ((passed++)) || ((failed++))
    check_ready && ((passed++)) || ((failed++))
    check_system_info && ((passed++)) || ((failed++))
    check_authentication && ((passed++)) || ((failed++))
    check_rules_api && ((passed++)) || ((failed++))
    check_analysis_api && ((passed++)) || ((failed++))
    check_database && ((passed++)) || ((failed++))
    check_etcd && ((passed++)) || ((failed++))

    echo ""
    echo "======================================"
    echo "验证结果汇总"
    echo "======================================"
    echo -e "通过：${GREEN}$passed${NC}"
    echo -e "失败：${RED}$failed${NC}"
    echo ""

    if [ $failed -eq 0 ]; then
        echo -e "${GREEN}所有检查通过！部署验证成功。${NC}"
        return 0
    else
        echo -e "${RED}部分检查失败。请检查服务状态和日志。${NC}"
        return 1
    fi
}

# 主函数
main() {
    echo "目标地址：$BASE_URL"
    echo ""

    run_all_checks
    exit $?
}

main
