package main

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/openware/rango/pkg/auth"
)

type JWTService struct {
	keys *auth.KeyStore
}

func NewJWTService(privKeyPath, pubKeyPath string) (*JWTService, error) {
	keys, err := auth.LoadOrGenerateKeys(privKeyPath, pubKeyPath)
	if err != nil {
		return nil, err
	}

	return &JWTService{keys: keys}, nil
}

func (j *JWTService) GenearateJWT(u User) (string, error) {
	return auth.ForgeToken("empty", u.Email, "empty", 0, j.keys.
		PrivateKey, nil)
}

func (j *JWTService) ParseJWT(jwt string) (auth.Auth, error) {
	return auth.ParseAndValidate(jwt, j.keys.PublicKey)
}

type JWTParams struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (u *UserService) JWT(
	w http.ResponseWriter,
	r *http.Request,
	jwtService *JWTService,
) {
	params := &JWTParams{}
	err := json.NewDecoder(r.Body).Decode(params)
	if err != nil {
		handleError(errors.New("could not read params"), w)
		return
	}

	passwordDigest := md5.New().Sum([]byte(params.Password))
	user, err := u.repository.Get(params.Email)
	if err != nil {
		handleError(err, w)
		return
	}

	if UserHasBan(user) {
		writeResponse(w, 401, "user "+user.Email+" has ban due to '"+user.BanHistory[len(user.BanHistory)-1].WhyBanned+"'")
		return
	}

	if string(passwordDigest) != user.PasswordDigest {
		handleError(errors.New("invalid login params"), w)
		return
	}

	token, err := jwtService.GenearateJWT(user)
	if err != nil {
		handleError(err, w)
		return
	}

	writeResponse(w, http.StatusOK, token)
}

type ProtectedHandler func(rw http.ResponseWriter, r *http.Request, u User)

func (j *JWTService) JWTAuth(
	users UserRepository,
	h ProtectedHandler,
) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")

		auth, err := j.ParseJWT(token)
		if err != nil {
			rw.WriteHeader(401)
			rw.Write([]byte("unauthorized"))
			return
		}

		user, err := users.Get(auth.Email)
		if err != nil {
			rw.WriteHeader(401)
			rw.Write([]byte("unauthorized"))
			return
		}

		if UserHasBan(user) {
			writeResponse(rw, 401, "user "+user.Email+" has ban due to '"+user.BanHistory[len(user.BanHistory)-1].WhyBanned+"'")
			return
		}

		h(rw, r, user)
	}
}
