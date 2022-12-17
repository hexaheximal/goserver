package protocol

import (
	"goserver/blocks"
	"github.com/aquilax/go-perlin"
	"compress/gzip"
	"encoding/binary"
	"io/ioutil"
	"bytes"
	"strings"
	"crypto/sha256"
	"time"
	"log"
)

const (
	HASH_LENGTH = 32
	STRING_LENGTH = 64
	BYTE_ARRAY_LENGTH = 1024
	PROTOCOL_VERSION = 0x07
)

const (
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

const (
	LEVEL_FLAT = 0
	LEVEL_CLASSIC = 1
	LEVEL_EXPERIMENTAL = 2
)

const (
	LEVEL_TYPE_NORMAL = 0 // Normal levels contain the level data and nothing else.
	LEVEL_TYPE_CHAIN = 1 // Chain levels contain a chain of block updates instead of the level data. Very useful if you need to do a level rollback.
)

// Protocol types

type Level struct {
	Width int // X size
	Height int // Y size
	Depth int // Z size
	Data []byte // block array
	Spawnpoint Spawnpoint // Spawnpoint
	Type int // Level type
	Chain []BlockUpdate // Chain data
}

type BlockUpdate struct {
	X int // X
	Y int // Y
	Z int // Z
	ID byte // Block ID
	Name string // Player name (or blank if it is not by a player)
	PreviousBlock []byte // Hash of the previous block in the chain
}

type Spawnpoint struct {
	X int
	Y int
	Z int
	Yaw byte
	Pitch byte
}

func (blockUpdate BlockUpdate) Serialize() []byte {
	buffer := make([]byte, 2 + 2 + 2 + 1 + STRING_LENGTH + HASH_LENGTH)
	
	CopyData(0, EncodeShort(blockUpdate.X), buffer) // X
	CopyData(2, EncodeShort(blockUpdate.Y), buffer) // Y
	CopyData(4, EncodeShort(blockUpdate.Z), buffer) // Z
	
	buffer[6] = blockUpdate.ID // Block ID
	
	CopyData(7, EncodeString(blockUpdate.Name), buffer) // Player name (or blank if it is not by a player)
	CopyData(7 + STRING_LENGTH, blockUpdate.PreviousBlock, buffer) // Hash of the previous block in the chain
	
	return buffer
}

func DeserializeBlockUpdate(data []byte) BlockUpdate {
	x := DecodeShort(data, 0) // X
	y := DecodeShort(data, 2) // Y
	z := DecodeShort(data, 4) // Z
	
	id := data[6] // Block ID
	
	name := DecodeString(data, 7) // Player name (or blank if it is not by a player)
	
	hashIndex := 7 + STRING_LENGTH
	hashEndIndex := hashIndex + HASH_LENGTH
	previousBlock := data[hashIndex:hashEndIndex] // Hash of the previous block in the chain
	
	return BlockUpdate{x, y, z, id, name, previousBlock}
}

func (level Level) IsOOB(x int, y int, z int) bool {
	if (y * level.Depth + z) * level.Width + x > len(level.Data) - 1 {
		return true
	}
	
	return false
}

func (level Level) GetBlock(x int, y int, z int) byte {
	if level.IsOOB(x, y, z) {
		return blocks.BLOCK_AIR
	}

	return level.Data[(y * level.Depth + z) * level.Width + x]
}

func (level *Level) SetBlock(x int, y int, z int, id byte) {
	level.SetBlockPlayer(x, y, z, id, "")
}

func (level *Level) SetBlockPlayer(x int, y int, z int, id byte, name string) {	
	level.Data[(y * level.Depth + z) * level.Width + x] = id
	
	if level.Type == LEVEL_TYPE_CHAIN {
		block := BlockUpdate{x, y, z, id, name, make([]byte, 32)}
		
		if len(level.Chain) != 0 {
			previousBlockHash := sha256.Sum256(level.Chain[len(level.Chain) - 1].Serialize())
			block.PreviousBlock = previousBlockHash[:]
		}
		
		level.Chain = append(level.Chain, block)
	}
}

func (level Level) Encode() []byte {
	buffer := make([]byte, 4 + len(level.Data))
	
	CopyData(0, EncodeInt(len(level.Data)), buffer)
	CopyData(4, level.Data, buffer)
	
	return buffer
}

func (level Level) Serialize() []byte {
	if level.Type == LEVEL_TYPE_CHAIN {
		blockSize := 2 + 2 + 2 + 1 + STRING_LENGTH + HASH_LENGTH
		
		buffer := make([]byte, 5 + 1 + 2 + 2 + 2 + 2 + 2 + 2 + 1 + 1 + (blockSize * len(level.Chain)))
	
		CopyData(0, []byte("CHAIN"), buffer) // Header
		buffer[5] = 0x01 // Format Version
		
		CopyData(6, EncodeShort(level.Width), buffer) // Width
		CopyData(8, EncodeShort(level.Height), buffer) // Height
		CopyData(10, EncodeShort(level.Depth), buffer) // Depth
	
		CopyData(12, EncodeShort(level.Spawnpoint.X), buffer) // Spawn X
		CopyData(14, EncodeShort(level.Spawnpoint.Y), buffer) // Spawn Y
		CopyData(16, EncodeShort(level.Spawnpoint.Z), buffer) // Spawn Z
	
		buffer[18] = byte(level.Spawnpoint.Yaw) // Spawn Yaw
		buffer[19] = byte(level.Spawnpoint.Pitch) // Spawn Pitch

		for i := 0; i < len(level.Chain); i++ {
			CopyData(20 + (blockSize * i), level.Chain[i].Serialize(), buffer) // Block
		}
		
		return buffer
	}
	
	buffer := make([]byte, 5 + 1 + 2 + 2 + 2 + 2 + 2 + 2 + 1 + 1 + len(level.Data))
	
	CopyData(0, []byte("LEVEL"), buffer) // Header
	buffer[5] = 0x01 // Format Version
	
	CopyData(6, EncodeShort(level.Width), buffer) // Width
	CopyData(8, EncodeShort(level.Height), buffer) // Height
	CopyData(10, EncodeShort(level.Depth), buffer) // Depth
	
	CopyData(12, EncodeShort(level.Spawnpoint.X), buffer) // Spawn X
	CopyData(14, EncodeShort(level.Spawnpoint.Y), buffer) // Spawn Y
	CopyData(16, EncodeShort(level.Spawnpoint.Z), buffer) // Spawn Z
	
	buffer[18] = byte(level.Spawnpoint.Yaw) // Spawn Yaw
	buffer[19] = byte(level.Spawnpoint.Pitch) // Spawn Pitch
	
	CopyData(20, level.Data, buffer)
	
	return buffer
}

func DeserializeLevel(data []byte) Level {
	if bytes.Equal(data[0:5], []byte("CHAIN")) {
		//log.Println("DeserializeLevel(): Level type: Chain")
		
		blockSize := 2 + 2 + 2 + 1 + STRING_LENGTH + HASH_LENGTH
		headerSize := 5 + 1 + 2 + 2 + 2 + 2 + 2 + 2 + 1 + 1
		
		//log.Println("DeserializeLevel(): Deserializing level header...")
		
		if data[5] != 0x01 {
			log.Fatalln("Error: Invalid level format version! Please update goserver!")
		}
		
		width := DecodeShort(data, 6) // Width
		height := DecodeShort(data, 6 + 2) // Height
		depth := DecodeShort(data, 6 + 4) // Depth
		
		spawnX := DecodeShort(data, 6 + 6) // Spawn X
		spawnY := DecodeShort(data, 6 + 8) // Spawn Y
		spawnZ := DecodeShort(data, 6 + 10) // Spawn Z
		
		spawnYaw := data[6 + 12] // Spawn Yaw
		spawnPitch := data[6 + 13] // Spawn Pitch
		
		blockData := data[headerSize:]
		
		level := Level{
			width,
			height,
			depth,
			make([]byte, width * height * depth),
			Spawnpoint{spawnX, spawnY, spawnZ, spawnYaw, spawnPitch},
			LEVEL_TYPE_CHAIN,
			make([]BlockUpdate, 0),
		}
		
		blocks := int(float32(len(blockData)) / float32(blockSize))
		
		//log.Println("DeserializeLevel(): Deserializing and Iterating block updates...")
		
		for i := 0; i < blocks; i++ {
			startIndex := blockSize * i
			endIndex := startIndex + blockSize
			block := DeserializeBlockUpdate(blockData[startIndex:endIndex])
			blockHash := sha256.Sum256(block.Serialize())
			
			if len(level.Chain) > 0 {
				previousBlockHash := sha256.Sum256(level.Chain[len(level.Chain) - 1].Serialize())
				
				if !bytes.Equal(block.PreviousBlock, previousBlockHash[:]) {
					log.Fatalln("Block", blockHash, "contains an invalid previous block hash!")
				}
			}
			
			level.Chain = append(level.Chain, block)
			
			if level.IsOOB(block.X, block.Y, block.Z) {
				log.Fatalln("Block", blockHash, "contains an invalid position!")
			}
			
			level.Data[(block.Y * level.Depth + block.Z) * level.Width + block.X] = byte(block.ID)
		}
		
		//log.Println("DeserializeLevel(): Finished!")
		
		return level
	}
	
	if bytes.Equal(data[0:5], []byte("LEVEL")) {
		headerSize := 5 + 1 + 2 + 2 + 2 + 2 + 2 + 2 + 1 + 1 // header bytes, byte (format version), short, short, short (Level Size), short, short, short (Spawnpoint Position), byte, byte (Spawnpoint Yaw & Pitch)
		
		//log.Println("DeserializeLevel(): Level type: Normal")
		
		//log.Println("DeserializeLevel(): Deserializing level header...")
		
		if data[5] != 0x01 {
			log.Fatalln("Error: Invalid level format version! Please update goserver!")
		}
		
		width := DecodeShort(data, 6) // Width
		height := DecodeShort(data, 8) // Height
		depth := DecodeShort(data, 10) // Depth
		
		spawnX := DecodeShort(data, 12) // Spawn X
		spawnY := DecodeShort(data, 14) // Spawn Y
		spawnZ := DecodeShort(data, 16) // Spawn Z
		
		spawnYaw := data[18] // Spawn Yaw
		spawnPitch := data[19] // Spawn Pitch
		
		level := Level{
			width,
			height,
			depth,
			data[headerSize:],
			Spawnpoint{spawnX, spawnY, spawnZ, spawnYaw, spawnPitch},
			LEVEL_TYPE_NORMAL,
			make([]BlockUpdate, 0),
		}
		
		//log.Println("DeserializeLevel(): Finished!")
		
		return level
	}
	
	log.Fatalln("Invalid level format!")
	
	return Level{16, 16, 16, make([]byte, 16 * 16 * 16), Spawnpoint{0, 0, 0, 0, 0}, LEVEL_TYPE_NORMAL, make([]BlockUpdate, 0)}
}

func GenerateLevel(width int, height int, depth int, level_generation_type int, level_type int) Level {
	//if level_type == LEVEL_TYPE_CHAIN {
	//	panic("Chain levels are not implemented yet!")
	//}
	
	level := Level{
		width,
		height,
		depth,
		make([]byte, width * height * depth),
		Spawnpoint{int(float32(width) / 2.0), 0, int(float32(depth) / 2.0), 0, 0},
		level_type,
		make([]BlockUpdate, 0),
	}
	
	if level_generation_type == LEVEL_FLAT {
		FlatLevelGenerator(&level)
	}
	
	if level_generation_type == LEVEL_EXPERIMENTAL {
		ExperimentalLevelGenerator(&level)
	}
	
	for y := 0; y < height; y++ {
		if level.GetBlock(level.Spawnpoint.X, y, level.Spawnpoint.Z) == blocks.BLOCK_AIR {
			level.Spawnpoint.Y = y + 1
			break
		}
	}
	
	return level
}

func FlatLevelGenerator(level *Level) {
	for y := 0; y < 5; y++ {
		for x := 0; x < level.Width; x++ {
			for z := 0; z < level.Depth; z++ {
				level.SetBlock(x, 0 + y, z, blocks.BLOCK_STONE)
			}
		}
	}
	
	for y := 0; y < 2; y++ {
		for x := 0; x < level.Width; x++ {
			for z := 0; z < level.Depth; z++ {
				level.SetBlock(x, 5 + y, z, blocks.BLOCK_DIRT)
			}
		}
	}
	
	for x := 0; x < level.Width; x++ {
		for z := 0; z < level.Depth; z++ {
			level.SetBlock(x, 7, z, blocks.BLOCK_GRASS)
		}
	}
}

func ExperimentalLevelGenerator(level *Level) {
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
				level.SetBlock(x, 0 + y, z, blocks.BLOCK_STONE)
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
					level.SetBlock(x, 5 + y, z, blocks.BLOCK_STONE)
				}
			} else {
				for y := 0; y < noiseHeight; y++ {
					level.SetBlock(x, 5 + y, z, blocks.BLOCK_DIRT)
				}
				
				level.SetBlock(x, 5 + noiseHeight, z, blocks.BLOCK_GRASS)
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

func SpawnPlayer(name string, id byte, x int, y int, z int, yaw byte, pitch byte) []byte {
	buffer := make([]byte, 1 + 1 + STRING_LENGTH + 2 + 2 + 2 + 1 + 1)
	
	buffer[0] = SERVER_SPAWN_PLAYER // Packet ID
	buffer[1] = id // Player ID
	CopyData(2, EncodeString(name), buffer) // Player Name
	CopyData(2 + STRING_LENGTH, EncodeShort(x), buffer) // X
	CopyData(2 + STRING_LENGTH + 2, EncodeShort(y), buffer) // Y
	CopyData(2 + STRING_LENGTH + 2 + 2, EncodeShort(z), buffer) // Z
	buffer[2 + STRING_LENGTH + 2 + 2 + 2] = yaw // Yaw (Heading)
	buffer[2 + STRING_LENGTH + 2 + 2 + 2 + 1] = pitch // Pitch
	
	return buffer
}

func DespawnPlayer(id byte) []byte {
	buffer := make([]byte, 1 + 1)
	
	buffer[0] = SERVER_DESPAWN_PLAYER // Packet ID
	buffer[1] = id // Player ID
	
	return buffer
}

func Disconnect(reason string) []byte {
	buffer := make([]byte, 1 + STRING_LENGTH)
	
	buffer[0] = SERVER_DISCONNECT // Packet ID
	CopyData(1, EncodeString(reason), buffer) // Disconnect reason
	
	return buffer
}

func Message(id byte, message string) []byte {
	buffer := make([]byte, 1 + 1 + STRING_LENGTH)
	
	buffer[0] = SERVER_MESSAGE // Packet ID
	buffer[1] = id // Player ID
	CopyData(2, EncodeString(message), buffer) // Message
	
	return buffer
}

func SetBlock(x int, y int, z int, id byte) []byte {
	buffer := make([]byte, 1 + 2 + 2 + 2 + 1)
	
	buffer[0] = SERVER_SET_BLOCK // Packet ID
	CopyData(1, EncodeShort(x), buffer) // X
	CopyData(3, EncodeShort(y), buffer) // Y
	CopyData(5, EncodeShort(z), buffer) // Z
	buffer[7] = id // Block Type
	
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
