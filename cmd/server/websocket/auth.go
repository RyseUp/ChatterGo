package websocket

import (
	"errors"
	"net/http"
	"strings"

	"github.com/RyseUp/ChatterGo/utils"
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
