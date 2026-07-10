package constants

import "time"

var EnvKeys = envKeys{
	Env:                "ENV",
	ServerAddress:      "SERVER_ADDRESS",
	CorsAllowedOrigin:  "CORS_ALLOWED_ORIGIN",
	AWSAccessKeyID:     "AWS_ACCESS_KEY_ID",
	AWSSecretAccessKey: "AWS_SECRET_ACCESS_KEY",
	AWSRegion:          "AWS_REGION",
	AWSEndpoint:        "AWS_ENDPOINT",
	DBHost:             "DB_HOST",
	DBPort:             "DB_PORT",
	DBUser:             "DB_USER",
	DBPassword:         "DB_PASSWORD",
	DBName:             "DB_NAME",
	DBSSLMode:          "DB_SSLMODE",
	DBSchema:           "DB_SCHEMA",
	AuthDomain:         "AUTH_DOMAIN",
	AuthAudience:       "AUTH_AUDIENCE",
}

var Headers = headers{
	Origin:        "Origin",
	ContentLength: "Content-Length",
}

var ContextKeys = contextKeys{
	Identity: "auth.identity",
}

var MaxAge = 12 * time.Hour

type contextKeys struct {
	Identity string
}

type envKeys struct {
	Env                string
	ServerAddress      string
	CorsAllowedOrigin  string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	AWSRegion          string
	AWSEndpoint        string
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	DBSSLMode          string
	DBSchema           string
	AuthDomain         string
	AuthAudience       string
}

type headers struct {
	Origin        string
	ContentLength string
}
