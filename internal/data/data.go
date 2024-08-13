package data

import (
	"context"
	"os"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gitlab.calendaria.team/services/media/ent"
	"gitlab.calendaria.team/services/media/internal/conf"
	u_config "gitlab.calendaria.team/services/utils/v1/config"
	u_jwt "gitlab.calendaria.team/services/utils/v2/jwt"

	_ "github.com/lib/pq"
	_ "gitlab.calendaria.team/services/media/ent/runtime"
)

const (
	Calendaria = "calendaria"
	Pms        = "pms"

	DefaultTimeout = 10 * time.Minute
)

// ProviderSet is data providers.
//
//nolint:gochecknoglobals // global variable, used in wire
var ProviderSet = wire.NewSet(
	NewData,
	u_config.NewConfig,
	u_jwt.NewJwtProcessor,
	NewNatsClient,
	NewS3Uploader,
	NewMediaRepo,
)

// Data .
type Data struct {
	db *ent.Client
}

// NewData .
func NewData(bc *conf.Bootstrap, c *u_config.Config, logger log.Logger) (*Data, func(), error) {
	l := log.NewHelper(logger)

	dbDsn := bc.GetDb() // read from local config
	if dbDsn == "" {
		// read from vault
		secret, err := c.ReadSecretsFor(context.Background(), "db-dsn")
		if err != nil {
			l.Fatalf("db dsn not found: %v", err)
			return nil, nil, err
		}

		var ok bool

		dbDsn, ok = secret["data"].(string)
		if !ok {
			l.Fatalf("db dsn not found: %v", err)

			return nil, nil, err
		}
	}

	autoMigrate := os.Getenv("AUTOMIGRATE")
	entLogging := os.Getenv("ENT_LOGGING")

	var options []ent.Option

	if entLogging == "true" {
		options = append(options, ent.Debug(), ent.Log(l.Info))
	}

	client, err := ent.Open("postgres", dbDsn, options...)
	if err != nil {
		l.Fatalf("failed opening connection to postgres: %v", err)
		return nil, nil, err
	}

	if autoMigrate != "" {
		if err = client.Schema.Create(context.Background()); err != nil {
			l.Errorf("failed creating schema resources: %v", err)
			return nil, nil, err
		}
	}

	l.Info("Connected to postgres")

	cleanup := func() {
		if err = client.Close(); err != nil {
			l.Error(err)
		}
	}

	return &Data{
		db: client,
	}, cleanup, nil
}
