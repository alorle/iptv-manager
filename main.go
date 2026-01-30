package main

import (
	"fmt"
	"net/http"
	"os"
)

var (
	httpAddress            = os.Getenv("HTTP_ADDRESS")
	httpPort               = os.Getenv("HTTP_PORT")
	acestreamPlayerBaseUrl = os.Getenv("ACESTREAM_PLAYER_BASE_URL")
)

func main() {
	if httpAddress == "" {
		httpAddress = "127.0.0.1"
	}
	fmt.Printf("httpAddress: %v\n", httpAddress)

	if httpPort == "" {
		httpPort = "8080"
	}
	fmt.Printf("httpPort: %v\n", httpPort)

	if acestreamPlayerBaseUrl == "" {
		acestreamPlayerBaseUrl = "http://127.0.0.1:6878/ace/getstream"
	}
	fmt.Printf("acestreamPlayerBaseUrl: %v\n", acestreamPlayerBaseUrl)

	handler := http.NewServeMux()

	handler.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	s := &http.Server{
		Handler: handler,
		Addr:    fmt.Sprintf("%s:%s", httpAddress, httpPort),
	}

	if err := s.ListenAndServe(); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		return
	}
}
