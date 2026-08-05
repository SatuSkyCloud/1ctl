package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func testModel() linearModel {
	return linearModel{
		Version:   "linear-v1",
		Bias:      0.2,
		Weights:   []float64{0.6, -0.4},
		Threshold: 0,
	}
}

func TestLoadAndScoreModel(t *testing.T) {
	model, err := loadModel("model/model-v1.json")
	if err != nil {
		t.Fatalf("loadModel() error = %v", err)
	}
	got, err := model.score([]float64{1, 0})
	if err != nil {
		t.Fatalf("score() error = %v", err)
	}
	if got != 0.8 {
		t.Fatalf("score() = %v, want 0.8", got)
	}
}

func TestPredictBoundaries(t *testing.T) {
	handler := newAPIServer(testModel(), 0).routes()
	tests := []struct {
		name       string
		method     string
		body       string
		wantStatus int
	}{
		{"valid", http.MethodPost, `{"features":[1,0]}`, http.StatusOK},
		{"wrong feature count", http.MethodPost, `{"features":[1]}`, http.StatusBadRequest},
		{"unknown field", http.MethodPost, `{"features":[1,0],"extra":true}`, http.StatusBadRequest},
		{"trailing object", http.MethodPost, `{"features":[1,0]} {}`, http.StatusBadRequest},
		{"wrong method", http.MethodGet, "", http.StatusMethodNotAllowed},
		{
			"oversized",
			http.MethodPost,
			`{"features":[` + strings.Repeat("0,", 600) + `0]}`,
			http.StatusRequestEntityTooLarge,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, "/predict", strings.NewReader(test.body))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%q", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

func TestPredictRejectsOverload(t *testing.T) {
	handler := newAPIServer(testModel(), 200*time.Millisecond).routes()
	start := make(chan struct{})
	statuses := make(chan int, maxConcurrentPredictions+1)
	var requests sync.WaitGroup

	for range maxConcurrentPredictions + 1 {
		requests.Add(1)
		go func() {
			defer requests.Done()
			<-start
			request := httptest.NewRequest(
				http.MethodPost,
				"/predict",
				bytes.NewBufferString(`{"features":[1,0]}`),
			)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			statuses <- response.Code
		}()
	}

	close(start)
	requests.Wait()
	close(statuses)

	tooManyRequests := 0
	for status := range statuses {
		if status == http.StatusTooManyRequests {
			tooManyRequests++
		}
	}
	if tooManyRequests == 0 {
		t.Fatal("expected at least one overloaded request to return 429")
	}
}
