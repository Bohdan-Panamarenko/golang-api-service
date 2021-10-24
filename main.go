package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"api-service/api"
	"api-service/cake_websocket"
	"api-service/logging"

	"github.com/gorilla/mux"
)

func main() {
	os.Setenv("CAKE_ADMIN_EMAIL", "superadmin@openware.com")
	os.Setenv("CAKE_ADMIN_PASSWORD", "12345678")

	hub := cake_websocket.NewHub()
	go hub.Run()

	r := mux.NewRouter()

	users := api.NewInMemoryUserStorage()
	userService := api.NewUserService(users)

	userService.AddSuperadmin()

	jwtService, err := api.NewJWTService("pubkey.rsa", "privkey.rsa")
	if err != nil {
		panic(err)
	}

	r.HandleFunc("/cake", logging.LogRequest(jwtService.JWTAuth(users, api.GetCakeHandler))).Methods(http.MethodGet)
	r.HandleFunc("/user/me", logging.LogRequest(jwtService.JWTAuth(users, api.GetCakeHandler))).Methods(http.MethodGet)
	r.HandleFunc("/user/register", logging.LogRequest(userService.Register)).Methods(http.MethodPost)
	r.HandleFunc("/user/favorite_cake", logging.LogRequest(jwtService.
		JWTAuth(users, userService.UpdateFavoriteCakeHandler))).Methods(http.MethodPost)
	r.HandleFunc("/user/email", logging.LogRequest(jwtService.
		JWTAuth(users, userService.UpdateEmailHandler))).Methods(http.MethodPost)
	r.HandleFunc("/user/password", logging.LogRequest(jwtService.
		JWTAuth(users, userService.UpdatePasswordHandler))).Methods(http.MethodPost)
	r.HandleFunc("/user/jwt", logging.LogRequest(api.WrapJwt(jwtService, userService.JWT))).Methods(http.MethodPost)

	r.HandleFunc("/admin/promote", logging.LogRequest(jwtService.JWTAuth(users, userService.PromoteUser))).Methods(http.MethodPost)
	r.HandleFunc("/admin/fire", logging.LogRequest(jwtService.JWTAuth(users, userService.FireUser))).Methods(http.MethodPost)
	r.HandleFunc("/admin/ban", logging.LogRequest(jwtService.JWTAuth(users, userService.BanUserHandler))).Methods(http.MethodPost)
	r.HandleFunc("/admin/unban", logging.LogRequest(jwtService.JWTAuth(users, userService.UnbanUserHandler))).Methods(http.MethodPost)
	r.HandleFunc("/admin/inspect", logging.LogRequest(jwtService.JWTAuth(users, userService.InspectUserHandler))).Methods(http.MethodGet)

	r.HandleFunc("/ws", jwtService.JWTAuth(users, cake_websocket.WsHandshakeHandler(hub)))

	go hub.SendMessages(5 * time.Second)

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
