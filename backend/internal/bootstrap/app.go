package bootstrap

import (
	"fmt"

	apihttp "store-mind/api/http"
	app "store-mind/application/customerqa"
	"store-mind/infra/config"
	"store-mind/infra/logger"
	"store-mind/infra/persistence/mysql"

	"go.uber.org/zap"
)

type App struct {
	Router any
	Logger *zap.Logger
	Config config.Config
}

func Build() (*App, error) {
	cfg := config.Load()
	l := logger.New()
	appLogger := logger.NewAppLogger(l)

	db, err := mysql.Open(cfg.MySQLDSN)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	repo := mysql.NewCustomerQARepository(db)
	svc := app.NewService(repo, appLogger)
	h := apihttp.NewCustomerQAHandler(svc, l)
	r := apihttp.NewRouter(l, h)

	return &App{Router: r, Logger: l, Config: cfg}, nil
}
