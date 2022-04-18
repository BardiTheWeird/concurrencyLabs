package server

import (
	"fmt"
	"parallel-computations-5/websockets"
	"unicode"
)

func (s *Server) logIn(ws *websockets.WebSocketConnection, username string) (bool, string) {
	isValid, _ := validateUsername(username)
	if !isValid {
		return false, "bad_username"
	}

	s.currentConnectionMutex.Lock()
	_, userOnline := s.currentConnection[username]
	if userOnline {
		s.currentConnectionMutex.Unlock()
		return false, "already_logged_in"
	}
	s.currentConnection[username] = ws
	s.currentConnectionMutex.Unlock()

	fmt.Println(username, "logged in")
	s.broadcastProtoMsgExcept("user_logged_in", username, username)
	s.users.Store(username, true)
	return true, "ok"
}

func (s *Server) logOut(username string) {
	s.currentConnectionMutex.Lock()
	_, isLoggedIn := s.currentConnection[username]
	if !isLoggedIn {
		fmt.Println("tried logging out", username, "who is not logged in")
		s.currentConnectionMutex.Unlock()
		return
	}
	delete(s.currentConnection, username)
	s.currentConnectionMutex.Unlock()

	fmt.Println(username, "logged out")
	s.broadcastProtoMsgExcept("user_logged_out", username, username)
	s.users.Store(username, false)
}

type user struct {
	Username string `json:"username"`
	Online   bool   `json:"online"`
}

func (s *Server) getAllUsers() []user {
	users := make([]user, 0)
	s.users.Range(func(key, value any) bool {
		users = append(users, user{
			Username: key.(string),
			Online:   value.(bool),
		})
		return true
	})
	return users
}

func validateUsername(s string) (bool, string) {
	minLength := 1
	maxLength := 32

	sRune := []rune(s)
	if len(sRune) < minLength || len(sRune) > maxLength {
		return false, "invalid length"
	}

	for _, r := range sRune {
		if !unicode.IsGraphic(r) {
			return false, "invalid character " + string(r)
		}
	}

	return true, ""
}
