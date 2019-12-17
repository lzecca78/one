package config

import (
	"log"

	"github.com/spf13/viper"
)

// GetConfig initialize all configuration from file and from environment variable
func GetConfig() *viper.Viper {
	v := viper.New()
	v.SetConfigName("conf")
	v.AddConfigPath(".")
	v.AddConfigPath("..")
	v.AddConfigPath("/one")
	v.AutomaticEnv()
	v.SetEnvPrefix("ONE")
	err := v.ReadInConfig()
	if err != nil {
		log.Fatalf("Fatal error config file: %s \n", err)
	}
	return v
}

// CheckAndGetString handles the error and apply the GetString viper function
func CheckAndGetString(v *viper.Viper, key string) string {
	if !v.IsSet(key) {
		log.Fatalf("key %s is not set", key)
	}
	return v.GetString(key)
}

// CheckAndGetBool handles the error and apply the GetBool viper function
func CheckAndGetBool(v *viper.Viper, key string) bool {
	if !v.IsSet(key) {
		log.Fatalf("key %s is not set", key)
	}
	return v.GetBool(key)
}
