package middleware

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

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
		"sub":  float64(42),
		"role": "agent",
		"exp":  jwt.NewNumericDate(time.Now().Add(time.Minute)),
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
		"role": "agent",
	})
	wrongString, err := wrongMethod.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseToken(wrongString, secret); err == nil {
		t.Fatal("parseToken() accepted HS384 token")
	}
}
