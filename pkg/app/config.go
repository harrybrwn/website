package app

import (
	"os"

	hydra "github.com/ory/hydra-client-go"
	"github.com/sendgrid/sendgrid-go"
)

const (
	envHydraAdminURL = "HYDRA_ADMIN_URL"
	//envHydraPublicURL = "HYDRA_PUBLIC_URL"
	envSendgridAPIKey = "SENDGRID_API_KEY"
)

func HydraAdminConfig() *hydra.Configuration {
	return &hydra.Configuration{
		UserAgent:     "hrry.me/api",
		DefaultHeader: map[string]string{"X-Forwarded-Proto": "https"},
		Debug:         Debug,
		Servers: hydra.ServerConfigurations{
			{URL: getenv(envHydraAdminURL, "http://hydra:4445")},
		},
	}
}

func SendgridClient() *sendgrid.Client {
	key, ok := os.LookupEnv(envSendgridAPIKey)
	if !ok {
		return nil
	}
	return sendgrid.NewSendClient(key)
}

func getenv(key, defaultValue string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	return v
}
