package protocol

import (
	"goserver/level"
	"goserver/serialization"
	"compress/gzip"
	"io/ioutil"
	"bytes"
)

const (
	PROTOCOL_VERSION = 0x07
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

// Util functions

func CompressData(source []byte) []byte {
	var buf bytes.Buffer
	
	zw := gzip.NewWriter(&buf)

	_, err := zw.Write(source)
	
	if err != nil {
		panic(err)
	}

	if err := zw.Close(); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

func DecompressData(source []byte) []byte {
	reader := bytes.NewReader(source)
	
	gzreader, err := gzip.NewReader(reader)
	
	if err != nil {
		panic(err)
	}
	
	output, err2 := ioutil.ReadAll(gzreader)
	
	if err2 != nil {
		panic(err2)
	}
	
	return output
}

// Packets

func ServerIdentification(serverName string, serverMOTD string, isOP bool) []byte {
	buffer := make([]byte, 1 + 1 + serialization.STRING_LENGTH + serialization.STRING_LENGTH + 1)
	
	buffer[0] = SERVER_IDENTIFICATION // Packet ID
	buffer[1] = PROTOCOL_VERSION // Protocol version
	
	serialization.CopyData(2, serialization.EncodeString(serverName), buffer) // Server name
	serialization.CopyData(2 + serialization.STRING_LENGTH, serialization.EncodeString(serverMOTD), buffer) // Server MOTD
	
	if isOP {
		buffer[2 + serialization.STRING_LENGTH + serialization.STRING_LENGTH] = 0x64 // OP User type
	} else {
		buffer[2 + serialization.STRING_LENGTH + serialization.STRING_LENGTH] = 0x00 // non-OP User type
	}
	
	return buffer
}

func LevelDataChunk(data []byte, percentComplete int) []byte {
	buffer := make([]byte, 1 + 2 + serialization.BYTE_ARRAY_LENGTH + 1)
	
	buffer[0] = SERVER_LEVEL_DATA_CHUNK // Packet ID
	
	serialization.CopyData(1, serialization.EncodeShort(len(data)), buffer) // Chunk Length
	serialization.CopyData(3, serialization.EncodeByteArray(data), buffer) // Chunk Data
	
	buffer[3 + 1024] = byte(percentComplete) // Percent Complete
	
	return buffer
}

func LevelFinalize(level level.Level) []byte {
	buffer := make([]byte, 1 + 2 + 2 + 2)
	
	buffer[0] = SERVER_LEVEL_FINALIZE // Packet ID
	
	serialization.CopyData(1, serialization.EncodeShort(level.Width), buffer) // X size
	serialization.CopyData(3, serialization.EncodeShort(level.Height), buffer) // Y size
	serialization.CopyData(5, serialization.EncodeShort(level.Depth), buffer) // Z size
	
	return buffer
}

func SpawnPlayer(name string, id byte, x int, y int, z int, yaw byte, pitch byte) []byte {
	buffer := make([]byte, 1 + 1 + serialization.STRING_LENGTH + 2 + 2 + 2 + 1 + 1)
	
	buffer[0] = SERVER_SPAWN_PLAYER // Packet ID
	buffer[1] = id // Player ID
	serialization.CopyData(2, serialization.EncodeString(name), buffer) // Player Name
	serialization.CopyData(2 + serialization.STRING_LENGTH, serialization.EncodeShort(x), buffer) // X
	serialization.CopyData(2 + serialization.STRING_LENGTH + 2, serialization.EncodeShort(y), buffer) // Y
	serialization.CopyData(2 + serialization.STRING_LENGTH + 2 + 2, serialization.EncodeShort(z), buffer) // Z
	buffer[2 + serialization.STRING_LENGTH + 2 + 2 + 2] = yaw // Yaw (Heading)
	buffer[2 + serialization.STRING_LENGTH + 2 + 2 + 2 + 1] = pitch // Pitch
	
	return buffer
}

func DespawnPlayer(id byte) []byte {
	buffer := make([]byte, 1 + 1)
	
	buffer[0] = SERVER_DESPAWN_PLAYER // Packet ID
	buffer[1] = id // Player ID
	
	return buffer
}

func Disconnect(reason string) []byte {
	buffer := make([]byte, 1 + serialization.STRING_LENGTH)
	
	buffer[0] = SERVER_DISCONNECT // Packet ID
	serialization.CopyData(1, serialization.EncodeString(reason), buffer) // Disconnect reason
	
	return buffer
}

func Message(id byte, message string) []byte {
	buffer := make([]byte, 1 + 1 + serialization.STRING_LENGTH)
	
	buffer[0] = SERVER_MESSAGE // Packet ID
	buffer[1] = id // Player ID
	serialization.CopyData(2, serialization.EncodeString(message), buffer) // Message
	
	return buffer
}

func SetBlock(x int, y int, z int, id byte) []byte {
	buffer := make([]byte, 1 + 2 + 2 + 2 + 1)
	
	buffer[0] = SERVER_SET_BLOCK // Packet ID
	serialization.CopyData(1, serialization.EncodeShort(x), buffer) // X
	serialization.CopyData(3, serialization.EncodeShort(y), buffer) // Y
	serialization.CopyData(5, serialization.EncodeShort(z), buffer) // Z
	buffer[7] = id // Block Type
	
	return buffer
}

func PositionAndOrientation(id int, x int, y int, z int, yaw int, pitch int) []byte {
	buffer := make([]byte, 1 + 1 + 2 + 2 + 2 + 1 + 1)
	
	buffer[0] = SERVER_POSITION_AND_ORIENTATION // Packet ID
	buffer[1] = byte(id) // Player ID
	serialization.CopyData(2, serialization.EncodeShort(x), buffer) // X
	serialization.CopyData(4, serialization.EncodeShort(y), buffer) // Y
	serialization.CopyData(6, serialization.EncodeShort(z), buffer) // Z
	buffer[8] = byte(yaw) // Yaw
	buffer[9] = byte(pitch) // Pitch
	
	return buffer
}

func PositionAndOrientationUpdate(id byte, oldX int, oldY int, oldZ int, newX int, newY int, newZ int, yaw byte, pitch byte) []byte {
	buffer := make([]byte, 1 + 1 + 1 + 1 + 1 + 1 + 1)
	
	buffer[0] = SERVER_POSITION_AND_ORIENTATION_UPDATE // Packet ID
	buffer[1] = id // Player ID
	
	buffer[2] = byte(newX - oldX)// Change in X
	buffer[3] = byte(newY - oldY) // Change in Y
	buffer[4] = byte(newZ - oldZ) // Change in Z
	
	buffer[5] = yaw // Yaw
	buffer[6] = pitch // Pitch
	
	return buffer
}
