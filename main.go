package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"goserver/config"
	"goserver/protocol"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
)

var level protocol.Level
var clients []Client
var serverConfig config.Config

type Client struct {
	Username string
	ID       byte
	X        int
	Y        int
	Z        int
	Yaw      byte
	Pitch    byte
	Socket   net.Conn
}

var NULL_CLIENT Client

const (
	MAIN_LEVEL_FILE = "main.level"
)

// Server code

func main() {
	if runtime.GOOS == "windows" {
		log.Fatalln("Windows is not supported.")
	}

	if len(os.Args) > 1 && os.Args[1] == "levelhistory" {
		if _, err := os.Stat(MAIN_LEVEL_FILE); errors.Is(err, os.ErrNotExist) {
			log.Fatalln("The level file does not exist!")
		} else {
			content, err := ioutil.ReadFile(MAIN_LEVEL_FILE)

			if err != nil {
				panic(err)
			}

			level = protocol.DeserializeLevel(protocol.DecompressData(content))

			if level.Type == protocol.LEVEL_TYPE_NORMAL {
				log.Fatalln("Level history is only available in chain levels.")
			}

			for i := 0; i < len(level.Chain); i++ {
				block := level.Chain[i]

				if block.Name == "" {
					if block.ID == 0 {
						fmt.Printf("%x: Block at %d, %d, %d removed\n", sha256.Sum256(block.Serialize()), block.X, block.Y, block.Z)
					} else {
						fmt.Printf("%x: Block at %d, %d, %d set to ID %d\n", sha256.Sum256(block.Serialize()), block.X, block.Y, block.Z, block.ID)
					}

					continue
				}

				if block.ID == 0 {
					fmt.Printf("%x: Block at %d, %d, %d removed by %s\n", sha256.Sum256(block.Serialize()), block.X, block.Y, block.Z, block.Name)
				} else {
					fmt.Printf("%x: Block at %d, %d, %d set to ID %d by %s\n", sha256.Sum256(block.Serialize()), block.X, block.Y, block.Z, block.ID, block.Name)
				}
			}
		}

		return
	}

	log.Println("Starting server...")

	NULL_CLIENT = Client{"", 0, 0, 0, 0, 0, 0, nil}

	clients = make([]Client, 32)

	// Load config

	if _, err := os.Stat("server.properties"); errors.Is(err, os.ErrNotExist) {
		log.Println("Creating server.properties...")

		configData := "# Minecraft server properties (goserver)\nserver-name=Minecraft Server\nmotd=Welcome to my Minecraft Server!\npublic=false\nport=25565\nverify-names=false\nmax-players=32\nmax-connections=1\ngrow-trees=false\nadmin-slot=false"
		err := ioutil.WriteFile("server.properties", []byte(configData), 0644)

		if err != nil {
			log.Fatalln("Failed to create server.properties:", err)
		}

		serverConfig = config.ParseConfig(string(configData))
	} else {
		log.Println("Reading server.properties...")

		content, err := ioutil.ReadFile("server.properties")

		if err != nil {
			panic(err)
		}

		serverConfig = config.ParseConfig(string(content))
	}

	// Load level

	if _, err := os.Stat(MAIN_LEVEL_FILE); errors.Is(err, os.ErrNotExist) {
		log.Println("Generating level...")

		levelType := protocol.LEVEL_TYPE_NORMAL

		if len(os.Args) > 1 && os.Args[1] == "--chain-level" {
			levelType = protocol.LEVEL_TYPE_CHAIN
		}

		level = protocol.GenerateLevel(128, 64, 128, protocol.LEVEL_EXPERIMENTAL, levelType)
	} else {
		log.Println("Loading level...")
		content, err := ioutil.ReadFile(MAIN_LEVEL_FILE)

		if err != nil {
			panic(err)
		}

		level = protocol.DeserializeLevel(protocol.DecompressData(content))
	}

	listen, err := net.Listen("tcp", "127.0.0.1:"+serverConfig.GetString("port"))

	if err != nil {
		log.Fatalln(err)
	}

	// close listener

	defer listen.Close()

	c := make(chan os.Signal)

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down...")
		SaveLevel()
		os.Exit(0)
	}()

	log.Println("Starting level save thread...")

	go LevelSaveThread()

	log.Println("Listening for clients...")

	for {
		conn, err := listen.Accept()

		if err != nil {
			log.Fatalln(err)
		}

		log.Println("Accepted Connection:", conn.RemoteAddr())
		go HandleConnection(conn)
	}
}

