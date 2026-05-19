package git

import (
	"context"
	"testing"
	"time"

	"github.com/entro314-labs/git-herd/pkg/types"
)

func TestOperationContextWithoutTimeout(t *testing.T) {
	t.Parallel()

	processor := NewProcessor(&types.Config{})
	parentCtx := context.Background()

	ctx, cancel := processor.operationContext(parentCtx)
	defer cancel()

	if _, ok := ctx.Deadline(); ok {
		t.Fatal("expected no deadline when timeout is disabled")
	}

	if ctx != parentCtx {
		t.Fatal("expected operationContext to reuse the parent context when timeout is disabled")
	}
}

func TestOperationContextWithTimeout(t *testing.T) {
	t.Parallel()

	processor := NewProcessor(&types.Config{Timeout: 250 * time.Millisecond})
	parentCtx := context.Background()

	ctx, cancel := processor.operationContext(parentCtx)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected deadline when timeout is configured")
	}

	remaining := time.Until(deadline)
	if remaining <= 0 {
		t.Fatalf("expected future deadline, got %v", remaining)
	}

	if remaining > 400*time.Millisecond {
		t.Fatalf("expected deadline close to configured timeout, got %v", remaining)
	}

	if _, ok := parentCtx.Deadline(); ok {
		t.Fatal("expected parent context to remain without a deadline")
	}
}

func TestOperationContextHonorsEarlierParentDeadline(t *testing.T) {
	t.Parallel()

	processor := NewProcessor(&types.Config{Timeout: 2 * time.Second})
	parentCtx, parentCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer parentCancel()

	ctx, cancel := processor.operationContext(parentCtx)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected derived context to have a deadline")
	}

	remaining := time.Until(deadline)
	if remaining > 300*time.Millisecond {
		t.Fatalf("expected parent deadline to win, got %v", remaining)
	}
}