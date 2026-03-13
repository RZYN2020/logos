module github.com/log-system/logos/examples/simple-demo

go 1.21

require (
	github.com/log-system/log-sdk v0.0.0
	github.com/segmentio/kafka-go v0.4.47
)

replace github.com/log-system/log-sdk => ../../log-sdk/log-sdk