func SaveLevel() {
	log.Println("Saving level...")

	err := ioutil.WriteFile(MAIN_LEVEL_FILE, protocol.CompressData(level.Serialize()), 0644)

	if err != nil {
		log.Println("Failed to save level:", err)
	} else {
		log.Println("Level saved!")
	}
}

func LevelSaveThread() {
	for {
		SaveLevel()
		time.Sleep(time.Minute * 5)
	}
}

func SendToAllClients(exclude byte, data []byte) {
	for i := 0; i < len(clients); i++ {
		if clients[i] == NULL_CLIENT {
			continue
		}

		if exclude != 0xff && byte(i) == exclude {
			continue
		}

		clients[i].Socket.Write(data)
	}
}

func SendInitialData(conn net.Conn, buffer []byte, id byte) {
	if buffer[1] != byte(0x07) {
		conn.Write(protocol.Disconnect("Incorrect protocol version!"))
		conn.Close()
		return
	}

	username := protocol.DecodeString(buffer, 2)

	conn.Write(protocol.ServerIdentification(serverConfig.GetString("server-name"), serverConfig.GetString("motd"), false)) // Server Identification

	conn.Write([]byte{protocol.SERVER_LEVEL_INITIALIZE}) // Level Initialize

	splitCompressedEncodedLevel := protocol.SplitData(protocol.CompressData(level.Encode()), 1024)

	for i := 0; i < len(splitCompressedEncodedLevel); i++ {
		percentage := int((float32(i+1) / float32(len(splitCompressedEncodedLevel))) * 100)
		conn.Write(protocol.LevelDataChunk(splitCompressedEncodedLevel[i], percentage)) // Level Data Chunk
	}

	conn.Write(protocol.LevelFinalize(level)) // Level Finalize

	clients[id].X = int(float32(level.Spawnpoint.X) * 32.0)
	clients[id].Y = int(float32(level.Spawnpoint.Y) * 32.0)
	clients[id].Z = int(float32(level.Spawnpoint.Z) * 32.0)
	clients[id].Yaw = level.Spawnpoint.Yaw
	clients[id].Pitch = level.Spawnpoint.Pitch

	// Spawn Player

	conn.Write(protocol.SpawnPlayer(username, 0xff, (level.Spawnpoint.X<<5)+16, (level.Spawnpoint.Y<<5)+16, (level.Spawnpoint.Z<<5)+16, clients[id].Yaw, clients[id].Pitch))

	SendToAllClients(id, protocol.SpawnPlayer(username, id, clients[id].X, clients[id].Y, clients[id].Z, clients[id].Yaw, clients[id].Pitch))

	if _, err := os.Stat("welcome.txt"); errors.Is(err, os.ErrNotExist) {
		log.Println("Cannot find welcome.txt, not showing welcome message.")
	} else {
		welcomeMessageData, err := ioutil.ReadFile("welcome.txt")

		if err != nil {
			log.Println("Failed to load welcome.txt, but the file exists! Something is broken!")
			log.Println("Here is the complete error message:")
			log.Println(err)
			return
		}

		lines := strings.Split(string(welcomeMessageData), "\n")

		// Send the welcome message to the client

		for _, line := range lines {
			conn.Write(protocol.Message(126, line))
		}

		// Send a blank line at the end if it wasn't already sent

		log.Println(len(lines[len(lines) - 1]))

		if len(lines[len(lines) - 1]) != 0 {
			conn.Write(protocol.Message(126, ""))
		}
	}

	SendToAllClients(0xff, protocol.Message(0xff, username+" joined the game")) // Send join message

	for i := 0; i < len(clients); i++ {
		if i == int(id) || clients[i] == NULL_CLIENT {
			continue
		}

		conn.Write(protocol.SpawnPlayer(clients[i].Username, byte(i), clients[i].X, clients[i].Y, clients[i].Z, clients[i].Yaw, clients[i].Pitch))
	}
}

