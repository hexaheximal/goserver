package packet

import (
	"goserver/serialization"
	"net"
)

type PacketWriter struct {
	Buffer []byte
}

func (w *PacketWriter) WriteBytes(data []byte) {
	w.Buffer = append(w.Buffer, data...)
}

func (w *PacketWriter) WriteString(data string) {
	w.WriteBytes(serialization.EncodeString(data))
}

func (w *PacketWriter) WriteShort(data int) {
	w.WriteBytes(serialization.EncodeShort(data))
}

func (w *PacketWriter) WriteByte(data byte) {
	w.WriteBytes([]byte{data})
}

func (w *PacketWriter) WriteByteArray(data []byte) {
	w.WriteBytes(serialization.EncodeByteArray(data))
}

func (w *PacketWriter) WriteToSocket(conn net.Conn) {
	conn.Write(w.Buffer)
	w.Buffer = make([]byte, 0)
}

func CreatePacketWriter() PacketWriter {
	return PacketWriter{make([]byte, 0)}
}