// Package detector 提供非结构化日志识别功能
package detector

import (
	"regexp"
	"strings"
	"unicode"
)

// UnstructuredDetector 非结构化日志检测器
type UnstructuredDetector struct {
	config *UnstructuredConfig
}

// UnstructuredConfig 非结构化检测配置
type UnstructuredConfig struct {
	MinTextRatio     float64   // 最小文本比例
	MinWordCount     int       // 最小单词数
	MaxPatternScore  float64   // 最大模式匹配分数
	CommonPatterns   []*regexp.Regexp
	LanguageHints    []string  // 语言提示
}

// DefaultUnstructuredConfig 返回默认配置
func DefaultUnstructuredConfig() *UnstructuredConfig {
	return &UnstructuredConfig{
		MinTextRatio:    0.7,
		MinWordCount:    5,
		MaxPatternScore: 0.3,
		CommonPatterns: []*regexp.Regexp{
			regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`), // IP
			regexp.MustCompile(`\d{2}:\d{2}:\d{2}`),                   // 时间
			regexp.MustCompile(`\[.*?\]`),                             // 括号内容
		},
		LanguageHints: []string{"en", "zh"},
	}
}

// NewUnstructuredDetector 创建非结构化日志检测器
func NewUnstructuredDetector(config *UnstructuredConfig) *UnstructuredDetector {
	if config == nil {
		config = DefaultUnstructuredConfig()
	}
	return &UnstructuredDetector{
		config: config,
	}
}

// Detect 检测是否为非结构化日志
func (d *UnstructuredDetector) Detect(log []byte) *DetectionResult {
	content := string(log)

	// 快速检查：如果是明显的结构化格式，直接返回 nil
	trimmed := strings.TrimSpace(content)
	if len(trimmed) > 0 {
		// JSON 格式检查
		if (trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}') ||
			(trimmed[0] == '[' && trimmed[len(trimmed)-1] == ']') {
			return nil
		}

		// XML 格式检查
		if strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">") {
			return nil
		}
	}

	// 计算文本特征
	features := d.extractFeatures(content)

	// 计算非结构化分数
	score := d.calculateScore(features)

	if score < 0.5 {
		return nil
	}

	return &DetectionResult{
		Format:     FormatUnstructured,
		Confidence: score,
		Metadata: map[string]interface{}{
			"text_ratio":    features.textRatio,
			"word_count":    features.wordCount,
			"pattern_score": features.patternScore,
			"has_timestamp": features.hasTimestamp,
			"has_level":     features.hasLevel,
		},
	}
}

// textFeatures 文本特征
type textFeatures struct {
	textRatio       float64
	wordCount       int
	patternScore    float64
	hasTimestamp    bool
	hasLevel        bool
	averageWordLen  float64
	specialCharRatio float64
}

// extractFeatures 提取文本特征
func (d *UnstructuredDetector) extractFeatures(content string) *textFeatures {
	features := &textFeatures{}

	// 计算文本比例（字母和汉字占总字符的比例）
	totalChars := len(content)
	if totalChars == 0 {
		return features
	}

	textChars := 0
	for _, r := range content {
		if unicode.IsLetter(r) || unicode.Is(unicode.Han, r) || unicode.IsSpace(r) {
			textChars++
		}
	}
	features.textRatio = float64(textChars) / float64(totalChars)

	// 计算单词数
	words := strings.Fields(content)
	features.wordCount = len(words)

	// 计算平均单词长度
	if len(words) > 0 {
		totalLen := 0
		for _, w := range words {
			totalLen += len(w)
		}
		features.averageWordLen = float64(totalLen) / float64(len(words))
	}

	// 计算模式匹配分数
	features.patternScore = d.calculatePatternScore(content)

	// 检查是否包含时间戳
	features.hasTimestamp = d.containsTimestamp(content)

	// 检查是否包含日志级别
	features.hasLevel = d.containsLogLevel(content)

	// 计算特殊字符比例
	specialChars := 0
	for _, r := range content {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r) && !unicode.Is(unicode.Han, r) {
			specialChars++
		}
	}
	features.specialCharRatio = float64(specialChars) / float64(totalChars)

	return features
}

// calculatePatternScore 计算模式匹配分数
func (d *UnstructuredDetector) calculatePatternScore(content string) float64 {
	score := 0.0
	matches := 0

	for _, pattern := range d.config.CommonPatterns {
		if pattern.MatchString(content) {
			matches++
		}
	}

	if len(d.config.CommonPatterns) > 0 {
		score = float64(matches) / float64(len(d.config.CommonPatterns))
	}

	return score
}

// containsTimestamp 检查是否包含时间戳
func (d *UnstructuredDetector) containsTimestamp(content string) bool {
	timestampPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\d{4}-\d{2}-\d{2}`),
		regexp.MustCompile(`\d{2}/\d{2}/\d{4}`),
		regexp.MustCompile(`\d{2}:\d{2}:\d{2}`),
		regexp.MustCompile(`\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}`),
	}

	for _, pattern := range timestampPatterns {
		if pattern.MatchString(content) {
			return true
		}
	}

	return false
}

