package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/segmentio/kafka-go"
)

// Event — структура для события
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
	Time time.Time   `json:"time"`
}

const (
	User    string = "user-events"
	Payment string = "payment-events"
	Movie   string = "movie-events"
)

var (
	kafkaBrokers = getEnv("KAFKA_BROKERS", "kafka:9092")
	groupID      = getEnv("KAFKA_GROUP", "events-service-group")
)

// producer
func produceEvent(eventType string, data interface{}) error {
	event := Event{
		Type: eventType,
		Data: data,
		Time: time.Now(),
	}
	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}
	w := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBrokers),
		Topic:    eventType,
		Balancer: &kafka.LeastBytes{},
	}
	defer w.Close()

	return w.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(eventType),
			Value: bytes,
		},
	)
}

// consumer — читает топик и пишет в лог
func consumeEvents() {
	for _, value := range []string{User, Payment, Movie} {
		go func(value string) {
			r := kafka.NewReader(kafka.ReaderConfig{
				Brokers:  []string{kafkaBrokers},
				Topic:    value,
				GroupID:  groupID,
				MinBytes: 1,
				MaxBytes: 10e6,
			})

			for {
				msg, err := r.ReadMessage(context.Background())
				if err != nil {
					log.Printf("[consumer] error: %v", err)
					continue
				}
				log.Printf("[consumer] received: topic=%s partition=%d offset=%d key=%s value=%s",
					msg.Topic, msg.Partition, msg.Offset, string(msg.Key), string(msg.Value))
			}
		}(value)

	}

}

func main() {
	e := echo.New()

	e.Use(middleware.Logger())  // стандартное логирование
	e.Use(middleware.Recover()) // защита от паник

	e.GET("/api/events/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{"status": true})
	})

	e.POST("/api/events/user", func(c echo.Context) error {
		body := map[string]interface{}{}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if err := produceEvent(User, body); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusCreated, map[string]any{"status": "success"})
	})

	e.POST("/api/events/payment", func(c echo.Context) error {
		body := map[string]interface{}{}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if err := produceEvent(Payment, body); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusCreated, map[string]any{"status": "success"})
	})

	e.POST("/api/events/movie", func(c echo.Context) error {
		body := map[string]interface{}{}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if err := produceEvent(Movie, body); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusCreated, map[string]any{"status": "success"})
	})

	go consumeEvents()

	port := getEnv("PORT", "8082")
	log.Printf("Events service running on :%s", port)
	e.Logger.Fatal(e.Start(":" + port))
}

// helper
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
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
