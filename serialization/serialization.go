package serialization

import (
	"encoding/binary"
	"strings"
)

const (
	HASH_LENGTH = 32
	STRING_LENGTH = 64
	BYTE_ARRAY_LENGTH = 1024
)

func EncodeString(data string) []byte {
	input_bytes := []byte(data)
	bytes := make([]byte, STRING_LENGTH)
	
	i := 0
	
	for i = 0; i < STRING_LENGTH; i++ {
		// Unused bytes use 0x20
		
		bytes[i] = byte(0x20)
	}
	
	for i = 0; i < len(data); i++ {
		if i == STRING_LENGTH {
			break
		}
		
		bytes[i] = input_bytes[i]
	}
	
	return bytes
}

func DecodeString(data []byte, index int) string {
	output := ""
	
	for i := 0; i < STRING_LENGTH; i++ {
		output += string(data[index + i])
	}
	
	return strings.TrimRight(output, " ")
}

func DecodeShort(data []byte, index int) int {
	return int((data[index + 0] << 8) + data[index + 1])
}

func EncodeByteArray(data []byte) []byte {
	bytes := make([]byte, BYTE_ARRAY_LENGTH)
	
	i := 0
	
	for i = 0; i < BYTE_ARRAY_LENGTH; i++ {
		// Unused bytes use 0x00
		
		bytes[i] = byte(0x00)
	}
	
	for i = 0; i < len(data); i++ {
		bytes[i] = data[i]
	}
	
	return bytes
}

func EncodeShort(value int) []byte {
	return []byte{
		byte((value >> 8) & 0xff),
		byte(value & 0xff),
	}
}

func EncodeInt(value int) []byte {
	bytes := make([]byte, 4)
	
	binary.BigEndian.PutUint32(bytes, uint32(value))
	
	return bytes
}

func CopyData(index int, source []byte, target []byte) {
	for i := 0; i < len(source); i++ {
		target[index + i] = source[i]
	}
}

func SplitData(data []byte, chunkSize int) [][]byte {
	var chunks [][]byte
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		
		if end > len(data) {
			end = len(data)
		}

		chunks = append(chunks, data[i:end])
	}

	return chunks
}