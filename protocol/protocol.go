package protocol

import (
	"goserver/level"
	"goserver/packet"
)

const (
	// Misc.

	PROTOCOL_VERSION = 0x07
	
	// Client -> Server

	CLIENT_IDENTIFICATION = 0x00
	CLIENT_SET_BLOCK = 0x05
	CLIENT_POSITION_AND_ORIENTATION = 0x08
	CLIENT_MESSAGE = 0x0d
	
	// Server -> Client

	SERVER_IDENTIFICATION = 0x00
	SERVER_PING = 0x01 // Unused
	SERVER_LEVEL_INITIALIZE = 0x02
	SERVER_LEVEL_DATA_CHUNK = 0x03
	SERVER_LEVEL_FINALIZE = 0x04
	SERVER_SET_BLOCK = 0x06
	SERVER_SPAWN_PLAYER = 0x07
	SERVER_POSITION_AND_ORIENTATION = 0x08
	SERVER_POSITION_AND_ORIENTATION_UPDATE = 0x09
	SERVER_POSITION_UPDATE = 0x0a
	SERVER_ORIENTATION_UPDATE = 0x0b
	SERVER_DESPAWN_PLAYER = 0x0c
	SERVER_MESSAGE = 0x0d
	SERVER_DISCONNECT = 0x0e
	SERVER_UPDATE_USER_TYPE = 0x0f
)

// Packets

func WriteServerIdentification(w *packet.PacketWriter, name string, motd string, op bool) {
	w.WriteByte(SERVER_IDENTIFICATION) // Packet ID
	w.WriteByte(PROTOCOL_VERSION) // Protocol version
	w.WriteString(name) // Server name
	w.WriteString(motd) // Server MOTD
	
	if op {
		w.WriteByte(0x64) // OP User type
	} else {
		w.WriteByte(0x00) // non-OP User type
	}
}

func WriteLevelDataChunk(w *packet.PacketWriter, data []byte, percentage byte) {
	w.WriteByte(SERVER_LEVEL_DATA_CHUNK) // Packet ID
	w.WriteShort(len(data)) // Chunk Length
	w.WriteByteArray(data) // Chunk Data
	w.WriteByte(percentage) // Progress Percentage
}

func WriteLevelFinalize(w *packet.PacketWriter, level level.Level) {
	w.WriteByte(SERVER_LEVEL_FINALIZE) // Packet ID
	w.WriteShort(level.Width) // Width
	w.WriteShort(level.Height) // Height
	w.WriteShort(level.Depth) // Depth
}

func WriteSpawnPlayer(w *packet.PacketWriter, name string, id byte, x int, y int, z int, yaw byte, pitch byte) {
	w.WriteByte(SERVER_SPAWN_PLAYER) // Packet ID
	w.WriteByte(id) // Player ID
	w.WriteString(name) // Player Name
	w.WriteShort(x) // X
	w.WriteShort(y) // Y
	w.WriteShort(z) // Z
	w.WriteByte(yaw) // Yaw (Heading)
	w.WriteByte(pitch) // Pitch
}

func WriteDespawnPlayer(w *packet.PacketWriter, id byte) {
	w.WriteByte(SERVER_DESPAWN_PLAYER)
	w.WriteByte(id)
}

func WriteDisconnect(w *packet.PacketWriter, message string) {
	w.WriteByte(SERVER_DISCONNECT) // Packet ID
	w.WriteString(message) // Disconnect message
}

func WriteMessage(w *packet.PacketWriter, id byte, message string) {
	w.WriteByte(SERVER_MESSAGE) // Packet ID
	w.WriteByte(id) // Player ID
	w.WriteString(message) // Message
}

func WriteSetBlock(w *packet.PacketWriter, x int, y int, z int, id byte) {
	w.WriteByte(SERVER_SET_BLOCK) // Packet ID
	w.WriteShort(x) // X
	w.WriteShort(y) // Y
	w.WriteShort(z) // Z
	w.WriteByte(id) // Block Type
}

func WritePositionAndOrientation(w *packet.PacketWriter, id byte, x int, y int, z int, yaw byte, pitch byte) {
	w.WriteByte(SERVER_POSITION_AND_ORIENTATION) // Packet ID
	w.WriteByte(id) // Player ID
	w.WriteShort(x) // X
	w.WriteShort(y) // Y
	w.WriteShort(z) // Z
	w.WriteByte(yaw) // Yaw
	w.WriteByte(pitch) // Pitch
}

func WritePositionAndOrientationUpdate(w *packet.PacketWriter, id byte, oldX int, oldY int, oldZ int, newX int, newY int, newZ int, yaw byte, pitch byte) {
	w.WriteByte(SERVER_POSITION_AND_ORIENTATION_UPDATE) // Packet ID
	w.WriteByte(id) // Player ID
	w.WriteByte(byte(newX - oldX)) // X difference
	w.WriteByte(byte(newY - oldY)) // Y difference
	w.WriteByte(byte(newZ - oldZ)) // Z difference
	w.WriteByte(yaw) // Yaw
	w.WriteByte(pitch) // Pitch
}
