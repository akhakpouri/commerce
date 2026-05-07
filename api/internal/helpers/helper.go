package helpers

import (
	"commerce/api/configs"
	"commerce/api/container"
	"commerce/api/internal/auth"
	"strconv"

	"github.com/gin-gonic/gin"
)

func ParseParamToUint(param string) (*uint, error) {
	id, err := strconv.ParseUint(param, 10, 64)

	if err != nil {
		return nil, err
	}
	val := uint(id)
	return &val, nil
}

func ParseParamToBool(param string) bool {
	p, err := strconv.ParseBool(param)

	if err != nil {
		return false
	}
	return bool(p)
}

func RegisterAuthRoutes(router *gin.Engine, c *container.Container, config *configs.Config) {
	valid, err := auth.NewValidator(config.Auth.Domain, config.Auth.Audience)
	if err != nil {
		panic("")
	}
	mw, err := auth.NewMiddleware(valid)
	if err != nil {
		panic("")
	}
	ginAuth := auth.Gin(*mw)

	// Apply to a route group
	authedAPI := router.Group("/api")
	authedAPI.Use(ginAuth)
	{
		authHandler := authH.NewAuthHandler(authSvc)
		authHandler.RegisterRoutes(authedAPI.Group("/auth")) // /api/auth/whoami
	}
}
