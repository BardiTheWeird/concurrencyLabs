package websockets

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"sync"
)

type WebSocketConnection struct {
	C chan Message

	conn                  net.Conn
	writeMutex            sync.Mutex
	fragmentedPayload     [][]byte
	fragmentedFrameOpcode WebSocketOpcode

	isClosed      bool
	isClosedMutex sync.Mutex
}

type Message struct {
	ConnectionClosed bool
	Payload          []byte
}

var requiredHeaders []string = []string{
	"upgrade",
	"connection",
	"sec-websocket-key",
}

var webSocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

func UpgradeFromTCP(conn net.Conn) (*WebSocketConnection, error) {
	var ok bool
	defer func() {
		if !ok {
			conn.Close()
		}
	}()

	reader := bufio.NewReader(conn)
	readLine := func() (string, error) {
		outLine := ""
		for {
			l, err := reader.ReadString('\n')
			if err != nil {
				return "", err
			}
			outLine += l

			if len(outLine) >= 2 && outLine[len(outLine)-2:] == "\r\n" {
				return outLine[:len(outLine)-2], nil
			}
		}
	}

	/// Parse HTTP Request-line
	requestLine, err := readLine()
	if err != nil {
		return nil, err
	}

	if len(requestLine) < 6 {
		return nil, fmt.Errorf("HTTP Request-line too short")
	}
	firstSpaceIndex := strings.Index(requestLine, " ")
	if firstSpaceIndex == -1 {
		return nil, fmt.Errorf("HTTP Request-line format invalid")
	}
	// secondSpaceIndex := strings.Index(requestLine[firstSpaceIndex+1:], " ")
	// if firstSpaceIndex == -1 {
	// 	return nil, fmt.Errorf("HTTP Request-line format invalid")
	// }

	method := requestLine[:firstSpaceIndex]
	if method != "GET" {
		conn.Write([]byte("HTTP/1.1 405 Method Not Allowed\r\n\r\n"))
	}
	// uri := requestLine[firstSpaceIndex+1 : secondSpaceIndex]
	// httpVersion := requestLine[secondSpaceIndex+1:]

	/// Parse headers
	headers := make(map[string]string)
	for {
		line, err := readLine()
		if err != nil {
			return nil, err
		}

		if line == "" {
			break
		}

		colonIndex := strings.Index(line, ":")
		if colonIndex == -1 {
			return nil, fmt.Errorf("HTTP Invalid Header")
		}
		headers[strings.ToLower(line[:colonIndex])] =
			strings.TrimSpace(line[colonIndex+1:])
	}

	for _, header := range requiredHeaders {
		_, present := headers[header]
		if !present {
			return nil, fmt.Errorf("HTTP Header %s Not Present", header)
		}
	}

	webSocketKey := headers["sec-websocket-key"]
	webSocketAcceptBytes := sha1.Sum([]byte(webSocketKey + webSocketGUID))
	webSocketAcceptB64 := base64.StdEncoding.EncodeToString(webSocketAcceptBytes[:])

	// Writing a response
	responseString := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + webSocketAcceptB64 + "\r\n" +
		"\r\n"

	conn.Write([]byte(responseString))

	ok = true
	ws := WebSocketConnection{
		conn: conn,
		C:    make(chan Message),
	}
	go ws.listen()

	return &ws, nil
}

func (ws *WebSocketConnection) SendData(b []byte) {
	f := Frame{
		IsFin:   true,
		Opcode:  Text,
		Payload: b,
	}
	transportBytes := f.ToTransport()
	ws.write(transportBytes)
}

func (ws *WebSocketConnection) Close(reason string) {
	ws.isClosedMutex.Lock()
	if ws.isClosed {
		ws.isClosedMutex.Unlock()
		return
	}
	ws.isClosed = true
	ws.isClosedMutex.Unlock()

	payload := make([]byte, 2)
	binary.BigEndian.PutUint16(payload, 1009)
	f := Frame{
		IsFin:   true,
		Opcode:  Close,
		Payload: append(payload, []byte(reason)...),
	}
	ws.write(f.ToTransport())
}

func (ws *WebSocketConnection) listen() {
	for {
		frame, err := ws.ReadFrame()
		if err != nil {
			fmt.Println("error reading frame:", err)
		}
		frame.UnmaskPayload()

		if !frame.IsFin {
			if frame.Opcode != None {
				ws.fragmentedFrameOpcode = frame.Opcode
			}
			ws.fragmentedPayload =
				append(ws.fragmentedPayload, frame.Payload)

			continue
		} else if frame.Opcode == None {
			combinedPayloadLen := len(frame.Payload)
			for _, v := range ws.fragmentedPayload {
				combinedPayloadLen += len(v)
			}

			combinedPayload := make([]byte, 0, combinedPayloadLen)
			for _, v := range ws.fragmentedPayload {
				combinedPayload = append(combinedPayload, v...)
			}
			combinedPayload = append(combinedPayload, frame.Payload...)

			frame.Payload = combinedPayload
			frame.Opcode = ws.fragmentedFrameOpcode

			ws.fragmentedPayload = [][]byte{}
		}

		switch frame.Opcode {
		case Text, Binary:
			ws.C <- Message{false, frame.Payload}
		case Close:
			if len(frame.Payload) >= 2 {
				statusCode := binary.BigEndian.Uint16(frame.Payload[:2])
				reason := string(frame.Payload[2:])
				fmt.Println("received Close frame:", statusCode, reason)
			}

			ws.isClosedMutex.Lock()
			if !ws.isClosed {
				ws.isClosed = true
				ws.isClosedMutex.Unlock()
				ws.write(frame.ToTransport())
				ws.C <- Message{true, []byte{}}
			} else {
				ws.isClosedMutex.Unlock()
			}
			ws.conn.Close()
			fmt.Println("connection closed")
			return
		case Ping:
			pongFrame := Frame{
				IsFin:   true,
				Opcode:  Pong,
				Payload: frame.Payload,
			}
			ws.write(pongFrame.ToTransport())
		case Pong:
			fmt.Println("https://www.youtube.com/watch?v=CXpuRIZzJog")
		default:
			fmt.Println("what is wrong with you")
		}
	}
}

func (ws *WebSocketConnection) write(b []byte) {
	ws.writeMutex.Lock()
	defer ws.writeMutex.Unlock()

	for {
		bytesWritten, err := ws.conn.Write(b)
		if err != nil {
			fmt.Println("error writing to ws")
			return
		}

		if bytesWritten == len(b) {
			return
		}
		b = b[bytesWritten:]
	}
}
