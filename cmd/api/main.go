package main

import (
	"fmt"
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"time"
)

const version = "1.0.0"

type config struct {
	Port int `env:"PORT" envDefault:"8080"`
}

type application struct {
	config config
	logger *log.Logger
}

func main() {
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	if err := godotenv.Load(".env"); err != nil {
		log.Println(".env файл не найден!")
	}

	var cfg config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("Ошибка чтения переменных среды: %v", err)
	}

	log.Printf("Конфигурация: %+v", cfg)

	app := &application{
		config: cfg,
		logger: logger,
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	logger.Printf("Запуск сервера на %s", srv.Addr)
	err := srv.ListenAndServe()
	logger.Fatal(err)
}
