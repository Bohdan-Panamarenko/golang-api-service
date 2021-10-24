package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUsers_Register(t *testing.T) {
	doRequest := createRequester(t)

	getResp := func(params Params) parsedResponse {
		u := newTestUserService()

		regs := httptest.NewServer(http.HandlerFunc(u.Register))
		defer regs.Close()

		return doRequest(http.NewRequest(http.MethodPost, regs.URL, prepareParams(t, params)))
	}

	t.Run("succesfull registration", func(t *testing.T) {
		regParams := Params{
			"email":         "test@mail.com",
			"password":      "somepass",
			"favorite_cake": "cheesecake",
		}

		resp := getResp(regParams)
		assertStatus(t, 201, resp)
		assertBody(t, "registered", resp)
	})

	t.Run("wrong email", func(t *testing.T) {
		regParams := Params{
			"email":         "wrongemail",
			"password":      "somepass",
			"favorite_cake": "cheesecake",
		}

		resp := getResp(regParams)
		assertStatus(t, 422, resp)
		assertBody(t, "Unvalid email address", resp)
	})

	t.Run("user already exists", func(t *testing.T) {
		u := newTestUserService()

		regs := httptest.NewServer(http.HandlerFunc(u.Register))
		defer regs.Close()

		regParams := Params{
			"email":         "test@mail.com",
			"password":      "somepass",
			"favorite_cake": "cheesecake",
		}

		doRequest(http.NewRequest(http.MethodPost, regs.URL, prepareParams(t, regParams)))
		resp := doRequest(http.NewRequest(http.MethodPost, regs.URL, prepareParams(t, regParams)))
		assertStatus(t, 422, resp)
		assertBody(t, "Key 'test@mail.com' already exists", resp)
	})

	t.Run("wrong registration password", func(t *testing.T) {
		regParams := Params{
			"email":         "test@mail.com",
			"password":      "some",
			"favorite_cake": "cheesecake",
		}

		resp := getResp(regParams)
		assertStatus(t, 422, resp)
		assertBody(t, "Password too short", resp)
	})

	t.Run("favorit cake can not be empty", func(t *testing.T) {
		regParams := Params{
			"email":    "test@mail.com",
			"password": "somepass",
		}

		resp := getResp(regParams)
		assertStatus(t, 422, resp)
		assertBody(t, "Favorit cake can't be empty", resp)
	})

	t.Run("favorit cake can contain only letters", func(t *testing.T) {
		regParams := Params{
			"email":         "test@mail.com",
			"password":      "somepass",
			"favorite_cake": "cheese cake",
		}

		resp := getResp(regParams)
		assertStatus(t, 422, resp)
		assertBody(t, "Favorit cake can contain only letters", resp)
	})
}
