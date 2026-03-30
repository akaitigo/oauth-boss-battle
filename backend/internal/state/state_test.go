package state_test

import (
	"testing"

	"github.com/akaitigo/oauth-boss-battle/backend/internal/state"
)

func TestGenerate(t *testing.T) {
	store := state.NewStore()
	s, err := store.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(s) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("Generate() length = %d, want 32", len(s))
	}
}

func TestGenerate_Unique(t *testing.T) {
	store := state.NewStore()
	s1, _ := store.Generate()
	s2, _ := store.Generate()
	if s1 == s2 {
		t.Error("Generate() should produce unique values")
	}
}

func TestValidate_Success(t *testing.T) {
	store := state.NewStore()
	s, _ := store.Generate()
	if err := store.Validate(s, s); err != nil {
		t.Errorf("Validate() unexpected error = %v", err)
	}
}

func TestValidate_Mismatch(t *testing.T) {
	store := state.NewStore()
	s, _ := store.Generate()
	err := store.Validate(s, "different-state")
	if err != state.ErrStateMismatch {
		t.Errorf("Validate() error = %v, want ErrStateMismatch", err)
	}
}

func TestValidate_Missing(t *testing.T) {
	store := state.NewStore()
	err := store.Validate("", "some-state")
	if err != state.ErrMissingState {
		t.Errorf("Validate() error = %v, want ErrMissingState", err)
	}
}

func TestValidate_ConsumedState(t *testing.T) {
	store := state.NewStore()
	s, _ := store.Generate()
	_ = store.Validate(s, s)    // First use
	err := store.Validate(s, s) // Second use should fail
	if err != state.ErrStateExpired {
		t.Errorf("Validate() error = %v, want ErrStateExpired", err)
	}
}

func TestValidate_UnknownState(t *testing.T) {
	store := state.NewStore()
	err := store.Validate("unknown", "unknown")
	if err != state.ErrStateExpired {
		t.Errorf("Validate() error = %v, want ErrStateExpired", err)
	}
}

func TestIsIssued(t *testing.T) {
	store := state.NewStore()
	s, _ := store.Generate()
	if !store.IsIssued(s) {
		t.Error("IsIssued() = false, want true")
	}
	if store.IsIssued("nonexistent") {
		t.Error("IsIssued() = true for nonexistent, want false")
	}
}
