//sdk/context.go
package sdk

import "context"

type contextKey string

const traceIDKey contextKey = "trace_id"

func InjectTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

func ExtractTraceID(ctx context.Context) string {
	if id, ok := ctx.Value(traceIDKey).(string); ok {
		return id
	}
	return ""
}