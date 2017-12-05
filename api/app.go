package api

import (
	"os"

	"github.com/go-redis/redis"
	"github.com/keratin/authn-server/config"
	"github.com/keratin/authn-server/data"
	"github.com/keratin/authn-server/ops"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	dataRedis "github.com/keratin/authn-server/data/redis"
)

type pinger func() bool

type App struct {
	DbCheck           pinger
	RedisCheck        pinger
	Config            *config.Config
	AccountStore      data.AccountStore
	RefreshTokenStore data.RefreshTokenStore
	KeyStore          data.KeyStore
	Actives           data.Actives
	Reporter          ops.ErrorReporter
}

func NewApp() (*App, error) {
	cfg := config.ReadEnv()

	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(os.Stdout)

	db, err := data.NewDB(cfg.DatabaseURL)
	if err != nil {
		return nil, errors.Wrap(err, "data.NewDB")
	}

	var redis *redis.Client
	if cfg.RedisURL != nil {
		redis, err = dataRedis.New(cfg.RedisURL)
		if err != nil {
			return nil, errors.Wrap(err, "redis.New")
		}
	}

	accountStore, err := data.NewAccountStore(db)
	if err != nil {
		return nil, errors.Wrap(err, "NewAccountStore")
	}

	tokenStore, err := data.NewRefreshTokenStore(db, redis, cfg.ErrorReporter, cfg.RefreshTokenTTL)
	if err != nil {
		return nil, errors.Wrap(err, "NewRefreshTokenStore")
	}

	blobStore, err := data.NewBlobStore(cfg.AccessTokenTTL, redis, db, cfg.ErrorReporter)
	if err != nil {
		return nil, errors.Wrap(err, "NewBlobStore")
	}

	keyStore := data.NewRotatingKeyStore()
	if cfg.IdentitySigningKey == nil {
		m := data.NewKeyStoreRotater(
			data.NewEncryptedBlobStore(blobStore, cfg.DBEncryptionKey),
			cfg.AccessTokenTTL,
		)
		err := m.Maintain(keyStore, cfg.ErrorReporter)
		if err != nil {
			return nil, errors.Wrap(err, "Maintain")
		}
	} else {
		keyStore.Rotate(cfg.IdentitySigningKey)
	}

	var actives data.Actives
	if redis != nil {
		actives = dataRedis.NewActives(
			redis,
			cfg.StatisticsTimeZone,
			cfg.DailyActivesRetention,
			cfg.WeeklyActivesRetention,
			5*12,
		)
	}

	return &App{
		DbCheck:           func() bool { return db.Ping() == nil },
		RedisCheck:        func() bool { return redis != nil && redis.Ping().Err() == nil },
		Config:            cfg,
		AccountStore:      accountStore,
		RefreshTokenStore: tokenStore,
		KeyStore:          keyStore,
		Actives:           actives,
		Reporter:          cfg.ErrorReporter,
	}, nil
}
