package server

import (
	"encoding/json"
	"fmt"
	"net"
	"parallel-computations-5/websockets"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Server struct {
	messages      []userMessage
	messagesMutex sync.RWMutex

	// id -> userMessage
	scheduledMessages sync.Map

	getNextMessageId func() int

	// username -> connection
	currentConnection      map[string]*websockets.WebSocketConnection
	currentConnectionMutex sync.RWMutex

	// username -> isOnline
	users sync.Map
}

func InitializeServer() *Server {
	srv := Server{}
	srv.currentConnection = make(map[string]*websockets.WebSocketConnection)
	srv.getNextMessageId = func() func() int {
		var id int32 = 0
		return func() int {
			curId := atomic.LoadInt32(&id)
			atomic.AddInt32(&id, 1)
			return int(curId)
		}
	}()
	return &srv
}

type Connection struct {
	net.Conn
	writeMutex sync.Mutex
}

func (c *Connection) Write(b []byte) (int, error) {
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()
	return c.Conn.Write(b)
}

func (s *Server) Start() {
	listener, _ := net.Listen("tcp", "127.0.0.1:8080")
	fmt.Println("listening on", listener.Addr())
	s.startMessageScheduler()

	for {
		conn, err := listener.Accept()
		if err != nil {
			break
		}
		go func() {
			ws, err := websockets.UpgradeFromTCP(conn)
			if err != nil {
				fmt.Println("error creating connection:", err)
			} else {
				fmt.Println("successfully created a ws connection!")
			}
			s.handleWsMessages(ws)
		}()
	}
}

func (s *Server) handleWsMessages(ws *websockets.WebSocketConnection) {
	username := ""
	for {
		msg := <-ws.C
		if msg.ConnectionClosed {
			if username != "" {
				s.logOut(username)
			}
			return
		}
		fmt.Println("received", string(msg.Payload))

		var clientMessage protoMsg
		json.Unmarshal(msg.Payload, &clientMessage)

		switch clientMessage.Kind {
		case "log_in":
			_username, isString := clientMessage.Data.(string)
			if !isString {
				sendProtoMsg(ws, "login_status", "bad_username")
				break
			}
			_username = strings.TrimSpace(_username)

			loggedIn, statusMsg := s.logIn(ws, _username)
			sendProtoMsg(ws, "login_status", statusMsg)
			if !loggedIn {
				break
			}
			username = _username
			sendProtoMsg(ws, "message_history",
				s.getUserMessageHistory(username))
			sendProtoMsg(ws, "users",
				s.getAllUsers())
		case "send_message":
			m, isMap := clientMessage.Data.(map[string]interface{})
			if !isMap {
				sendProtoMsg(ws, "send_fail", "data is not a map")
				break
			}
			msgBody, bodyPresent := m["body"].(string)
			if !bodyPresent {
				sendProtoMsg(ws, "send_fail", "invalid message payload")
				break
			}
			msgReceivers, receiversPresent := m["receivers"].([]interface{})
			timestamp, timestampPresent := m["timestamp"].(string)

			msg := userMessage{
				Id:     s.getNextMessageId(),
				Sender: username,
				Body:   msgBody,
			}
			if receiversPresent {
				msg.Receivers = make([]string, 0, len(msgReceivers))
				for _, v := range msgReceivers {
					msg.Receivers = append(msg.Receivers, v.(string))
				}
			}

			// scheduling a message
			if timestampPresent {
				msg.Timestamp, _ = time.Parse(time.RFC3339, timestamp)
				s.scheduledMessages.Store(msg.Id, msg)
				sendProtoMsg(ws, "schedule_success", "")
				fmt.Println("scheduled", msg)
				break
			}

			msg = s.pushUserMessageToHistory(msg)
			go s.sendMessageToOnlineUsers(msg, true)

			confirmationMessage := struct {
				Id        int       `json:"id"`
				Timestamp time.Time `json:"timestamp"`
			}{
				msg.Id,
				msg.Timestamp,
			}
			sendProtoMsg(ws, "send_success", confirmationMessage)
			fmt.Println("sent", msg)
		default:
			sendProtoMsg(ws, "status",
				"unknown message kind "+clientMessage.Kind)
		}
	}
}
