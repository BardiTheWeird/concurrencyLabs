package server

import (
	"encoding/json"
	"fmt"
	"net"
	"parallel-computations-5/websockets"
	"strings"
	"sync"
	"time"
)

type Server struct {
	messages      []userMessage
	messagesMutex sync.RWMutex

	// username -> connection
	currentConnection      map[string]*websockets.WebSocketConnection
	currentConnectionMutex sync.RWMutex

	// username -> isOnline
	users sync.Map
}

func InitializeServer() *Server {
	srv := Server{}
	srv.currentConnection = make(map[string]*websockets.WebSocketConnection)
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
				return
			}
			_username = strings.TrimSpace(_username)

			loggedIn, statusMsg := s.logIn(ws, _username)
			sendProtoMsg(ws, "login_status", statusMsg)
			if !loggedIn {
				return
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
				return
			}
			msgBody, bodyPresent := m["body"].(string)
			if !bodyPresent {
				sendProtoMsg(ws, "send_fail", "invalid message payload")
				return
			}
			msgReceivers, receiversPresent := m["receivers"].([]interface{})

			msg := userMessage{
				Sender: username,
				Body:   msgBody,
			}
			if receiversPresent {
				msg.Receivers = make([]string, 0, len(msgReceivers))
				for _, v := range msgReceivers {
					msg.Receivers = append(msg.Receivers, v.(string))
				}
			}
			msg = s.pushUserMessageToHistory(msg)
			go s.sendMessageToOnlineUsers(msg)

			confirmationMessage := struct {
				Id        int       `json:"id"`
				Timestamp time.Time `json:"timestamp"`
			}{
				msg.Id,
				msg.Timestamp,
			}
			sendProtoMsg(ws, "send_success", confirmationMessage)
		default:
			sendProtoMsg(ws, "status",
				"unknown message kind "+clientMessage.Kind)
		}
	}
}

// func (s *Server) HandleConnection(username string, conn *Connection) {
// 	reader := bufio.NewReader(conn)
// 	for {
// 		action, err := reader.ReadString('\n')
// 		if err != nil {
// 			if errors.Is(err, net.ErrClosed) {
// 				fmt.Println(conn.RemoteAddr(), "closed the connection")
// 				s.currentConnectionMutex.Lock()
// 				delete(s.currentConnection, username)
// 				s.currentConnectionMutex.Unlock()
// 				return
// 			}
// 			break
// 		}

// 		switch action {
// 		case "get messages":
// 			messages := s.GetUserMessages(username)
// 			var response []byte = []byte("messages\n")
// 			for _, msg := range messages {
// 				response = append(response, msg.ToDTO()...)
// 			}
// 			response = append(response, []byte("end\n")...)
// 			conn.Write(response)
// 		case "send message":
// 			msgString, err := reader.ReadString('\n')
// 			if err != nil {
// 				conn.Write([]byte("error\nsending message\nerror reading message"))
// 				continue
// 			}
// 			msgSplit := strings.Split(msgString, ";")
// 			if len(msgSplit) != 4 {
// 				conn.Write([]byte("error\nsending message\nmessage isn't split into 4 parts by ';'"))
// 				continue
// 			}

// 			var msg UserMessage
// 			msg.SenderName = msgSplit[0]
// 			msg.ReceiversNames = strings.Split(msgSplit[1], ",")

// 			t, err := time.Parse(time.RFC3339, msgSplit[2])
// 			if err != nil {
// 				conn.Write([]byte("error\nsending message\nerror parsing time"))
// 				continue
// 			}
// 			msg.SendTime = t

// 			b64bytes, err := base64.StdEncoding.DecodeString(msgSplit[3])
// 			if err != nil {
// 				conn.Write([]byte("error\nsending message\nerror decoding body"))
// 				continue
// 			}
// 			msg.Body = string(b64bytes)

// 			sendTime := s.AppendUserMessage(msg)
// 			sendTimeEncoded, _ := sendTime.MarshalText()

// 			conn.Write(append([]byte("ok\n"), sendTimeEncoded...))
// 		default:
// 			fmt.Println(username, "sent a jibberish action", action, "ignoring it")
// 			continue
// 		}
// 	}
// }
