package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"time"
)

const version = "1.0.0"

type config struct {
	Port int `env:"PORT" envDefault:"8080"`
	Db   struct {
		User         string `env:"DATABASE_USER"`
		Password     string `env:"DATABASE_PASSWORD"`
		Host         string `env:"DATABASE_HOST"`
		Port         string `env:"DATABASE_PORT"`
		Name         string `env:"DATABASE_NAME"`
		MaxOpenConns int    `env:"MAX_OPEN_CONNS"`
		MaxIdleConns int    `env:"MAX_IDLE_CONNS"`
		MaxIdleTime  string `env:"MAX_IDLE_TIME"`
	}
}

func (cfg *config) getDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.Db.User, cfg.Db.Password, cfg.Db.Host, cfg.Db.Port, cfg.Db.Name)
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

	db, err := openDB(cfg)
	if err != nil {
		logger.Fatal(err)
	}

	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {

		}
	}(db)

	logger.Printf("Подключение к БД установлено")

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
	err = srv.ListenAndServe()
	logger.Fatal(err)
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.getDSN())
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.Db.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Db.MaxIdleConns)

	duration, err := time.ParseDuration(cfg.Db.MaxIdleTime)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}
	return db, nil
}
