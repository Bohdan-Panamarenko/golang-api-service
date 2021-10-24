package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCake(t *testing.T) {
	doRequest := createRequester(t)

	t.Run("unauthorized", func(t *testing.T) {
		u := newTestUserService()
		j := newTestJwtService(t)

		jwts := httptest.NewServer(http.HandlerFunc(WrapJwt(j, u.JWT)))
		cks := httptest.NewServer(http.HandlerFunc(j.JWTAuth(u.repository, GetCakeHandler)))
		regs := httptest.NewServer(http.HandlerFunc(u.Register))
		defer func() {
			cks.Close()
			regs.Close()
			jwts.Close()
		}()

		resp := doRequest(http.NewRequest(http.MethodGet, cks.URL, nil))
		assertStatus(t, 401, resp)
		assertBody(t, "unauthorized", resp)

		regParams := map[string]interface{}{
			"email":         "test@mail.com",
			"password":      "somepass",
			"favorite_cake": "cheesecake",
		}

		jwtParams := map[string]interface{}{
			"email":    "test@mail.com",
			"password": "somepass",
		}

		doRequest(http.NewRequest(http.MethodPost, regs.URL, prepareParams(t, regParams))) // register

		resp = doRequest(http.NewRequest(http.MethodPost, jwts.URL, prepareParams(t, jwtParams))) // get jwt

		req, err := http.NewRequest(http.MethodGet, cks.URL, nil)
		req.Header.Add(
			"Authorization",
			"Bearer "+string(resp.body),
		)

		u.repository.Delete("test@mail.com")

		resp = doRequest(req, err)
		assertStatus(t, 401, resp)
		assertBody(t, "unauthorized", resp)
	})

	t.Run("authorized", func(t *testing.T) {
		u := newTestUserService()
		j := newTestJwtService(t)

		jwts := httptest.NewServer(http.HandlerFunc(WrapJwt(j, u.JWT)))
		regs := httptest.NewServer(http.HandlerFunc(u.Register))
		cks := httptest.NewServer(http.HandlerFunc(j.JWTAuth(u.repository, GetCakeHandler)))
		defer func() {
			jwts.Close()
			regs.Close()
			cks.Close()
		}()

		regParams := map[string]interface{}{
			"email":         "test@mail.com",
			"password":      "somepass",
			"favorite_cake": "cheesecake",
		}

		jwtParams := map[string]interface{}{
			"email":    "test@mail.com",
			"password": "somepass",
		}

		doRequest(http.NewRequest(http.MethodPost, regs.URL, prepareParams(t, regParams)))
		resp := doRequest(http.NewRequest(http.MethodPost, jwts.URL, prepareParams(t, jwtParams)))

		req, err := http.NewRequest(http.MethodGet, cks.URL, nil)
		req.Header.Add(
			"Authorization",
			"Bearer "+string(resp.body),
		)

		resp = doRequest(req, err)
		assertStatus(t, 200, resp)
		assertBody(t, "cheesecake", resp)
	})
}
