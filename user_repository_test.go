package main

import (
	"testing"
)

func makeTestError(t *testing.T, expected, actual string) {
	t.Errorf("Unexpected response status. Expected: %s, actual: %s", expected, actual)
}

func TestUsers_Repository(t *testing.T) {

	t.Run("test add", func(t *testing.T) {
		users := NewInMemoryUserStorage()

		user := User{
			Email:          "test@mail.com",
			PasswordDigest: "passtest",
			FavoriteCake:   "cheesecake",
		}

		users.Add(user.Email, user)

		if users.storage[user.Email] != user {
			t.Errorf("User %v has not been added", user.Email)
		}

		err := users.Add(user.Email, user)
		if err == nil {
			t.Errorf("Expected `Key 'test@mail.com' already exists` error")
		}
	})

	t.Run("test get", func(t *testing.T) {
		users := NewInMemoryUserStorage()

		user := User{
			Email:          "test@mail.com",
			PasswordDigest: "passtest",
			FavoriteCake:   "cheesecake",
		}

		users.Add(user.Email, user)

		u, err := users.Get(user.Email)

		if u != user || err != nil {
			t.Errorf("Expected '%s' user and 'nil' error\n But get '%s' user and '%s' error",
				user.Email, u.Email, err)
		}

		u, err = users.Get("wrongemail")
		if u != (User{}) || err == nil {
			t.Errorf("Expected empty user and `Key 'wrongemail' doesn't exist`\n"+
				"But get '%s' user and '%s' error", u.Email, err)
		}
	})

	t.Run("test update", func(t *testing.T) {
		users := NewInMemoryUserStorage()

		oldUser := User{
			Email:          "test@mail.com",
			PasswordDigest: "passtest",
			FavoriteCake:   "cheesecake",
		}

		newUser := User{
			Email:          "test@mail.com",
			PasswordDigest: "testpass",
			FavoriteCake:   "napoleon",
		}

		users.Add(oldUser.Email, oldUser)

		err := users.Update(newUser.Email, newUser)

		actualUser, getErr := users.Get(oldUser.Email)

		if actualUser == oldUser || actualUser != newUser || err != nil || getErr != nil {
			t.Errorf("Expected %v user\nBut got %v user", newUser, actualUser)
		}

		err = users.Update("wrongemail", newUser)
		if err == nil {
			t.Errorf("Expected `Key 'wrongemail' does not exist` but got 'nil'")
		}
	})

	t.Run("test add", func(t *testing.T) {
		users := NewInMemoryUserStorage()

		user := User{
			Email:          "test@mail.com",
			PasswordDigest: "passtest",
			FavoriteCake:   "cheesecake",
		}

		users.Add(user.Email, user)

		users.Delete(user.Email)

		if val, _ := users.Get(user.Email); val != (User{}) {
			t.Errorf("Expected empty user\nBut got %v", val)
		}

		_, err := users.Delete(user.Email)
		if err == nil {
			t.Errorf("Expected `Key 'test@mail.com' does not exists` error but got 'nil'")
		}
	})
}
