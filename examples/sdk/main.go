package main

import (
	"errors"

	"github.com/log-system/log-sdk/pkg/logger"
)

func main() {
	// 初始化 Logger
	log := logger.New(logger.Config{
		ServiceName:         "sdk-example",
		Environment:         "dev",
		Cluster:             "local",
		Pod:                 "sdk-example-pod-1",
		KafkaBrokers:        []string{"localhost:9092"},
		KafkaTopic:          "logs",
		EtcdEndpoints:       []string{"http://localhost:2379"},
		BatchSize:           100,
		BatchTimeout:        100,
		FallbackToConsole:   true,
		MaxBufferSize:       10000,
	})
	defer log.Close()

	// 添加 Hook 过滤 DEBUG 日志
	log = log.AddHook(logger.LevelHook(logger.LevelInfo))

	// 基本日志
	log.Info("Hello World",
		logger.F("user_id", "123"),
		logger.F("product_id", "abc"))

	// 错误日志
	err := someFunction()
	if err != nil {
		log.Error("Failed to execute function",
			logger.F("error", err.Error()),
			logger.F("function", "someFunction"))
	}
}

func someFunction() error {
	return errors.New("internal error")
}
