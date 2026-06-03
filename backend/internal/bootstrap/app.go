package bootstrap

import (
	"fmt"

	apihttp "store-mind/api/http"
	app "store-mind/application/customerqa"
	domain "store-mind/domain/customerqa"
	ai "store-mind/infra/ai"
	"store-mind/infra/config"
	"store-mind/infra/logger"
	"store-mind/infra/persistence/mysql"
	"store-mind/infra/retrieval"

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

	db, err := mysql.Open(cfg.MySQLDSN)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	repo := mysql.NewCustomerQARepository(db)
	svc := newCustomerQAService(repo, nil, nil, nil)
	h := apihttp.NewCustomerQAHandler(svc, l)
	r := apihttp.NewRouter(l, h)

	return &App{Router: r, Logger: l, Config: cfg}, nil
}

func newCustomerQAService(repo domain.Repository, analyzer app.IntentAnalyzer, composer app.AnswerComposer, retriever app.Retriever) app.Service {
	appLogger := logger.NewAppLogger(zap.NewNop())
	if analyzer == nil || composer == nil || retriever == nil {
		return app.NewService(repo, appLogger)
	}
	return app.NewServiceWithOrchestrator(repo, appLogger, app.NewDefaultOrchestrator(repo, appLogger, analyzer, composer, retriever))
}

func serviceUsesPrimaryOrchestrator(svc app.Service) bool {
	return app.UsesPrimaryOrchestrator(svc)
}

func defaultAnalyzer() app.IntentAnalyzer {
	return ai.FakeIntentAnalyzer{}
}

func defaultComposer() app.AnswerComposer {
	return ai.FakeAnswerComposer{}
}

func defaultRetriever(repo domain.KnowledgeRepository) app.Retriever {
	return retrieval.NewMySQLRetriever(repo)
}
