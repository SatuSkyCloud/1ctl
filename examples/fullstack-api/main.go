package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// ── Domain types ──────────────────────────────────────────────────────

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime"`
}

type ConfigInfo struct {
	AppEnv    string `json:"app_env"`
	DBHost    string `json:"db_host"`
	LogLevel  string `json:"log_level"`
	UploadDir string `json:"upload_dir"`
	NodeEnv   string `json:"node_env"`
	FeatureX  bool   `json:"feature_x_enabled"`
}

type Task struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type TaskCreateRequest struct {
	Title string `json:"title"`
}

type UploadResponse struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size_bytes"`
	Path     string `json:"path"`
}

type StatsResponse struct {
	TotalTasks  int    `json:"total_tasks"`
	DBConnected bool   `json:"db_connected"`
	Uptime      string `json:"uptime"`
}

// ── Globals ────────────────────────────────────────────────────────────

var (
	startTime = time.Now()
	db        *sql.DB
	appConfig ConfigInfo
)

// ── Main ───────────────────────────────────────────────────────────────

func main() {
	port := getEnv("PORT", "8080")
	appConfig = loadConfig()

	log.Printf("fullstack-api starting on :%s", port)
	log.Printf("config: env=%s db=%s log=%s uploads=%s",
		appConfig.AppEnv, appConfig.DBHost,
		appConfig.LogLevel, appConfig.UploadDir)

	// Attempt database connection (non-fatal — the app serves health
	// endpoints regardless, which is important for platform smoke checks).
	if appConfig.DBHost != "" {
		if err := connectDB(); err != nil {
			log.Printf("WARNING: database unavailable (will serve health endpoints anyway): %v", err)
		} else {
			defer db.Close()
			migrateDB()
		}
	}

	// Ensure upload directory exists.
	uploadDir := appConfig.UploadDir
	if uploadDir == "" {
		uploadDir = "/data/uploads"
	}
	if err := os.MkdirAll(uploadDir, 0750); err != nil {
		log.Printf("WARNING: cannot create upload dir %s: %v", uploadDir, err)
	}

	mux := http.NewServeMux()

	// ── Platform health (used by SatuSky smoke checks) ──
	mux.HandleFunc("/health", handleHealth)

	// ── Home / info ──
	mux.HandleFunc("/", handleHome)
	mux.HandleFunc("/api/config", handleConfig)

	// ── CRUD ──
	mux.HandleFunc("/api/tasks", handleTasks)
	mux.HandleFunc("/api/tasks/", handleTaskByID)

	// ── File upload (uses volume mount) ──
	mux.HandleFunc("/api/upload", handleUpload)
	mux.HandleFunc("/api/files/", handleServeFile)

	// ── Stats ──
	mux.HandleFunc("/api/stats", handleStats)

	// ── Secrets demo ──
	mux.HandleFunc("/api/secrets-info", handleSecretsInfo)

	addr := fmt.Sprintf(":%s", port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      withCORS(withLogging(mux)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// ── Configuration ──────────────────────────────────────────────────────

func loadConfig() ConfigInfo {
	return ConfigInfo{
		AppEnv:    getEnv("APP_ENV", "development"),
		DBHost:    getEnv("DB_HOST", ""),
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		UploadDir: getEnv("UPLOAD_DIR", "/data/uploads"),
		NodeEnv:   getEnv("NODE_ENV", ""),
		FeatureX:  getEnvBool("FEATURE_X_ENABLED", false),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := strings.ToLower(os.Getenv(key))
	switch v {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return fallback
	}
}

// ── Database ───────────────────────────────────────────────────────────

func connectDB() error {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", ""),
		getEnv("DB_NAME", "app"),
	)
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db.Ping()
}

func migrateDB() {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id         SERIAL PRIMARY KEY,
			title      TEXT NOT NULL,
			status     TEXT NOT NULL DEFAULT 'pending',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Printf("WARNING: migration failed: %v", err)
	}
}

// ── HTTP Handlers ──────────────────────────────────────────────────────

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
		Version:   "2.0.0",
		Uptime:    time.Since(startTime).Round(time.Second).String(),
	})
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":    "fullstack-api",
		"version": "2.0.0",
		"docs":    "/api/tasks",
		"health":  "/health",
		"upload":  "/api/upload",
		"config":  "/api/config",
		"stats":   "/api/stats",
	})
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, appConfig)
}

func handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleListTasks(w)
	case http.MethodPost:
		handleCreateTask(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleTaskByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	if idStr == "" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		handleGetTask(w, idStr)
	case http.MethodDelete:
		handleDeleteTask(w, idStr)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleListTasks(w http.ResponseWriter) {
	if db == nil {
		writeJSON(w, http.StatusOK, []Task{
			{ID: 1, Title: "Example task (no DB)", Status: "pending", CreatedAt: time.Now()},
		})
		return
	}
	rows, err := db.Query("SELECT id, title, status, created_at FROM tasks ORDER BY created_at DESC LIMIT 100")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	tasks := []Task{}
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Status, &t.CreatedAt); err != nil {
			continue
		}
		tasks = append(tasks, t)
	}
	writeJSON(w, http.StatusOK, tasks)
}

func handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req TaskCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
		return
	}

	if db == nil {
		writeJSON(w, http.StatusCreated, Task{
			ID: 1, Title: req.Title, Status: "pending", CreatedAt: time.Now(),
		})
		return
	}

	var t Task
	err := db.QueryRow(
		"INSERT INTO tasks (title) VALUES ($1) RETURNING id, title, status, created_at",
		req.Title,
	).Scan(&t.ID, &t.Title, &t.Status, &t.CreatedAt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func handleGetTask(w http.ResponseWriter, idStr string) {
	if db == nil {
		writeJSON(w, http.StatusOK, Task{ID: 1, Title: "example", Status: "pending", CreatedAt: time.Now()})
		return
	}
	var t Task
	if err := db.QueryRow(
		"SELECT id, title, status, created_at FROM tasks WHERE id = $1", idStr,
	).Scan(&t.ID, &t.Title, &t.Status, &t.CreatedAt); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func handleDeleteTask(w http.ResponseWriter, idStr string) {
	if db == nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		return
	}
	if _, err := db.Exec("DELETE FROM tasks WHERE id = $1", idStr); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit upload to 10 MB.
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file too large (max 10 MB)"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file"})
		return
	}
	defer file.Close()

	uploadDir := appConfig.UploadDir
	if uploadDir == "" {
		uploadDir = "/data/uploads"
	}

	// Sanitize filename.
	safeName := filepath.Base(header.Filename)
	destPath := filepath.Join(uploadDir, safeName)

	dst, err := os.Create(destPath) // #nosec G304 -- destPath is constrained to the configured upload directory
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cannot save file"})
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "write failed"})
		return
	}

	writeJSON(w, http.StatusCreated, UploadResponse{
		Filename: safeName,
		Size:     written,
		Path:     destPath,
	})
}

func handleServeFile(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/api/files/")
	if filename == "" {
		http.NotFound(w, r)
		return
	}
	uploadDir := appConfig.UploadDir
	if uploadDir == "" {
		uploadDir = "/data/uploads"
	}
	safeName := filepath.Base(filename)
	http.ServeFile(w, r, filepath.Join(uploadDir, safeName))
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	total := 0
	if db != nil {
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&total); err != nil {
			log.Printf("WARNING: cannot read task count: %v", err)
		}
	}
	writeJSON(w, http.StatusOK, StatsResponse{
		TotalTasks:  total,
		DBConnected: db != nil,
		Uptime:      time.Since(startTime).Round(time.Second).String(),
	})
}

func handleSecretsInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"db_password_set":   os.Getenv("DB_PASSWORD") != "",
		"api_key_set":       os.Getenv("API_KEY") != "",
		"smtp_password_set": os.Getenv("SMTP_PASSWORD") != "",
		"jwt_secret_set":    os.Getenv("JWT_SECRET") != "",
		// Never echo secret values — only their presence.
	})
}

// ── Middleware ─────────────────────────────────────────────────────────

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Microsecond))
	})
}

// ── Helpers ────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("WARNING: cannot encode JSON response: %v", err)
	}
}
