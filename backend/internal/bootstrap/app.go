package bootstrap

import (
	"fmt"

	apihttp "store-mind/api/http"
	app "store-mind/application/customerqa"
	domain "store-mind/domain/customerqa"
	"store-mind/infra/ai"
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

	// S0: 尝试接入 Python LLM Sidecar
	pythonLLMClient := ai.NewPythonLLMClient(cfg.PythonLLMEndpoint)

	// S1: 创建会话管理器（基于 repo 加载 context_stack）
	sessionMgr := app.NewSessionManager(repo, logger.NewAppLogger(l))

	// S1: 创建上下文消解器（L1 规则 / L2 通过 Python LLM sidecar）
	contextResolver := app.NewContextResolver(pythonLLMClient, logger.NewAppLogger(l))

	// S1: 创建引导引擎（规则驱动）
	guideEngine := app.NewGuideEngine(logger.NewAppLogger(l))

	orch := app.NewDefaultOrchestrator(repo, logger.NewAppLogger(l), pythonLLMClient, pythonLLMClient, defaultRetriever(repo, pythonLLMClient))
	svc := app.NewServiceWithConfig(app.ServiceConfig{
		Repo:            repo,
		Log:             logger.NewAppLogger(l),
		Orchestrator:    orch,
		SessionManager:  sessionMgr,
		ContextResolver: contextResolver,
		GuideEngine:     guideEngine,
	})

	// S0.5: 反馈端点
	h := apihttp.NewCustomerQAHandler(svc, l)
	fh := apihttp.NewFeedbackHandler(svc, l)

	// S2.4: 管理后台 CRUD
	adminRepo := mysql.NewAdminRepository(db)
	ah := apihttp.NewAdminHandler(adminRepo, l)

	r := apihttp.NewRouter(l, h, fh, ah)

	return &App{Router: r, Logger: l, Config: cfg}, nil
}

func newCustomerQAService(
	repo domain.Repository,
	analyzer app.IntentAnalyzer,
	composer app.AnswerComposer,
	retriever app.Retriever,
) app.Service {
	appLogger := logger.NewAppLogger(zap.NewNop())
	// S0: 当 Python LLM endpoint 已配置时，传入 analyzer/composer/retriever
	// orchestrator.Run() 中若 analyzer 为 nil 则自动降级到 fallbackOrchestrator
	if analyzer != nil && composer != nil && retriever != nil {
		return app.NewServiceWithOrchestrator(
			repo,
			appLogger,
			app.NewDefaultOrchestrator(repo, appLogger, analyzer, composer, retriever),
		)
	}
	return app.NewService(repo, appLogger)
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

func defaultRetriever(repo domain.KnowledgeRepository, reranker retrieval.SemanticReranker) app.Retriever {
	r := retrieval.NewMySQLRetriever(repo)
	if reranker != nil {
		r.SetLLMClient(reranker)
	}
	return r
}
