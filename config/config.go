package config

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type AppEnvironment uint8

const (
	_ AppEnvironment = iota
	Local
	Prod
)

type Config struct {
	AppEnv          AppEnvironment
	MainHost        string
	MainHTTPport    uint16
	DripperHost     string
	DripperHTTPport uint16
	DatabaseURL     *url.URL
	MigrationFolder string
	Extra           map[string]string
}

func Load(extraEnv ...string) (Config, error) {
	getEnv := func(key string) string {
		val := os.Getenv(key)
		if len(strings.TrimSpace(val)) == 0 {
			slog.Error("Env var empty or all whitespace", "key", key)
			os.Exit(1)
		}
		return val
	}

	ae := getEnv("APP_ENV")
	var appEnv AppEnvironment
	switch ae {
	case "local":
		appEnv = Local
	case "prod":
		appEnv = Prod
	default:
		slog.Error("Failed to parse app environment", "APP_ENV", ae)
		os.Exit(1)
	}

	MainHost := getEnv("MAIN_HOST")
	mhp := getEnv("MAIN_HTTP_PORT")
	MainHTTPport, err := strconv.ParseUint(mhp, 10, 16)
	if err != nil {
		slog.Error("Failed to parse port number", "error", err)
		os.Exit(1)
	}

	DripperHost := getEnv("DRIPPER_HOST")
	dhp := getEnv("DRIPPER_HTTP_PORT")
	DripperHTTPport, err := strconv.ParseUint(dhp, 10, 16)
	if err != nil {
		slog.Error("Failed to parse port number", "error", err)
		os.Exit(1)
	}

	dbu := getEnv("DATABASE_URL")
	dbURL, err := url.Parse(dbu)
	if err != nil {
		return Config{}, fmt.Errorf("failed parsing database url: %w", err)
	}

	migrationFolder := getEnv("MIGRATION_FOLDER")

	extraEnvMap := make(map[string]string)
	for _, key := range extraEnv {
		value := getEnv(key)
		extraEnvMap[key] = value
	}

	return Config{
		AppEnv:          appEnv,
		MainHost:        MainHost,
		MainHTTPport:    uint16(MainHTTPport),
		DripperHost:     DripperHost,
		DripperHTTPport: uint16(DripperHTTPport),
		DatabaseURL:     dbURL,
		MigrationFolder: migrationFolder,
		Extra:           extraEnvMap,
	}, nil
}

// this might go in a /common package or something in the future
func HealthCheck(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			slog.Error("Database ping failed", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
