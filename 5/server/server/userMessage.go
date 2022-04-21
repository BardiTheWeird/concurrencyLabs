package server

import (
	"encoding/json"
	"time"

	"golang.org/x/exp/slices"
)

type userMessage struct {
	Id        int       `json:"id"`
	Sender    string    `json:"sender,omitempty"`
	Receivers []string  `json:"receivers,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Body      string    `json:"body"`
}

func (u *userMessage) toTransport() []byte {
	bytes, _ := json.Marshal(u)
	return bytes
}

func (s *Server) sendMessageToOnlineUsers(msg userMessage, exceptSender bool) {
	s.currentConnectionMutex.RLock()
	defer s.currentConnectionMutex.RUnlock()

	for username, ws := range s.currentConnection {
		if msg.isReceiver(username) && (!exceptSender || username != msg.Sender) {
			sendProtoMsg(ws, "new_message", msg)
		}
	}
}

func (s *Server) pushUserMessageToHistory(msg userMessage) userMessage {
	s.messagesMutex.Lock()
	defer s.messagesMutex.Unlock()

	msg.Timestamp = time.Now()
	s.messages = append(s.messages, msg)

	return msg
}

func (s *Server) getUserMessageHistory(username string) []userMessage {
	messages := make([]userMessage, 0)
	s.messagesMutex.RLock()

	for _, msg := range s.messages {
		if msg.isVisibleToUser(username) {
			messages = append(messages, msg)
		}
	}

	s.messagesMutex.RUnlock()
	return messages
}

func (u *userMessage) isVisibleToUser(username string) bool {
	return u.isReceiver(username) || u.Sender == username
}

func (u *userMessage) isReceiver(username string) bool {
	return len(u.Receivers) == 0 || slices.Contains(u.Receivers, username)
}
