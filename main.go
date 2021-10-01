package main

import (
	"context"
	"crypto/md5"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
)

func getCakeHandler(w http.ResponseWriter, r *http.Request, u User) {
	w.Write([]byte(u.FavoriteCake))
}

func wrapJwt(
	jwt *JWTService,
	f func(http.ResponseWriter, *http.Request, *JWTService),
) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		f(rw, r, jwt)
	}
}

func (s *UserService) addSuperadmin() error {
	superadminEmail, err := os.LookupEnv("CAKE_ADMIN_EMAIL")
	if !err {
		return errors.New("Undefined superadmin email")
	}
	superadminPassword, err := os.LookupEnv("CAKE_ADMIN_PASSWORD")
	if !err {
		return errors.New("Undefined superadmin password")
	}

	superadmin := User{
		Email:          superadminEmail,
		PasswordDigest: string(md5.New().Sum([]byte(superadminPassword))),
		FavoriteCake:   "napoleon",
		Role:           superadminRole,
	}

	addErr := s.repository.Add(superadmin.Email, superadmin)
	if addErr != nil {
		return addErr
	}

	return nil
}

func main() {
	os.Setenv("CAKE_ADMIN_EMAIL", "superadmin@openware.com")
	os.Setenv("CAKE_ADMIN_PASSWORD", "12345678")

	r := mux.NewRouter()

	users := NewInMemoryUserStorage()
	userService := UserService{
		repository: users,
	}

	userService.addSuperadmin()

	jwtService, err := NewJWTService("pubkey.rsa", "privkey.rsa")
	if err != nil {
		panic(err)
	}

	r.HandleFunc("/cake", logRequest(jwtService.JWTAuth(users, getCakeHandler))).Methods(http.MethodGet)
	r.HandleFunc("/user/me", logRequest(jwtService.JWTAuth(users, getCakeHandler))).Methods(http.MethodGet)
	r.HandleFunc("/user/register", logRequest(userService.Register)).Methods(http.MethodPost)
	r.HandleFunc("/user/favorite_cake", logRequest(jwtService.
		JWTAuth(users, userService.UpdateFavoriteCakeHandler))).Methods(http.MethodPost)
	r.HandleFunc("/user/email", logRequest(jwtService.
		JWTAuth(users, userService.UpdateEmailHandler))).Methods(http.MethodPost)
	r.HandleFunc("/user/password", logRequest(jwtService.
		JWTAuth(users, userService.UpdatePasswordHandler))).Methods(http.MethodPost)
	r.HandleFunc("/user/jwt", logRequest(wrapJwt(jwtService, userService.JWT))).Methods(http.MethodPost)

	r.HandleFunc("/admin/promote", logRequest(jwtService.JWTAuth(users, userService.promoteUser))).Methods(http.MethodPost)
	r.HandleFunc("/admin/fire", logRequest(jwtService.JWTAuth(users, userService.fireUser))).Methods(http.MethodPost)
	r.HandleFunc("/admin/ban", logRequest(jwtService.JWTAuth(users, userService.banUserHandler))).Methods(http.MethodPost)
	r.HandleFunc("/admin/unban", logRequest(jwtService.JWTAuth(users, userService.unbanUserHandler))).Methods(http.MethodPost)
	r.HandleFunc("/admin/inspect", logRequest(jwtService.JWTAuth(users, userService.inspectUserHandler))).Methods(http.MethodGet)

	srv := http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	log.Println("Server started, hit Ctrl+C to stop")
	err = srv.ListenAndServe()
	if err != nil {
		log.Println("Server exited with error: ", err)
	}

	log.Println("Good bye :)")
}
