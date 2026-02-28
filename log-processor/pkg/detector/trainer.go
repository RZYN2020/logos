// Package detector 提供格式检测的训练和优化功能
package detector

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// DetectorTrainer 检测器训练器
type DetectorTrainer struct {
	mu            sync.Mutex
	trainingData  []TrainingSample
	weights       map[FormatType]float64
	thresholds    map[FormatType]float64
}

// TrainingSample 训练样本
type TrainingSample struct {
	Log          string     `json:"log"`
	ExpectedFormat FormatType `json:"expected_format"`
	Source       string     `json:"source,omitempty"`
}

// TrainingResult 训练结果
type TrainingResult struct {
	Accuracy    float64
	Precision   map[FormatType]float64
	Recall      map[FormatType]float64
	F1Score     map[FormatType]float64
	ConfusionMatrix map[string]int
}

// NewDetectorTrainer 创建训练器
func NewDetectorTrainer() *DetectorTrainer {
	return &DetectorTrainer{
		trainingData: make([]TrainingSample, 0),
		weights:      make(map[FormatType]float64),
		thresholds:   make(map[FormatType]float64),
	}
}

// AddTrainingData 添加训练数据
func (t *DetectorTrainer) AddTrainingData(sample TrainingSample) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.trainingData = append(t.trainingData, sample)
}

// LoadTrainingData 从文件加载训练数据
func (t *DetectorTrainer) LoadTrainingData(filename string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read training data: %w", err)
	}

	var samples []TrainingSample
	if err := json.Unmarshal(data, &samples); err != nil {
		return fmt.Errorf("failed to parse training data: %w", err)
	}

	t.trainingData = append(t.trainingData, samples...)
	return nil
}

// SaveTrainingData 保存训练数据到文件
func (t *DetectorTrainer) SaveTrainingData(filename string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := json.MarshalIndent(t.trainingData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal training data: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write training data: %w", err)
	}

	return nil
}

// Train 训练检测器
func (t *DetectorTrainer) Train(detector *FormatDetectorImpl) *TrainingResult {
	t.mu.Lock()
	defer t.mu.Unlock()

	result := &TrainingResult{
		Precision:      make(map[FormatType]float64),
		Recall:         make(map[FormatType]float64),
		F1Score:        make(map[FormatType]float64),
		ConfusionMatrix: make(map[string]int),
	}

	correct := 0
	total := len(t.trainingData)

	// 统计每个格式的真实数量和预测数量
	actualCounts := make(map[FormatType]int)
	predictedCounts := make(map[FormatType]int)
	truePositives := make(map[FormatType]int)

	for _, sample := range t.trainingData {
		logData := []byte(sample.Log)
		predicted := detector.Detect(logData)

		actualCounts[sample.ExpectedFormat]++
		predictedCounts[predicted.Format]++

		matrixKey := fmt.Sprintf("%s->%s", sample.ExpectedFormat, predicted.Format)
		result.ConfusionMatrix[matrixKey]++

		if predicted.Format == sample.ExpectedFormat {
			correct++
			truePositives[sample.ExpectedFormat]++
		}
	}

	// 计算准确率
	if total > 0 {
		result.Accuracy = float64(correct) / float64(total)
	}

	// 计算每个格式的精确率、召回率和 F1 分数
	formats := []FormatType{FormatJSON, FormatKeyValue, FormatSyslog, FormatApache, FormatNginx, FormatUnstructured}
	for _, format := range formats {
		// 精确率 = TP / (TP + FP)
		if predictedCounts[format] > 0 {
			result.Precision[format] = float64(truePositives[format]) / float64(predictedCounts[format])
		}

		// 召回率 = TP / (TP + FN)
		if actualCounts[format] > 0 {
			result.Recall[format] = float64(truePositives[format]) / float64(actualCounts[format])
		}

		// F1 分数 = 2 * (Precision * Recall) / (Precision + Recall)
		if result.Precision[format]+result.Recall[format] > 0 {
			result.F1Score[format] = 2 * (result.Precision[format] * result.Recall[format]) / (result.Precision[format] + result.Recall[format])
		}
	}

	return result
}

