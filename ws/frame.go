package ws

import (
	"log"
	"io"
	"bufio"
	"math/rand"
	"encoding/binary"
	"errors"
)

type FrameType byte

const (
	ContinuationFrame	= 0
	TextFrame		= 1
	BinaryFrame		= 2
	CloseFrame		= 8
	PingFrame               = 9
	PongFrame               = 10
)

type Frame struct {
	Type FrameType
	Mask bool
	Payload []byte

	fin  bool
	rsv1 bool
	rsv2 bool
	rsv3 bool

	mask []byte
}

func NewTextFrame(s string) *Frame {
	return &Frame{
		Type: TextFrame,
		Payload: []byte(s),
		fin: true,
	}
}

func NewBinaryFrame(b []byte) *Frame {
	return &Frame{
		Type: BinaryFrame,
		Payload: b,
		fin: true,
	}
}

func newCloseFrame(code uint16, message string)  *Frame {
	data := make([]byte, 2)
	binary.BigEndian.PutUint16(data, code)
	data = append(data, []byte(message)...)

	return &Frame{
		Type: CloseFrame,
		Payload: data,
		fin: true,
	}
}

func newPingFrame(data []byte) *Frame {
	return &Frame{
		Type: PingFrame,
	        Payload: data,
		fin: true,
	}
}

func newPongFrame(data []byte) *Frame {
	return &Frame{
		Type: PongFrame,
	        Payload: data,
		fin: true,
	}
}


type frameReader struct {
	* bufio.Reader
}

type frameWriter struct {
	* bufio.Writer
}

type frameReadWriter struct {
	* frameReader
	* frameWriter
}

func newFrameReader(reader * bufio.Reader) * frameReader {
	return &frameReader{
		Reader: reader,
	}
}

func newframeWriter(writer * bufio.Writer) * frameWriter {
	return &frameWriter{
		Writer: writer,
	}
}

func newFrameReadWriter(readWriter * bufio.ReadWriter) * frameReadWriter {
	return &frameReadWriter{
		frameReader: newFrameReader(readWriter.Reader),
		frameWriter: newframeWriter(readWriter.Writer),
	}
}

func (reader * frameReader) ReadFrame() (*Frame, error) {

	frame := &Frame{}

	b, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}

	frame.fin    = (b & 0x80) >> 7 == 1
	frame.rsv1   = (b & 0x40) >> 6 == 1
	frame.rsv2   = (b & 0x20) >> 5 == 1
	frame.rsv3   = (b & 0x10) >> 4 == 1
	frame.Type   = FrameType(b & 0x0f)

	b, err = reader.ReadByte()
	if err != nil {
		return nil, err
	}

	frame.Mask = (b & 0x80) >> 7 == 1

	length  := b & 0x7f

	var payloadLength uint64
	switch(length) {
	case 126:
		var length uint16
		err := binary.Read(reader, binary.BigEndian, &length)
		if err != nil {
			return nil, err
		}
		payloadLength = uint64(length)
	case 127:
		var length uint64
		err := binary.Read(reader, binary.BigEndian, &length)
		if err != nil {
			return nil, err
		}
		payloadLength = uint64(length)
	default:
		payloadLength = uint64(length)
	}

	if frame.Mask {
		frame.mask = make([]byte, 4)
		_, err := io.ReadFull(reader, frame.mask)
		if err != nil {
			return nil, err
		}
	}

	frame.Payload = make([]byte, payloadLength)
	_, err = io.ReadFull(reader, frame.Payload)
	if err != nil {
		return nil, err
	}

	if frame.Mask {
		for i := uint64(0); i<payloadLength; i++ {
			frame.Payload[i] = frame.Payload[i] ^ frame.mask[i % 4]
		}
	}

	return frame, nil
}

func (writer * frameWriter) WriteFrame(frame *Frame) error {

	var b byte

	if frame.fin  { b |= 0x80 }
	if frame.rsv1 { b |= 0x40 }
	if frame.rsv2 { b |= 0x20 }
	if frame.rsv3 { b |= 0x10 }

	b |= byte(frame.Type & 0x0f)

	err := writer.WriteByte(b)
	if err != nil {
		return err
	}

	b = 0
	var length byte

	if frame.Mask {
		length = 0x80
	}

	payloadLength := uint64(len(frame.Payload))

	switch {
	case payloadLength < 126:
		length |= byte(payloadLength) & 0x7f
		writer.WriteByte(length)
	case payloadLength < 2^16 + 1:
		length |= 126
		writer.WriteByte(length)
		err := binary.Write(writer, binary.BigEndian, uint16(payloadLength))
		if err != nil {
			panic(err)
		}
	case payloadLength < 2^63:
		length |= 127
		writer.WriteByte(length)
		err := binary.Write(writer, binary.BigEndian, payloadLength)
		if err != nil {
			panic(err)
		}

	default:
		log.Panicf("frame length %d too large", payloadLength)
	}

	if frame.Mask {
		mask := make([]byte, 4)
		rand.Read(mask)
		frame.mask = mask

		_, err := writer.Write(frame.mask)
		if err != nil {
			return err
		}

		for i:=uint64(0); i<payloadLength; i++ {
			frame.Payload[i] = frame.Payload[i] ^ frame.mask[i % 4]
		}
	}

	_, err = writer.Write(frame.Payload)
	if err != nil {
		return err
	}

	writer.Flush()

	return nil
}

func ParseCloseFrame(frame *Frame) (code uint16, message string, err error) {

	if frame.Type != CloseFrame {
		err = errors.New("not a close frame")
		return
	}

	if len(frame.Payload) == 1 {
		err = errors.New("invalid paylaoad size, expecting 16 bit code or nothing")
		return
	}

	code = binary.BigEndian.Uint16(frame.Payload[:2])

	message = string(frame.Payload[2:])

	return
}
