// Package analyzer 提供情感分析和语言检测功能
package analyzer

import (
	"strings"
)

// SentimentAnalyzer 情感分析器
type SentimentAnalyzer struct {
	positiveWords map[string]float64
	negativeWords map[string]float64
	intensifiers  map[string]float64
	negators      map[string]bool
}

// NewSentimentAnalyzer 创建情感分析器
func NewSentimentAnalyzer() *SentimentAnalyzer {
	return &SentimentAnalyzer{
		positiveWords: map[string]float64{
			"good": 0.5, "great": 0.8, "excellent": 1.0, "amazing": 0.9,
			"success": 0.7, "successful": 0.8, "passed": 0.6, "complete": 0.5,
			"fast": 0.4, "quick": 0.4, "efficient": 0.5, "helpful": 0.5,
			"positive": 0.5, "happy": 0.7, "satisfied": 0.6, "perfect": 1.0,
			"好": 0.5, "很好": 0.7, "优秀": 0.9, "成功": 0.7, "完美": 1.0,
			"快": 0.4, "快速": 0.5, "高效": 0.6, "满意": 0.6,
		},
		negativeWords: map[string]float64{
			"bad": -0.5, "terrible": -0.9, "horrible": -1.0, "awful": -0.8,
			"error": -0.6, "failed": -0.7, "failure": -0.8, "exception": -0.5,
			"slow": -0.4, "crash": -0.8, "broken": -0.7, "wrong": -0.5,
			"negative": -0.5, "sad": -0.6, "angry": -0.7, "disappointed": -0.7,
			"错误": -0.6, "失败": -0.7, "异常": -0.5,
			"慢": -0.4, "崩溃": -0.8, "损坏": -0.7, "糟糕": -0.8,
		},
		intensifiers: map[string]float64{
			"very": 1.5, "extremely": 2.0, "really": 1.5, "incredibly": 2.0,
			"so": 1.3, "highly": 1.5, "completely": 1.8, "totally": 1.8,
			"非常": 1.5, "极其": 2.0, "真的": 1.5, "十分": 1.5,
		},
		negators: map[string]bool{
			"not": true, "no": true, "never": true, "neither": true,
			"nobody": true, "nothing": true, "nowhere": true,
			"不": true, "没": true, "没有": true, "从未": true,
		},
	}
}

// Analyze 分析文本情感
func (s *SentimentAnalyzer) Analyze(text string) SentimentResult {
	textLower := strings.ToLower(text)
	words := tokenize(textLower)

	score := 0.0
	negatorActive := false
	intensifier := 1.0

	for _, word := range words {
		// 检查否定词
		if s.negators[word] {
			negatorActive = true
			continue
		}

		// 检查程度词
		if intensifierVal, ok := s.intensifiers[word]; ok {
			intensifier = intensifierVal
			continue
		}

		// 检查积极词
		if polarity, ok := s.positiveWords[word]; ok {
			wordScore := polarity * intensifier
			if negatorActive {
				wordScore = -wordScore * 0.5 // 否定后减弱
			}
			score += wordScore
			negatorActive = false
			intensifier = 1.0
			continue
		}

		// 检查消极词
		if polarity, ok := s.negativeWords[word]; ok {
			wordScore := polarity * intensifier
			if negatorActive {
				wordScore = -wordScore * 0.5 // 否定后减弱
			}
			score += wordScore
			negatorActive = false
			intensifier = 1.0
			continue
		}

		// 重置状态
		if !isPunctuation(word) {
			negatorActive = false
			intensifier = 1.0
		}
	}

	// 归一化分数到 [-1, 1] 范围
	if score > 1.0 {
		score = 1.0
	} else if score < -1.0 {
		score = -1.0
	}

	// 确定情感标签
	var label string
	if score > 0.1 {
		label = "positive"
	} else if score < -0.1 {
		label = "negative"
	} else {
		label = "neutral"
	}

	// 检测混合情感
	hasPositive := false
	hasNegative := false
	for _, word := range words {
		if _, ok := s.positiveWords[word]; ok {
			hasPositive = true
		}
		if _, ok := s.negativeWords[word]; ok {
			hasNegative = true
		}
	}

	return SentimentResult{
		Score:  score,
		Label:  label,
		Mixed:  hasPositive && hasNegative,
	}
}

// tokenize 简单分词
func tokenize(text string) []string {
	// 简单实现：按空格和标点分割
	words := strings.Fields(text)
	result := make([]string, 0, len(words))

	for _, word := range words {
		// 清理标点符号
		cleaned := strings.Trim(word, ".,!?;:()[]{}\"'`-")
		if len(cleaned) > 0 {
			result = append(result, cleaned)
		}
	}

	return result
}

