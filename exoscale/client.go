package exoscale

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	exov2 "github.com/exoscale/egoscale/v2"
	"github.com/exoscale/terraform-provider-exoscale/version"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"

	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
)

const (
	DefaultEnvironment = "api"
)

var UserAgent = fmt.Sprintf("Exoscale-Terraform-Provider/%s (%s) Terraform-SDK/%s Terraform-framework/%s %s",
	version.Version,
	version.Commit,
	getModVersion("github.com/hashicorp/terraform-plugin-sdk/v2"),
	getModVersion("github.com/hashicorp/terraform-plugin-framework"),
	exov2.UserAgent)

func getModVersion(module string) string {
	// Read Build info
	bi, ok := debug.ReadBuildInfo()
	if ok {
		for _, mod := range bi.Deps {
			if mod.Path == module {
				return mod.Version
			}
		}
	}
	return "err"
}

func getConfig(meta interface{}) providerConfig.BaseConfig {
	t := meta.(map[string]interface{})
	return t["config"].(providerConfig.BaseConfig)
}

func getClient(meta interface{}) *exov2.Client {
	config := getConfig(meta)

	httpClient := cleanhttp.DefaultPooledClient()
	httpClient.Transport = &defaultTransport{next: httpClient.Transport}
	if logging.IsDebugOrHigher() {
		httpClient.Transport = logging.NewSubsystemLoggingHTTPTransport(
			"exoscale",
			httpClient.Transport,
		)
	}

	// During the Exoscale API V1 -> V2 transition, we need to initialize the
	// V2 client independently from the V1 client because of HTTP middleware
	// (http.Transport) clashes.
	// This can be removed once the only API used is V2.
	clientExoV2, err := exov2.NewClient(
		config.Key,
		config.Secret,
		exov2.ClientOptWithTimeout(config.Timeout),
		exov2.ClientOptWithHTTPClient(func() *http.Client {
			rc := retryablehttp.NewClient()
			rc.Logger = LeveledTFLogger{Verbose: logging.IsDebugOrHigher()}
			hc := rc.StandardClient()
			if logging.IsDebugOrHigher() {
				hc.Transport = logging.NewSubsystemLoggingHTTPTransport("exoscale", hc.Transport)
			}
			return hc
		}()),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to initialize Exoscale API V2 client: %v", err))
	}

	return clientExoV2
}

func getEnvironment(meta interface{}) string {
	config := getConfig(meta)
	if config.Environment == "" {
		return DefaultEnvironment
	}
	return config.Environment
}

type defaultTransport struct {
	next http.RoundTripper
}

// RoundTrip executes a single HTTP transaction while augmenting requests with custom headers.
func (t *defaultTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("User-Agent", UserAgent)

	resp, err := t.next.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// LeveledTFLogger is a thin wrapper around stdlib.log that satisfies retryablehttp.LeveledLogger interface.
type LeveledTFLogger struct {
	Verbose bool
}

func (l LeveledTFLogger) Error(msg string, keysAndValues ...interface{}) {
	log.Println("[ERROR]", msg, keysAndValues)
}
func (l LeveledTFLogger) Info(msg string, keysAndValues ...interface{}) {
	log.Println("[INFO]", msg, keysAndValues)
}
func (l LeveledTFLogger) Debug(msg string, keysAndValues ...interface{}) {
	if l.Verbose {
		log.Println("[DEBUG]", msg, keysAndValues)
	}
}
func (l LeveledTFLogger) Warn(msg string, keysAndValues ...interface{}) {
	log.Println("[WARN]", msg, keysAndValues)
}
