package config

import (
	"context"
	"fmt"
	"os"

	"github.com/go-kratos/consul/registry"
	"github.com/go-kratos/kratos/contrib/config/consul/v2"
	kconfig "github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/env"
	"github.com/hashicorp/consul/api"
	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/approle"
)

type Config struct {
	kconfig.Config

	appName      string
	appPath      string
	consulClient *api.Client

	vault *vault.Client
}

func NewConfig() (*Config, error) {
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

	envPrefix := os.Getenv("ENV_PREFIX")

	cfg := kconfig.New(kconfig.WithSource(env.NewSource(envPrefix), globalSource, source))
	if err := cfg.Load(); err != nil {
		return nil, err
	}

	return &Config{
		consulClient: consulClient,
		appName:      appName,
		appPath:      appPath,
		Config:       cfg,
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

	vconf := vault.DefaultConfig()

	vaultAddress, _ := c.Value("VAULT_ADDRESS").String()
	if vaultAddress != "" {
		vconf.Address = vaultAddress
	}

	client, err := vault.NewClient(vconf)
	if err != nil {
		return nil, fmt.Errorf("Unable to initialize Vault client: %w", err)
	}

	token, _ := c.Value("VAULT_TOKEN").String()
	if token == "" {
		roleID, _ := c.Value("VAULT_ROLE_ID").String()
		if roleID == "" {
			return nil, fmt.Errorf("No role ID was provided in VAULT_ROLE_ID env var")
		}

		var secretID *auth.SecretID
		secretIDpath, _ := c.Value("VAULT_SECRET_ID_PATH").String()
		if secretIDpath == "" {
			secretIDstring, _ := c.Value("VAULT_SECRET_ID").String()
			if secretIDstring == "" {
				return nil, fmt.Errorf("No secret ID file path was provided in VAULT_SECRET_ID/VAULT_SECRET_ID_PATH env vars")
			}
			secretID = &auth.SecretID{FromString: secretIDstring}
		} else {
			secretID = &auth.SecretID{FromFile: secretIDpath}
		}

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
		return nil, fmt.Errorf("Unable to read global secret (%s): %s", subpath, err.Error())
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
		return nil, fmt.Errorf("Unable to read app secret (%s): %s", subpath, err.Error())
	}

	return secret.Data, nil
}

func (c *Config) ReadJwt(ctx context.Context, app string) ([]byte, error) {
	vault, err := c.GetVault(ctx)
	if err != nil {
		return nil, err
	}

	secret, err := vault.KVv2("secret").Get(ctx, fmt.Sprintf("app/%s/jwt", app))
	if err != nil {
		return nil, fmt.Errorf("Unable to read jwt for %s: %s", app, err.Error())
	}

	jwt := secret.Data["data"].([]byte)
	if jwt == nil {
		return nil, fmt.Errorf("Unable to read jwt for %s: data is empty", app)
	}

	return jwt, nil
}
