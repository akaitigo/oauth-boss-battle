package main

import (
	"log"
	"net/http"
	"os"

	"github.com/akaitigo/oauth-boss-battle/backend/internal/boss"
	"github.com/akaitigo/oauth-boss-battle/backend/internal/middleware"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Boss 1: PKCE Missing Attack
	boss1 := boss.NewBoss1Handler()
	mux.HandleFunc("POST /api/boss/1/authorize", boss1.Authorize)
	mux.HandleFunc("POST /api/boss/1/token", boss1.Token)
	mux.HandleFunc("POST /api/boss/1/verify", boss1.Verify)

	// Boss 2: State Mismatch (CSRF)
	boss2 := boss.NewBoss2Handler()
	mux.HandleFunc("POST /api/boss/2/authorize", boss2.Authorize)
	mux.HandleFunc("POST /api/boss/2/callback", boss2.Callback)
	mux.HandleFunc("POST /api/boss/2/attack", boss2.Attack)
	mux.HandleFunc("POST /api/boss/2/verify", boss2.Verify)
	mux.HandleFunc("POST /api/boss/2/generate-state", boss2.GenerateState)

	// Boss 3: Nonce Replay Attack
	boss3 := boss.NewBoss3Handler()
	mux.HandleFunc("POST /api/boss/3/authorize", boss3.Authorize)
	mux.HandleFunc("POST /api/boss/3/token", boss3.Token)
	mux.HandleFunc("POST /api/boss/3/replay", boss3.Replay)
	mux.HandleFunc("POST /api/boss/3/verify", boss3.Verify)
	mux.HandleFunc("POST /api/boss/3/generate-nonce", boss3.GenerateNonce)

	// Boss 4: JWKS Rotation Failure
	boss4 := boss.NewBoss4Handler()
	mux.HandleFunc("POST /api/boss/4/jwks", boss4.JWKS)
	mux.HandleFunc("POST /api/boss/4/rotate", boss4.Rotate)
	mux.HandleFunc("POST /api/boss/4/sign", boss4.Sign)
	mux.HandleFunc("POST /api/boss/4/verify", boss4.Verify)
	mux.HandleFunc("POST /api/boss/4/configure-cache", boss4.ConfigureCache)

	handler := middleware.CORS(mux)

	log.Printf("Server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
