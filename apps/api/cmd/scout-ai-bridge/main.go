package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/kyle/scout/open-core/apps/api/internal/bridge"
)

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func resolveToken(tokenFlag, tokenFileFlag string) (string, error) {
	// Token file takes precedence over the inline flag so the flag can be
	// omitted entirely when a file is provided.
	if tokenFileFlag != "" {
		data, err := os.ReadFile(tokenFileFlag)
		if err != nil {
			return "", fmt.Errorf("reading token file %q: %w", tokenFileFlag, err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	return tokenFlag, nil
}

func main() {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		fmt.Fprintf(os.Stderr, "scout-ai-bridge: unsupported OS %q (darwin and linux only)\n", runtime.GOOS)
		os.Exit(1)
	}

	defaultBind := envOrDefault("SCOUT_AI_BRIDGE_BIND", "127.0.0.1:4761")
	defaultToken := envOrDefault("SCOUT_AI_BRIDGE_TOKEN", "")
	defaultTokenFile := envOrDefault("SCOUT_AI_BRIDGE_TOKEN_FILE", "")

	bind := flag.String("bind", defaultBind, "address to listen on (host:port)")
	tokenFlag := flag.String("token", defaultToken, "bearer token (min 32 chars); prefer --token-file to avoid exposure in process listings")
	tokenFileFlag := flag.String("token-file", defaultTokenFile, "path to file containing the bearer token (preferred over --token)")
	allowPrivate := flag.Bool("allow-private-bind", false, "allow binding to private (RFC-1918) addresses")
	flag.Parse()

	if *tokenFlag != "" && *tokenFileFlag != "" {
		fmt.Fprintf(os.Stderr, "scout-ai-bridge: --token and --token-file are mutually exclusive; --token-file takes precedence\n")
	}

	if err := bridge.ValidateBindAddress(*bind, *allowPrivate); err != nil {
		fmt.Fprintf(os.Stderr, "scout-ai-bridge: %v\n", err)
		os.Exit(1)
	}

	token, err := resolveToken(*tokenFlag, *tokenFileFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scout-ai-bridge: %v\n", err)
		os.Exit(1)
	}
	if len(token) < 32 {
		fmt.Fprintf(os.Stderr, "scout-ai-bridge: token must be at least 32 characters (got %d)\n", len(token))
		os.Exit(1)
	}

	listenURL, err := bridge.LocalhostURLForBind(*bind)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scout-ai-bridge: %v\n", err)
		os.Exit(1)
	}

	handler := bridge.NewHandler(bridge.ServerConfig{
		BearerToken:    token,
		Registry:       bridge.NewRegistry(5*time.Minute, bridge.NewCopilotRuntime(), bridge.NewClaudeRuntime()),
		MaxBodyBytes:   1 << 20,
		RequestTimeout: 5 * time.Minute,
	})

	srv := &http.Server{
		Addr:              *bind,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Minute,
		WriteTimeout:      10 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}

	log.Printf("scout-ai-bridge: listening on %s", listenURL)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("scout-ai-bridge: server error: %v", err)
	}
}
