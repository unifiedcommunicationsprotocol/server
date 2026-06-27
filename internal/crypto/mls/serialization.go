// Package mls implements RFC 9420 Messaging Layer Security in pure Go.
package mls

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Serializer handles TLS wire encoding/decoding per RFC 9420.
type Serializer struct{}

// Encode encodes a value using TLS wire format.
func (s *Serializer) EncodeUint8(val uint8) []byte {
	return []byte{val}
}

// EncodeUint16 encodes a 16-bit unsigned integer (big-endian).
func (s *Serializer) EncodeUint16(val uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, val)
	return b
}

// EncodeUint32 encodes a 32-bit unsigned integer (big-endian).
func (s *Serializer) EncodeUint32(val uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, val)
	return b
}

// EncodeUint64 encodes a 64-bit unsigned integer (big-endian).
func (s *Serializer) EncodeUint64(val uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, val)
	return b
}

// EncodeBytes encodes a variable-length byte string with 2-byte length prefix.
func (s *Serializer) EncodeBytes(data []byte) []byte {
	return append(s.EncodeUint16(uint16(len(data))), data...)
}

// EncodeOpaque encodes with size_t length prefix (4 bytes per RFC 9420).
func (s *Serializer) EncodeOpaque(data []byte) []byte {
	return append(s.EncodeUint32(uint32(len(data))), data...)
}

// DecodeUint8 decodes a single byte.
func (s *Serializer) DecodeUint8(data []byte) (uint8, int, error) {
	if len(data) < 1 {
		return 0, 0, fmt.Errorf("insufficient data for uint8")
	}
	return data[0], 1, nil
}

// DecodeUint16 decodes a 16-bit unsigned integer.
func (s *Serializer) DecodeUint16(data []byte) (uint16, int, error) {
	if len(data) < 2 {
		return 0, 0, fmt.Errorf("insufficient data for uint16")
	}
	return binary.BigEndian.Uint16(data), 2, nil
}

// DecodeUint32 decodes a 32-bit unsigned integer.
func (s *Serializer) DecodeUint32(data []byte) (uint32, int, error) {
	if len(data) < 4 {
		return 0, 0, fmt.Errorf("insufficient data for uint32")
	}
	return binary.BigEndian.Uint32(data), 4, nil
}

// DecodeUint64 decodes a 64-bit unsigned integer.
func (s *Serializer) DecodeUint64(data []byte) (uint64, int, error) {
	if len(data) < 8 {
		return 0, 0, fmt.Errorf("insufficient data for uint64")
	}
	return binary.BigEndian.Uint64(data), 8, nil
}

// DecodeBytes decodes a variable-length byte string.
func (s *Serializer) DecodeBytes(data []byte) ([]byte, int, error) {
	if len(data) < 2 {
		return nil, 0, fmt.Errorf("insufficient data for bytes length")
	}
	length := binary.BigEndian.Uint16(data)
	if len(data) < 2+int(length) {
		return nil, 0, fmt.Errorf("insufficient data for bytes content")
	}
	return data[2 : 2+length], 2 + int(length), nil
}

// DecodeOpaque decodes a variable-length opaque value.
func (s *Serializer) DecodeOpaque(data []byte) ([]byte, int, error) {
	if len(data) < 4 {
		return nil, 0, fmt.Errorf("insufficient data for opaque length")
	}
	length := binary.BigEndian.Uint32(data)
	if len(data) < 4+int(length) {
		return nil, 0, fmt.Errorf("insufficient data for opaque content")
	}
	return data[4 : 4+length], 4 + int(length), nil
}

// Builder provides a convenient interface for constructing byte sequences.
type Builder struct {
	buf *bytes.Buffer
}

// NewBuilder creates a new byte sequence builder.
func NewBuilder() *Builder {
	return &Builder{buf: &bytes.Buffer{}}
}

// WriteUint8 writes an 8-bit value.
func (b *Builder) WriteUint8(val uint8) {
	b.buf.WriteByte(val)
}

// WriteUint16 writes a 16-bit value.
func (b *Builder) WriteUint16(val uint16) {
	data := make([]byte, 2)
	binary.BigEndian.PutUint16(data, val)
	b.buf.Write(data)
}

// WriteUint32 writes a 32-bit value.
func (b *Builder) WriteUint32(val uint32) {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, val)
	b.buf.Write(data)
}

// WriteUint64 writes a 64-bit value.
func (b *Builder) WriteUint64(val uint64) {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, val)
	b.buf.Write(data)
}

// WriteBytes writes a variable-length byte string.
func (b *Builder) WriteBytes(data []byte) {
	binary.Write(b.buf, binary.BigEndian, uint16(len(data)))
	b.buf.Write(data)
}

// WriteOpaque writes an opaque variable-length value.
func (b *Builder) WriteOpaque(data []byte) {
	binary.Write(b.buf, binary.BigEndian, uint32(len(data)))
	b.buf.Write(data)
}

// Bytes returns the constructed byte sequence.
func (b *Builder) Bytes() []byte {
	return b.buf.Bytes()
}
