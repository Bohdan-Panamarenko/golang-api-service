package main

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

type Ban struct {
	WhoBanned    string
	WhenBanned   int64
	WhyBanned    string
	WhoUnbanned  string
	WhenUnbanned int64
}

type User struct {
	Email          string
	PasswordDigest string
	Role           Role
	FavoriteCake   string
	BanHistory     []Ban
}

func UserHasBan(u User) bool {
	if u.BanHistory == nil || len(u.BanHistory) == 0 || (u.BanHistory)[len(u.BanHistory)-1].WhenUnbanned != 0 {
		return false
	}
	return true
}

func (s *UserService) BanUser(key string, adminEmail string, reason string) error {
	u, err := s.repository.Get(key)
	if err != nil {
		return err
	}

	if UserHasBan(u) {
		return errors.New("user " + u.Email + " already have ban")
	}

	u.BanHistory = append(u.BanHistory, Ban{
		WhoBanned:  adminEmail,
		WhenBanned: time.Now().UnixNano(),
		WhyBanned:  reason,
	})

	err = s.repository.Update(u.Email, u)
	if err != nil {
		return err
	}

	return nil
}

func (s *UserService) UnbanUser(key string, adminEmail string) error {
	u, err := s.repository.Get(key)
	if err != nil {
		return err
	}

	if !UserHasBan(u) {
		return errors.New("user " + u.Email + " does not have any active bans")
	}

	lastBan := u.BanHistory[len(u.BanHistory)-1]

	lastBan.WhoUnbanned = adminEmail
	lastBan.WhenUnbanned = time.Now().UnixNano()

	u.BanHistory[len(u.BanHistory)-1] = lastBan

	err = s.repository.Update(u.Email, u)
	if err != nil {
		return err
	}

	return nil
}

func InspectUser(user User) string {
	var response string = "user " + user.Email + " have next bans: \n"

	for _, ban := range user.BanHistory {
		response += fmt.Sprintf("-- Banned %v by %s because '%s'.", ban.WhenBanned, ban.WhoBanned, ban.WhyBanned)
		if ban.WhenUnbanned != 0 {
			response += fmt.Sprintf(" Unbanned %v by %s.", ban.WhenUnbanned, ban.WhoUnbanned)
		}
		response += "\n"
	}

	return response
}

type UserRepository interface {
	Add(string, User) error
	Get(string) (User, error)
	Update(string, User) error
	Delete(string) (User, error)
}

type UserService struct {
	repository UserRepository
}

type UserRegisterParams struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	FavoriteCake string `json:"favorite_cake"`
}

type EmailParams struct {
	Email string `json:"email"`
}

type CakeParams struct {
	Cake string `json:"favorite_cake"`
}

type PasswordParams struct {
	Password string `json:"password"`
}

func validateRegisterParams(p *UserRegisterParams) error {
	err := validateEmail(p.Email)
	if err != nil {
		return err
	}

	err = validatePassword(p.Password)
	if err != nil {
		return err
	}

	return validateCake(p.FavoriteCake)
}

func validateEmail(email string) error {
	// 1. Email is valid
	match, _ := regexp.Match(`(^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+$)`, []byte(email))
	if !match {
		return errors.New("Unvalid email address")
	}
	return nil
}

func validatePassword(password string) error {
	// 2. Password at least 8 symbols
	if len(password) < 8 {
		return errors.New("Password too short")
	}
	return nil
}

func validateCake(cake string) error {
	// 3. Favorite cake not empty
	if len(cake) == 0 {
		return errors.New("Favorit cake can't be empty")
	}

	// 4. Favorite cake only alphabetic
	match, _ := regexp.Match(`\W`, []byte(cake))
	if match {
		return errors.New("Favorit cake can contain only letters")
	}

	return nil
}

func (u *UserService) Register(w http.ResponseWriter, r *http.Request) {
	params, err := readRegisterParams(r)
	if err != nil {
		handleError(err, w)
		return
	}

	if err := validateRegisterParams(params); err != nil {
		handleError(err, w)
		return
	}

	passwordDigest := md5.New().Sum([]byte(params.Password))
	newUser := User{
		Email:          params.Email,
		PasswordDigest: string(passwordDigest),
		FavoriteCake:   params.FavoriteCake,
		Role:           userRole,
	}

	err = u.repository.Add(params.Email, newUser)
	if err != nil {
		handleError(err, w)
		return
	}

	writeResponse(w, http.StatusCreated, "registered")
}

func (u *UserService) UpdateFavoriteCakeHandler(w http.ResponseWriter, r *http.Request, user User) {
	cake, err := readCake(r)
	if err != nil {
		handleError(err, w)
		return
	}

	if err := validateCake(cake); err != nil {
		handleError(err, w)
		return
	}

	newUser := user
	newUser.FavoriteCake = cake

	err = u.repository.Update(newUser.Email, newUser)
	if err != nil {
		handleError(err, w)
		return
	}

	writeResponse(w, http.StatusOK, "favorite cake changed")
}

func (u *UserService) UpdateEmailHandler(w http.ResponseWriter, r *http.Request, user User) {
	email, err := readEmail(r)
	if err != nil {
		handleError(err, w)
		return
	}

	if err := validateEmail(email); err != nil {
		handleError(err, w)
		return
	}

	newUser := user
	newUser.Email = email

	_, err = u.repository.Delete(user.Email)
	if err != nil {
		handleError(err, w)
		return
	}

	err = u.repository.Add(newUser.Email, newUser)
	if err != nil {
		handleError(err, w)
		return
	}

	writeResponse(w, http.StatusOK, "email changed")
}

func (u *UserService) UpdatePasswordHandler(w http.ResponseWriter, r *http.Request, user User) {
	password, err := readPassword(r)
	if err != nil {
		handleError(err, w)
		return
	}

	if err := validatePassword(password); err != nil {
		handleError(err, w)
		return
	}

	newUser := user
	newUser.PasswordDigest = string(md5.New().Sum([]byte(password)))

	err = u.repository.Update(newUser.Email, newUser)
	if err != nil {
		handleError(err, w)
		return
	}

	writeResponse(w, http.StatusOK, "password changed")
}

func handleError(err error, w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnprocessableEntity)
	w.Write([]byte(err.Error()))
}

func readRegisterParams(r *http.Request) (*UserRegisterParams, error) {
	params := &UserRegisterParams{}

	err := json.NewDecoder(r.Body).Decode(params)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("could not read params")
	}

	return params, nil
}

func readEmail(r *http.Request) (string, error) {
	email := &EmailParams{}

	err := json.NewDecoder(r.Body).Decode(email)
	if err != nil {
		fmt.Println(err)
		return "", errors.New("could not read params")
	}

	return email.Email, nil
}

func readPassword(r *http.Request) (string, error) {
	password := &PasswordParams{}

	err := json.NewDecoder(r.Body).Decode(password)
	if err != nil {
		fmt.Println(err)
		return "", errors.New("could not read params")
	}

	return password.Password, nil
}

func readCake(r *http.Request) (string, error) {
	cake := &CakeParams{}

	err := json.NewDecoder(r.Body).Decode(cake)
	if err != nil {
		fmt.Println(err)
		return "", errors.New("could not read params")
	}

	return cake.Cake, nil
}

func writeResponse(w http.ResponseWriter, status int, response string) {
	w.WriteHeader(status)
	w.Write([]byte(response))
}
