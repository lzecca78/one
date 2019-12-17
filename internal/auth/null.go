package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// NullAdapter return a null authentication adapter method, aka as no auth at all
func NullAdapter(v *viper.Viper, r *gin.RouterGroup, routerGroupPath string) *gin.RouterGroup {
	return r.Group(routerGroupPath)
}