// OptimizeThresholds 优化检测阈值
func (t *DetectorTrainer) OptimizeThresholds(detector *FormatDetectorImpl) map[FormatType]float64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	bestThresholds := make(map[FormatType]float64)
	formats := []FormatType{FormatJSON, FormatKeyValue, FormatSyslog, FormatApache, FormatNginx, FormatUnstructured}

	for _, format := range formats {
		bestThreshold := 0.5
		bestScore := 0.0

		// 尝试不同的阈值
		for threshold := 0.3; threshold <= 0.9; threshold += 0.05 {
			// 临时设置阈值
			score := t.evaluateThreshold(detector, format, threshold)
			if score > bestScore {
				bestScore = score
				bestThreshold = threshold
			}
		}

		bestThresholds[format] = bestThreshold
	}

	return bestThresholds
}

// evaluateThreshold 评估阈值
func (t *DetectorTrainer) evaluateThreshold(detector *FormatDetectorImpl, format FormatType, threshold float64) float64 {
	truePositives := 0
	falsePositives := 0
	falseNegatives := 0

	for _, sample := range t.trainingData {
		logData := []byte(sample.Log)
		result := detector.Detect(logData)

		if result.Format == format && result.Confidence >= threshold {
			if sample.ExpectedFormat == format {
				truePositives++
			} else {
				falsePositives++
			}
		} else if result.Format != format && sample.ExpectedFormat == format {
			falseNegatives++
		}
	}

	// 计算 F1 分数
	if truePositives+falsePositives == 0 || truePositives+falseNegatives == 0 {
		return 0
	}

	precision := float64(truePositives) / float64(truePositives+falsePositives)
	recall := float64(truePositives) / float64(truePositives+falseNegatives)

	if precision+recall == 0 {
		return 0
	}

	return 2 * (precision * recall) / (precision + recall)
}

// ExportMetrics 导出训练指标
func (t *DetectorTrainer) ExportMetrics(result *TrainingResult) map[string]interface{} {
	metrics := make(map[string]interface{})

	metrics["accuracy"] = result.Accuracy
	metrics["total_samples"] = len(t.trainingData)

	for format, precision := range result.Precision {
		metrics[fmt.Sprintf("precision_%s", format)] = precision
	}

	for format, recall := range result.Recall {
		metrics[fmt.Sprintf("recall_%s", format)] = recall
	}

	for format, f1 := range result.F1Score {
		metrics[fmt.Sprintf("f1_%s", format)] = f1
	}

	metrics["confusion_matrix"] = result.ConfusionMatrix

	return metrics
}

// GenerateReport 生成训练报告
func (t *DetectorTrainer) GenerateReport(result *TrainingResult) string {
	report := "=== Detector Training Report ===\n\n"

	report += fmt.Sprintf("Total Samples: %d\n", len(t.trainingData))
	report += fmt.Sprintf("Overall Accuracy: %.2f%%\n\n", result.Accuracy*100)

	report += "Per-Format Metrics:\n"
	report += "-------------------\n"

	formats := []FormatType{FormatJSON, FormatKeyValue, FormatSyslog, FormatApache, FormatNginx, FormatUnstructured}
	for _, format := range formats {
		report += fmt.Sprintf("\n%s:\n", format)
		report += fmt.Sprintf("  Precision: %.2f%%\n", result.Precision[format]*100)
		report += fmt.Sprintf("  Recall: %.2f%%\n", result.Recall[format]*100)
		report += fmt.Sprintf("  F1 Score: %.2f%%\n", result.F1Score[format]*100)
	}

	report += "\nConfusion Matrix:\n"
	report += "-----------------\n"
	for key, count := range result.ConfusionMatrix {
		report += fmt.Sprintf("  %s: %d\n", key, count)
	}

	return report
}
