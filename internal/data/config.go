package data

import (
	"context"
	"fmt"
	"os"

	"github.com/go-kratos/consul/registry"
	"github.com/go-kratos/kratos/contrib/config/consul/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/hashicorp/consul/api"
	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/approle"
	"gitlab.calendaria.team/services/media/internal/conf"
)

type Config struct {
	config.Config

	appName      string
	appPath      string
	consulClient *api.Client

	Bootstrap *conf.Bootstrap
	vault     *vault.Client
}

func NewConfig(bootstrap *conf.Bootstrap) (*Config, error) {
	appName := os.Getenv("SERVICE_NAME")
	if appName == "" {
		return nil, fmt.Errorf("SERVICE_NAME not found")
	}

	consulConfig := api.DefaultConfig()
	consulAddress := os.Getenv("CONSUL_ADDRESS")
	if consulAddress != "" {
		consulConfig.Address = consulAddress
	}

	// new consul client
	consulClient, err := api.NewClient(consulConfig)
	if err != nil {
		panic(err)
	}

	globalSource, err := consul.New(consulClient, consul.WithPath("app/global/"))
	if err != nil {
		return nil, fmt.Errorf("Global source is not found: %w", err)
	}

	appPath := fmt.Sprintf("app/%s/", appName)
	source, err := consul.New(consulClient, consul.WithPath(appPath))
	if err != nil {
		return nil, fmt.Errorf("Source '%s' is not found: %w", appPath, err)
	}

	cfg := config.New(config.WithSource(globalSource, source))
	if err := cfg.Load(); err != nil {
		return nil, err
	}

	return &Config{
		consulClient: consulClient,
		appName:      appName,
		appPath:      appPath,
		Config:       cfg,
		Bootstrap:    bootstrap,
	}, nil
}

func (c *Config) GetAppName() string {
	return c.appName
}

func (c *Config) GetRegistry() *registry.Registry {
	return registry.New(c.consulClient)

}

func (c *Config) GetVault(ctx context.Context) (*vault.Client, error) {
	if c.vault != nil {
		return c.vault, nil
	}

	config := vault.DefaultConfig()

	vaultAddress := os.Getenv("VAULT_ADDRESS")
	if vaultAddress != "" {
		config.Address = vaultAddress
	}

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("Unable to initialize Vault client: %w", err)
	}

	token := os.Getenv("VAULT_TOKEN")
	if token == "" {
		roleID := os.Getenv("VAULT_ROLE_ID")
		if roleID == "" {
			return nil, fmt.Errorf("No role ID was provided in VAULT_ROLE_ID env var")
		}
		secretIDpath := os.Getenv("VAULT_SECRET_ID_PATH")
		if secretIDpath == "" {
			return nil, fmt.Errorf("No secret ID file path was provided in VAULT_SECRET_ID_PATH env var")
		}

		secretID := &auth.SecretID{FromFile: secretIDpath}

		appRoleAuth, err := auth.NewAppRoleAuth(roleID, secretID)
		if err != nil {
			return nil, fmt.Errorf("Unable to initialize AppRole auth method: %w", err)
		}

		authInfo, err := client.Auth().Login(context.Background(), appRoleAuth)
		if err != nil {
			return nil, fmt.Errorf("Unable to login to AppRole auth method: %w", err)
		}
		if authInfo == nil {
			return nil, fmt.Errorf("No auth info was returned after login")
		}
	} else {
		client.SetToken(token)
	}

	c.vault = client

	return client, nil
}

func (c *Config) ReadGlobalSecretsFor(ctx context.Context, subpath string) (map[string]interface{}, error) {
	vault, err := c.GetVault(ctx)
	if err != nil {
		return nil, err
	}

	secret, err := vault.KVv2("secret").Get(ctx, "app/global/"+subpath)
	if err != nil {
		return nil, fmt.Errorf("Unable to read global secret (%s): %w", subpath, err)
	}

	return secret.Data, nil
}

func (c *Config) ReadSecretsFor(ctx context.Context, subpath string) (map[string]interface{}, error) {
	vault, err := c.GetVault(ctx)
	if err != nil {
		return nil, err
	}

	secret, err := vault.KVv2("secret").Get(ctx, c.appPath+subpath)
	if err != nil {
		return nil, fmt.Errorf("Unable to read app secret (%s): %w", subpath, err)
	}

	return secret.Data, nil
}
