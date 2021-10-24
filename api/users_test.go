package api

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

const DefaultPassword = "12345678"

func encrypt(str string) string {
	return string(md5.New().Sum([]byte(str)))
}

type parsedResponse struct {
	status int
	body   []byte
}

func createRequester(t *testing.T) func(req *http.Request, err error) parsedResponse {
	return func(req *http.Request, err error) parsedResponse {
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return parsedResponse{}
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return parsedResponse{}
		}

		resp, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return parsedResponse{}
		}

		return parsedResponse{res.StatusCode, resp}
	}
}

func prepareParams(t *testing.T, params map[string]interface{}) io.Reader {
	body, err := json.Marshal(params)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	return bytes.NewBuffer(body)
}

func newTestUserService() *UserService {
	return &UserService{
		repository: NewInMemoryUserStorage(),
	}
}

func newTestJwtService(t *testing.T) *JWTService {
	j, err := NewJWTService("pubkey.rsa", "privkey.rsa")
	if err != nil {
		t.Error("failad to make new jwt service")
	}
	return j
}

func assertStatus(t *testing.T, expected int, r parsedResponse) {
	if r.status != expected {
		t.Errorf("Unexpected response status. Expected: %d, actual: %d", expected, r.status)
	}
}

func assertBody(t *testing.T, expected string, r parsedResponse) {
	actual := string(r.body)
	if actual != expected {
		t.Errorf("Unexpected response body. Expected: %s, actual: %s", expected, actual)
	}
}

func assertResponse(t *testing.T, status int, body string, r parsedResponse) {
	assertStatus(t, status, r)
	assertBody(t, body, r)
}

func randomNum() string {
	return strconv.FormatInt(int64(rand.Intn(1000)), 10)
}

func newUser() User {
	return User{
		Email:          randomNum() + "user@mail.com",
		PasswordDigest: encrypt("12345678"),
		FavoriteCake:   "cheesecake",
		Role:           userRole,
	}
}

func newAdmin() User {
	return User{
		Email:          randomNum() + "admin@mail.com",
		PasswordDigest: encrypt("12345678"),
		FavoriteCake:   "cheesecake",
		Role:           adminRole,
	}
}

func newSuperadmin() User {
	return User{
		Email:          randomNum() + "superadmin@mail.com",
		PasswordDigest: encrypt("12345678"),
		FavoriteCake:   "cheesecake",
		Role:           superadminRole,
	}
}

