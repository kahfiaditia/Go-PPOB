package middleware

import (
	"strings"

	"pulsa-backend/util"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthAgent — untuk endpoint agent
func AuthAgent(secret []byte) gin.HandlerFunc {
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

		if role, _ := claims["role"].(string); role != "agent" {
			util.Forbidden(c, "agent access only")
			c.Abort()
			return
		}

		c.Set("agent_id", int64(claims["sub"].(float64)))
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

		c.Set("admin_id", int64(claims["sub"].(float64)))
		c.Set("admin_role", claims["role"].(string))
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
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
