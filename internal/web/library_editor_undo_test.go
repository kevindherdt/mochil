package web

import (
	"errors"
	"testing"
)

func TestEditorUndoKeepsTheTokenUntilTheUndoActuallySucceeds(t *testing.T) {
	undo := &editorUndo{}
	failure := errors.New("the store could not be written")
	token := undo.remember(func() error { return failure })

	run, ok := undo.peek(token)
	if !ok {
		t.Fatal("a freshly remembered batch must be available")
	}
	if err := run(); !errors.Is(err, failure) {
		t.Fatalf("undo error = %v, want the injected failure", err)
	}
	if _, ok := undo.peek(token); !ok {
		t.Fatal("a failed undo must leave the token usable, or a retry answers 410")
	}

	undo.drop(token)
	if _, ok := undo.peek(token); ok {
		t.Fatal("a successful undo must consume the token")
	}

	older := undo.remember(func() error { return nil })
	undo.remember(func() error { return nil })
	if _, ok := undo.peek(older); ok {
		t.Fatal("only the newest batch may be undone")
	}
}
