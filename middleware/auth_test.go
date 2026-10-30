package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type testAgentSessionVerifier bool

func (v testAgentSessionVerifier) ValidateAgentSession(context.Context, int64, int64, string) (bool, error) {
	return bool(v), nil
}

func TestSubjectID(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  int64
		ok    bool
	}{
		{name: "number", value: float64(42), want: 42, ok: true},
		{name: "numeric string", value: "42", want: 42, ok: true},
		{name: "json number", value: json.Number("42"), want: 42, ok: true},
		{name: "zero", value: float64(0), ok: false},
		{name: "fraction", value: float64(1.5), ok: false},
		{name: "invalid string", value: "agent", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := subjectID(jwt.MapClaims{"sub": tt.value})
			if got != tt.want || ok != tt.ok {
				t.Fatalf("subjectID() = (%d, %t), want (%d, %t)", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestParseTokenRequiresHS256(t *testing.T) {
	secret := []byte("test-secret-with-more-than-32-characters")

	validToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      float64(7),
		"agent_id": float64(42),
		"role":     "owner",
		"exp":      jwt.NewNumericDate(time.Now().Add(time.Minute)),
	})
	validString, err := validToken.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseToken(validString, secret); err != nil {
		t.Fatalf("parseToken() rejected valid token: %v", err)
	}

	wrongMethod := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{
		"sub":  float64(42),
		"role": "owner",
	})
	wrongString, err := wrongMethod.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseToken(wrongString, secret); err == nil {
		t.Fatal("parseToken() accepted HS384 token")
	}
}

func TestRequireAgentRoleBlocksEmployeesFromOwnerActions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := []byte("test-secret-with-more-than-32-characters")

	for _, test := range []struct {
		role       string
		wantStatus int
	}{
		{role: "owner", wantStatus: http.StatusNoContent},
		{role: "employee", wantStatus: http.StatusForbidden},
	} {
		t.Run(test.role, func(t *testing.T) {
			token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"sub":      float64(7),
				"agent_id": float64(42),
				"role":     test.role,
				"exp":      jwt.NewNumericDate(time.Now().Add(time.Minute)),
			}).SignedString(secret)
			if err != nil {
				t.Fatal(err)
			}

			router := gin.New()
			router.GET("/owner-action", AuthAgent(secret, testAgentSessionVerifier(true)), RequireAgentRole("owner"), func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodGet, "/owner-action", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}

func TestAuthAgentRejectsInactiveSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := []byte("test-secret-with-more-than-32-characters")
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      float64(7),
		"agent_id": float64(42),
		"role":     "employee",
		"exp":      jwt.NewNumericDate(time.Now().Add(time.Minute)),
	}).SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.GET("/agent", AuthAgent(secret, testAgentSessionVerifier(false)), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/agent", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}
