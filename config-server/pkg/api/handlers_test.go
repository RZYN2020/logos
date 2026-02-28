// Package api 提供 Config Server API 测试
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return New()
}

// ============== 策略管理测试 ==============

func TestListStrategies(t *testing.T) {
	router := setupRouter()

	req, _ := http.NewRequest("GET", "/api/v1/strategies", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("Expected code 0, got %d", resp.Code)
	}

	data, ok := resp.Data.([]interface{})
	if !ok {
		t.Error("Expected data to be array")
	}

	if len(data) == 0 {
		t.Error("Expected at least one strategy")
	}
}

func TestCreateStrategy(t *testing.T) {
	router := setupRouter()

	strategy := Strategy{
		Name:        "test-strategy",
		Description: "Test Strategy",
		Rules: []StrategyRule{
			{
				Condition: map[string]interface{}{"level": "ERROR"},
				Action:    map[string]interface{}{"enabled": true},
			},
		},
	}

	body, _ := json.Marshal(strategy)
	req, _ := http.NewRequest("POST", "/api/v1/strategies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("Expected code 0, got %d: %s", resp.Code, resp.Message)
	}
}

func TestGetStrategy(t *testing.T) {
	router := setupRouter()

	// First create a strategy
	strategy := Strategy{
		Name:        "test-get-strategy",
		Description: "Test Get Strategy",
		Rules: []StrategyRule{
			{
				Condition: map[string]interface{}{"level": "ERROR"},
				Action:    map[string]interface{}{"enabled": true},
			},
		},
	}

	body, _ := json.Marshal(strategy)
	req, _ := http.NewRequest("POST", "/api/v1/strategies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResp Response
	json.Unmarshal(w.Body.Bytes(), &createResp)
	data := createResp.Data.(map[string]interface{})
	id := data["id"].(string)

	// Then get it
	req, _ = http.NewRequest("GET", "/api/v1/strategies/"+id, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("Expected code 0, got %d", resp.Code)
	}
}

func TestUpdateStrategy(t *testing.T) {
	router := setupRouter()

	// First create a strategy
	strategy := Strategy{
		Name:        "test-update-strategy",
		Description: "Test Update Strategy",
		Rules: []StrategyRule{
			{
				Condition: map[string]interface{}{"level": "ERROR"},
				Action:    map[string]interface{}{"enabled": true},
			},
		},
	}

	body, _ := json.Marshal(strategy)
	req, _ := http.NewRequest("POST", "/api/v1/strategies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResp Response
	json.Unmarshal(w.Body.Bytes(), &createResp)
	data := createResp.Data.(map[string]interface{})
	id := data["id"].(string)

	// Update it
	strategy.Description = "Updated Description"
	body, _ = json.Marshal(strategy)
	req, _ = http.NewRequest("PUT", "/api/v1/strategies/"+id, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteStrategy(t *testing.T) {
	router := setupRouter()

	// First create a strategy
	strategy := Strategy{
		Name:        "test-delete-strategy",
		Description: "Test Delete Strategy",
		Rules: []StrategyRule{
			{
				Condition: map[string]interface{}{"level": "ERROR"},
				Action:    map[string]interface{}{"enabled": true},
			},
		},
	}

	body, _ := json.Marshal(strategy)
	req, _ := http.NewRequest("POST", "/api/v1/strategies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResp Response
	json.Unmarshal(w.Body.Bytes(), &createResp)
	data := createResp.Data.(map[string]interface{})
	id := data["id"].(string)

	// Delete it
	req, _ = http.NewRequest("DELETE", "/api/v1/strategies/"+id, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ============== 解析器配置测试 ==============

func TestListParserConfigs(t *testing.T) {
	router := setupRouter()

	req, _ := http.NewRequest("GET", "/api/v1/parsers", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("Expected code 0, got %d", resp.Code)
	}

	data, ok := resp.Data.([]interface{})
	if !ok {
		t.Error("Expected data to be array")
	}

	if len(data) == 0 {
		t.Error("Expected at least one parser config")
	}
}

func TestCreateParserConfig(t *testing.T) {
	router := setupRouter()

	config := ParserConfig{
		Name:     "test-parser",
		Type:     "json",
		Enabled:  true,
		Priority: 50,
		Config:   map[string]interface{}{"strict": true},
	}

	body, _ := json.Marshal(config)
	req, _ := http.NewRequest("POST", "/api/v1/parsers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("Expected code 0, got %d: %s", resp.Code, resp.Message)
	}
}

func TestCreateParserConfigInvalidType(t *testing.T) {
	router := setupRouter()

	config := ParserConfig{
		Name:     "test-parser",
		Type:     "invalid_type",
		Enabled:  true,
		Priority: 50,
	}

	body, _ := json.Marshal(config)
	req, _ := http.NewRequest("POST", "/api/v1/parsers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestGetParserConfig(t *testing.T) {
	router := setupRouter()

	// Get existing parser config
	req, _ := http.NewRequest("GET", "/api/v1/parsers/parser-json", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("Expected code 0, got %d", resp.Code)
	}
}

func TestUpdateParserConfig(t *testing.T) {
	router := setupRouter()

	config := ParserConfig{
		Name:     "test-update-parser",
		Type:     "json",
		Enabled:  true,
		Priority: 50,
	}

	body, _ := json.Marshal(config)
	req, _ := http.NewRequest("POST", "/api/v1/parsers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResp Response
	json.Unmarshal(w.Body.Bytes(), &createResp)
	data := createResp.Data.(map[string]interface{})
	id := data["id"].(string)

	// Update it
	config.Priority = 100
	body, _ = json.Marshal(config)
	req, _ = http.NewRequest("PUT", "/api/v1/parsers/"+id, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteParserConfig(t *testing.T) {
	router := setupRouter()

	config := ParserConfig{
		Name:     "test-delete-parser",
		Type:     "json",
		Enabled:  true,
		Priority: 50,
	}

	body, _ := json.Marshal(config)
	req, _ := http.NewRequest("POST", "/api/v1/parsers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResp Response
	json.Unmarshal(w.Body.Bytes(), &createResp)
	data := createResp.Data.(map[string]interface{})
	id := data["id"].(string)

	// Delete it
	req, _ = http.NewRequest("DELETE", "/api/v1/parsers/"+id, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ============== 转换规则测试 ==============

func TestListTransformRules(t *testing.T) {
	router := setupRouter()

	req, _ := http.NewRequest("GET", "/api/v1/transforms", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("Expected code 0, got %d", resp.Code)
	}

	data, ok := resp.Data.([]interface{})
	if !ok {
		t.Error("Expected data to be array")
	}

	if len(data) == 0 {
		t.Error("Expected at least one transform rule")
	}
}

func TestCreateTransformRule(t *testing.T) {
	router := setupRouter()

	config := TransformRuleConfig{
		Name:        "test-transform",
		Description: "Test Transform Rule",
		Enabled:     true,
		Priority:    50,
		Rules: []TransformRule{
			{
				ID:          "rule-001",
				SourceField: "message",
				TargetField: "method",
				Extractor:   "regex",
				Config:      map[string]interface{}{"pattern": "(GET|POST)"},
				OnError:     "skip",
			},
		},
	}

	body, _ := json.Marshal(config)
	req, _ := http.NewRequest("POST", "/api/v1/transforms", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("Expected code 0, got %d: %s", resp.Code, resp.Message)
	}
}

func TestCreateTransformRuleInvalidExtractor(t *testing.T) {
	router := setupRouter()

	config := TransformRuleConfig{
		Name:        "test-transform",
		Description: "Test Transform Rule",
		Enabled:     true,
		Rules: []TransformRule{
			{
				ID:          "rule-001",
				SourceField: "message",
				TargetField: "method",
				Extractor:   "invalid_extractor",
			},
		},
	}

	body, _ := json.Marshal(config)
	req, _ := http.NewRequest("POST", "/api/v1/transforms", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestCreateTransformRuleInvalidRegex(t *testing.T) {
	router := setupRouter()

	config := TransformRuleConfig{
		Name:        "test-transform",
		Description: "Test Transform Rule",
		Enabled:     true,
		Rules: []TransformRule{
			{
				ID:          "rule-001",
				SourceField: "message",
				TargetField: "method",
				Extractor:   "regex",
				Config:      map[string]interface{}{"pattern": "[invalid"},
			},
		},
	}

	body, _ := json.Marshal(config)
	req, _ := http.NewRequest("POST", "/api/v1/transforms", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// ============== 过滤器配置测试 ==============

func TestListFilterConfigs(t *testing.T) {
	router := setupRouter()

	req, _ := http.NewRequest("GET", "/api/v1/filters", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("Expected code 0, got %d", resp.Code)
	}

	data, ok := resp.Data.([]interface{})
	if !ok {
		t.Error("Expected data to be array")
	}

	if len(data) == 0 {
		t.Error("Expected at least one filter config")
	}
}

func TestCreateFilterConfig(t *testing.T) {
	router := setupRouter()

	config := FilterConfig{
		Name:        "test-filter",
		Description: "Test Filter Config",
		Enabled:     true,
		Priority:    50,
		Rules: []FilterRule{
			{
				ID:      "rule-001",
				Name:    "drop-debug",
				Field:   "level",
				Pattern: "^DEBUG$",
				Action:  ActionDrop,
			},
		},
	}

	body, _ := json.Marshal(config)
	req, _ := http.NewRequest("POST", "/api/v1/filters", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("Expected code 0, got %d: %s", resp.Code, resp.Message)
	}
}

func TestCreateFilterConfigInvalidPattern(t *testing.T) {
	router := setupRouter()

	config := FilterConfig{
		Name:        "test-filter",
		Description: "Test Filter Config",
		Enabled:     true,
		Rules: []FilterRule{
			{
				ID:      "rule-001",
				Name:    "invalid-pattern",
				Field:   "level",
				Pattern: "[invalid",
				Action:  ActionDrop,
			},
		},
	}

	body, _ := json.Marshal(config)
	req, _ := http.NewRequest("POST", "/api/v1/filters", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// ============== 配置验证测试 ==============

func TestValidateParserConfig(t *testing.T) {
	router := setupRouter()

	config := ParserConfig{
		Name:     "valid-parser",
		Type:     "json",
		Enabled:  true,
		Priority: 50,
	}

	body, _ := json.Marshal(config)
	req, _ := http.NewRequest("POST", "/api/v1/validate/parser", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("Expected code 0, got %d", resp.Code)
	}
}

func TestValidateParserConfigInvalid(t *testing.T) {
	router := setupRouter()

	config := ParserConfig{
		Name:     "invalid-parser",
		Type:     "unknown",
		Enabled:  true,
		Priority: 50,
	}

	body, _ := json.Marshal(config)
	req, _ := http.NewRequest("POST", "/api/v1/validate/parser", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestValidateTransformRule(t *testing.T) {
	router := setupRouter()

	config := TransformRuleConfig{
		Name:        "valid-transform",
		Description: "Valid Transform",
		Enabled:     true,
		Rules: []TransformRule{
			{
				ID:          "rule-001",
				SourceField: "message",
				TargetField: "method",
				Extractor:   "regex",
				Config:      map[string]interface{}{"pattern": "(GET|POST)"},
			},
		},
	}

	body, _ := json.Marshal(config)
	req, _ := http.NewRequest("POST", "/api/v1/validate/transform", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestValidateFilterConfig(t *testing.T) {
	router := setupRouter()

	config := FilterConfig{
		Name:        "valid-filter",
		Description: "Valid Filter",
		Enabled:     true,
		Rules: []FilterRule{
			{
				ID:      "rule-001",
				Name:    "drop-debug",
				Field:   "level",
				Pattern: "^DEBUG$",
				Action:  ActionDrop,
			},
		},
	}

	body, _ := json.Marshal(config)
	req, _ := http.NewRequest("POST", "/api/v1/validate/filter", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ============== 系统信息测试 ==============

func TestHealthCheck(t *testing.T) {
	router := setupRouter()

	req, _ := http.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetSystemInfo(t *testing.T) {
	router := setupRouter()

	req, _ := http.NewRequest("GET", "/api/v1/info", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("Expected code 0, got %d", resp.Code)
	}
}

// ============== 版本工具函数测试 ==============

func TestIncrementVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"v1.0.0", "v1.1.0"},
		{"v1.1.0", "v1.2.0"},
		{"v2.5.3", "v2.6.3"}, // Function increments minor, preserves patch
		{"invalid", "v1.0.0"},
		{"", "v1.0.0"},
		{"v1", "v1.1.0"}, // Parses as v1.0.0, increments to v1.1.0
	}

	for _, tt := range tests {
		result := incrementVersion(tt.input)
		if result != tt.expected {
			t.Errorf("incrementVersion(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if len(id1) != 8 {
		t.Errorf("Expected ID length 8, got %d", len(id1))
	}

	if id1 == id2 {
		t.Error("Expected different IDs")
	}
}

// ============== 历史版本测试 ==============

func TestGetParserConfigHistory(t *testing.T) {
	router := setupRouter()

	// Create a parser config first
	config := ParserConfig{
		Name:     "test-history-parser",
		Type:     "json",
		Enabled:  true,
		Priority: 50,
	}

	body, _ := json.Marshal(config)
	req, _ := http.NewRequest("POST", "/api/v1/parsers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResp Response
	json.Unmarshal(w.Body.Bytes(), &createResp)
	data := createResp.Data.(map[string]interface{})
	id := data["id"].(string)

	// Get history
	req, _ = http.NewRequest("GET", "/api/v1/parsers/"+id+"/history", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetTransformRuleHistory(t *testing.T) {
	router := setupRouter()

	config := TransformRuleConfig{
		Name:        "test-history-transform",
		Description: "Test History",
		Enabled:     true,
		Rules: []TransformRule{
			{
				ID:          "rule-001",
				SourceField: "message",
				TargetField: "method",
				Extractor:   "regex",
				Config:      map[string]interface{}{"pattern": "(GET|POST)"},
			},
		},
	}

	body, _ := json.Marshal(config)
	req, _ := http.NewRequest("POST", "/api/v1/transforms", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResp Response
	json.Unmarshal(w.Body.Bytes(), &createResp)
	data := createResp.Data.(map[string]interface{})
	id := data["id"].(string)

	// Get history
	req, _ = http.NewRequest("GET", "/api/v1/transforms/"+id+"/history", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetFilterConfigHistory(t *testing.T) {
	router := setupRouter()

	config := FilterConfig{
		Name:        "test-history-filter",
		Description: "Test History",
		Enabled:     true,
		Rules: []FilterRule{
			{
				ID:      "rule-001",
				Name:    "drop-debug",
				Field:   "level",
				Pattern: "^DEBUG$",
				Action:  ActionDrop,
			},
		},
	}

	body, _ := json.Marshal(config)
	req, _ := http.NewRequest("POST", "/api/v1/filters", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResp Response
	json.Unmarshal(w.Body.Bytes(), &createResp)
	data := createResp.Data.(map[string]interface{})
	id := data["id"].(string)

	// Get history
	req, _ = http.NewRequest("GET", "/api/v1/filters/"+id+"/history", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// ============== 并发测试 ==============

func TestConcurrentParserConfigAccess(t *testing.T) {
	router := setupRouter()

	// Create initial config
	config := ParserConfig{
		Name:     "concurrent-parser",
		Type:     "json",
		Enabled:  true,
		Priority: 50,
	}

	body, _ := json.Marshal(config)
	req, _ := http.NewRequest("POST", "/api/v1/parsers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResp Response
	json.Unmarshal(w.Body.Bytes(), &createResp)
	data := createResp.Data.(map[string]interface{})
	id := data["id"].(string)

	done := make(chan bool, 10)

	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			req, _ := http.NewRequest("GET", "/api/v1/parsers/"+id, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			done <- w.Code == http.StatusOK
		}()
	}

	// Concurrent updates
	for i := 0; i < 5; i++ {
		go func(iter int) {
			config.Priority = 50 + iter
			body, _ := json.Marshal(config)
			req, _ := http.NewRequest("PUT", "/api/v1/parsers/"+id, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			done <- w.Code == http.StatusOK
		}(i)
	}

	successCount := 0
	for i := 0; i < 10; i++ {
		if <-done {
			successCount++
		}
	}

	if successCount != 10 {
		t.Errorf("Expected 10 successful concurrent operations, got %d", successCount)
	}
}
