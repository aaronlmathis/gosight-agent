module github.com/aaronlmathis/gosight-agent

go 1.23.7

toolchain go1.24.3

require (
	github.com/aaronlmathis/gosight-shared v0.0.0-20250529171634-55ec3c7de783
	github.com/coreos/go-systemd/v22 v22.5.0
	github.com/docker/docker v25.0.6+incompatible
	github.com/google/uuid v1.6.0
	github.com/nxadm/tail v1.4.11
	github.com/shirou/gopsutil/v4 v4.25.3
	go.opentelemetry.io/proto/otlp v1.7.0
	golang.org/x/sys v0.33.0
	google.golang.org/grpc v1.72.2
	google.golang.org/protobuf v1.36.6
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/ebitengine/purego v0.8.2 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/foxboron/go-tpm-keyfiles v0.0.0-20250323135004-b31fac66206e // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/go-tpm v0.9.5 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.26.3 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/knadh/koanf/providers/confmap v1.0.0 // indirect
	github.com/knadh/koanf/v2 v2.2.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/mostynb/go-grpc-compression v1.2.3 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/rs/cors v1.11.1 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/collector v0.127.0 // indirect
	go.opentelemetry.io/collector/client v1.33.0 // indirect
	go.opentelemetry.io/collector/component v1.33.0 // indirect
	go.opentelemetry.io/collector/component/componentstatus v0.127.0 // indirect
	go.opentelemetry.io/collector/config/configauth v0.127.0 // indirect
	go.opentelemetry.io/collector/config/configcompression v1.33.0 // indirect
	go.opentelemetry.io/collector/config/configgrpc v0.127.0 // indirect
	go.opentelemetry.io/collector/config/confighttp v0.127.0 // indirect
	go.opentelemetry.io/collector/config/configmiddleware v0.127.0 // indirect
	go.opentelemetry.io/collector/config/confignet v1.33.0 // indirect
	go.opentelemetry.io/collector/config/configopaque v1.33.0 // indirect
	go.opentelemetry.io/collector/config/configtls v1.33.0 // indirect
	go.opentelemetry.io/collector/confmap v1.33.0 // indirect
	go.opentelemetry.io/collector/consumer v1.33.0 // indirect
	go.opentelemetry.io/collector/consumer/consumererror v0.127.0 // indirect
	go.opentelemetry.io/collector/consumer/xconsumer v0.127.0 // indirect
	go.opentelemetry.io/collector/extension/extensionauth v1.33.0 // indirect
	go.opentelemetry.io/collector/extension/extensionmiddleware v0.127.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.33.0 // indirect
	go.opentelemetry.io/collector/internal/sharedcomponent v0.127.0 // indirect
	go.opentelemetry.io/collector/internal/telemetry v0.127.0 // indirect
	go.opentelemetry.io/collector/pdata v1.33.0 // indirect
	go.opentelemetry.io/collector/pdata/pprofile v0.127.0 // indirect
	go.opentelemetry.io/collector/pipeline v0.127.0 // indirect
	go.opentelemetry.io/collector/receiver v1.33.0 // indirect
	go.opentelemetry.io/collector/receiver/otlpreceiver v0.127.0 // indirect
	go.opentelemetry.io/collector/receiver/receiverhelper v0.127.0 // indirect
	go.opentelemetry.io/collector/receiver/xreceiver v0.127.0 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.10.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.60.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.60.0 // indirect
	go.opentelemetry.io/otel v1.35.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.35.0 // indirect
	go.opentelemetry.io/otel/log v0.11.0 // indirect
	go.opentelemetry.io/otel/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/sdk v1.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.38.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250528174236-200df99c418a // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250528174236-200df99c418a // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gotest.tools/v3 v3.5.2 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

replace github.com/docker/distribution => github.com/docker/distribution v2.7.1+incompatible
