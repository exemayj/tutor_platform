package layouts

import (
	"context"
	"tutor_platform/internal/middleware"
)

type UserInfo struct {
	IsLoggedIn bool
	IsTutor    bool
}

func GetUserInfo(ctx context.Context) UserInfo {
	claims := middleware.GetUserFromContext(ctx)
	if claims == nil {
		return UserInfo{}
	}
	return UserInfo{
		IsLoggedIn: true,
		IsTutor:    claims.Role == "tutor",
	}
}
