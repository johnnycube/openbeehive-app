package auth

import (
	"context"

	"connectrpc.com/connect"
)

// LocalUser is the fixed identity used in self-host single-user mode, i.e.
// when no OIDC provider is configured. It lets the offline-first app work
// out of the box without a login flow: one beekeeper, one tenant.
var LocalUser = Identity{
	UserID: "local",
	OrgID:  "local",
	Email:  "local@openbeehive",
	Role:   "owner",
}

// devInterceptor injects LocalUser into every request context (unary and
// streaming). It is wired in only when authentication is disabled (selfhost
// without OIDC). Never enable it together with the real OIDC interceptor.
type devInterceptor struct{}

// DevInterceptor returns the self-host single-user interceptor.
func DevInterceptor() connect.Interceptor { return devInterceptor{} }

func (devInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return next(WithIdentity(ctx, LocalUser), req)
	}
}

func (devInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (devInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return next(WithIdentity(ctx, LocalUser), conn)
	}
}
