// Package analyzer 提供文本分析功能
package analyzer

import (
	"strings"
	"sync"
	"time"
)

// TextAnalyzer 文本分析器接口
type TextAnalyzer interface {
	Analyze(text string) (*AnalysisResult, error)
	RegisterExtractor(name string, extractor EntityExtractor)
	RegisterAnalyzer(name string, analyzer TextFeatureAnalyzer)
}

// AnalysisResult 分析结果
type AnalysisResult struct {
	Text        string                 `json:"text"`
	Entities    []Entity               `json:"entities,omitempty"`
	Keywords    []string               `json:"keywords,omitempty"`
	KeyPhrases  []string               `json:"key_phrases,omitempty"`
	Sentiment   SentimentResult        `json:"sentiment,omitempty"`
	Language    string                 `json:"language,omitempty"`
	Category    string                 `json:"category,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	AnalyzedAt  time.Time              `json:"analyzed_at"`
}

// Entity 实体
type Entity struct {
	Type     string  `json:"type"`
	Value    string  `json:"value"`
	Confidence float64 `json:"confidence,omitempty"`
	Start    int     `json:"start"`
	End      int     `json:"end"`
}

// SentimentResult 情感分析结果
type SentimentResult struct {
	Score   float64 `json:"score"`   // -1.0 到 1.0，负数表示负面，正数表示正面
	Label   string  `json:"label"`   // positive, negative, neutral
	Mixed   bool    `json:"mixed"`   // 是否混合情感
}

// EntityExtractor 实体提取器接口
type EntityExtractor interface {
	Extract(text string) []Entity
}

// TextFeatureAnalyzer 文本特征分析器接口
type TextFeatureAnalyzer interface {
	Analyze(text string) map[string]interface{}
}

// TextAnalyzerImpl 文本分析器实现
type TextAnalyzerImpl struct {
	mu              sync.RWMutex
	extractors      map[string]EntityExtractor
	featureAnalyzers map[string]TextFeatureAnalyzer
	languageDetector *LanguageDetector
	sentimentAnalyzer *SentimentAnalyzer
}

// NewTextAnalyzer 创建文本分析器
func NewTextAnalyzer() *TextAnalyzerImpl {
	a := &TextAnalyzerImpl{
		extractors:      make(map[string]EntityExtractor),
		featureAnalyzers: make(map[string]TextFeatureAnalyzer),
		languageDetector: NewLanguageDetector(),
		sentimentAnalyzer: NewSentimentAnalyzer(),
	}

	// 注册默认提取器
	a.registerDefaultExtractors()

	return a
}

// registerDefaultExtractors 注册默认提取器
func (a *TextAnalyzerImpl) registerDefaultExtractors() {
	a.RegisterExtractor("ip", &IPAddressExtractor{})
	a.RegisterExtractor("url", &URLEXtractor{})
	a.RegisterExtractor("email", &EmailExtractor{})
	a.RegisterExtractor("timestamp", &TimestampExtractor{})
	a.RegisterExtractor("error", &ErrorPatternExtractor{})
}

// RegisterExtractor 注册实体提取器
func (a *TextAnalyzerImpl) RegisterExtractor(name string, extractor EntityExtractor) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.extractors[name] = extractor
}

// RegisterAnalyzer 注册文本特征分析器
func (a *TextAnalyzerImpl) RegisterAnalyzer(name string, analyzer TextFeatureAnalyzer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.featureAnalyzers[name] = analyzer
}

// Analyze 分析文本
func (a *TextAnalyzerImpl) Analyze(text string) (*AnalysisResult, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := &AnalysisResult{
		Text:       text,
		Entities:   make([]Entity, 0),
		Keywords:   make([]string, 0),
		KeyPhrases: make([]string, 0),
		Metadata:   make(map[string]interface{}),
		AnalyzedAt: time.Now(),
	}

	// 提取实体
	for _, extractor := range a.extractors {
		entities := extractor.Extract(text)
		result.Entities = append(result.Entities, entities...)
	}

	// 检测语言
	result.Language = a.languageDetector.Detect(text)

	// 情感分析
	result.Sentiment = a.sentimentAnalyzer.Analyze(text)

	// 提取关键词
	result.Keywords = extractKeywords(text)
	result.KeyPhrases = extractKeyPhrases(text)

	// 应用特征分析器
	for _, analyzer := range a.featureAnalyzers {
		features := analyzer.Analyze(text)
		for k, v := range features {
			result.Metadata[k] = v
		}
	}

	// 推断类别
	result.Category = inferCategory(text, result)

	return result, nil
}

// extractKeywords 提取关键词
func extractKeywords(text string) []string {
	// 简单的关键词提取
	stopWords := map[string]string{
		"a": "", "an": "", "the": "", "and": "", "or": "",
		"is": "", "are": "", "was": "", "were": "",
		"in": "", "on": "", "at": "", "to": "", "for": "",
		"of": "", "with": "", "by": "", "from": "",
		"this": "", "that": "", "these": "", "those": "",
		"我": "", "的": "", "了": "", "是": "", "在": "", "就": "", "都": "",
	}

	words := strings.Fields(strings.ToLower(text))
	keywords := make([]string, 0)

	for _, word := range words {
		// 过滤标点符号
		word = strings.Trim(word, ".,!?;:()[]{}\"'`")
		if len(word) > 2 && stopWords[word] == "" {
			keywords = append(keywords, word)
		}
	}

	// 去重
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, kw := range keywords {
		if !seen[kw] {
			seen[kw] = true
			result = append(result, kw)
		}
	}

	// 只返回前 10 个关键词
	if len(result) > 10 {
		result = result[:10]
	}

	return result
}

// extractKeyPhrases 提取关键短语
func extractKeyPhrases(text string) []string {
	// 简单的句子提取
	sentences := splitSentences(text)
	phrases := make([]string, 0)

	for _, sentence := range sentences {
		if len(sentence) > 10 && len(sentence) < 200 {
			phrases = append(phrases, sentence)
		}
	}

	// 只返回前 5 个短语
	if len(phrases) > 5 {
		phrases = phrases[:5]
	}

	return phrases
}

// splitSentences 分割句子
func splitSentences(text string) []string {
	separators := []string{". ", "! ", "? ", ".\n", "!\n", "?\n"}
	sentences := []string{text}

	for _, sep := range separators {
		newSentences := make([]string, 0)
		for _, s := range sentences {
			parts := strings.Split(s, sep)
			newSentences = append(newSentences, parts...)
		}
		sentences = newSentences
	}

	result := make([]string, 0)
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			result = append(result, s)
		}
	}

	return result
}

// inferCategory 推断文本类别
func inferCategory(text string, result *AnalysisResult) string {
	textLower := strings.ToLower(text)

	// 基于关键词的简单分类
	if strings.Contains(textLower, "error") || strings.Contains(textLower, "exception") ||
		strings.Contains(textLower, "failed") || strings.Contains(textLower, "failure") {
		return "error"
	}

	if strings.Contains(textLower, "http") || strings.Contains(textLower, "request") ||
		strings.Contains(textLower, "response") || strings.Contains(textLower, "api") {
		return "http"
	}

	if strings.Contains(textLower, "database") || strings.Contains(textLower, "sql") ||
		strings.Contains(textLower, "query") {
		return "database"
	}

	if strings.Contains(textLower, "auth") || strings.Contains(textLower, "login") ||
		strings.Contains(textLower, "token") || strings.Contains(textLower, "permission") {
		return "security"
	}

	return "general"
}
