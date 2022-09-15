package main

import (
	"errors"
	"goserver/protocol"
	"goserver/config"
	"io/ioutil"
	"log"
	"net"
	"os"
	"runtime"
	"os/signal"
	"syscall"
	"time"
)

var level protocol.Level
var clients []Client
var serverConfig config.Config

type Client struct {
	Username string
	ID int
	X int
	Y int
	Z int
	Yaw int
	Pitch int
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
	
	log.Println("Starting server...")

	NULL_CLIENT = Client{"", 0, 0, 0, 0, 0, 0, nil}

	clients = make([]Client, 32)
	
	if _, err := os.Stat("server.properties"); errors.Is(err, os.ErrNotExist) {
		log.Println("Creating server.properties...")
		
		configData := "# Minecraft server properties (goserver)\nserver-name=Minecraft Server\nmotd=Welcome to my Minecraft Server!\npublic=false\nport=25565\nverify-names=false\nmax-players=32\nmax-connections=1\ngrow-trees=false\nadmin-slot=false"
		err := ioutil.WriteFile("server.properties", []byte(configData), 0644)
		
		if err != nil {
			log.Fatalln("Failed to create server.properties:", err)
		}
		
		serverConfig = config.ParseConfig(string(configData))
		
		log.Println(serverConfig)
	} else {
		log.Println("Reading server.properties...")
		
		content, err := ioutil.ReadFile("server.properties")

		if err != nil {
			panic(err)
		}

		serverConfig = config.ParseConfig(string(content))
		
		log.Println(serverConfig)
	}

	if _, err := os.Stat(MAIN_LEVEL_FILE); errors.Is(err, os.ErrNotExist) {
		log.Println("Generating level...")
		level = protocol.GenerateLevel(128, 64, 128, protocol.LEVEL_EXPERIMENTAL)
	} else {
		log.Println("Loading level...")
		content, err := ioutil.ReadFile(MAIN_LEVEL_FILE)

		if err != nil {
			panic(err)
		}

		level = protocol.DeserializeLevel(protocol.DecompressData(content))
	}

	listen, err := net.Listen("tcp", "127.0.0.1:" + serverConfig.GetString("port"))

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

func SendToAllClients(exclude int, data []byte) {
	for i := 0; i < len(clients); i++ {
		if clients[i] == NULL_CLIENT || i == exclude {
			continue
		}

		clients[i].Socket.Write(data)
	}
}

func SendInitialData(conn net.Conn, buffer []byte, id int) {
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
	
	clients[id].X = level.Spawnpoint.X
	clients[id].Y = level.Spawnpoint.Y
	clients[id].Z = level.Spawnpoint.Z
	clients[id].Yaw = level.Spawnpoint.Yaw
	clients[id].Pitch = level.Spawnpoint.Pitch
	
	// Spawn Player

	conn.Write(protocol.SpawnPlayer(username, 0xff, clients[id].X, clients[id].Y, clients[id].Z, clients[id].Yaw, clients[id].Pitch))
	
	// TODO: FIX THIS!!!
	//SendToAllClients(id, protocol.SpawnPlayer(username, id, clients[id].X, clients[id].Y, clients[id].Z, clients[id].Yaw, clients[id].Pitch))

	joinMessage := protocol.Message(0xff, username+" joined the game")

	SendToAllClients(-1, joinMessage) // Send join message
	
	// TODO: FIX THIS!!!
	/*for i := 0; i < len(clients); i++ {
		if i == id || clients[i] == NULL_CLIENT {
			continue
		}
		
		conn.Write(protocol.SpawnPlayer(clients[i].Username, i, clients[i].X, clients[i].Y, clients[i].Z, clients[i].Yaw, clients[i].Pitch))
		//conn.Write(protocol.PositionAndOrientation(i, clients[i].X, clients[i].Y, clients[i].Z, clients[i].Yaw, clients[i].Pitch))
	}*/
}

func HandleMessage(conn net.Conn, buffer []byte, username string, id int) {
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

		update_type := int(buffer[7])
		block_type := int(buffer[8])

		if update_type != 0x01 {
			block_type = 0
		}

		// Set the block type to grass if the original block type is dirt and there is an air block above it

		if block_type == protocol.BLOCK_DIRT && !level.IsOOB(x, y+1, z) && level.GetBlock(x, y+1, z) == 0 {
			block_type = protocol.BLOCK_GRASS
		}

		level.SetBlock(x, y, z, block_type)
		log.Println(x, y, z, "updated with ID", block_type)

		SendToAllClients(-1, protocol.SetBlock(x, y, z, block_type))

		// If there is grass below the block (and block_type is not air, glass, or leaves), set it to dirt

		if block_type != protocol.BLOCK_AIR && block_type != protocol.BLOCK_GLASS && block_type != protocol.BLOCK_LEAVES && !level.IsOOB(x, y-1, z) && level.GetBlock(x, y-1, z) == protocol.BLOCK_GRASS {
			level.SetBlock(x, y-1, z, protocol.BLOCK_DIRT)
			SendToAllClients(-1, protocol.SetBlock(x, y-1, z, protocol.BLOCK_DIRT))
		}

		// If there is dirt below the block (and block_type is air), set it to grass

		if block_type == protocol.BLOCK_AIR && !level.IsOOB(x, y-1, z) && level.GetBlock(x, y-1, z) == protocol.BLOCK_DIRT {
			level.SetBlock(x, y-1, z, protocol.BLOCK_GRASS)
			SendToAllClients(-1, protocol.SetBlock(x, y-1, z, protocol.BLOCK_GRASS))
		}

		return
	}
	
	if buffer[0] == byte(0x08) {
		x := protocol.DecodeShort(buffer, 2)
		y := protocol.DecodeShort(buffer, 4)
		z := protocol.DecodeShort(buffer, 6)
		
		clients[id].Yaw = int(buffer[8])
		clients[id].Pitch = int(buffer[9])
		
		// TODO: FIX THIS!!!
		//SendToAllClients(id, protocol.PositionAndOrientation(id, x, y, z, clients[id].Yaw, clients[id].Pitch))
		//SendToAllClients(id, protocol.PositionAndOrientationUpdate(id, clients[id].X, clients[id].Y, clients[id].Z, x, y, z, clients[id].Yaw, clients[id].Pitch))
		
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

		SendToAllClients(-1, protocol.Message(0x00, username+": "+message))
		return
	}
}

func HandleConnection(conn net.Conn) {
	client_index := -1

	for i := 0; i < len(clients); i++ {
		if clients[i] == NULL_CLIENT {
			client_index = i
			break
		}
	}

	if client_index == -1 {
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

			// TODO: FIX THIS!!!
			//SendToAllClients(-1, protocol.DespawnPlayer(client_index))
			SendToAllClients(-1, protocol.Message(0xff, clients[client_index].Username + " left the game"))
			
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