// isPunctuation 检查是否是标点
func isPunctuation(s string) bool {
	return s == "." || s == "," || s == "!" || s == "?" || s == ";" || s == ":"
}

// LanguageDetector 语言检测器
type LanguageDetector struct {
	languageProfiles map[string]*LanguageProfile
}

// LanguageProfile 语言特征
type LanguageProfile struct {
	Name          string
	CommonWords   map[string]float64
	CharacterFreq map[rune]float64
	SpecialChars  []rune
}

// NewLanguageDetector 创建语言检测器
func NewLanguageDetector() *LanguageDetector {
	d := &LanguageDetector{
		languageProfiles: make(map[string]*LanguageProfile),
	}

	// 注册语言特征
	d.registerLanguages()

	return d
}

// registerLanguages 注册语言特征
func (d *LanguageDetector) registerLanguages() {
	// 英语
	d.languageProfiles["en"] = &LanguageProfile{
		Name: "English",
		CommonWords: map[string]float64{
			"the": 1.0, "a": 0.9, "an": 0.8, "is": 0.8, "are": 0.7,
			"was": 0.7, "were": 0.6, "be": 0.6, "been": 0.5, "being": 0.5,
			"have": 0.7, "has": 0.7, "had": 0.6, "do": 0.6, "does": 0.5,
			"did": 0.5, "will": 0.6, "would": 0.5, "could": 0.5, "should": 0.5,
			"error": 0.4, "info": 0.3, "warning": 0.3, "debug": 0.3,
		},
		SpecialChars: []rune{},
	}

	// 中文
	d.languageProfiles["zh"] = &LanguageProfile{
		Name: "Chinese",
		CommonWords: map[string]float64{
			"的": 1.0, "了": 0.9, "是": 0.8, "在": 0.7, "就": 0.6,
			"都": 0.5, "而": 0.5, "及": 0.4, "与": 0.5, "着": 0.5,
			"错误": 0.4, "信息": 0.3, "警告": 0.3, "调试": 0.3,
		},
		SpecialChars: []rune{'的', '了', '是', '在', '就', '都', '而', '及', '与', '着'},
	}

	// 日语
	d.languageProfiles["ja"] = &LanguageProfile{
		Name: "Japanese",
		CommonWords: map[string]float64{
			"は": 1.0, "が": 0.9, "を": 0.8, "に": 0.8, "で": 0.7,
			"と": 0.6, "も": 0.6, "の": 0.9, "ます": 0.5, "です": 0.5,
		},
		SpecialChars: []rune{'ñ'}, // 使用单字符 rune
	}

	// 西班牙语
	d.languageProfiles["es"] = &LanguageProfile{
		Name: "Spanish",
		CommonWords: map[string]float64{
			"el": 1.0, "la": 1.0, "de": 0.9, "que": 0.8, "y": 0.8,
			"a": 0.7, "en": 0.7, "un": 0.6, "una": 0.6, "es": 0.6,
		},
		SpecialChars: []rune{'ñ', '¿', '¡'},
	}
}

// Detect 检测语言
func (d *LanguageDetector) Detect(text string) string {
	// 检查特殊字符
	charLang := d.detectBySpecialChars(text)
	if charLang != "" {
		return charLang
	}

	// 基于常见词检测
	wordLang := d.detectByCommonWords(text)
	if wordLang != "" {
		return wordLang
	}

	// 默认返回英语
	return "en"
}

// detectBySpecialChars 通过特殊字符检测语言
func (d *LanguageDetector) detectBySpecialChars(text string) string {
	for lang, profile := range d.languageProfiles {
		for _, char := range profile.SpecialChars {
			if strings.ContainsRune(text, char) {
				return lang
			}
		}
	}
	return ""
}

// detectByCommonWords 通过常见词检测语言
func (d *LanguageDetector) detectByCommonWords(text string) string {
	textLower := strings.ToLower(text)
	words := strings.Fields(textLower)

	langScores := make(map[string]float64)

	for lang, profile := range d.languageProfiles {
		score := 0.0
		matchCount := 0

		for _, word := range words {
			if weight, ok := profile.CommonWords[word]; ok {
				score += weight
				matchCount++
			}
		}

		if matchCount > 0 {
			langScores[lang] = score / float64(matchCount)
		}
	}

	// 找到最高分的语言
	maxScore := 0.0
	result := ""
	for lang, score := range langScores {
		if score > maxScore {
			maxScore = score
			result = lang
		}
	}

	// 只有分数超过阈值才返回
	if maxScore > 0.3 {
		return result
	}

	return ""
}
