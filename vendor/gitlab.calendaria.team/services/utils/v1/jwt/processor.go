package jwt

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"gitlab.calendaria.team/services/utils/v1/config"
)

type JwtProcessor struct {
	jwtSecret []byte
}

// NewJwtProcessor .
func NewJwtProcessor(c *config.Config) (*JwtProcessor, error) {
	secret, err := c.ReadGlobalSecretsFor(context.Background(), "jwt")
	if err != nil {
		return nil, fmt.Errorf("jwt secret not found, error: %w", err)
	}

	return &JwtProcessor{
		jwtSecret: []byte(secret["data"].(string)),
	}, nil
}

func (j *JwtProcessor) GetSecret() []byte {
	return j.jwtSecret
}

func (j *JwtProcessor) GetClaimsFromContext(ctx context.Context) (*TenantClaims, bool) {
	token, ok := jwt.FromContext(ctx)
	if !ok {
		return nil, false
	}

	claims, ok := token.(*TenantClaims)
	if !ok {
		return nil, false
	}

	return claims, true
}

func (j *JwtProcessor) GetUserIdFromContext(ctx context.Context) int64 {
	claims, ok := j.GetClaimsFromContext(ctx)
	if !ok || !claims.IsUserRequest() {
		return 0
	}

	return claims.GetUserId()
}
