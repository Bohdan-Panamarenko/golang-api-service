package main

import (
	"encoding/json"
	"errors"
	"net/http"
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

func isSuperadmin(u User) bool {
	if u.Role != superadminRole {
		return false
	}
	return true
}

func (s *UserService) promoteUser(w http.ResponseWriter, r *http.Request, u User) {
	if !isSuperadmin(u) {
		writeResponse(w, 401, "attempt to acces superadmin api without superadmin rights")
		return
	}

	email, err := readEmail(r)
	if err != nil {
		handleError(err, w)
		return
	}

	user, err := s.repository.Get(email)
	if err != nil {
		handleError(err, w)
		return
	}

	if isSuperadmin(user) {
		writeResponse(w, 401, "attempt to change superadmin rights")
		return
	}

	user.Role = adminRole

	err = s.repository.Update(user.Email, user)
	if err != nil {
		handleError(err, w)
		return
	}

	writeResponse(w, http.StatusOK, "user "+user.Email+" is admin now")
}

func (s *UserService) fireUser(w http.ResponseWriter, r *http.Request, u User) {
	if !isSuperadmin(u) {
		writeResponse(w, 401, "attempt to acces superadmin api without superadmin rights")
		return
	}

	email, err := readEmail(r)
	if err != nil {
		handleError(err, w)
		return
	}

	user, err := s.repository.Get(email)
	if err != nil {
		handleError(err, w)
		return
	}

	if isSuperadmin(user) {
		writeResponse(w, 401, "attempt to change superadmin rights")
		return
	}

	user.Role = userRole

	s.repository.Update(user.Email, user)
	if err != nil {
		handleError(err, w)
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

func (s *UserService) banUserHandler(w http.ResponseWriter, r *http.Request, u User) {
	params := &UserBanParams{}
	err := json.NewDecoder(r.Body).Decode(params)
	if err != nil {
		handleError(errors.New("could not read params"), w)
		return
	}

	err = validateUserBanParams(*params)
	if err != nil {
		handleError(err, w)
		return
	}

	target, err := s.repository.Get(params.Email)
	if err != nil {
		handleError(err, w)
		return
	}

	if !validateAdminAction(w, u, target) {
		return
	}

	if UserHasBan(target) {
		handleError(errors.New("user "+target.Email+" is already banned"), w)
		return
	}

	err = s.BanUser(params.Email, u.Email, params.Reason)
	if err != nil {
		handleError(err, w)
		return
	}

	writeResponse(w, http.StatusOK, "user "+target.Email+" is banned now")
}

func (s *UserService) unbanUserHandler(w http.ResponseWriter, r *http.Request, u User) {
	params := &UserBanParams{}
	err := json.NewDecoder(r.Body).Decode(params)
	if err != nil {
		handleError(errors.New("could not read params"), w)
	}

	err = validateUserBanParams(*params)
	if err != nil {
		handleError(err, w)
	}

	target, err := s.repository.Get(params.Email)
	if err != nil {
		handleError(err, w)
	}

	if !validateAdminAction(w, u, target) {
		return
	}

	err = s.UnbanUser(params.Email, u.Email)
	if err != nil {
		handleError(err, w)
		return
	}

	writeResponse(w, http.StatusOK, "user "+target.Email+" is unbanned now")
}

func (s *UserService) inspectUserHandler(w http.ResponseWriter, r *http.Request, u User) {
	if u.Role != adminRole && u.Role != superadminRole {
		writeResponse(w, 401, "not enough rights to performe this action")
		return
	}

	email := r.URL.Query().Get("email")
	err := validateEmail(email)
	if err != nil {
		handleError(err, w)
		return
	}

	target, err := s.repository.Get(email)
	if err != nil {
		handleError(err, w)
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
