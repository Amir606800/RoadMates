package requestmeta

import "context"

type key struct{}

type Metadata struct {
	ID     string
	UserID string // Populated by authentication in Phase 2.
}

func With(ctx context.Context, meta *Metadata) context.Context {
	return context.WithValue(ctx, key{}, meta)
}

func From(ctx context.Context) *Metadata {
	meta, _ := ctx.Value(key{}).(*Metadata)
	if meta == nil {
		return &Metadata{}
	}
	return meta
}
