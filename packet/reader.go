package packet

import (
	"goserver/serialization"
	"bytes"
)

type PacketReader struct {
	Buffer []byte
	Index int
}

func (r *PacketReader) ReadBytes(length int) []byte {
	// If we are trying to read past the end of the buffer, return null bytes
	
	if r.Index >= len(r.Buffer) {
		return make([]byte, length)
	}

	// If we are trying to read more data than the buffer has (but some data will still be read), copy it into a new buffer to fit the length
	
	if r.Index + length > len(r.Buffer) {
		buffer := make([]byte, length)
		copy(buffer, r.Buffer[r.Index:])
		r.Index += length
		return buffer
	}

	data := r.Buffer[r.Index:r.Index+length]
	r.Index += length
	return data
}

func (r *PacketReader) ReadString() string {
	return serialization.DecodeString(bytes.ReplaceAll(r.ReadBytes(serialization.STRING_LENGTH), []byte{0}, []byte{0x20}), 0)
}

func (r *PacketReader) ReadShort() int {
	return serialization.DecodeShort(r.ReadBytes(2), 0)
}

func (r *PacketReader) ReadByte() byte {
	return r.ReadBytes(1)[0]
}

func CreatePacketReader(buffer []byte) PacketReader {
	return PacketReader{buffer, 0}
}