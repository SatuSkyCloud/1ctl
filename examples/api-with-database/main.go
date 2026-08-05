package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	db *pgxpool.Pool
}

type note struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
}

func main() {
	ctx := context.Background()
	var pool *pgxpool.Pool

	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		config, err := pgxpool.ParseConfig(databaseURL)
		if err != nil {
			log.Fatal(err)
		}
		if value := os.Getenv("DB_MAX_CONNECTIONS"); value != "" {
			maxConnections, err := strconv.Atoi(value)
			if err != nil {
				log.Fatal(err)
			}
			config.MaxConns = int32(maxConnections)
		}

		pool, err = pgxpool.NewWithConfig(ctx, config)
		if err != nil {
			log.Fatal(err)
		}
		if err = pool.Ping(ctx); err != nil {
			log.Fatal(err)
		}
		if _, err = pool.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS notes (
				id BIGSERIAL PRIMARY KEY,
				body TEXT NOT NULL,
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)
		`); err != nil {
			log.Fatal(err)
		}
		log.Println("database connected and migrated")
		defer pool.Close()
	} else {
		log.Println("DATABASE_URL not attached yet")
	}

	server := &app{db: pool}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", server.health)
	mux.HandleFunc("/notes", server.notes)

	log.Println("listening on 0.0.0.0:8080")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", mux))
}

func (a *app) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	if a.db == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "configuring"})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":   "ok",
		"database": "connected",
	})
}

func (a *app) notes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	if a.db == nil {
		http.Error(w, `{"error":"database not configured"}`, http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		rows, err := a.db.Query(r.Context(), "SELECT id, body FROM notes ORDER BY id")
		if err != nil {
			http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		notes := make([]note, 0)
		for rows.Next() {
			var item note
			if err = rows.Scan(&item.ID, &item.Body); err != nil {
				http.Error(w, `{"error":"scan failed"}`, http.StatusInternalServerError)
				return
			}
			notes = append(notes, item)
		}
		if err = rows.Err(); err != nil {
			http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(notes)

	case http.MethodPost:
		var input struct {
			Body string `json:"body"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Body == "" {
			http.Error(w, `{"error":"body is required"}`, http.StatusBadRequest)
			return
		}

		var created note
		err := a.db.QueryRow(
			r.Context(),
			"INSERT INTO notes(body) VALUES($1) RETURNING id, body",
			input.Body,
		).Scan(&created.ID, &created.Body)
		if err != nil {
			http.Error(w, `{"error":"insert failed"}`, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(created)

	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
