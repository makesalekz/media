package auth

import (
	"context"
	"strconv"
	"strings"
	"time"

	u_auth "gitlab.calendaria.team/services/utils/v2/auth"
	u_jwt "gitlab.calendaria.team/services/utils/v2/jwt"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/middleware"
	kjwt "github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/golang-jwt/jwt/v5"
)

const S2S_TOKEN_DURATION = 60 * time.Minute

// Server is a middleware that extracts the claims from the jwt token and adds them to the context.
func Server(jwtp u_jwt.IJwtProcessor) middleware.Middleware {
	return kjwt.Server(
		func(token *jwt.Token) (interface{}, error) {
			return jwtp.GetSecret(), nil
		}, kjwt.WithSigningMethod(jwt.SigningMethodHS256),
		kjwt.WithClaims(func() jwt.Claims { return &u_jwt.TenantClaims{} }),
	)
}

// BffMetaServer is a middleware that extracts the actor id and tenant id from the jwt token and adds them to the metadata in global context.
// It allows to use the actor id and tenant id in the whole service calls chain.
// Requires the jwt & metadata middlewares to be used before this middleware.
// Requires user claims to be passed in the jwt token.
func BffMetaServer(jwtp u_jwt.IJwtProcessor, appId string) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			claims, _ := jwtp.GetClaimsFromContext(ctx)

			if !claims.IsUserRequest() {
				return nil, errors.New(401, "UNAUTHORIZED", "user claims are required")
			}

			actorId := claims.GetUserId()
			ctx = u_auth.NewActorContext(ctx, actorId)
			ctx = metadata.AppendToClientContext(ctx, "x-md-global-actor-id", strconv.FormatInt(actorId, 10))

			if appId != "" {
				ctx = metadata.AppendToClientContext(ctx, "x-md-global-app-id", appId)
			}

			tenantId := claims.GetTenantId()
			if tenantId != 0 {
				ctx = u_auth.NewTenantContext(ctx, tenantId)
				ctx = metadata.AppendToClientContext(ctx, "x-md-global-tenant-id", strconv.FormatInt(tenantId, 10))

				identities := claims.GetIdentities()
				if len(identities) > 0 {
					ctx = metadata.AppendToClientContext(ctx, "x-md-global-identities", strings.Join(identities, ","))
				}
			}

			return handler(ctx, req)
		}
	}
}

// Client is a middleware that adds the jwt token to the client grpc request.
func Client(
	jwtp u_jwt.IJwtProcessor,
	issuer string,
	audience jwt.ClaimStrings,
) middleware.Middleware {
	return kjwt.Client(
		func(token *jwt.Token) (interface{}, error) {
			return jwtp.GetSecret(), nil
		}, kjwt.WithSigningMethod(jwt.SigningMethodHS256), kjwt.WithClaims(
			func() jwt.Claims {
				return &jwt.RegisteredClaims{
					Issuer:    issuer,
					Audience:  audience,
					Subject:   "s2s",
					IssuedAt:  jwt.NewNumericDate(time.Now()),
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(S2S_TOKEN_DURATION)),
				}
			},
		),
	)
}
