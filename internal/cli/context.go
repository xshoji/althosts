package cli

import "context"

type ctxKey struct{}

func withApp(ctx context.Context, a *app) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxKey{}, a)
}

func appFrom(ctx context.Context) *app {
	if ctx == nil {
		return nil
	}
	v, _ := ctx.Value(ctxKey{}).(*app)
	return v
}
