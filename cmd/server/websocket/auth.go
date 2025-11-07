package websocket

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/RyseUp/ChatterGo/utils"
	socketio "github.com/googollee/go-socket.io"
)

func VerifySocketJWT(r *http.Request) (uint64, error) {
	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}
	if tokenString == "" {
		return 0, errors.New("missing token")
	}

	claims, err := utils.PraseToken(tokenString)
	if err != nil {
		return 0, err
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, errors.New("invalid token payload")
	}

	return uint64(userID), nil
}

// VerifySocketJWTFromConn extracts JWT token from Socket.IO connection
// Checks URL query parameter and Authorization header
// Note: Socket.IO v4 client sends auth.token in the handshake, which may be
// accessible through the connection's request context or query parameters
func VerifySocketJWTFromConn(s socketio.Conn, r *http.Request) (uint64, error) {
	var tokenString string

	// Socket.IO v4 sends auth in the handshake payload, but googollee/go-socket.io
	// may expose it through the request URL or headers during the initial handshake
	// First, try URL query parameter
	tokenString = r.URL.Query().Get("token")
	
	// Check all query parameters for token (in case it's sent differently)
	if tokenString == "" {
		// Check for auth.token in query
		tokenString = r.URL.Query().Get("auth.token")
	}
	
	// Check Authorization header (some clients send it this way)
	if tokenString == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	// For Socket.IO v4, the auth object is sent in the handshake payload
	// The googollee library may not expose it directly, so we rely on
	// query parameters or headers. The frontend should send the token
	// in the query string or Authorization header during connection.

	if tokenString == "" {
		// Log all query parameters for debugging
		queryParams := r.URL.Query()
		return 0, fmt.Errorf("missing token - ensure token is sent in query parameter 'token' or Authorization header. Query params: %v", queryParams)
	}

	claims, err := utils.PraseToken(tokenString)
	if err != nil {
		return 0, fmt.Errorf("failed to parse token: %w", err)
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, errors.New("invalid token payload - missing user_id")
	}

	return uint64(userID), nil
}
