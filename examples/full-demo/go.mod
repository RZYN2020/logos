module github.com/log-system/logos/examples/full-demo

go 1.25

require (
	github.com/log-system/log-sdk v0.0.0
	github.com/segmentio/kafka-go v0.4.47
)

require (
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/log-system/logos/pkg/rule v0.0.0-00010101000000-000000000000 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	go.etcd.io/etcd/api/v3 v3.5.9 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.9 // indirect
	go.etcd.io/etcd/client/v3 v3.5.9 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.17.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c // indirect
	google.golang.org/grpc v1.41.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)

replace (
	github.com/log-system/log-sdk => ../../log-sdk/log-sdk
	github.com/log-system/logos/pkg/rule => ../../pkg/rule
)
