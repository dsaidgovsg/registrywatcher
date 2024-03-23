package server

import (
	"fmt"

	"registrywatcher/app/health"
	"registrywatcher/app/repository"
	"registrywatcher/utilities/config"

	"github.com/gin-gonic/gin"
)

func setupGin(env config.Environment) (r *gin.Engine) {
	switch env {
	case config.Production, config.Staging:
		gin.SetMode(gin.ReleaseMode)
		r = gin.Default()
		err := r.SetTrustedProxies(nil)
		if err != nil {
			panic(fmt.Sprintf("Failed to set trusted proxies: %v\n", err))
		}
	case config.Testing, config.CI:
		gin.SetMode(gin.ReleaseMode)
		r = gin.New()
	case config.Development:
		r = gin.Default()
	default:
		panic(fmt.Sprintf("Invalid environment: %s", env))
	}
	return
}

func InitRoutes(repoService repository.RepositoryService) *gin.Engine {
	r := setupGin(config.GetEnv())

	// Add your routes here
	r.GET("/ping", health.PingHandler)

	// Repositories routes
	r.GET("/repos", repoService.GetRepositories)

	return r
}
