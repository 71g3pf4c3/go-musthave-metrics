package sign

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestComputeAndEqualHMAC(t *testing.T) {
	body := []byte(`{"id":"cpu","type":"gauge","value":42.5}`)
	key := "test-key"

	h := ComputeHMAC(body, key)
	if h == "" {
		t.Fatal("expected non-empty hash")
	}
	if !EqualHMAC(h, body, key) {
		t.Fatal("expected equal hmac")
	}
	if EqualHMAC(h, body, "other-key") {
		t.Fatal("expected non-equal hmac for different key")
	}
}

func TestMiddleware_ValidSignature(t *testing.T) {
	key := "test-key"
	body := `{"id":"cpu","type":"gauge","value":42.5}`
	goodHash := ComputeHMAC([]byte(body), key)

	h := Middleware(key)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	r.Header.Set(HeaderHashSHA256, goodHash)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestMiddleware_InvalidSignature(t *testing.T) {
	key := "test-key"
	body := `{"id":"cpu","type":"gauge","value":42.5}`

	h := Middleware(key)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	r.Header.Set(HeaderHashSHA256, "deadbeef")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestMiddleware_SignsResponse(t *testing.T) {
	key := "test-key"
	respBody := "hello"

	h := Middleware(key)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(respBody))
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	gotHash := w.Header().Get(HeaderHashSHA256)
	if gotHash == "" {
		t.Fatal("expected response HashSHA256 header")
	}
	if !EqualHMAC(gotHash, []byte(respBody), key) {
		t.Fatal("response hash mismatch")
	}
}

func TestMiddleware_MissingHeaderSkipsVerify(t *testing.T) {
	h := Middleware("test-key")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader("body"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when HashSHA256 header absent, got %d", w.Code)
	}
}

func TestMiddleware_NoKeySkipsAll(t *testing.T) {
	called := false
	h := Middleware("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader("body"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if !called {
		t.Fatal("expected handler to be called when key is empty")
	}
	if w.Header().Get(HeaderHashSHA256) != "" {
		t.Fatal("expected no HashSHA256 header when key is empty")
	}
}
