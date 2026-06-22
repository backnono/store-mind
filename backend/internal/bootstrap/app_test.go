package bootstrap

import (
	"context"
	"testing"
	"time"

	app "store-mind/application/customerqa"
	domain "store-mind/domain/customerqa"
)

func TestNewCustomerQAServiceUsesFallbackWithoutAIDependencies(t *testing.T) {
	svc := newCustomerQAService(&bootstrapRepoStub{}, nil, nil, nil)
	if serviceUsesPrimaryOrchestrator(svc) {
		t.Fatalf("expected fallback wiring when AI dependencies are missing")
	}
}

func TestNewCustomerQAServiceUsesPrimaryWithAIDependencies(t *testing.T) {
	svc := newCustomerQAService(&bootstrapRepoStub{}, &bootstrapAnalyzerStub{}, &bootstrapComposerStub{}, &bootstrapRetrieverStub{})
	if !serviceUsesPrimaryOrchestrator(svc) {
		t.Fatalf("expected primary orchestrator wiring when AI dependencies are present")
	}
}

type bootstrapRepoStub struct{}

func (bootstrapRepoStub) CreateSession(context.Context, *domain.Session) (*domain.Session, error) {
	return &domain.Session{ID: 1, StoreID: 1, Channel: "miniapp"}, nil
}
func (bootstrapRepoStub) GetSession(context.Context, int64) (*domain.Session, error) {
	return &domain.Session{ID: 1, StoreID: 1, Channel: "miniapp"}, nil
}
func (bootstrapRepoStub) CreateMessage(context.Context, *domain.Message) (*domain.Message, error) {
	return &domain.Message{ID: 1, SessionID: 1}, nil
}
func (bootstrapRepoStub) CreateToolCall(context.Context, *domain.ToolCall) (*domain.ToolCall, error) {
	return &domain.ToolCall{ID: 1}, nil
}
func (bootstrapRepoStub) ListSessions(context.Context, int64, int) ([]domain.Session, error) {
	return nil, nil
}
func (bootstrapRepoStub) ListToolCalls(context.Context, int64, int) ([]domain.ToolCall, error) {
	return nil, nil
}
func (bootstrapRepoStub) SearchFAQ(context.Context, int64, string, int) ([]domain.FAQ, error) {
	return nil, nil
}
func (bootstrapRepoStub) SearchProducts(context.Context, int64, string, int) ([]domain.Product, error) {
	return nil, nil
}
func (bootstrapRepoStub) ListProductsByLocation(_ context.Context, storeID int64, zoneID, shelfID *int64, limit int) ([]domain.Product, error) {
	return nil, nil
}
func (bootstrapRepoStub) GetProductLocation(context.Context, int64, int64) (*domain.ProductLocation, error) {
	return nil, domain.ErrNotFound
}
func (bootstrapRepoStub) GetInventory(context.Context, int64, int64) (*domain.Inventory, error) {
	return nil, domain.ErrNotFound
}
func (bootstrapRepoStub) ListActivePromotions(context.Context, int64, time.Time, int) ([]domain.Promotion, error) {
	return nil, nil
}
func (bootstrapRepoStub) SearchKnowledge(context.Context, int64, string, []string, int) ([]domain.KnowledgeChunk, error) {
	return nil, nil
}
func (bootstrapRepoStub) ListRecentMessages(_ context.Context, sessionID int64, limit int) ([]domain.Message, error) {
	return nil, nil
}

type bootstrapAnalyzerStub struct{}

func (bootstrapAnalyzerStub) AnalyzeIntent(context.Context, app.IntentRequest) (app.Decision, error) {
	return app.Decision{Intent: "faq", Route: app.RouteRAG, RewrittenQuery: "支付方式", Confidence: 0.9}, nil
}

type bootstrapComposerStub struct{}

func (bootstrapComposerStub) ComposeAnswer(context.Context, app.AnswerRequest) (*app.AnswerResult, error) {
	return &app.AnswerResult{Answer: "支持微信和支付宝扫码支付"}, nil
}

type bootstrapRetrieverStub struct{}

func (bootstrapRetrieverStub) Retrieve(context.Context, app.RetrievalRequest) ([]app.Evidence, error) {
	return []app.Evidence{{Kind: "faq", Title: "支付方式", Content: "支持微信和支付宝扫码支付"}}, nil
}
