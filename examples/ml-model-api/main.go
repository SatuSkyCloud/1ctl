package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	maxRequestBytes          = 1024
	maxConcurrentPredictions = 4
)

type linearModel struct {
	Version   string    `json:"version"`
	Bias      float64   `json:"bias"`
	Weights   []float64 `json:"weights"`
	Threshold float64   `json:"threshold"`
}

func loadModel(path string) (linearModel, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return linearModel{}, fmt.Errorf("read model: %w", err)
	}

	var model linearModel
	if err := json.Unmarshal(data, &model); err != nil {
		return linearModel{}, fmt.Errorf("decode model: %w", err)
	}
	if model.Version == "" {
		return linearModel{}, errors.New("model version is required")
	}
	if len(model.Weights) != 2 {
		return linearModel{}, errors.New("model must contain exactly two weights")
	}
	return model, nil
}

func (m linearModel) score(features []float64) (float64, error) {
	if len(features) != len(m.Weights) {
		return 0, fmt.Errorf("features must contain exactly %d numbers", len(m.Weights))
	}

	value := m.Bias
	for i, weight := range m.Weights {
		value += weight * features[i]
	}
	return value, nil
}

type predictRequest struct {
	Features []float64 `json:"features"`
}

type predictResponse struct {
	Class        string  `json:"class"`
	Score        float64 `json:"score"`
	ModelVersion string  `json:"model_version"`
}

type apiServer struct {
	model           linearModel
	predictionSlots chan struct{}
	predictionDelay time.Duration
}

func newAPIServer(model linearModel, predictionDelay time.Duration) *apiServer {
	return &apiServer{
		model:           model,
		predictionSlots: make(chan struct{}, maxConcurrentPredictions),
		predictionDelay: predictionDelay,
	}
}

func (s *apiServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/readyz", s.ready)
	mux.HandleFunc("/predict", s.predict)
	return mux
}

func (s *apiServer) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *apiServer) ready(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":        "ready",
		"model_version": s.model.Version,
	})
}

func (s *apiServer) predict(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	select {
	case s.predictionSlots <- struct{}{}:
		defer func() { <-s.predictionSlots }()
	default:
		http.Error(w, "inference queue is full", http.StatusTooManyRequests)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	defer r.Body.Close()

	var request predictRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeDecodeError(w, err)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "request body must contain one JSON object", http.StatusBadRequest)
		return
	}

	value, err := s.model.score(request.Features)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if s.predictionDelay > 0 {
		time.Sleep(s.predictionDelay)
	}

	class := "negative"
	if value >= s.model.Threshold {
		class = "positive"
	}
	writeJSON(w, http.StatusOK, predictResponse{
		Class:        class,
		Score:        value,
		ModelVersion: s.model.Version,
	})
}

func writeDecodeError(w http.ResponseWriter, err error) {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		http.Error(w, "request body exceeds 1024 bytes", http.StatusRequestEntityTooLarge)
		return
	}
	http.Error(w, "invalid JSON request", http.StatusBadRequest)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func configuredDelay() time.Duration {
	raw := os.Getenv("PREDICTION_DELAY_MS")
	if raw == "" {
		return 100 * time.Millisecond
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 || value > 10_000 {
		log.Fatalf("PREDICTION_DELAY_MS must be between 0 and 10000")
	}
	return time.Duration(value) * time.Millisecond
}

func main() {
	modelPath := os.Getenv("MODEL_PATH")
	if modelPath == "" {
		modelPath = "/model/model.json"
	}
	model, err := loadModel(modelPath)
	if err != nil {
		log.Fatalf("model validation failed: %v", err)
	}

	server := &http.Server{
		Addr:              "0.0.0.0:8080",
		Handler:           newAPIServer(model, configuredDelay()).routes(),
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	log.Printf("model API listening on %s with model=%s", server.Addr, model.Version)
	log.Fatal(server.ListenAndServe())
}
