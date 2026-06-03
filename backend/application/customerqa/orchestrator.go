package customerqa

import (
	"context"
	"fmt"
	"strings"
	"time"

	domain "store-mind/domain/customerqa"
)

const (
	RouteTool     = "tool"
	RouteRAG      = "rag"
	RouteHybrid   = "hybrid"
	RouteFallback = "fallback"
)

type Orchestrator interface {
	Run(ctx context.Context, req OrchestratorRequest) (OrchestratorResult, error)
}

type OrchestratorRequest struct {
	RequestID string
	StoreID   int64
	SessionID int64
	MessageID int64
	UserID    *int64
	Channel   string
	Message   string
}

type Decision struct {
	Intent         string
	RewrittenQuery string
	Route          string
	NeedsHandoff   bool
	Confidence     float64
	ReasoningTags  []string
	FallbackUsed   bool
}

type Evidence struct {
	Source   string
	Kind     string
	RecordID int64
	Title    string
	Content  string
}

type OrchestratorResult struct {
	Decision Decision
	Answer   string
	Cards    []ChatCard
	Evidence []Evidence
}

type defaultOrchestrator struct {
	analyzer  IntentAnalyzer
	composer  AnswerComposer
	retriever Retriever
	fallback  Orchestrator
	repo      domain.Repository
	log       Logger
}

func newDefaultOrchestrator(repo domain.Repository, log Logger) Orchestrator {
	return NewDefaultOrchestrator(repo, log, nil, nil, nil)
}

func NewDefaultOrchestrator(repo domain.Repository, log Logger, analyzer IntentAnalyzer, composer AnswerComposer, retriever Retriever) Orchestrator {
	if log == nil {
		log = nopLogger{}
	}
	return &defaultOrchestrator{
		analyzer:  analyzer,
		composer:  composer,
		retriever: retriever,
		repo:      repo,
		log:       log,
		fallback:  newFallbackOrchestrator(repo, log),
	}
}

func (o *defaultOrchestrator) Run(ctx context.Context, req OrchestratorRequest) (OrchestratorResult, error) {
	if o.analyzer == nil {
		if o.fallback != nil {
			return o.fallback.Run(ctx, req)
		}
		return OrchestratorResult{}, nil
	}

	decision, err := o.analyzer.AnalyzeIntent(ctx, IntentRequest{
		StoreID:   req.StoreID,
		SessionID: req.SessionID,
		Message:   req.Message,
	})
	if err != nil {
		if o.fallback != nil {
			return o.fallback.Run(ctx, req)
		}
		return OrchestratorResult{}, nil
	}

	decision.Route = o.normalizeRoute(decision)
	evidence, cards, err := o.collectEvidence(ctx, req, decision)
	if err != nil {
		if o.fallback != nil {
			return o.fallback.Run(ctx, req)
		}
		return OrchestratorResult{}, err
	}

	answer := conservativeAnswer(decision.Route)
	if len(evidence) > 0 && o.composer != nil {
		answer, err = o.composer.ComposeAnswer(ctx, AnswerRequest{
			Decision: decision,
			Message:  req.Message,
			Evidence: evidence,
		})
		if err != nil {
			if o.fallback != nil {
				return o.fallback.Run(ctx, req)
			}
			return OrchestratorResult{}, err
		}
	} else if len(evidence) == 0 {
		answer = "暂时没有找到可靠依据回答这个问题，你可以换个问法，或联系人工客服。"
	}

	return OrchestratorResult{Decision: decision, Answer: answer, Cards: cards, Evidence: evidence}, nil
}

func (o *defaultOrchestrator) normalizeRoute(decision Decision) string {
	intent := strings.TrimSpace(decision.Intent)
	switch {
	case intent == "inventory" || intent == "product_location":
		return RouteTool
	case intent == "faq":
		return RouteRAG
	case intent == "product_policy":
		return RouteHybrid
	case decision.Route == RouteTool || decision.Route == RouteRAG || decision.Route == RouteHybrid || decision.Route == RouteFallback:
		return decision.Route
	default:
		return RouteFallback
	}
}

func (o *defaultOrchestrator) collectEvidence(ctx context.Context, req OrchestratorRequest, decision Decision) ([]Evidence, []ChatCard, error) {
	switch decision.Route {
	case RouteTool:
		return o.collectToolEvidence(ctx, req, decision)
	case RouteRAG:
		return o.collectRAGEvidence(ctx, req, decision)
	case RouteHybrid:
		toolEvidence, cards, err := o.collectToolEvidence(ctx, req, Decision{Intent: "inventory", RewrittenQuery: decision.RewrittenQuery})
		if err != nil {
			return nil, nil, err
		}
		ragEvidence, _, err := o.collectRAGEvidence(ctx, req, decision)
		if err != nil {
			return nil, nil, err
		}
		return append(toolEvidence, ragEvidence...), cards, nil
	default:
		if o.fallback != nil {
			result, err := o.fallback.Run(ctx, req)
			if err != nil {
				return nil, nil, err
			}
			return result.Evidence, result.Cards, nil
		}
		return nil, nil, nil
	}
}

