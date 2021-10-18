package main

import (
	"errors"
	"reflect"
	"sync"
)

type InMemoryUserStorage struct {
	lock    sync.RWMutex
	storage map[string]User
}

func NewInMemoryUserStorage() *InMemoryUserStorage {
	return &InMemoryUserStorage{
		lock:    sync.RWMutex{},
		storage: make(map[string]User),
	}
}

func (s *InMemoryUserStorage) Add(key string, user User) error {
	if !reflect.DeepEqual(s.storage[key], User{}) {
		return errors.New("Key '" + key + "' already exists")
	}

	s.storage[key] = user
	return nil
}

func (s *InMemoryUserStorage) Update(key string, user User) error {
	if reflect.DeepEqual(s.storage[key], User{}) {
		return errors.New("Key '" + key + "' doesn't exist")
	}

	s.storage[key] = user
	return nil
}

func (s *InMemoryUserStorage) Get(key string) (user User, err error) {
	user, exists := s.storage[key]
	if exists {
		return user, nil
	}
	return (User{}), errors.New("Key '" + key + "' doesn't exist")
}

func (s *InMemoryUserStorage) Delete(key string) (user User, err error) {
	user, exists := s.storage[key]
	if exists {
		delete(s.storage, key)
		return user, nil
	}
	return (User{}), errors.New("Key '" + key + "' doesn't exist")
}
