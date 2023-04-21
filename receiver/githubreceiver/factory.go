package githubreceiver // import "github.com/liatrio/otel-liatrio-contrib/receiver/githubreceiver"

import (
	"context"
	"errors"
	//"fmt"
	"time"

	"github.com/liatrio/otel-liatrio-contrib/receiver/githubreceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

const (
	typeStr         = "github"
	defaultInterval = 60 * time.Second
	//defaultIgnoreTLS = false
	defaultTimeout = 15 * time.Second
	stability      = component.StabilityLevelAlpha
)

var (
	ghConfigNotValid = errors.New("config is not a valid github receiver configuration")
)

// NewFactory creates a factory for the githubreceiver according to OTEL's conventions
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		// TODO: pattern seems to create internal/metadata from where stability is defined
		// look into this later
		receiver.WithMetrics(createMetricsReceiver, stability),
	)
}

// Create the default config based on the const(s) defined above.
func createDefaultConfig() component.Config {
	//logger.Sugar().Infof("Creating the default config")
	return &Config{
		// Not sure if this is right but we'll see
		ScraperControllerSettings: scraperhelper.ScraperControllerSettings{
			CollectionInterval: 10 * time.Second,
		},
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Timeout: defaultTimeout,
		},
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),

		//Interval: fmt.sprintf("%v", defaultInterval),
		//InsecureSkipVerify: defaultIgnoreTLS,
	}
}

// Create the metrics receiver according to the OTEL conventions taking in the
// context, receiver params, configuration from the component, and consumer (process or exporter)
func createMetricsReceiver(
	ctx context.Context,
	params receiver.CreateSettings,
	cfg component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {

	// check that the configuration is valid
	conf, ok := cfg.(*Config)
	if !ok {
		return nil, ghConfigNotValid
	}

	params.Logger.Sugar().Infof("Creating the metrics receiver")
	sc := newScraper(conf, params)

	params.Logger.Sugar().Infof("Creating the scraper")
	sch, err := scraperhelper.NewScraper(typeStr, sc.scrape, scraperhelper.WithStart(sc.start))
	if err != nil {
		params.Logger.Sugar().Fatalf("Error creating scraper", err)
		return nil, err
	}

	//httpClient, err := ghCfg.HTTPClientSettings.ToClient(host component.Host, settings params.TelemetrySettings{})
	// create the http client to pass to the scraper
	//httpClient, err := ghCfg.HTTPClientSettings.ToClient(component.Host, params.TelemetrySettings)

	//logger := params.Logger
	//ghs := &ghScraper{
	//    client:      httpClient,
	//	logger:       logger,
	//	nextConsumer: consumer,
	//	config:       ghCfg,
	//}

	//httpClient, err := ghCfg.HTTPClientSettings.ToClient(host component.Host, settings params.TelemetrySettings{} )
	//hc, err := ghCfg.HTTPClientSettings.ToClient(host component.Host, settings component.TelemetrySettings{})

	//if err != nil {
	//	params.Logger.Sugar().Fatalf("Error creating HTTP client", err)
	//}

	// create the scraper passing the httpclient that was created
	//sc := newScraper(httpClient, params.Logger, consumer, ghCfg)

	//scHelper, err := scraperhelper.NewScraper(typeStr, sc, scraperhelper.WithStart(sc.start), scraperhelper.WithShutdown(sc.shutdown))

	// if the next consumer (processer or exporter) in the pipeline has an issue
	// or is passed as nil then throw the next consumer error
	//if consumer == nil {
	//	return nil, component.ErrNilNextConsumer
	//}

	params.Logger.Sugar().Infof("Returning the scraper")
	return scraperhelper.NewScraperControllerReceiver(&conf.ScraperControllerSettings, params, consumer, scraperhelper.AddScraper(sch))
	//return sch, nil

	// TODO:
	// define a variable create a new scraper function which takes config and params
	// then instatiate the scraper with scraper helper new scraper taking in the type string
	// the scrape from the scraper, and a with start.
	// make the scraper with start pass the *http.Client to the GH client and actually scrape
	// return the proper metrics
	// note: the metrics might need to come from a metrics builder which possibly is generated from metadata
}
