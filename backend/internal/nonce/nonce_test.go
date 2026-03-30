package nonce_test

import (
	"testing"

	"github.com/akaitigo/oauth-boss-battle/backend/internal/nonce"
)

func TestGenerate(t *testing.T) {
	store := nonce.NewStore()
	n, err := store.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(n) != 32 {
		t.Errorf("Generate() length = %d, want 32", len(n))
	}
}

func TestGenerate_Unique(t *testing.T) {
	store := nonce.NewStore()
	n1, _ := store.Generate()
	n2, _ := store.Generate()
	if n1 == n2 {
		t.Error("Generate() should produce unique values")
	}
}

func TestConsume_Success(t *testing.T) {
	store := nonce.NewStore()
	n, _ := store.Generate()
	if err := store.Consume(n); err != nil {
		t.Errorf("Consume() unexpected error = %v", err)
	}
}

func TestConsume_Empty(t *testing.T) {
	store := nonce.NewStore()
	err := store.Consume("")
	if err != nonce.ErrMissingNonce {
		t.Errorf("Consume() error = %v, want ErrMissingNonce", err)
	}
}

func TestConsume_Unknown(t *testing.T) {
	store := nonce.NewStore()
	err := store.Consume("unknown-nonce")
	if err != nonce.ErrNonceUnknown {
		t.Errorf("Consume() error = %v, want ErrNonceUnknown", err)
	}
}

func TestConsume_AlreadyConsumed(t *testing.T) {
	store := nonce.NewStore()
	n, _ := store.Generate()
	_ = store.Consume(n)
	err := store.Consume(n)
	if err != nonce.ErrNonceConsumed {
		t.Errorf("Consume() error = %v, want ErrNonceConsumed", err)
	}
}

func TestIsConsumed(t *testing.T) {
	store := nonce.NewStore()
	n, _ := store.Generate()
	if store.IsConsumed(n) {
		t.Error("IsConsumed() = true before consume, want false")
	}
	_ = store.Consume(n)
	if !store.IsConsumed(n) {
		t.Error("IsConsumed() = false after consume, want true")
	}
}
