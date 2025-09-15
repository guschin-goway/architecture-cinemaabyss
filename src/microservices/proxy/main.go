package main

import (
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()

	e.Use(middleware.Logger())  // стандартное логирование
	e.Use(middleware.Recover()) // защита от паник

	// env vars
	port := getEnv("PORT", "8000")
	monolithURL := getEnv("MONOLITH_URL", "http://monolith:8080")
	moviesURL := getEnv("MOVIES_SERVICE_URL", "http://movies-service:8081")
	eventsURL := getEnv("EVENTS_SERVICE_URL", "http://events-service:8082")
	gradualMigration := getEnv("GRADUAL_MIGRATION", "false") == "true"
	moviesPercent := getEnvInt("MOVIES_MIGRATION_PERCENT", 0)

	// movies route
	e.Any("/api/movies/*", func(c echo.Context) error {
		target := monolithURL
		if gradualMigration {
			if rand.Intn(100) < moviesPercent {
				target = moviesURL
			}
		}
		return proxyRequest(c, target)
	})

	// events route
	e.Any("/api/events/*", func(c echo.Context) error {
		return proxyRequest(c, eventsURL)
	})

	// fallback -> monolith
	e.Any("/*", func(c echo.Context) error {
		return proxyRequest(c, monolithURL)
	})

	log.Printf("Proxy service running on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}

// proxyRequest forwards request to target service
func proxyRequest(c echo.Context, target string) error {
	url := target + c.Request().URL.Path
	if c.Request().URL.RawQuery != "" {
		url += "?" + c.Request().URL.RawQuery
	}

	req, err := http.NewRequest(c.Request().Method, url, c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// copy headers
	for k, v := range c.Request().Header {
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "Bad gateway: " + err.Error()})
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return c.Blob(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}

// helpers
func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func getEnvInt(key string, def int) int {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return def
	}
	return n
}
