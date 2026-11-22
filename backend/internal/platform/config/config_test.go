package config

import (
	"net/url"
	"testing"
)

func setDBEnv(t *testing.T, user, password, host, port, database, sslmode string) {
	t.Helper()

	t.Setenv("POSTGRES_USER", user)
	t.Setenv("POSTGRES_PASSWORD", password)
	t.Setenv("POSTGRES_HOST", host)
	t.Setenv("POSTGRES_PORT", port)
	t.Setenv("POSTGRES_DB", database)
	t.Setenv("POSTGRES_SSLMODE", sslmode)
}

func TestBuildDatabaseURL(t *testing.T) {
	t.Run("all values", func(t *testing.T) {
		setDBEnv(t, "postgres", "secret", "db", "5433", "urlshortener", "require")

		raw, err := buildDatabaseURL()
		if err != nil {
			t.Fatalf("buildDatabaseURL: %v", err)
		}

		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if u.Scheme != "postgres" || u.Host != "db:5433" || u.Path != "/urlshortener" {
			t.Fatalf("url = %q", raw)
		}
		if got := u.Query().Get("sslmode"); got != "require" {
			t.Fatalf("sslmode = %q, want require", got)
		}
		password, ok := u.User.Password()
		if !ok || u.User.Username() != "postgres" || password != "secret" {
			t.Fatalf("userinfo = %q", u.User.String())
		}
	})

	t.Run("defaults", func(t *testing.T) {
		setDBEnv(t, "postgres", "", "", "", "urlshortener", "")

		raw, err := buildDatabaseURL()
		if err != nil {
			t.Fatalf("buildDatabaseURL: %v", err)
		}

		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if u.Host != "localhost:5432" {
			t.Fatalf("host = %q, want localhost:5432", u.Host)
		}
		if got := u.Query().Get("sslmode"); got != "disable" {
			t.Fatalf("sslmode = %q, want disable", got)
		}
		if _, ok := u.User.Password(); ok {
			t.Fatalf("password should be absent, got %q", u.User.String())
		}
		if u.User.Username() != "postgres" {
			t.Fatalf("username = %q, want postgres", u.User.Username())
		}
	})

	t.Run("escapes special characters in password", func(t *testing.T) {
		password := "p@ss:w/rd?#"
		setDBEnv(t, "postgres", password, "localhost", "", "urlshortener", "disable")

		raw, err := buildDatabaseURL()
		if err != nil {
			t.Fatalf("buildDatabaseURL: %v", err)
		}

		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		got, ok := u.User.Password()
		if !ok || got != password {
			t.Fatalf("password = %q, want %q (raw %q)", got, password, raw)
		}
	})

	t.Run("requires user", func(t *testing.T) {
		setDBEnv(t, "", "secret", "localhost", "", "urlshortener", "")

		if _, err := buildDatabaseURL(); err == nil {
			t.Fatal("expected error for missing user")
		}
	})

	t.Run("requires database", func(t *testing.T) {
		setDBEnv(t, "postgres", "secret", "localhost", "", "", "")

		if _, err := buildDatabaseURL(); err == nil {
			t.Fatal("expected error for missing database")
		}
	})
}

func setRedisEnv(t *testing.T, host, port, database, user, password string) {
	t.Helper()

	t.Setenv("REDIS_HOST", host)
	t.Setenv("REDIS_PORT", port)
	t.Setenv("REDIS_DB", database)
	t.Setenv("REDIS_USER", user)
	t.Setenv("REDIS_PASSWORD", password)
}

func TestBuildRedisURL(t *testing.T) {
	t.Run("disabled without host", func(t *testing.T) {
		setRedisEnv(t, "", "6379", "0", "", "secret")

		if got := buildRedisURL(); got != "" {
			t.Fatalf("buildRedisURL = %q, want empty", got)
		}
	})

	t.Run("defaults", func(t *testing.T) {
		setRedisEnv(t, "redis", "", "", "", "")

		raw := buildRedisURL()
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if u.Host != "redis:6379" || u.Path != "/0" {
			t.Fatalf("url = %q", raw)
		}
		if u.User != nil {
			t.Fatalf("userinfo = %q, want none", u.User.String())
		}
	})

	t.Run("custom port and db", func(t *testing.T) {
		setRedisEnv(t, "cache", "6380", "2", "", "")

		raw := buildRedisURL()
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if u.Host != "cache:6380" || u.Path != "/2" {
			t.Fatalf("url = %q", raw)
		}
	})

	t.Run("password only", func(t *testing.T) {
		setRedisEnv(t, "redis", "", "", "", "secret")

		raw := buildRedisURL()
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		password, ok := u.User.Password()
		if !ok || password != "secret" {
			t.Fatalf("password = %q, ok = %v", password, ok)
		}
		if u.User.Username() != "" {
			t.Fatalf("username = %q, want empty", u.User.Username())
		}
	})

	t.Run("user and password", func(t *testing.T) {
		setRedisEnv(t, "redis", "", "", "cache-user", "secret")

		raw := buildRedisURL()
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		password, ok := u.User.Password()
		if !ok || password != "secret" || u.User.Username() != "cache-user" {
			t.Fatalf("userinfo = %q", u.User.String())
		}
	})
}
