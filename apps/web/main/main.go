package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"text/template"
	"time"

	"github.com/somethingsoftware/cadencereader/config"
	"github.com/somethingsoftware/cadencereader/database"
)

func main() {
	slog.Info("CadenceReader Starting")

	cfg, err := config.Load("VIEW_FOLDER")
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	db, queries, err := database.Open(ctx, cfg)
	if err != nil {
		slog.Error("Failed to open database", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", config.HealthCheck(db))
	mux.HandleFunc("GET /", index(ctx, cfg, queries))
	mux.HandleFunc("GET /post/{id}", post(ctx, cfg, queries))
	mux.HandleFunc("GET /add-blog",
		static(fmt.Sprintf("%s/%s", cfg.Extra["VIEW_FOLDER"], "add-blog.html")))
	mux.HandleFunc("POST /add-blog", addBlog(queries))

	intrC := make(chan os.Signal, 1)
	signal.Notify(intrC, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.MainHost, cfg.MainHTTPport),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		<-intrC
		slog.Warn("received interrupt, shutting down gracefully")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("graceful shutdown failed", "error", err)
		}
	}()

	slog.Info("server starting")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("HTTP server exit", "error", err)
		os.Exit(1)
	}
	slog.Info("shutdown complete")
}

func index(ctx context.Context, cfg config.Config, queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		posts, err := queries.ListBlogPosts(ctx, 20)
		if err != nil {
			http.Error(w, "failed to query database", http.StatusInternalServerError)
			slog.Error("Failed to query blog posts", "error", err)
			return
		}
		tmpl := template.Must(template.ParseFiles(cfg.Extra["VIEW_FOLDER"] + "/index.gohtml"))
		if err := tmpl.Execute(w, posts); err != nil {
			http.Error(w, "failed to render", http.StatusInternalServerError)
			slog.Error("Failed to execute index template with posts", "error", err)
			return
		}
	}
}

func post(ctx context.Context, cfg config.Config, queries *database.Queries) http.HandlerFunc {
	type postPlusDomain struct {
		database.BlogPost
		Domain string
	}

	return func(w http.ResponseWriter, r *http.Request) {
		postIDstr := r.PathValue("id")
		postID, err := strconv.Atoi(postIDstr)
		if err != nil {
			http.Error(w, "failed to parse post id in path", http.StatusInternalServerError)
			slog.Error("Failed to parse post id", "error", err)
			return
		}
		post, err := queries.GetBlogPost(ctx, int32(postID))
		if err != nil {
			http.Error(w, "failed to find post", http.StatusInternalServerError)
			slog.Error("Failed to get blog post", "error", err)
			return
		}
		proto := "http"
		if cfg.AppEnv == config.Prod {
			proto += "s"
		}
		pph := postPlusDomain{
			BlogPost: post,
			Domain:   fmt.Sprintf("%s://%s:%d", proto, cfg.DripperDomain, cfg.DripperHTTPport),
		}
		tmpl := template.Must(template.ParseFiles(cfg.Extra["VIEW_FOLDER"] + "/post.gohtml"))
		if err := tmpl.Execute(w, pph); err != nil {
			http.Error(w, "failed to render", http.StatusInternalServerError)
			slog.Error("Failed to execute template with post data", "error", err)
			return
		}
	}
}

func static(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fh, err := os.Open(path)
		if err != nil {
			slog.Error("Failed to open index html view", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		b, err := io.ReadAll(fh)
		if err != nil {
			slog.Error("Failed to read index view", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = w.Write(b)
		if err != nil {
			slog.Error("Failed to write index view", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
