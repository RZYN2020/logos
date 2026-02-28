// Package tests 用户场景验收测试
package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 测试配置
const (
	BaseURL      = "http://localhost:8080"
	APIBaseURL   = BaseURL + "/api/v1"
	TestUsername = "testuser"
	TestPassword = "testpass123"
)

// TestScenario_UserRegistration 用户注册场景
func TestScenario_UserRegistration(t *testing.T) {
	// 1. 注册新用户
	registerReq := map[string]string{
		"username": TestUsername,
		"password": TestPassword,
		"email":    "testuser@example.com",
	}

	resp, err := postJSON(fmt.Sprintf("%s/auth/register", APIBaseURL), registerReq)
	if err != nil {
		t.Skipf("跳过测试：服务不可用 (%v)", err)
		return
	}

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	t.Log("用户注册成功")

	defer resp.Body.Close()
}

// TestScenario_UserLogin 用户登录场景
func TestScenario_UserLogin(t *testing.T) {
	// 1. 使用默认管理员登录
	loginReq := map[string]string{
		"username": "admin",
		"password": "admin123",
	}

	resp, err := postJSON(fmt.Sprintf("%s/auth/login", APIBaseURL), loginReq)
	if err != nil {
		t.Skipf("跳过测试：服务不可用 (%v)", err)
		return
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.NotNil(t, result["token"])
	t.Log("管理员登录成功")
}

// TestScenario_RuleLifecycle 规则完整生命周期场景
func TestScenario_RuleLifecycle(t *testing.T) {
	// 1. 登录获取 token
	token := loginAndGetToken(t)
	if token == "" {
		return
	}

	headers := map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	}

	// 2. 创建规则
	t.Log("步骤 1: 创建规则")
	createReq := map[string]interface{}{
		"name":        "scenario-test-rule",
		"description": "规则生命周期测试",
		"enabled":     true,
		"priority":    1,
		"conditions": []map[string]interface{}{
			{"field": "level", "operator": "=", "value": "ERROR"},
		},
		"actions": []map[string]interface{}{
			{"type": "filter", "config": map[string]interface{}{"sampling": 1.0}},
		},
	}

	resp, err := postJSONWithHeaders(fmt.Sprintf("%s/rules", APIBaseURL), createReq, headers)
	if err != nil {
		t.Fatalf("创建规则失败：%v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResult map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&createResult)
	ruleID := createResult["id"].(string)
	assert.NotEmpty(t, ruleID)
	t.Logf("规则创建成功，ID: %s", ruleID)

	// 3. 获取规则列表
	t.Log("步骤 2: 获取规则列表")
	resp, err = getWithHeaders(fmt.Sprintf("%s/rules", APIBaseURL), headers)
	if err != nil {
		t.Fatalf("获取规则列表失败：%v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	t.Log("规则列表获取成功")

	// 4. 获取规则详情
	t.Log("步骤 3: 获取规则详情")
	resp, err = getWithHeaders(fmt.Sprintf("%s/rules/%s", APIBaseURL, ruleID), headers)
	if err != nil {
		t.Fatalf("获取规则详情失败：%v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	t.Log("规则详情获取成功")

	// 5. 验证规则
	t.Log("步骤 4: 验证规则")
	resp, err = postJSONWithHeaders(fmt.Sprintf("%s/rules/%s/validate", APIBaseURL, ruleID), nil, headers)
	if err != nil {
		t.Fatalf("验证规则失败：%v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	t.Log("规则验证成功")

	// 6. 测试规则
	t.Log("步骤 5: 测试规则")
	resp, err = postJSONWithHeaders(fmt.Sprintf("%s/rules/%s/test", APIBaseURL, ruleID), nil, headers)
	if err != nil {
		t.Fatalf("测试规则失败：%v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	t.Log("规则测试成功")

	// 7. 更新规则
	t.Log("步骤 6: 更新规则")
	updateReq := map[string]interface{}{
		"name":        "updated-scenario-rule",
		"description": "规则已更新",
		"enabled":     false,
		"priority":    2,
		"conditions":  []interface{}{},
		"actions":     []interface{}{},
	}

	resp, err = putJSONWithHeaders(fmt.Sprintf("%s/rules/%s", APIBaseURL, ruleID), updateReq, headers)
	if err != nil {
		t.Fatalf("更新规则失败：%v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	t.Log("规则更新成功")

	// 8. 获取规则历史
	t.Log("步骤 7: 获取规则历史")
	resp, err = getWithHeaders(fmt.Sprintf("%s/rules/%s/history", APIBaseURL, ruleID), headers)
	if err != nil {
		t.Fatalf("获取规则历史失败：%v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	t.Log("规则历史获取成功")

	// 9. 删除规则
	t.Log("步骤 8: 删除规则")
	resp, err = deleteWithHeaders(fmt.Sprintf("%s/rules/%s", APIBaseURL, ruleID), headers)
	if err != nil {
		t.Fatalf("删除规则失败：%v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	t.Log("规则删除成功")
}

// TestScenario_LogAnalysis 日志分析场景
func TestScenario_LogAnalysis(t *testing.T) {
	token := loginAndGetToken(t)
	if token == "" {
		return
	}

	headers := map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	}

	// 1. 日志模式挖掘
	t.Log("步骤 1: 日志模式挖掘")
	mineReq := map[string]interface{}{
		"logs": []map[string]interface{}{
			{"timestamp": time.Now().Format(time.RFC3339), "level": "ERROR", "service": "api", "message": "Connection failed"},
			{"timestamp": time.Now().Format(time.RFC3339), "level": "ERROR", "service": "api", "message": "Connection failed"},
			{"timestamp": time.Now().Format(time.RFC3339), "level": "INFO", "service": "api", "message": "Request processed"},
		},
	}

	resp, err := postJSONWithHeaders(fmt.Sprintf("%s/analysis/mine", APIBaseURL), mineReq, headers)
	if err != nil {
		t.Fatalf("模式挖掘失败：%v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	t.Log("日志模式挖掘成功")

	// 2. 智能规则推荐
	t.Log("步骤 2: 智能规则推荐")
	recommendReq := map[string]interface{}{
		"logs": []map[string]interface{}{
			{"timestamp": time.Now().Format(time.RFC3339), "level": "ERROR", "service": "payment", "message": "Payment failed"},
			{"timestamp": time.Now().Format(time.RFC3339), "level": "ERROR", "service": "payment", "message": "Payment failed"},
			{"timestamp": time.Now().Format(time.RFC3339), "level": "ERROR", "service": "payment", "message": "Payment failed"},
		},
		"min_frequency": 1,
	}

	resp, err = postJSONWithHeaders(fmt.Sprintf("%s/analysis/recommend", APIBaseURL), recommendReq, headers)
	if err != nil {
		t.Fatalf("规则推荐失败：%v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	t.Log("智能规则推荐成功")
}

// TestScenario_ImportExport 规则导入导出场景
func TestScenario_ImportExport(t *testing.T) {
	token := loginAndGetToken(t)
	if token == "" {
		return
	}

	headers := map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	}

	// 1. 导出规则
	t.Log("步骤 1: 导出规则")
	resp, err := getWithHeaders(fmt.Sprintf("%s/rules/export", APIBaseURL), headers)
	if err != nil {
		t.Fatalf("导出规则失败：%v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var exportResult map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&exportResult)
	t.Logf("规则导出成功，共 %v 条规则", len(exportResult["rules"].([]interface{})))

	// 2. 导入规则
	t.Log("步骤 2: 导入规则")
	importReq := map[string]interface{}{
		"rules": []map[string]interface{}{
			{
				"name":        "imported-rule-1",
				"description": "Imported test rule 1",
				"enabled":     true,
				"priority":    1,
				"conditions":  []interface{}{},
				"actions":     []interface{}{},
			},
		},
	}

	resp, err = postJSONWithHeaders(fmt.Sprintf("%s/rules/import", APIBaseURL), importReq, headers)
	if err != nil {
		t.Fatalf("导入规则失败：%v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	t.Log("规则导入成功")
}

// 辅助函数

func loginAndGetToken(t *testing.T) string {
	loginReq := map[string]string{
		"username": "admin",
		"password": "admin123",
	}

	resp, err := postJSON(fmt.Sprintf("%s/auth/login", APIBaseURL), loginReq)
	if err != nil {
		t.Skipf("跳过测试：服务不可用 (%v)", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skip("跳过测试：登录失败")
		return ""
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	token, ok := result["token"].(string)
	if !ok {
		t.Fatal("无法获取 token")
	}

	return token
}

func postJSON(url string, data interface{}) (*http.Response, error) {
	return postJSONWithHeaders(url, data, nil)
}

func postJSONWithHeaders(url string, data interface{}, headers map[string]string) (*http.Response, error) {
	body, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}

func putJSONWithHeaders(url string, data interface{}, headers map[string]string) (*http.Response, error) {
	body, _ := json.Marshal(data)
	req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}

func getWithHeaders(url string, headers map[string]string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", url, nil)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}

func deleteWithHeaders(url string, headers map[string]string) (*http.Response, error) {
	req, _ := http.NewRequest("DELETE", url, nil)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}
