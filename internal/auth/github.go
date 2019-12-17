package auth

import (
	"fmt"

	"github.com/lzecca78/one/internal/auth/github"
	"github.com/lzecca78/one/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// GithubAdapter applies the github oauth2 authentication to all routes specificied in a router group
func GithubAdapter(v *viper.Viper, r *gin.RouterGroup, routerGroupPath string) *gin.RouterGroup {
	scopes := []string{
		"repo",
	}
	fmt.Println("enabling github authentication")
	githubSecret := config.CheckAndGetString(v, "GITHUB_OAUTH_SECRET")
	credFilePath := config.CheckAndGetString(v, "GITHUB_OAUTH_CRED_PATH")
	authRedirectURL := config.CheckAndGetString(v, "GITHUB_OAUTH_AUTHORIZED_REDIRECT_URL")
	organizationRequired := config.CheckAndGetString(v, "GITHUB_OAUTH_REQUIRED_ORG")
	sessionName := "one-session-github"
	github.Setup(credFilePath, scopes, []byte(githubSecret), authRedirectURL, organizationRequired)
	r.Use(github.Session(sessionName))
	r.GET("/login", github.LoginHandler)
	r.GET("/auth", github.Auth())
	rg := r.Group(routerGroupPath)
	rg.Use(github.CheckAuthenticatedUser())
	return rg
}
