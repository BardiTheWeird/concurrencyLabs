package websockets

import (
	"encoding/binary"
	"io"
	"io/ioutil"
)

// FIN + RSV1-3 + opcode = 1 byte
// MASK + Payload len = 1 byte
// Extended payload length = 0/2/8 bytes
// Masking-key = 0/4 bytes
// Header sums up to 2/4/6/8/10/14 bytes
// Payload Data = 0/some/a lot of bytes (see Payload len)
// https://datatracker.ietf.org/doc/html/rfc6455#section-5
type Frame struct {
	IsFin bool
	// RSV1, RSV2, RSV3 are implicitly discarded/set to 0
	Opcode     WebSocketOpcode
	IsMasked   bool
	MaskingKey []byte // 0 or 4 bytes
	Payload    []byte
}

type WebSocketOpcode byte

const (
	None   WebSocketOpcode = 0
	Text   WebSocketOpcode = 1
	Binary WebSocketOpcode = 2
	// future non-control frames
	Close WebSocketOpcode = 8
	Ping  WebSocketOpcode = 9
	Pong  WebSocketOpcode = 10
	// future control frames
)

func (f *Frame) ToTransport() []byte {
	extendedPayloadLenLength := 0
	if len(f.Payload) > 65536 {
		extendedPayloadLenLength = 8
	} else if len(f.Payload) > 125 {
		extendedPayloadLenLength = 2
	}
	var transport []byte = make([]byte,
		2+extendedPayloadLenLength,
		2+extendedPayloadLenLength+len(f.MaskingKey)+len(f.Payload))

	if f.IsFin {
		transport[0] += 0x80
	}
	transport[0] += byte(f.Opcode)

	if extendedPayloadLenLength == 0 {
		transport[1] += byte(len(f.Payload))
	} else if extendedPayloadLenLength == 2 {
		transport[1] += 126
		binary.BigEndian.PutUint16(
			transport[2:4],
			uint16(len(f.Payload)),
		)
	} else {
		transport[1] += 127
		binary.BigEndian.PutUint64(
			transport[2:10],
			uint64(len(f.Payload)),
		)
	}

	if f.IsMasked {
		transport[1] += 0x80
		transport = append(transport, f.MaskingKey...)
	}

	transport = append(transport, f.Payload...)

	return transport
}

func (ws *WebSocketConnection) ReadFrame() (Frame, error) {
	f := Frame{}
	buffer := make([]byte, 8)

	if _, err := io.ReadFull(ws.conn, buffer[:2]); err != nil {
		return Frame{}, err
	}

	f.IsFin = buffer[0]&0x80 != 0
	f.Opcode = WebSocketOpcode(buffer[0] & 0xF)

	f.IsMasked = buffer[1]&0x80 != 0

	payloadLen := uint64(buffer[1] & 0x7F)
	if payloadLen == 126 {
		if _, err := io.ReadFull(ws.conn, buffer[:2]); err != nil {
			return Frame{}, err
		}
		payloadLen = uint64(binary.BigEndian.Uint16(buffer[:2]))
	} else if payloadLen == 127 {
		if _, err := io.ReadFull(ws.conn, buffer[:8]); err != nil {
			return Frame{}, err
		}

		payloadLen = binary.BigEndian.Uint64(buffer[:8])
	}

	if f.IsMasked {
		if _, err := io.ReadFull(ws.conn, buffer[:4]); err != nil {
			return Frame{}, err
		}

		f.MaskingKey = make([]byte, 4)
		copy(f.MaskingKey, buffer)
	}

	payload, err := ioutil.ReadAll(io.LimitReader(ws.conn, int64(payloadLen)))

	if err != nil {
		return Frame{}, err
	}
	f.Payload = payload

	return f, nil
}

func (f *Frame) UnmaskPayload() {
	if !f.IsMasked && len(f.MaskingKey) == 4 {
		return
	}

	for i, v := range f.Payload {
		f.Payload[i] = v ^ f.MaskingKey[i%4]
	}
	f.IsMasked = false
	f.MaskingKey = []byte{}
}
