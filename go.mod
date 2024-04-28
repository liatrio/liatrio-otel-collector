module github.com/liatrio/liatrio-otel-collector

go 1.21.11

toolchain go1.22.2

require (
	github.com/liatrio/liatrio-otel-collector/extension/githubappauthextension v0.0.0-00010101000000-000000000000
	github.com/liatrio/liatrio-otel-collector/receiver/gitproviderreceiver v0.0.0-00010101000000-000000000000
)

require (
	github.com/Khan/genqlient v0.7.0 // indirect
	github.com/bradleyfalzon/ghinstallation/v2 v2.10.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.0.0-alpha.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-github/v60 v60.0.0 // indirect
	github.com/google/go-github/v61 v61.0.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.5 // indirect
	github.com/hashicorp/go-version v1.6.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.8 // indirect
	github.com/knadh/koanf/maps v0.1.1 // indirect
	github.com/knadh/koanf/providers/confmap v0.1.0 // indirect
	github.com/knadh/koanf/v2 v2.1.1 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/rs/cors v1.10.1 // indirect
	github.com/vektah/gqlparser/v2 v2.5.11 // indirect
	github.com/xanzy/go-gitlab v0.103.0 // indirect
	go.opentelemetry.io/collector v0.99.0 // indirect
	go.opentelemetry.io/collector/component v0.99.0 // indirect
	go.opentelemetry.io/collector/config/configauth v0.99.0 // indirect
	go.opentelemetry.io/collector/config/configcompression v1.6.0 // indirect
	go.opentelemetry.io/collector/config/confighttp v0.99.0 // indirect
	go.opentelemetry.io/collector/config/configopaque v1.6.0 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.99.0 // indirect
	go.opentelemetry.io/collector/config/configtls v0.99.0 // indirect
	go.opentelemetry.io/collector/config/internal v0.99.0 // indirect
	go.opentelemetry.io/collector/confmap v0.99.0 // indirect
	go.opentelemetry.io/collector/consumer v0.99.0 // indirect
	go.opentelemetry.io/collector/extension v0.99.0 // indirect
	go.opentelemetry.io/collector/extension/auth v0.99.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.6.0 // indirect
	go.opentelemetry.io/collector/filter v0.99.0 // indirect
	go.opentelemetry.io/collector/pdata v1.6.0 // indirect
	go.opentelemetry.io/collector/receiver v0.99.0 // indirect
	go.opentelemetry.io/collector/semconv v0.99.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.50.0 // indirect
	go.opentelemetry.io/otel v1.25.0 // indirect
	go.opentelemetry.io/otel/metric v1.25.0 // indirect
	go.opentelemetry.io/otel/trace v1.25.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/net v0.24.0 // indirect
	golang.org/x/oauth2 v0.19.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240401170217-c3f982113cda // indirect
	google.golang.org/grpc v1.63.2 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

replace github.com/liatrio/liatrio-otel-collector/receiver/gitproviderreceiver => ./receiver/gitproviderreceiver

replace github.com/liatrio/liatrio-otel-collector/extension/githubappauthextension => ./extension/githubappauthextension
