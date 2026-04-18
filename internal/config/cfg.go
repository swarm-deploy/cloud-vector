package config

import "log/slog"

type Config struct {
	Proxy struct {
		Log struct {
			Level slog.Level `env:"level"`
		} `env:"LOG"`
	} `envPrefix:"PROXY_"`

	Cloudru Cloudru `envPrefix:"CLOUDRU_"`
}

type Cloudru struct {
	Logging struct {
		Endpoint string `env:"ENDPOINT,required"`
	} `envPrefix:"LOGGING_"`

	IAM struct {
		Address      string `env:"ADDRESS"`
		ClientID     string `env:"CLIENT_ID,required,file,notEmpty"`
		ClientSecret string `env:"CLIENT_SECRET,required,file,notEmpty"`
	} `envPrefix:"IAM_"`

	DiscoveryURL string `env:"DISCOVERY_URL" envDefault:"https://api.cloud.ru/endpoints"`
}
