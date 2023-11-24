package data

import (
	"context"
	"os"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gitlab.calendaria.team/services/media/ent"
	"gitlab.calendaria.team/services/media/internal/conf"

	_ "github.com/lib/pq"
	_ "gitlab.calendaria.team/services/media/ent/runtime"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewConfig, NewJwtProcessor, NewNatsClient, NewS3Uploader, NewMediaRepo)

// Data .
type Data struct {
	log *log.Helper
	db  *ent.Client
}

// NewData .
func NewData(c *conf.Bootstrap, logger log.Logger) (*Data, func(), error) {
	l := log.NewHelper(logger)

	automigrate := os.Getenv("AUTOMIGRATE")
	options := []ent.Option{}
	if automigrate != "" {
		options = append(options, ent.Debug(), ent.Log(l.Info))
	}

	client, err := ent.Open("postgres", c.Db, options...)
	if err != nil {
		l.Fatalf("failed opening connection to postgres: %v", err)
		return nil, nil, err
	}

	if automigrate != "" {
		if err := client.Schema.Create(context.Background()); err != nil {
			l.Errorf("failed creating schema resources: %v", err)
			return nil, nil, err
		}
	}

	l.Info("Connected to postgres")

	cleanup := func() {
		if err := client.Close(); err != nil {
			l.Error(err)
		}
	}

	return &Data{
		log: log.NewHelper(logger),
		db:  client,
	}, cleanup, nil
}
