package wsio

import (
	"fmt"
	"io"
	"bufio"
	"encoding/binary"
	"errors"
)

type Frame struct {
	Fin  bool
	Rsv1 bool
	Rsv2 bool
	Rsv3 bool
	Opcode byte
	Mask []byte
	Payload []byte
}

type FrameReader struct {
	reader * bufio.Reader
}

type FrameWriter struct {
	writer * bufio.Writer
}

func NewFrameReader(r * bufio.Reader) * FrameReader {
	return &FrameReader{
		reader: r,
	}
}

func NewFrameWriter(w * bufio.Writer) * FrameWriter {
	return &FrameWriter{
		writer: w,
	}
}

func (reader * FrameReader) ReadFrame() (*Frame, error) {

	frame := &Frame{}
	
	b, err := reader.reader.ReadByte()
	if err != nil {
		panic(err)
	}

	frame.Fin    = (b & 0x80) >> 7 == 1
	frame.Rsv1   = (b & 0x40) >> 6 == 1
	frame.Rsv2   = (b & 0x20) >> 5 == 1
	frame.Rsv3   = (b & 0x10) >> 4 == 1
	frame.Opcode = b & 0x0f
	
	b, err = reader.reader.ReadByte()
	if err != nil {
		panic(err)
	}

	hasMask := (b & 0x80) >> 7 == 1
	length  := b & 0x7f

	var payloadLength uint64
	switch(length) {
	case 126:
		var length uint16
		err := binary.Read(reader.reader, binary.BigEndian, &length)
		if err != nil {
			panic(err)
		}
		payloadLength = uint64(length)
	case 127:
		var length uint64
		err := binary.Read(reader.reader, binary.BigEndian, &length)
		if err != nil {
			panic(err)
		}
		payloadLength = uint64(length)
	default:
		payloadLength = uint64(length)
	}

	if hasMask {
		frame.Mask = make([]byte, 4)
		_, err := io.ReadFull(reader.reader, frame.Mask)
		if err != nil {
			panic(err)
		}
	}

	frame.Payload = make([]byte, payloadLength)
	_, err = io.ReadFull(reader.reader, frame.Payload)
	if err != nil {
		panic(err)
	}

	if hasMask {
		for i := uint64(0); i<payloadLength; i++ {
			frame.Payload[i] = frame.Payload[i] ^ frame.Mask[i % 4]
		}
	}

	return frame, nil
}

func (writer * FrameWriter) WriteFrame(frame *Frame) error {

	var b byte

	if frame.Fin  { b |= 0x80 }
	if frame.Rsv1 { b |= 0x40 }
	if frame.Rsv2 { b |= 0x20 }
	if frame.Rsv3 { b |= 0x10 }

	b |= frame.Opcode & 0x0f

	err := writer.writer.WriteByte(b)
	if err != nil {
		panic(err)
	}

	b = 0
	var length byte

	if len(frame.Mask) > 0 {
		length = 0x80
	}

	payloadLength := uint64(len(frame.Payload))
	
	switch {
	case payloadLength < 126:
		length |= byte(payloadLength) & 0x7f
		writer.writer.WriteByte(length)
	case payloadLength < 2^16 + 1:
		length |= 126
		writer.writer.WriteByte(length)
		err := binary.Write(writer.writer, binary.BigEndian, uint16(payloadLength))
		if err != nil {
			panic(err)
		}
	case payloadLength < 2^63:
		length |= 127
		writer.writer.WriteByte(length)
		err := binary.Write(writer.writer, binary.BigEndian, payloadLength)
		if err != nil {
			panic(err)
		}

	default:
		panic(errors.New(fmt.Sprintf("frame length %d too large", payloadLength)))
	}

	if len(frame.Mask) > 0 {
		_, err := writer.writer.Write(frame.Mask)
		if err != nil {
			panic(err)
		}
	}

	_, err = writer.writer.Write(frame.Payload)
	if err != nil {
		panic(err)
	}
	
	writer.writer.Flush()
	
	return nil
}
