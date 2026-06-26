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

	// Agent 循环架构：用 LLMClient + Agent + Fallback 替代了旧版的
	// Orchestrator + ContextResolver + SessionManager 三层依赖链

	// Python LLM sidecar 同时作为 LLMClient（Agent 循环 Chat 接口）
	pythonLLMClient := ai.NewPythonLLMClient(cfg.PythonLLMEndpoint)

	// Agent 工具依赖
	toolDeps := app.ToolDeps{
		Repo: repo,
		Log:  logger.NewAppLogger(l),
	}

	// Agent（LLM tool calling 循环）
	agent := app.NewAgent(pythonLLMClient, toolDeps, logger.NewAppLogger(l))

	// 降级编排器（LLM 不可用时自动启用，完全保留关键词路由逻辑）
	fallbackOrch := app.NewDefaultOrchestrator(repo, logger.NewAppLogger(l), nil, nil, nil)

	// 引导引擎（规则驱动）
	guideEngine := app.NewGuideEngine(logger.NewAppLogger(l))

	// 组装 Service
	svc := app.NewServiceWithConfig(app.ServiceConfig{
		Repo:        repo,
		Log:         logger.NewAppLogger(l),
		Agent:       agent,
		Fallback:    fallbackOrch,
		GuideEngine: guideEngine,
	})

	// HTTP 层
	h := apihttp.NewCustomerQAHandler(svc, l)
	fh := apihttp.NewFeedbackHandler(svc, l)

	// 管理后台 CRUD
	adminRepo := mysql.NewAdminRepository(db)
	ah := apihttp.NewAdminHandler(adminRepo, l)

	r := apihttp.NewRouter(l, h, fh, ah)

	return &App{Router: r, Logger: l, Config: cfg}, nil
}

// —— 以下保留供测试使用 ——

func newCustomerQAService(
	repo domain.Repository,
	analyzer app.IntentAnalyzer,
	composer app.AnswerComposer,
	retriever app.Retriever,
) app.Service {
	appLogger := logger.NewAppLogger(zap.NewNop())
	if analyzer != nil && composer != nil && retriever != nil {
		return app.NewServiceWithConfig(app.ServiceConfig{
			Repo:        repo,
			Log:         appLogger,
			Fallback:    app.NewDefaultOrchestrator(repo, appLogger, analyzer, composer, retriever),
			GuideEngine: app.NewGuideEngine(appLogger),
		})
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
