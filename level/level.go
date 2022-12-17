package level

import (
	"github.com/aquilax/go-perlin"
	"goserver/serialization"
	"goserver/blocks"
	"crypto/sha256"
	"log"
	"bytes"
	"time"
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
	buffer := make([]byte, 2 + 2 + 2 + 1 + serialization.STRING_LENGTH + serialization.HASH_LENGTH)
	
	serialization.CopyData(0, serialization.EncodeShort(blockUpdate.X), buffer) // X
	serialization.CopyData(2, serialization.EncodeShort(blockUpdate.Y), buffer) // Y
	serialization.CopyData(4, serialization.EncodeShort(blockUpdate.Z), buffer) // Z
	
	buffer[6] = blockUpdate.ID // Block ID
	
	serialization.CopyData(7, serialization.EncodeString(blockUpdate.Name), buffer) // Player name (or blank if it is not by a player)
	serialization.CopyData(7 + serialization.STRING_LENGTH, blockUpdate.PreviousBlock, buffer) // Hash of the previous block in the chain
	
	return buffer
}

func DeserializeBlockUpdate(data []byte) BlockUpdate {
	x := serialization.DecodeShort(data, 0) // X
	y := serialization.DecodeShort(data, 2) // Y
	z := serialization.DecodeShort(data, 4) // Z
	
	id := data[6] // Block ID
	
	name := serialization.DecodeString(data, 7) // Player name (or blank if it is not by a player)
	
	hashIndex := 7 + serialization.STRING_LENGTH
	hashEndIndex := hashIndex + serialization.HASH_LENGTH
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
	
	serialization.CopyData(0, serialization.EncodeInt(len(level.Data)), buffer)
	serialization.CopyData(4, level.Data, buffer)
	
	return buffer
}

func (level Level) Serialize() []byte {
	if level.Type == LEVEL_TYPE_CHAIN {
		blockSize := 2 + 2 + 2 + 1 + serialization.STRING_LENGTH + serialization.HASH_LENGTH
		
		buffer := make([]byte, 5 + 1 + 2 + 2 + 2 + 2 + 2 + 2 + 1 + 1 + (blockSize * len(level.Chain)))
	
		serialization.CopyData(0, []byte("CHAIN"), buffer) // Header
		buffer[5] = 0x01 // Format Version
		
		serialization.CopyData(6, serialization.EncodeShort(level.Width), buffer) // Width
		serialization.CopyData(8, serialization.EncodeShort(level.Height), buffer) // Height
		serialization.CopyData(10, serialization.EncodeShort(level.Depth), buffer) // Depth
	
		serialization.CopyData(12, serialization.EncodeShort(level.Spawnpoint.X), buffer) // Spawn X
		serialization.CopyData(14, serialization.EncodeShort(level.Spawnpoint.Y), buffer) // Spawn Y
		serialization.CopyData(16, serialization.EncodeShort(level.Spawnpoint.Z), buffer) // Spawn Z
	
		buffer[18] = byte(level.Spawnpoint.Yaw) // Spawn Yaw
		buffer[19] = byte(level.Spawnpoint.Pitch) // Spawn Pitch

		for i := 0; i < len(level.Chain); i++ {
			serialization.CopyData(20 + (blockSize * i), level.Chain[i].Serialize(), buffer) // Block
		}
		
		return buffer
	}
	
	buffer := make([]byte, 5 + 1 + 2 + 2 + 2 + 2 + 2 + 2 + 1 + 1 + len(level.Data))
	
	serialization.CopyData(0, []byte("LEVEL"), buffer) // Header
	buffer[5] = 0x01 // Format Version
	
	serialization.CopyData(6, serialization.EncodeShort(level.Width), buffer) // Width
	serialization.CopyData(8, serialization.EncodeShort(level.Height), buffer) // Height
	serialization.CopyData(10, serialization.EncodeShort(level.Depth), buffer) // Depth
	
	serialization.CopyData(12, serialization.EncodeShort(level.Spawnpoint.X), buffer) // Spawn X
	serialization.CopyData(14, serialization.EncodeShort(level.Spawnpoint.Y), buffer) // Spawn Y
	serialization.CopyData(16, serialization.EncodeShort(level.Spawnpoint.Z), buffer) // Spawn Z
	
	buffer[18] = byte(level.Spawnpoint.Yaw) // Spawn Yaw
	buffer[19] = byte(level.Spawnpoint.Pitch) // Spawn Pitch
	
	serialization.CopyData(20, level.Data, buffer)
	
	return buffer
}

func DeserializeLevel(data []byte) Level {
	if bytes.Equal(data[0:5], []byte("CHAIN")) {
		//log.Println("DeserializeLevel(): Level type: Chain")
		
		blockSize := 2 + 2 + 2 + 1 + serialization.STRING_LENGTH + serialization.HASH_LENGTH
		headerSize := 5 + 1 + 2 + 2 + 2 + 2 + 2 + 2 + 1 + 1
		
		//log.Println("DeserializeLevel(): Deserializing level header...")
		
		if data[5] != 0x01 {
			log.Fatalln("Error: Invalid level format version! Please update goserver!")
		}
		
		width := serialization.DecodeShort(data, 6) // Width
		height := serialization.DecodeShort(data, 6 + 2) // Height
		depth := serialization.DecodeShort(data, 6 + 4) // Depth
		
		spawnX := serialization.DecodeShort(data, 6 + 6) // Spawn X
		spawnY := serialization.DecodeShort(data, 6 + 8) // Spawn Y
		spawnZ := serialization.DecodeShort(data, 6 + 10) // Spawn Z
		
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
		
		width := serialization.DecodeShort(data, 6) // Width
		height := serialization.DecodeShort(data, 8) // Height
		depth := serialization.DecodeShort(data, 10) // Depth
		
		spawnX := serialization.DecodeShort(data, 12) // Spawn X
		spawnY := serialization.DecodeShort(data, 14) // Spawn Y
		spawnZ := serialization.DecodeShort(data, 16) // Spawn Z
		
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