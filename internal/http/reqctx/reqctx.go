package reqctx

import "context"

type key struct{}

func Set(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, key{}, id)
}

func Get(ctx context.Context) string {
	if id, ok := ctx.Value(key{}).(string); ok {
		return id
	}
	return ""
}
