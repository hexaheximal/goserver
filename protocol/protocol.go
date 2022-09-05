package protocol

import (
	"github.com/aquilax/go-perlin"
	"compress/gzip"
	"encoding/binary"
	"io/ioutil"
	"bytes"
	"strings"
	"time"
	"log"
)

const (
	BLOCK_AIR = 0
	BLOCK_STONE = 1
	BLOCK_GRASS = 2
	BLOCK_DIRT = 3
	BLOCK_COBBLESTONE = 4
	BLOCK_PLANKS = 5
	BLOCK_SAPLING = 6
	BLOCK_BEDROCK = 7
	BLOCK_FLOWING_WATER = 8
	BLOCK_STATIONARY_WATER = 9
	BLOCK_FLOWING_LAVA = 10
	BLOCK_STATIONARY_LAVA = 11
	BLOCK_SAND = 12
	BLOCK_GRAVEL = 13
	BLOCK_GOLD_ORE = 14
	BLOCK_IRON_ORE = 15
	BLOCK_COAL_ORE = 16
	BLOCK_WOOD = 17
	BLOCK_LEAVES = 18
	BLOCK_SPONGE = 19
	BLOCK_GLASS = 20
	BLOCK_RED_CLOTH = 21
	BLOCK_ORANGE_CLOTH = 22
	BLOCK_YELLOW_CLOTH = 23
	BLOCK_CHARTREUSE_CLOTH = 24
	BLOCK_GREEN_CLOTH = 25
	BLOCK_SPRING_GREEN_CLOTH = 26
	BLOCK_CYAN_CLOTH = 27
	BLOCK_CAPRI_CLOTH = 28
	BLOCK_ULTRAMARINE_CLOTH = 29
	BLOCK_VIOLET_CLOTH = 30
	BLOCK_PURPLE_CLOTH = 31
	BLOCK_MAGENTA_CLOTH = 32
	BLOCK_ROSE_CLOTH = 33
	BLOCK_DARK_GRAY_CLOTH = 34
	BLOCK_LIGHT_GRAY_CLOTH = 35
	BLOCK_WHITE_CLOTH = 36
	BLOCK_DANDELION = 37
	BLOCK_ROSE = 38
	BLOCK_BROWN_MUSHROOM = 39
	BLOCK_RED_MUSHROOM = 40
	BLOCK_GOLD = 41
	BLOCK_IRON = 42
	BLOCK_DOUBLE_SLAB = 43
	BLOCK_SLAB = 44
	BLOCK_BRICKS = 45
	BLOCK_TNT = 46
	BLOCK_BOOKSHELF = 47
	BLOCK_MOSSY_COBBLESTONE = 48
	BLOCK_OBSIDIAN = 49
)

const (
	STRING_LENGTH = 64
	BYTE_ARRAY_LENGTH = 1024
	PROTOCOL_VERSION = 0x07
)

