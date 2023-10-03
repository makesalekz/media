package data

import (
	"context"
	"os"

	"media/ent"
	"media/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	_ "github.com/lib/pq"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewConfig, NewJwtProcessor, NewS3Uploader, NewMediaRepo)

// Data .
type Data struct {
	log *log.Helper
	db  *ent.Client
}

// NewData .
func NewData(c *conf.Bootstrap, logger log.Logger) (*Data, func(), error) {
	l := log.NewHelper(logger)

	client, err := ent.Open("postgres", c.Db)
	if err != nil {
		l.Fatalf("failed opening connection to postgres: %v", err)
		return nil, nil, err
	}

	automigrate := os.Getenv("AUTOMIGRATE")
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
