package apiclient

import (
	"net/http"
	"os"
	"strings"
	"time"
)

// Config controls remote API and local fallback client behavior.
type Config struct {
	BaseURL      string
	APIToken     string
	MockDataDir  string
	HTTPClient   *http.Client
	RemoteStrict bool
}

// ResolveConfigFromEnv builds config from environment variables.
func ResolveConfigFromEnv() Config {
	cfg := Config{
		BaseURL:     strings.TrimSpace(firstNonEmpty(os.Getenv("SEH_API_BASE_URL"), os.Getenv("SEH_API_SERVER_URL"))),
		APIToken:    strings.TrimSpace(firstNonEmpty(os.Getenv("SEH_API_TOKEN"), os.Getenv("SEH_API_KEY"))),
		MockDataDir: strings.TrimSpace(os.Getenv("SEH_MOCK_DATA_DIR")),
	}
	if cfg.MockDataDir == "" {
		cfg.MockDataDir = ".mock-seh"
	}
	return cfg
}

func (c Config) normalizedBaseURL() string {
	return strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
}

func (c Config) clientOrDefault() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
