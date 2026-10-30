package middleware

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"pulsa-backend/util"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AgentSessionVerifier interface {
	ValidateAgentSession(ctx context.Context, userID, agentID int64, role string) (bool, error)
}

// AuthAgent — untuk endpoint agent
func AuthAgent(secret []byte, verifier AgentSessionVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractToken(c)
		if tokenStr == "" {
			util.Unauthorized(c, "authorization token required")
			c.Abort()
			return
		}

		claims, err := parseToken(tokenStr, secret)
		if err != nil {
			util.Log.Warn("[AUTH] invalid token: ", err)
			util.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		role, _ := claims["role"].(string)
		if role != "owner" && role != "employee" {
			util.Forbidden(c, "agent access only")
			c.Abort()
			return
		}

		userID, ok := subjectID(claims)
		if !ok {
			util.Unauthorized(c, "invalid token claims")
			c.Abort()
			return
		}
		agentID, ok := claimID(claims, "agent_id")
		if !ok {
			util.Unauthorized(c, "invalid token claims")
			c.Abort()
			return
		}
		active, err := verifier.ValidateAgentSession(c.Request.Context(), userID, agentID, role)
		if err != nil {
			util.Log.Errorf("[AUTH] agent session validation failed agent=%d user=%d: %v", agentID, userID, err)
			util.InternalError(c, "failed to validate agent session")
			c.Abort()
			return
		}
		if !active {
			util.Unauthorized(c, "agent account is inactive or invalid")
			c.Abort()
			return
		}
		c.Set("agent_id", agentID)
		c.Set("agent_user_id", userID)
		c.Set("agent_role", role)
		c.Next()
	}
}

func RequireAgentRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString("agent_role") != role {
			util.Forbidden(c, role+" access only")
			c.Abort()
			return
		}
		c.Next()
	}
}

// AuthAdmin — untuk endpoint admin
func AuthAdmin(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractToken(c)
		if tokenStr == "" {
			util.Unauthorized(c, "authorization token required")
			c.Abort()
			return
		}

		claims, err := parseToken(tokenStr, secret)
		if err != nil {
			util.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		if role, _ := claims["role"].(string); role != "admin" && role != "superadmin" {
			util.Forbidden(c, "admin access only")
			c.Abort()
			return
		}

		id, ok := subjectID(claims)
		if !ok {
			util.Unauthorized(c, "invalid token claims")
			c.Abort()
			return
		}
		role, _ := claims["role"].(string)
		c.Set("admin_id", id)
		c.Set("admin_role", role)
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if h == "" {
		return ""
	}
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return h
}

func parseToken(tokenStr string, secret []byte) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

func subjectID(claims jwt.MapClaims) (int64, bool) {
	return numericID(claims["sub"])
}

func claimID(claims jwt.MapClaims, name string) (int64, bool) {
	return numericID(claims[name])
}

func numericID(value interface{}) (int64, bool) {
	switch sub := value.(type) {
	case float64:
		id := int64(sub)
		if id <= 0 || float64(id) != sub {
			return 0, false
		}
		return id, true
	case json.Number:
		id, err := sub.Int64()
		return id, err == nil && id > 0
	case string:
		id, err := strconv.ParseInt(sub, 10, 64)
		return id, err == nil && id > 0
	default:
		return 0, false
	}
}
