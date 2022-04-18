package server

import (
	"time"
)

func (s *Server) startMessageScheduler() {
	go func() {
		ticker := time.NewTicker(time.Second)
		for {
			<-ticker.C
			s.scheduledMessages.Range(func(id, _msg any) bool {
				msg := _msg.(userMessage)
				if msg.Timestamp.Before(time.Now()) {
					s.scheduledMessages.Delete(id)
					s.pushUserMessageToHistory(msg)
					s.sendMessageToOnlineUsers(msg, false)
				}
				return true
			})
		}
	}()
}
