package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	httpserver "github.com/prasannakumar414/profile-retrieval-service/http"
)

func main() {
	loadDotEnv(".env")

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	server := httpserver.NewServer(httpserver.Config{
		Addr:                       addr,
		ProfileRateLimitRequests:   intEnv("PROFILE_RETRIEVE_RATE_LIMIT_REQUESTS", 0),
		ProfileRateLimitWindow:     durationEnv("PROFILE_RETRIEVE_RATE_LIMIT_WINDOW", 0),
		LinkedInRequestMinInterval: durationEnv("LINKEDIN_REQUEST_MIN_INTERVAL", 0),
	})

	log.Printf("profile retrieval service listening on %s", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func intEnv(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("invalid %s=%q: %v", key, value, err)
		return fallback
	}
	return parsed
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("invalid %s=%q: %v", key, value, err)
		return fallback
	}
	return parsed
}

func loadDotEnv(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if err := scanner.Err(); err != nil {
		log.Printf("error reading .env file: %v", err)
		return
	}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		value = strings.Trim(value, `"'`)
		_ = os.Setenv(key, value)
	}
}