func HandleMessage(conn net.Conn, buffer []byte, username string, id byte) {
	if buffer[0] == byte(0x00) && buffer[1] != byte(0x00) {
		SendInitialData(conn, buffer, id)
		return
	}

	if buffer[0] == byte(0x05) {
		// TODO: reimplement the anti-cheat code for this

		x := protocol.DecodeShort(buffer, 1)
		y := protocol.DecodeShort(buffer, 3)
		z := protocol.DecodeShort(buffer, 5)

		if level.IsOOB(x, y, z) {
			return
		}

		block_type := buffer[8]

		if buffer[7] != 0x01 {
			block_type = protocol.BLOCK_AIR
		}

		if block_type > protocol.BLOCK_OBSIDIAN {
			conn.Write(protocol.Disconnect("Invalid block!"))
			conn.Close()
			return
		}

		if block_type == protocol.BLOCK_DIRT && level.GetBlock(x, y + 1, z) == protocol.BLOCK_AIR {
			SendToAllClients(0xff, protocol.SetBlock(x, y, z, protocol.BLOCK_GRASS))
			return
		}

		SendToAllClients(0xff, protocol.SetBlock(x, y, z, block_type))

		return
	}

	if buffer[0] == byte(0x08) {
		x := protocol.DecodeShort(buffer, 2)
		y := protocol.DecodeShort(buffer, 4)
		z := protocol.DecodeShort(buffer, 6)

		clients[id].Yaw = buffer[8]
		clients[id].Pitch = buffer[9]

		SendToAllClients(id, protocol.PositionAndOrientationUpdate(id, clients[id].X, clients[id].Y, clients[id].Z, x, y, z, clients[id].Yaw, clients[id].Pitch))

		clients[id].X = x
		clients[id].Y = y
		clients[id].Z = z

		return
	}

	if buffer[0] == byte(0x0d) {
		message := protocol.DecodeString(buffer, 2)

		if len(message) == 0 {
			return
		}

		if message[0] == byte('/') {
			return
		}

		log.Println(username + ": " + message)
		SendToAllClients(0xff, protocol.Message(id, username+": "+message))
		return
	}
}

func HandleConnection(conn net.Conn) {
	client_index := byte(0)
	slot_assigned := false

	for i := byte(0); i < byte(len(clients)); i++ {
		if clients[i] == NULL_CLIENT {
			client_index = i
			slot_assigned = true
			break
		}
	}

	if slot_assigned == false {
		conn.Write(protocol.Disconnect("The server is full!"))
		log.Println("Closed Connection:", conn.RemoteAddr())
		return
	}

	clients[client_index] = Client{"", client_index, 0, 0, 0, 0, 0, conn}

	for {
		buffer := make([]byte, 512)
		_, err := conn.Read(buffer)

		if err != nil {
			conn.Close()

			SendToAllClients(0xff, protocol.DespawnPlayer(client_index))
			SendToAllClients(0xff, protocol.Message(0xff, clients[client_index].Username+" left the game"))

			clients[client_index] = NULL_CLIENT

			log.Println("Closed Connection:", conn.RemoteAddr())
			return
		}

		// respond

		if buffer[0] == byte(0x08) {
			packet_length := 1 + 1 + 2 + 2 + 2 + 1 + 1
			HandleMessage(conn, buffer, clients[client_index].Username, client_index)

			if buffer[packet_length] != byte(0x00) {
				HandleMessage(conn, buffer[packet_length:], clients[client_index].Username, client_index)
			}

			continue
		}

		if buffer[0] == byte(0x00) && buffer[1] != byte(0x00) {
			clients[client_index].Username = protocol.DecodeString(buffer, 2)
		}

		HandleMessage(conn, buffer, clients[client_index].Username, client_index)

		//conn.Close()
		//log.Println("Closed Connection:", conn.RemoteAddr())
		//return
	}

	conn.Close()
}
