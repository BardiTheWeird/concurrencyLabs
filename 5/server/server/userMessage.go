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

func (s *Server) sendMessageToOnlineUsers(msg userMessage) {
	s.currentConnectionMutex.RLock()
	defer s.currentConnectionMutex.RUnlock()

	for username, ws := range s.currentConnection {
		if msg.isReceiver(username) && username != msg.Sender {
			sendProtoMsg(ws, "new_message", msg)
		}
	}
}

func (s *Server) pushUserMessageToHistory(msg userMessage) userMessage {
	s.messagesMutex.Lock()
	defer s.messagesMutex.Unlock()

	msg.Timestamp = time.Now()
	msg.Id = len(s.messages) + 1
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

// transported as
// sender;receiver1,receiver2;sendTime;B64(body)\n
// func (msg *UserMessage) ToDTO() []byte {
// 	// sender
// 	var dto []byte = []byte(msg.SenderName + ";")
// 	// receivers
// 	dto = append(dto, []byte(strings.Join(msg.ReceiversNames, ",")+";")...)

// 	// send time
// 	t, _ := msg.SendTime.MarshalText()
// 	dto = append(dto, t...)
// 	dto = append(dto, ';')

// 	// body
// 	bodyBytes := []byte(msg.Body)
// 	b64Body := make([]byte, base64.StdEncoding.EncodedLen(len(bodyBytes)))
// 	base64.StdEncoding.Encode(b64Body, bodyBytes)
// 	dto = append(dto, b64Body...)

// 	return append(dto, '\n')
// }

// func (s *Server) AppendUserMessage(msg UserMessage) time.Time {
// 	s.messagesMutex.Lock()
// 	defer s.messagesMutex.Unlock()
// 	msg.SendTime = time.Now()
// 	s.messages = append(s.messages, msg)
// 	return msg.SendTime
// }
