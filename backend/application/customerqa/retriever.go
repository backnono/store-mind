package customerqa

import "context"

type RetrievalRequest struct {
	StoreID int64
	Query   string
	Intent  string
	Limit   int
}

type Retriever interface {
	Retrieve(ctx context.Context, req RetrievalRequest) ([]Evidence, error)
}