const (
	SERVER_IDENTIFICATION = 0x00
	SERVER_PING = 0x01
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

const (
	LEVEL_FLAT = 0
	LEVEL_CLASSIC = 1
	LEVEL_EXPERIMENTAL = 2
)

// Protocol types

type Level struct {
	Width int // X size
	Height int // Y size
	Depth int // Z size
	Data []byte // block array
	Spawnpoint Spawnpoint // Spawnpoint
}

type Spawnpoint struct {
	X int
	Y int
	Z int
	Yaw int
	Pitch int
}

func (level Level) IsOOB(x int, y int, z int) bool {
	if (y * level.Depth + z) * level.Width + x > len(level.Data) - 1 {
		return true
	}
	
	return false
}

func (level Level) GetBlock(x int, y int, z int) int {
	return int(level.Data[(y * level.Depth + z) * level.Width + x])
}

func (level Level) SetBlock(x int, y int, z int, id int) {
	level.Data[(y * level.Depth + z) * level.Width + x] = byte(id)
}

func (level Level) Encode() []byte {
	buffer := make([]byte, 4 + len(level.Data))
	
	CopyData(0, EncodeInt(len(level.Data)), buffer)
	CopyData(4, level.Data, buffer)
	
	return buffer
}

func (level Level) Serialize() []byte {
	buffer := make([]byte, 2 + 2 + 2 + 2 + 2 + 2 + 1 + 1 + len(level.Data))
	
	CopyData(0, EncodeShort(level.Width), buffer) // Width
	CopyData(2, EncodeShort(level.Height), buffer) // Height
	CopyData(4, EncodeShort(level.Depth), buffer) // Depth
	
	CopyData(6, EncodeShort(level.Spawnpoint.X), buffer) // Spawn X
	CopyData(8, EncodeShort(level.Spawnpoint.Y), buffer) // Spawn Y
	CopyData(10, EncodeShort(level.Spawnpoint.Z), buffer) // Spawn Z
	
	buffer[12] = byte(level.Spawnpoint.Yaw) // Spawn Yaw
	buffer[13] = byte(level.Spawnpoint.Pitch) // Spawn Pitch
	
	CopyData(14, level.Data, buffer)
	
	return buffer
}

func DeserializeLevel(data []byte) Level {
	header_size := 2 + 2 + 2 + 2 + 2 + 2 + 1 + 1 // short, short, short (Level Size), short, short, short (Spawnpoint Position), byte, byte (Spawnpoint Yaw & Pitch)
	
	width := DecodeShort(data, 0) // Width
	height := DecodeShort(data, 2) // Height
	depth := DecodeShort(data, 4) // Depth
	
	spawnX := DecodeShort(data, 6) // Spawn X
	spawnY := DecodeShort(data, 8) // Spawn Y
	spawnZ := DecodeShort(data, 10) // Spawn Z
	
	spawnYaw := int(data[12]) // Spawn Yaw
	spawnPitch := int(data[13]) // Spawn Pitch
	
	
	
	return Level{
		width,
		height,
		depth,
		data[header_size:],
		Spawnpoint{spawnX, spawnY, spawnZ, spawnYaw, spawnPitch},
	}
}

func GenerateLevel(width int, height int, depth int, level_type int) Level {
	level := Level{
		width,
		height,
		depth,
		make([]byte, width * height * depth),
		Spawnpoint{int(float32(width) / 2.0), 0, int(float32(depth) / 2.0), 0, 0},
	}
	
	if level_type == LEVEL_FLAT {
		FlatLevelGenerator(level)
	}
	
	if level_type == LEVEL_EXPERIMENTAL {
		ExperimentalLevelGenerator(level)
	}
	
	for y := 0; y < height; y++ {
		if level.GetBlock(level.Spawnpoint.X, y, level.Spawnpoint.Z) == BLOCK_AIR {
			level.Spawnpoint.Y = y + 1
			break
		}
	}
	
	return level
}

func FlatLevelGenerator(level Level) {
	for y := 0; y < 5; y++ {
		for x := 0; x < level.Width; x++ {
			for z := 0; z < level.Depth; z++ {
				level.SetBlock(x, 0 + y, z, BLOCK_STONE)
			}
		}
	}
	
	for y := 0; y < 2; y++ {
		for x := 0; x < level.Width; x++ {
			for z := 0; z < level.Depth; z++ {
				level.SetBlock(x, 5 + y, z, BLOCK_DIRT)
			}
		}
	}
	
	for x := 0; x < level.Width; x++ {
		for z := 0; z < level.Depth; z++ {
			level.SetBlock(x, 7, z, BLOCK_GRASS)
		}
	}
}

func ExperimentalLevelGenerator(level Level) {
	seed := time.Now().UnixNano()
	
	log.Println("Level seed:", seed)
	
	heightNoise1 := perlin.NewPerlin(2., 2., 3, int64(seed + 0))
	heightNoise2 := perlin.NewPerlin(2., 2., 3, int64(seed + 1))
	heightNoise3 := perlin.NewPerlin(2., 2., 3, int64(seed + 2))
	heightNoise4 := perlin.NewPerlin(2., 2., 3, int64(seed + 3))
	
	biomeNoise1 := perlin.NewPerlin(2., 2., 3, int64(seed + 4))
	biomeNoise2 := perlin.NewPerlin(2., 2., 3, int64(seed + 5))
	
	for y := 0; y < 5; y++ {
		for x := 0; x < level.Width; x++ {
			for z := 0; z < level.Depth; z++ {
				level.SetBlock(x, 0 + y, z, BLOCK_STONE)
			}
		}
	}
	
	for x := 0; x < level.Width; x++ {
		for z := 0; z < level.Depth; z++ {
			noiseValue1 := (heightNoise1.Noise2D(float64(x) / 10.0, float64(z) / 10.0) * 0.5) + 0.5
			noiseValue2 := (heightNoise2.Noise2D(float64(x) / 10.0, float64(z) / 10.0) * 0.5) + 0.5
			
			noiseValue3 := (heightNoise3.Noise2D(float64(x) / 20.0, float64(z) / 20.0) * 0.5) + 0.5
			noiseValue4 := (heightNoise4.Noise2D(float64(x) / 20.0, float64(z) / 20.0) * 0.5) + 0.5
			
			noiseValue5 := (biomeNoise1.Noise2D(float64(x) / 20.0, float64(z) / 20.0) * 0.5) + 0.5
			noiseValue6 := (biomeNoise2.Noise2D(float64(x) / 20.0, float64(z) / 20.0) * 0.5) + 0.5
			
			noiseHeight := int((noiseValue1 * noiseValue2) * 20.0)
			cliffHeight := int((noiseValue3 * noiseValue4) * 40.0)
			
			if (noiseValue5 * noiseValue6) > 0.3 {
				for y := 0; y < cliffHeight; y++ {
					level.SetBlock(x, 5 + y, z, BLOCK_STONE)
				}
			} else {
				for y := 0; y < noiseHeight; y++ {
					level.SetBlock(x, 5 + y, z, BLOCK_DIRT)
				}
				
				level.SetBlock(x, 5 + noiseHeight, z, BLOCK_GRASS)
			}
		}
	}
}

func EncodeString(data string) []byte {
	input_bytes := []byte(data)
	bytes := make([]byte, STRING_LENGTH)
	
	i := 0
	
	for i = 0; i < STRING_LENGTH; i++ {
		// Unused bytes use 0x20
		
		bytes[i] = byte(0x20)
	}
	
	for i = 0; i < len(data); i++ {
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

// Packets

func ServerIdentification(serverName string, serverMOTD string, isOP bool) []byte {
	buffer := make([]byte, 1 + 1 + STRING_LENGTH + STRING_LENGTH + 1)
	
	buffer[0] = SERVER_IDENTIFICATION // Packet ID
	buffer[1] = PROTOCOL_VERSION // Protocol version
	
	CopyData(2, EncodeString(serverName), buffer) // Server name
	CopyData(2 + STRING_LENGTH, EncodeString(serverMOTD), buffer) // Server MOTD
	
	if isOP {
		buffer[2 + STRING_LENGTH + STRING_LENGTH] = 0x64 // OP User type
	} else {
		buffer[2 + STRING_LENGTH + STRING_LENGTH] = 0x00 // non-OP User type
	}
	
	return buffer
}

func LevelDataChunk(data []byte, percentComplete int) []byte {
	buffer := make([]byte, 1 + 2 + BYTE_ARRAY_LENGTH + 1)
	
	buffer[0] = SERVER_LEVEL_DATA_CHUNK // Packet ID
	
	CopyData(1, EncodeShort(len(data)), buffer) // Chunk Length
	CopyData(3, EncodeByteArray(data), buffer) // Chunk Data
	
	buffer[3 + 1024] = byte(percentComplete) // Percent Complete
	
	return buffer
}

func LevelFinalize(level Level) []byte {
	buffer := make([]byte, 1 + 2 + 2 + 2)
	
	buffer[0] = SERVER_LEVEL_FINALIZE // Packet ID
	
	CopyData(1, EncodeShort(level.Width), buffer) // X size
	CopyData(3, EncodeShort(level.Height), buffer) // Y size
	CopyData(5, EncodeShort(level.Depth), buffer) // Z size
	
	return buffer
}

func SpawnPlayer(name string, id int, x int, y int, z int, yaw int, pitch int) []byte {
	buffer := make([]byte, 1 + 1 + STRING_LENGTH + 2 + 2 + 2 + 1 + 1)
	
	buffer[0] = SERVER_SPAWN_PLAYER // Packet ID
	buffer[1] = byte(id) // Player ID
	CopyData(2, EncodeString(name), buffer) // Player Name
	CopyData(2 + STRING_LENGTH, EncodeShort((x << 5) + 16), buffer) // X
	CopyData(2 + STRING_LENGTH + 2, EncodeShort((y << 5) + 16), buffer) // Y
	CopyData(2 + STRING_LENGTH + 2 + 2, EncodeShort((z << 5) + 16), buffer) // Z
	buffer[2 + STRING_LENGTH + 2 + 2 + 2] = byte(yaw) // Yaw (Heading)
	buffer[2 + STRING_LENGTH + 2 + 2 + 2 + 1] = byte(pitch) // Pitch
	
	return buffer
}

func DespawnPlayer(id int) []byte {
	buffer := make([]byte, 1 + 1)
	
	buffer[0] = SERVER_DESPAWN_PLAYER // Packet ID
	buffer[1] = byte(id) // Player ID
	
	return buffer
}

func Disconnect(reason string) []byte {
	buffer := make([]byte, 1 + STRING_LENGTH)
	
	buffer[0] = SERVER_DISCONNECT // Packet ID
	CopyData(1, EncodeString(reason), buffer) // Disconnect reason
	
	return buffer
}

func Message(id int, message string) []byte {
	buffer := make([]byte, 1 + 1 + STRING_LENGTH)
	
	buffer[0] = SERVER_MESSAGE // Packet ID
	buffer[1] = byte(id) // Player ID
	CopyData(2, EncodeString(message), buffer) // Message
	
	return buffer
}

func SetBlock(x int, y int, z int, id int) []byte {
	buffer := make([]byte, 1 + 2 + 2 + 2 + 1)
	
	buffer[0] = SERVER_SET_BLOCK // Packet ID
	CopyData(1, EncodeShort(x), buffer) // X
	CopyData(3, EncodeShort(y), buffer) // Y
	CopyData(5, EncodeShort(z), buffer) // Z
	buffer[7] = byte(id) // Block Type
	
	return buffer
}

func PositionAndOrientation(id int, x int, y int, z int, yaw int, pitch int) []byte {
	buffer := make([]byte, 1 + 1 + 2 + 2 + 2 + 1 + 1)
	
	buffer[0] = SERVER_POSITION_AND_ORIENTATION // Packet ID
	buffer[1] = byte(id) // Player ID
	CopyData(2, EncodeShort(x), buffer) // X
	CopyData(4, EncodeShort(y), buffer) // Y
	CopyData(6, EncodeShort(z), buffer) // Z
	buffer[8] = byte(yaw) // Yaw
	buffer[9] = byte(pitch) // Pitch
	
	return buffer
}

func PositionAndOrientationUpdate(id int, oldX int, oldY int, oldZ int, newX int, newY int, newZ int, yaw int, pitch int) []byte {
	buffer := make([]byte, 1 + 1 + 1 + 1 + 1 + 1 + 1)
	
	buffer[0] = SERVER_POSITION_AND_ORIENTATION_UPDATE // Packet ID
	buffer[1] = byte(id) // Player ID
	
	buffer[2] = byte(newX - oldX)// Change in X
	buffer[3] = byte(newY - oldY) // Change in Y
	buffer[4] = byte(newZ - oldZ) // Change in Z
	
	buffer[5] = byte(yaw) // Yaw
	buffer[6] = byte(pitch) // Pitch
	
	return buffer
}
