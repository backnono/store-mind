package retrieval

import (
	"context"

	app "store-mind/application/customerqa"
	domain "store-mind/domain/customerqa"
)

type MySQLRetriever struct {
	repo domain.KnowledgeRepository
}

func NewMySQLRetriever(repo domain.KnowledgeRepository) *MySQLRetriever {
	return &MySQLRetriever{repo: repo}
}

func (r *MySQLRetriever) Retrieve(ctx context.Context, req app.RetrievalRequest) ([]app.Evidence, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 3
	}

	chunks, err := r.repo.SearchKnowledge(ctx, req.StoreID, req.Query, knowledgeTypesForIntent(req.Intent), limit)
	if err != nil {
		return nil, err
	}

	out := make([]app.Evidence, 0, len(chunks))
	for _, chunk := range chunks {
		out = append(out, app.Evidence{
			Source:   "retriever",
			Kind:     chunk.KnowledgeType,
			RecordID: chunk.ID,
			Title:    chunk.Title,
			Content:  chunk.Content,
		})
	}
	return out, nil
}

func knowledgeTypesForIntent(intent string) []string {
	switch intent {
	case "faq":
		return []string{"faq", "store_policy"}
	default:
		return nil
	}
}
