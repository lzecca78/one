package auth

import (
	"fmt"

	"github.com/lzecca78/one/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// AdapterSet  is  a switch that choose the configuration based authorization method and apply its policies
func AdapterSet(v *viper.Viper, r *gin.RouterGroup, routerGroupPath string) (*gin.RouterGroup, error) {
	switch adapter := config.CheckAndGetString(v, "AUTH_METHOD"); adapter {
	case "github":
		return GithubAdapter(v, r, routerGroupPath), nil
	case "no-auth":
		return NullAdapter(v, r, routerGroupPath), nil
	default:
		return nil, fmt.Errorf("no adapter definition match %v", adapter)
	}
}
