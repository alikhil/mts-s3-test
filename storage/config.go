package storage

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/namsral/flag"
)

// Config - application configs
type Config struct {
	S3Bucket          string `split_words:"true"`
	S3Region          string `default:"ru-msk" split_words:"true"`
	S3Endpoint        string `split_words:"true"`
	S3AccessKeyID     string `split_words:"true"`
	S3SecretAccessKey string `split_words:"true"`
}

// App - application configs

func InitConfig() *Config {

	envPath := flag.String("env", ".env", "path to file with environment variables")

	// TODO: remove flag? do we really need it?
	flag.Parse()

	conf := InitConfigFrom(*envPath)
	return conf
}

// InitConfigFrom - initializes configs from env file
func InitConfigFrom(envPath string) *Config {

	App := &Config{}

	err := godotenv.Load(envPath)
	if err != nil {
		log.Printf("INFO: failed to read env file: %v", err)
	}

	err = envconfig.Process("", App)

	if err != nil {
		panic(err)
	}

	return App
}
