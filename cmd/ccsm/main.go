package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/luisgsluis/claude-code-session-manager/internal/config"
	"github.com/luisgsluis/claude-code-session-manager/internal/server"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config YAML file")
	hashPassword := flag.String("hash-password", "", "generate bcrypt hash for a password and exit")
	generateSecret := flag.Bool("generate-secret", false, "generate a random session secret and exit")
	flag.Parse()

	if *hashPassword != "" {
		hash, err := hashBcrypt(*hashPassword)
		if err != nil {
			log.Fatalf("hash password: %v", err)
		}
		fmt.Println(hash)
		return
	}

	if *generateSecret {
		fmt.Println(generateRandomSecret())
		return
	}

	// Find static files: prefer env var, then ./static, then /app/static (Docker)
	staticPath := os.Getenv("CCSM_STATIC_PATH")
	if staticPath == "" {
		for _, p := range []string{"static", "/app/static"} {
			if _, err := os.Stat(p); err == nil {
				staticPath = p
				break
			}
		}
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	srv := server.New(cfg, staticPath, *configPath)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
		os.Exit(1)
	}
}
