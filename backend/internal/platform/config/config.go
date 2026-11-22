package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	RedisURL    string
	Port        int
	BaseURL     string
	LogLevel    string
}

func Load() (Config, error) {
	loadDotEnv()

	databaseURL, err := buildDatabaseURL()
	if err != nil {
		return Config{}, err
	}

	c := Config{
		DatabaseURL: databaseURL,
		RedisURL:    buildRedisURL(),
		Port:        getEnvInt("PORT", 8080),
		BaseURL:     getEnv("BASE_URL", "http://localhost:8080"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}
	return c, nil
}

func buildDatabaseURL() (string, error) {
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	database := os.Getenv("POSTGRES_DB")
	if user == "" || database == "" {
		return "", fmt.Errorf("POSTGRES_USER and POSTGRES_DB are required")
	}

	u := &url.URL{
		Scheme:   "postgres",
		Host:     net.JoinHostPort(getEnv("POSTGRES_HOST", "localhost"), getEnv("POSTGRES_PORT", "5432")),
		Path:     "/" + database,
		RawQuery: url.Values{"sslmode": {getEnv("POSTGRES_SSLMODE", "disable")}}.Encode(),
	}
	if password == "" {
		u.User = url.User(user)
	} else {
		u.User = url.UserPassword(user, password)
	}
	return u.String(), nil
}

func buildRedisURL() string {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		return ""
	}

	u := &url.URL{
		Scheme: "redis",
		Host:   net.JoinHostPort(host, getEnv("REDIS_PORT", "6379")),
		Path:   "/" + getEnv("REDIS_DB", "0"),
	}
	user := os.Getenv("REDIS_USER")
	password := os.Getenv("REDIS_PASSWORD")
	if user != "" {
		u.User = url.UserPassword(user, password)
	} else if password != "" {
		u.User = url.UserPassword("", password)
	}
	return u.String()
}

func loadDotEnv() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for {
		path := filepath.Join(dir, ".env")
		if _, err := os.Stat(path); err == nil {
			_ = godotenv.Load(path)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
