package apperrors

import (
	"errors"
	"testing"
)

func TestNew_WithoutParams(t *testing.T) {
	// Arrange
	expected := "authentication token is empty"

	// Act
	err := New(ErrEmptyToken)

	// Assert
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestNew_WithSingleParam(t *testing.T) {
	// Arrange
	expected := "HTTP error [status_code=500]"

	// Act
	err := New(ErrHTTPError, P("status_code", 500))

	// Assert
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestNew_WithMultipleParams(t *testing.T) {
	// Arrange
	expected := "SAT error [code=5003, message=rate limited]"

	// Act
	err := New(ErrSATError, P("code", "5003"), P("message", "rate limited"))

	// Assert
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestWrap_WithCause(t *testing.T) {
	// Arrange
	cause := errors.New("connection refused")
	expected := "network error [url=https://sat.gob.mx]: connection refused"

	// Act
	err := Wrap(ErrNetworkError, cause, P("url", "https://sat.gob.mx"))

	// Assert
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestWrap_Unwrap(t *testing.T) {
	// Arrange
	cause := errors.New("original error")

	// Act
	err := Wrap(ErrAuthFailed, cause)

	// Assert
	if !errors.Is(err, cause) {
		t.Error("errors.Is should match the wrapped cause")
	}
}

func TestWrap_WithoutParams(t *testing.T) {
	// Arrange
	cause := errors.New("timeout")
	expected := "authentication failed: timeout"

	// Act
	err := Wrap(ErrAuthFailed, cause)

	// Assert
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestNew_ImplementsErrorInterface(t *testing.T) {
	// Act
	var err error = New(ErrEmptyToken)

	// Assert
	if err == nil {
		t.Error("AppError should implement error interface")
	}
}

func TestP_CreatesParam(t *testing.T) {
	// Act
	p := P("key", "value")

	// Assert
	if p.Key != "key" || p.Value != "value" {
		t.Errorf("expected key=key, value=value, got key=%s, value=%v", p.Key, p.Value)
	}
}

func TestWrap_NestedErrors(t *testing.T) {
	// Arrange
	root := errors.New("disk full")
	mid := Wrap(ErrReadBody, root)
	outer := Wrap(ErrAuthFailed, mid, P("operation", "CheckStatus"))

	// Act & Assert
	if !errors.Is(outer, root) {
		t.Error("errors.Is should match deeply nested cause")
	}

	expected := "authentication failed [operation=CheckStatus]: reading body error: disk full"
	if outer.Error() != expected {
		t.Errorf("expected %q, got %q", expected, outer.Error())
	}
}
