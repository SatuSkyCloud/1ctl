package main

import (
  "fmt"
  "log"
  "net/http"
)

func main() {
  http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
    _, _ = fmt.Fprintln(w, "Hello from SatuSky")
  })
  http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
    w.WriteHeader(http.StatusOK)
  })
  log.Fatal(http.ListenAndServe(":8080", nil))
}
