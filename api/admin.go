package api

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"net/http"
	"os"
)

type UserBanParams struct {
	Email  string `json:"email"`
	Reason string `json:"reason"`
}

func validateUserBanParams(p UserBanParams) error {
	err := validateEmail(p.Email)
	if err != nil {
		return err
	}
	return nil
}

func IsSuperadmin(u User) bool {
	return u.Role == superadminRole
}

func IsAdmin(u User) bool {
	return u.Role == adminRole
}

func (s *UserService) PromoteUser(w http.ResponseWriter, r *http.Request, u User) {
	if !IsSuperadmin(u) {
		writeResponse(w, 401, "attempt to acces superadmin api without superadmin rights")
		return
	}

	email, err := readEmail(r)
	if err != nil {
		HandleError(err, w)
		return
	}

	user, err := s.repository.Get(email)
	if err != nil {
		HandleError(err, w)
		return
	}

	if IsSuperadmin(user) {
		writeResponse(w, 401, "attempt to change superadmin rights")
		return
	}

	user.Role = adminRole

	err = s.repository.Update(user.Email, user)
	if err != nil {
		HandleError(err, w)
		return
	}

	writeResponse(w, http.StatusOK, "user "+user.Email+" is admin now")
}

func (s *UserService) FireUser(w http.ResponseWriter, r *http.Request, u User) {
	if !IsSuperadmin(u) {
		writeResponse(w, 401, "attempt to acces superadmin api without superadmin rights")
		return
	}

	email, err := readEmail(r)
	if err != nil {
		HandleError(err, w)
		return
	}

	user, err := s.repository.Get(email)
	if err != nil {
		HandleError(err, w)
		return
	}

	if IsSuperadmin(user) {
		writeResponse(w, 401, "attempt to change superadmin rights")
		return
	}

	user.Role = userRole

	s.repository.Update(user.Email, user)
	if err != nil {
		HandleError(err, w)
		return
	}

	writeResponse(w, http.StatusOK, "user "+user.Email+" is not admin now")
}

func validateAdminAction(w http.ResponseWriter, u User, target User) bool {
	if (u.Role == adminRole || u.Role == superadminRole) && target.Role == userRole ||
		u.Role == superadminRole && target.Role == adminRole && u.Email != target.Email {

		return true
	}

	writeResponse(w, 401, "not enough rights to performe this action")
	return false
}

func (s *UserService) BanUserHandler(w http.ResponseWriter, r *http.Request, u User) {
	params := &UserBanParams{}
	err := json.NewDecoder(r.Body).Decode(params)
	if err != nil {
		HandleError(errors.New("could not read params"), w)
		return
	}

	err = validateUserBanParams(*params)
	if err != nil {
		HandleError(err, w)
		return
	}

	target, err := s.repository.Get(params.Email)
	if err != nil {
		HandleError(err, w)
		return
	}

	if !validateAdminAction(w, u, target) {
		return
	}

	if UserHasBan(target) {
		HandleError(errors.New("user "+target.Email+" is already banned"), w)
		return
	}

	err = s.BanUser(params.Email, u.Email, params.Reason)
	if err != nil {
		HandleError(err, w)
		return
	}

	writeResponse(w, http.StatusOK, "user "+target.Email+" is banned now")
}

func (s *UserService) UnbanUserHandler(w http.ResponseWriter, r *http.Request, u User) {
	params := &UserBanParams{}
	err := json.NewDecoder(r.Body).Decode(params)
	if err != nil {
		HandleError(errors.New("could not read params"), w)
	}

	err = validateUserBanParams(*params)
	if err != nil {
		HandleError(err, w)
	}

	target, err := s.repository.Get(params.Email)
	if err != nil {
		HandleError(err, w)
	}

	if !validateAdminAction(w, u, target) {
		return
	}

	err = s.UnbanUser(params.Email, u.Email)
	if err != nil {
		HandleError(err, w)
		return
	}

	writeResponse(w, http.StatusOK, "user "+target.Email+" is unbanned now")
}

func (s *UserService) InspectUserHandler(w http.ResponseWriter, r *http.Request, u User) {
	if u.Role != adminRole && u.Role != superadminRole {
		writeResponse(w, 401, "not enough rights to performe this action")
		return
	}

	email := r.URL.Query().Get("email")
	err := validateEmail(email)
	if err != nil {
		HandleError(err, w)
		return
	}

	target, err := s.repository.Get(email)
	if err != nil {
		HandleError(err, w)
		return
	}

	history := target.BanHistory
	if history == nil {
		writeResponse(w, http.StatusOK, "user "+email+" does not have any bans")
		return
	}

	response := InspectUser(target)

	writeResponse(w, http.StatusOK, response)
}

func (s *UserService) AddSuperadmin() error {
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