func TestUsers_JWT(t *testing.T) {
	doRequest := createRequester(t)

	t.Run("write response test", func(t *testing.T) {
		// u := newTestUserService()
		// j := newTestJwtService(t)

		s := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			writeResponse(rw, 200, "hello world")
		}))
		defer s.Close()

		resp := doRequest(http.NewRequest(http.MethodPost, s.URL, nil))
		assertResponse(t, 200, "hello world", resp)
	})

	t.Run("user does not exist", func(t *testing.T) {
		u := newTestUserService()
		j := newTestJwtService(t)

		ts := httptest.NewServer(http.HandlerFunc(WrapJwt(j, u.JWT)))
		defer ts.Close()

		params := map[string]interface{}{
			"email":    "test@mail.com",
			"password": "somepass",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		assertStatus(t, 422, resp)
		assertBody(t, "Key 'test@mail.com' doesn't exist", resp)
	})

	t.Run("wrong password", func(t *testing.T) {
		u := newTestUserService()
		j := newTestJwtService(t)

		jwts := httptest.NewServer(http.HandlerFunc(WrapJwt(j, u.JWT)))
		defer func() {
			jwts.Close()
		}()

		user := newUser()
		u.repository.Add(user.Email, user)

		jwtParams := map[string]interface{}{
			"email":    user.Email,
			"password": "wrongpass",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, jwts.URL, prepareParams(t, jwtParams)))
		assertResponse(t, 422, "invalid login params", resp)
	})

	t.Run("right jwt", func(t *testing.T) {
		u := newTestUserService()
		j := newTestJwtService(t)

		jwts := httptest.NewServer(http.HandlerFunc(WrapJwt(j, u.JWT)))
		defer func() {
			jwts.Close()
		}()

		user := newUser()
		u.repository.Add(user.Email, user)

		jwtParams := map[string]interface{}{
			"email":    user.Email,
			"password": DefaultPassword,
		}

		resp := doRequest(http.NewRequest(http.MethodPost, jwts.URL, prepareParams(t, jwtParams)))
		jwt, _ := j.GenearateJWT(user)
		assertResponse(t, 200, jwt, resp)
	})

	t.Run("update cake", func(t *testing.T) {
		u := newTestUserService()
		j := newTestJwtService(t)

		jwts := httptest.NewServer(http.HandlerFunc(WrapJwt(j, u.JWT)))
		cks := httptest.NewServer(http.HandlerFunc(j.JWTAuth(u.repository, GetCakeHandler)))
		upds := httptest.NewServer(http.HandlerFunc(j.JWTAuth(u.repository, u.UpdateFavoriteCakeHandler)))
		defer func() {
			jwts.Close()
			cks.Close()
			upds.Close()
		}()

		user := newUser()
		u.repository.Add(user.Email, user)

		jwtParams := map[string]interface{}{
			"email":    user.Email,
			"password": DefaultPassword,
		}

		updateParams := map[string]interface{}{
			"favorite_cake": "napoleon",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, jwts.URL, prepareParams(t, jwtParams)))

		getCakeReq, getErr := http.NewRequest(http.MethodGet, cks.URL, nil)
		jwt := string(resp.body)
		getCakeReq.Header.Add(
			"Authorization",
			"Bearer "+jwt,
		)

		resp = doRequest(getCakeReq, getErr)
		assertStatus(t, 200, resp)
		assertBody(t, "cheesecake", resp)

		updateCakeReq, updateErr := http.NewRequest(http.MethodPost, upds.URL, prepareParams(t, updateParams))
		updateCakeReq.Header.Add(
			"Authorization",
			"Bearer "+jwt,
		)

		resp = doRequest(updateCakeReq, updateErr)
		assertStatus(t, http.StatusOK, resp)
		assertBody(t, "favorite cake changed", resp)

		resp = doRequest(getCakeReq, getErr)
		assertStatus(t, 200, resp)
		assertBody(t, "napoleon", resp)
	})

	t.Run("update email", func(t *testing.T) {
		u := newTestUserService()
		j, err := NewJWTService("pubkey.rsa", "privkey.rsa")
		if err != nil {
			t.FailNow()
		}

		jwts := httptest.NewServer(http.HandlerFunc(WrapJwt(j, u.JWT)))
		regs := httptest.NewServer(http.HandlerFunc(u.Register))
		upds := httptest.NewServer(http.HandlerFunc(j.JWTAuth(u.repository, u.UpdateEmailHandler)))
		defer func() {
			jwts.Close()
			regs.Close()
			upds.Close()
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

		updateParams := map[string]interface{}{
			"email": "new@mail.com",
		}

		jwtUpdatedParams := map[string]interface{}{
			"email":    "new@mail.com",
			"password": "somepass",
		}

		doRequest(http.NewRequest(http.MethodPost, regs.URL, prepareParams(t, regParams)))
		resp := doRequest(http.NewRequest(http.MethodPost, jwts.URL, prepareParams(t, jwtParams)))

		jwt := string(resp.body)

		updateEmailReq, updateErr := http.NewRequest(http.MethodPost, upds.URL, prepareParams(t, updateParams))
		updateEmailReq.Header.Add(
			"Authorization",
			"Bearer "+jwt,
		)

		resp = doRequest(updateEmailReq, updateErr)
		assertStatus(t, http.StatusOK, resp)
		assertBody(t, "email changed", resp)

		resp = doRequest(http.NewRequest(http.MethodGet, jwts.URL, prepareParams(t, jwtParams)))
		assertStatus(t, 422, resp)
		assertBody(t, "Key 'test@mail.com' doesn't exist", resp)

		resp = doRequest(http.NewRequest(http.MethodGet, jwts.URL, prepareParams(t, jwtUpdatedParams)))
		assertStatus(t, 200, resp)
		if jwt := string(resp.body); jwt == "Key 'new@mail.com' doesn't exist" {
			t.Errorf("Unexpected response body. Expected jwt, actual: %s", jwt)
		}
	})

	t.Run("update password", func(t *testing.T) {
		u := newTestUserService()
		j, err := NewJWTService("pubkey.rsa", "privkey.rsa")
		if err != nil {
			t.FailNow()
		}

		jwts := httptest.NewServer(http.HandlerFunc(WrapJwt(j, u.JWT)))
		regs := httptest.NewServer(http.HandlerFunc(u.Register))
		upds := httptest.NewServer(http.HandlerFunc(j.JWTAuth(u.repository, u.UpdatePasswordHandler)))
		defer func() {
			jwts.Close()
			regs.Close()
			upds.Close()
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

		updateParams := map[string]interface{}{
			"password": "newpassw",
		}

		jwtUpdatedParams := map[string]interface{}{
			"email":    "test@mail.com",
			"password": "newpassw",
		}

		doRequest(http.NewRequest(http.MethodPost, regs.URL, prepareParams(t, regParams)))
		resp := doRequest(http.NewRequest(http.MethodPost, jwts.URL, prepareParams(t, jwtParams)))

		jwt := string(resp.body)

		updateEmailReq, updateErr := http.NewRequest(http.MethodPost, upds.URL, prepareParams(t, updateParams))
		updateEmailReq.Header.Add(
			"Authorization",
			"Bearer "+jwt,
		)

		resp = doRequest(updateEmailReq, updateErr)
		assertStatus(t, http.StatusOK, resp)
		assertBody(t, "password changed", resp)

		resp = doRequest(http.NewRequest(http.MethodGet, jwts.URL, prepareParams(t, jwtParams)))
		assertStatus(t, 422, resp)
		assertBody(t, "invalid login params", resp)

		resp = doRequest(http.NewRequest(http.MethodGet, jwts.URL, prepareParams(t, jwtUpdatedParams)))
		assertStatus(t, 200, resp)
		if jwt := string(resp.body); jwt == "invalid login params" {
			t.Errorf("Unexpected response body. Expected jwt, actual: %s", jwt)
		}
	})
}