func (o *defaultOrchestrator) collectToolEvidence(ctx context.Context, req OrchestratorRequest, decision Decision) ([]Evidence, []ChatCard, error) {
	query := decision.RewrittenQuery
	if strings.TrimSpace(query) == "" {
		query = req.Message
	}

	switch decision.Intent {
	case "inventory", "product_policy":
		products, err := o.repo.SearchProducts(ctx, req.StoreID, extractProductQuery(query), 5)
		if err != nil {
			return nil, nil, err
		}
		if len(products) == 0 {
			return nil, nil, nil
		}
		location, err := o.repo.GetProductLocation(ctx, req.StoreID, products[0].ID)
		if err != nil {
			return nil, nil, err
		}
		if location.SKUID == nil {
			return []Evidence{{
				Source:   "tool",
				Kind:     "product_location",
				RecordID: products[0].ID,
				Title:    products[0].Name,
				Content:  fmt.Sprintf("%s 在 %s %s", products[0].Name, location.ZoneName, location.ShelfCode),
			}}, []ChatCard{{Type: "product", Name: products[0].Name, Location: fmt.Sprintf("%s %s 货架", location.ZoneName, location.ShelfCode)}}, nil
		}
		inventory, err := o.repo.GetInventory(ctx, req.StoreID, *location.SKUID)
		if err != nil {
			return nil, nil, err
		}
		evidence := []Evidence{
			{
				Source:   "tool",
				Kind:     "inventory",
				RecordID: inventory.SKUID,
				Title:    products[0].Name,
				Content:  fmt.Sprintf("系统显示%s还有 %d 件", products[0].Name, inventory.Quantity),
			},
		}
		card := ChatCard{Type: "inventory", SKUID: inventory.SKUID, Name: products[0].Name, Location: fmt.Sprintf("%s %s 货架", location.ZoneName, location.ShelfCode), Quantity: inventory.Quantity}
		return evidence, []ChatCard{card}, nil
	case "product_location":
		products, err := o.repo.SearchProducts(ctx, req.StoreID, extractProductQuery(query), 5)
		if err != nil {
			return nil, nil, err
		}
		if len(products) == 0 {
			return nil, nil, nil
		}
		location, err := o.repo.GetProductLocation(ctx, req.StoreID, products[0].ID)
		if err != nil {
			return nil, nil, err
		}
		evidence := []Evidence{{
			Source:   "tool",
			Kind:     "product_location",
			RecordID: products[0].ID,
			Title:    products[0].Name,
			Content:  fmt.Sprintf("%s 在 %s %s 货架第%d层", products[0].Name, location.ZoneName, location.ShelfCode, location.LayerNo),
		}}
		card := ChatCard{Type: "product", Name: products[0].Name, Location: fmt.Sprintf("%s %s 货架第%d层", location.ZoneName, location.ShelfCode, location.LayerNo)}
		if location.SKUID != nil {
			card.SKUID = *location.SKUID
		}
		return evidence, []ChatCard{card}, nil
	case "promotion":
		items, err := o.repo.ListActivePromotions(ctx, req.StoreID, time.Now(), 5)
		if err != nil {
			return nil, nil, err
		}
		if len(items) == 0 {
			return nil, nil, nil
		}
		evidence := []Evidence{{Source: "tool", Kind: "promotion", RecordID: items[0].ID, Title: items[0].Title, Content: items[0].Description}}
		card := ChatCard{Type: "promotion", Title: items[0].Title, Content: items[0].Description, Validity: items[0].EndAt.Format("01-02 15:04")}
		return evidence, []ChatCard{card}, nil
	default:
		return nil, nil, nil
	}
}

func (o *defaultOrchestrator) collectRAGEvidence(ctx context.Context, req OrchestratorRequest, decision Decision) ([]Evidence, []ChatCard, error) {
	if o.retriever == nil {
		return nil, nil, nil
	}
	query := decision.RewrittenQuery
	if strings.TrimSpace(query) == "" {
		query = req.Message
	}
	items, err := o.retriever.Retrieve(ctx, RetrievalRequest{
		StoreID: req.StoreID,
		Query:   query,
		Intent:  decision.Intent,
		Limit:   5,
	})
	if err != nil {
		return nil, nil, err
	}
	return items, nil, nil
}

func conservativeAnswer(route string) string {
	switch route {
	case RouteTool:
		return "系统已查询到相关门店数据。"
	case RouteRAG, RouteHybrid:
		return "已根据门店知识整理答案。"
	default:
		return ""
	}
}
