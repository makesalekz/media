package jwt

import (
	"strconv"

	"github.com/golang-jwt/jwt/v5"
)

type ITenantClaims interface {
	GetUserId() int64
	GetTenantId() int64
	GetIdentities() []string
	IsUserRequest() bool
	IsUserTenantRequest() bool
}

type TenantClaims struct {
	jwt.RegisteredClaims

	userId    *int64
	TenantId  int64    `json:"tenant,omitempty"`
	MemberId  string   `json:"member,omitempty"`
	GroupsIds []string `json:"groups,omitempty"`
}

func (tc *TenantClaims) GetUserId() int64 {
	if tc.userId != nil {
		return *tc.userId
	}

	userId, err := strconv.ParseInt(tc.Subject, 10, 64)
	if err != nil {
		userId = 0
	}
	tc.userId = &userId

	return userId
}

func (tc *TenantClaims) GetTenantId() int64 {
	return tc.TenantId
}

func (tc *TenantClaims) GetIdentities() []string {
	return append(tc.GroupsIds, tc.MemberId)
}

func (tc *TenantClaims) IsUserRequest() bool {
	return tc.Issuer == "iam" && tc.GetUserId() != 0
}

func (tc *TenantClaims) IsUserTenantRequest() bool {
	return tc.IsUserRequest() && tc.GetTenantId() != 0
}
