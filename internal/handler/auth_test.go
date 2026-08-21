package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"realtime-chat-api/internal/handler"
	"realtime-chat-api/internal/repository"
	"realtime-chat-api/internal/service"
)

const testJWTSecret = "test-secret-for-handler-tests-32bytes!"

func setupAuthHandler(t *testing.T) http.Handler {
	t.Helper()
	repo := repository.NewMockUserRepository()
	auth, err := service.NewAuthService(repo, []byte(testJWTSecret))
	if err != nil {
		t.Fatalf("NewAuthService failed: %v", err)
	}
	h := handler.NewAuthHandler(auth)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

func doRequest(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

// --- Register tests ---

func TestRegister_Success(t *testing.T) {
	h := setupAuthHandler(t)
	w := doRequest(h, http.MethodPost, "/register", `{"username":"alice","password":"secret123"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"alice"`) {
		t.Errorf("response should contain username: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"id"`) {
		t.Errorf("response should contain id: %s", w.Body.String())
	}
}

func TestRegister_Duplicate(t *testing.T) {
	h := setupAuthHandler(t)
	doRequest(h, http.MethodPost, "/register", `{"username":"alice","password":"secret123"}`)
	w := doRequest(h, http.MethodPost, "/register", `{"username":"alice","password":"secret123"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegister_InvalidJSON(t *testing.T) {
	h := setupAuthHandler(t)
	w := doRequest(h, http.MethodPost, "/register", `{bad json`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegister_EmptyFields(t *testing.T) {
	h := setupAuthHandler(t)
	w := doRequest(h, http.MethodPost, "/register", `{"username":"","password":"secret123"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegister_WrongMethod(t *testing.T) {
	h := setupAuthHandler(t)
	w := doRequest(h, http.MethodGet, "/register", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Login tests ---

func TestLogin_Success(t *testing.T) {
	h := setupAuthHandler(t)
	doRequest(h, http.MethodPost, "/register", `{"username":"alice","password":"secret123"}`)
	w := doRequest(h, http.MethodPost, "/login", `{"username":"alice","password":"secret123"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"token"`) {
		t.Errorf("response should contain token: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"user_id"`) {
		t.Errorf("response should contain user_id: %s", w.Body.String())
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	h := setupAuthHandler(t)
	doRequest(h, http.MethodPost, "/register", `{"username":"alice","password":"secret123"}`)
	w := doRequest(h, http.MethodPost, "/login", `{"username":"alice","password":"wrongpass"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	h := setupAuthHandler(t)
	w := doRequest(h, http.MethodPost, "/login", `{"username":"nobody","password":"secret123"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogin_InvalidJSON(t *testing.T) {
	h := setupAuthHandler(t)
	w := doRequest(h, http.MethodPost, "/login", `{bad json`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogin_WrongMethod(t *testing.T) {
	h := setupAuthHandler(t)
	w := doRequest(h, http.MethodGet, "/login", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", w.Code, w.Body.String())
	}
}
