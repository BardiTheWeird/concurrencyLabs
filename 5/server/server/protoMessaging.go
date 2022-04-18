package server

import (
	"encoding/json"
	"parallel-computations-5/websockets"

	"golang.org/x/exp/slices"
)

type protoMsg struct {
	Kind string      `json:"kind"`
	Data interface{} `json:"data"`
}

func (s *Server) broadcastProtoMsg(kind string, data interface{}) {
	s.currentConnectionMutex.RLock()
	defer s.currentConnectionMutex.RUnlock()

	for _, ws := range s.currentConnection {
		sendProtoMsg(ws, kind, data)
	}
}

func (s *Server) broadcastProtoMsgExcept(kind string, data interface{}, exceptionUsername string) {
	s.currentConnectionMutex.RLock()
	defer s.currentConnectionMutex.RUnlock()

	for username, ws := range s.currentConnection {
		if username != exceptionUsername {
			sendProtoMsg(ws, kind, data)
		}
	}
}

func (s *Server) broadcastProtoMsgFilterUsers(kind string, data interface{}, usernames []string) {
	s.currentConnectionMutex.RLock()
	defer s.currentConnectionMutex.RUnlock()

	for username, ws := range s.currentConnection {
		if slices.Contains(usernames, username) {
			sendProtoMsg(ws, kind, data)
		}
	}
}

func sendProtoMsg(ws *websockets.WebSocketConnection, kind string, data interface{}) {
	response := protoMsg{
		Kind: kind,
		Data: data,
	}
	bytes, _ := json.Marshal(response)
	ws.SendData(bytes)
}
