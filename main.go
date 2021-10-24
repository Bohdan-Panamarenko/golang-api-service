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
	"api-service/rabbitMQ"

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

	jobs := make(chan rabbitMQ.BrockerMessage, 10)

	go rabbitMQ.RunSender(10, jobs)
	go rabbitMQ.RunReciever(hub)

	r.HandleFunc("/cake", logging.LogRequest(jwtService.JWTAuth(users, api.GetCakeHandler))).Methods(http.MethodGet)
	r.HandleFunc("/user/me", logging.LogRequest(jwtService.JWTAuth(users, api.GetCakeHandler))).Methods(http.MethodGet)
	r.HandleFunc("/user/register", rabbitMQ.LogRequest(jobs, logging.LogRequest(userService.Register))).Methods(http.MethodPost)
	r.HandleFunc("/user/favorite_cake", logging.LogRequest(jwtService.
		JWTAuth(users, userService.UpdateFavoriteCakeHandler))).Methods(http.MethodPost)
	r.HandleFunc("/user/email", rabbitMQ.LogRequest(jobs, logging.LogRequest(jwtService.
		JWTAuth(users, userService.UpdateEmailHandler)))).Methods(http.MethodPost)
	r.HandleFunc("/user/password", rabbitMQ.LogRequest(jobs, logging.LogRequest(jwtService.
		JWTAuth(users, userService.UpdatePasswordHandler)))).Methods(http.MethodPost)
	r.HandleFunc("/user/jwt", logging.LogRequest(api.WrapJwt(jwtService, userService.JWT))).Methods(http.MethodPost)

	r.HandleFunc("/admin/promote", rabbitMQ.LogRequest(jobs, logging.LogRequest(jwtService.JWTAuth(users, userService.PromoteUser)))).Methods(http.MethodPost)
	r.HandleFunc("/admin/fire", rabbitMQ.LogRequest(jobs, logging.LogRequest(jwtService.JWTAuth(users, userService.FireUser)))).Methods(http.MethodPost)
	r.HandleFunc("/admin/ban", rabbitMQ.LogRequest(jobs, logging.LogRequest(jwtService.JWTAuth(users, userService.BanUserHandler)))).Methods(http.MethodPost)
	r.HandleFunc("/admin/unban", rabbitMQ.LogRequest(jobs, logging.LogRequest(jwtService.JWTAuth(users, userService.UnbanUserHandler)))).Methods(http.MethodPost)
	r.HandleFunc("/admin/inspect", logging.LogRequest(jwtService.JWTAuth(users, userService.InspectUserHandler))).Methods(http.MethodGet)

	r.HandleFunc("/ws", jwtService.JWTAuth(users, cake_websocket.WsHandler(hub)))

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