// containsLogLevel 检查是否包含日志级别
func (d *UnstructuredDetector) containsLogLevel(content string) bool {
	levelPatterns := []string{
		"DEBUG", "INFO", "WARN", "WARNING", "ERROR", "FATAL",
		"TRACE", "CRITICAL", "SUCCESS",
	}

	upperContent := strings.ToUpper(content)
	for _, level := range levelPatterns {
		if strings.Contains(upperContent, level) {
			return true
		}
	}

	return false
}

// calculateScore 计算非结构化分数
func (d *UnstructuredDetector) calculateScore(features *textFeatures) float64 {
	score := 0.0

	// 文本比例高 -> 更可能是非结构化
	if features.textRatio >= d.config.MinTextRatio {
		score += 0.3
	}

	// 单词数多 -> 更可能是非结构化
	if features.wordCount >= d.config.MinWordCount {
		score += 0.2
	}

	// 模式匹配少 -> 更可能是非结构化
	if features.patternScore <= d.config.MaxPatternScore {
		score += 0.2
	}

	// 没有明显的时间戳 -> 更可能是非结构化
	if !features.hasTimestamp {
		score += 0.15
	}

	// 没有明显的日志级别 -> 更可能是非结构化
	if !features.hasLevel {
		score += 0.15
	}

	return score
}

// AnalyzeContent 分析非结构化日志内容
func (d *UnstructuredDetector) AnalyzeContent(content string) *ContentAnalysis {
	analysis := &ContentAnalysis{
		OriginalText: content,
	}

	// 提取潜在的关键信息
	analysis.Entities = d.extractEntities(content)
	analysis.KeyPhrases = d.extractKeyPhrases(content)
	analysis.Sentences = d.splitSentences(content)

	return analysis
}

// ContentAnalysis 内容分析结果
type ContentAnalysis struct {
	OriginalText string
	Entities     []Entity
	KeyPhrases   []string
	Sentences    []string
}

// Entity 实体
type Entity struct {
	Type  string
	Value string
	Start int
	End   int
}

// extractEntities 提取实体
func (d *UnstructuredDetector) extractEntities(content string) []Entity {
	var entities []Entity

	// 提取 IP 地址
	ipPattern := regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	for _, match := range ipPattern.FindAllStringIndex(content, -1) {
		entities = append(entities, Entity{
			Type:  "IP_ADDRESS",
			Value: content[match[0]:match[1]],
			Start: match[0],
			End:   match[1],
		})
	}

	// 提取 URL
	urlPattern := regexp.MustCompile(`https?://[^\s<>"{}|\^\[\]]+`)
	for _, match := range urlPattern.FindAllStringIndex(content, -1) {
		entities = append(entities, Entity{
			Type:  "URL",
			Value: content[match[0]:match[1]],
			Start: match[0],
			End:   match[1],
		})
	}

	// 提取邮箱
	emailPattern := regexp.MustCompile(`[\w\-.]+@[\w\-.]+\.\w+`)
	for _, match := range emailPattern.FindAllStringIndex(content, -1) {
		entities = append(entities, Entity{
			Type:  "EMAIL",
			Value: content[match[0]:match[1]],
			Start: match[0],
			End:   match[1],
		})
	}

	return entities
}

// extractKeyPhrases 提取关键词
func (d *UnstructuredDetector) extractKeyPhrases(content string) []string {
	// 简单的关键词提取：按空格分割，过滤常见停用词
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "and": true, "or": true,
		"is": true, "are": true, "was": true, "were": true,
		"in": true, "on": true, "at": true, "to": true, "for": true,
		"of": true, "with": true, "by": true, "from": true,
	}

	words := strings.Fields(strings.ToLower(content))
	phrases := make([]string, 0)

	for _, word := range words {
		// 过滤标点符号
		word = strings.Trim(word, ".,!?;:()[]{}\"'`")
		if len(word) > 2 && !stopWords[word] {
			phrases = append(phrases, word)
		}
	}

	return phrases
}

// splitSentences 分割句子
func (d *UnstructuredDetector) splitSentences(content string) []string {
	// 简单的句子分割
	sentencePattern := regexp.MustCompile(`[.!?]+\s*`)
	sentences := sentencePattern.Split(content, -1)

	result := make([]string, 0)
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			result = append(result, s)
		}
	}

	return result
}
