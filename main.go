package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"goserver/config"
	"goserver/level"
	"goserver/packet"
	"goserver/protocol"
	"goserver/serialization"
	"goserver/compression"
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

var serverLevel level.Level
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

			serverLevel = level.DeserializeLevel(compression.DecompressData(content))

			if serverLevel.Type == level.LEVEL_TYPE_NORMAL {
				log.Fatalln("Level history is only available in chain levels.")
			}

			for i := 0; i < len(serverLevel.Chain); i++ {
				block := serverLevel.Chain[i]

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

		levelType := level.LEVEL_TYPE_NORMAL

		if len(os.Args) > 1 && os.Args[1] == "--chain-level" {
			levelType = level.LEVEL_TYPE_CHAIN
		}

		serverLevel = level.GenerateLevel(128, 64, 128, level.LEVEL_EXPERIMENTAL, levelType)
	} else {
		log.Println("Loading level...")
		content, err := ioutil.ReadFile(MAIN_LEVEL_FILE)

		if err != nil {
			panic(err)
		}

		serverLevel = level.DeserializeLevel(compression.DecompressData(content))
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

	err := ioutil.WriteFile(MAIN_LEVEL_FILE, compression.CompressData(serverLevel.Serialize()), 0644)

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

func SendToAllClients(exclude byte, w *packet.PacketWriter) {
	data := w.Buffer
	w.Buffer = make([]byte, 0)

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

func SendInitialData(r *packet.PacketReader, w *packet.PacketWriter, id byte) {
	r.Reset()
	r.ReadByte()

	if r.ReadByte() != byte(0x07) {
		protocol.WriteDisconnect(w, "Incorrect protocol version!")
		w.WriteToSocket(clients[id].Socket)
		clients[id].Socket.Close()
		return
	}

	username := r.ReadString()

	// TODO: player auth
	//token := r.ReadString()

	protocol.WriteServerIdentification(w, serverConfig.GetString("server-name"), serverConfig.GetString("motd"), false) // Server Identification
	w.WriteToSocket(clients[id].Socket)
	
	// TODO: change this
	clients[id].Socket.Write([]byte{protocol.SERVER_LEVEL_INITIALIZE}) // Level Initialize

	splitCompressedEncodedLevel := serialization.SplitData(compression.CompressData(serverLevel.Encode()), 1024)

	for i := 0; i < len(splitCompressedEncodedLevel); i++ {
		percentage := byte((float32(i+1) / float32(len(splitCompressedEncodedLevel))) * 100)
		protocol.WriteLevelDataChunk(w, splitCompressedEncodedLevel[i], percentage) // Level Data Chunk
		w.WriteToSocket(clients[id].Socket)
	}

	protocol.WriteLevelFinalize(w, serverLevel) // Level Finalize
	w.WriteToSocket(clients[id].Socket)

	clients[id].X = int(float32(serverLevel.Spawnpoint.X) * 32.0)
	clients[id].Y = int(float32(serverLevel.Spawnpoint.Y) * 32.0)
	clients[id].Z = int(float32(serverLevel.Spawnpoint.Z) * 32.0)
	clients[id].Yaw = serverLevel.Spawnpoint.Yaw
	clients[id].Pitch = serverLevel.Spawnpoint.Pitch

	// Spawn Player

	protocol.WriteSpawnPlayer(w, clients[id].Username, 0xff, (serverLevel.Spawnpoint.X<<5)+16, (serverLevel.Spawnpoint.Y<<5)+16, (serverLevel.Spawnpoint.Z<<5)+16, clients[id].Yaw, clients[id].Pitch)
	w.WriteToSocket(clients[id].Socket)

	protocol.WriteSpawnPlayer(w, clients[id].Username, clients[id].ID, clients[id].X, clients[id].Y, clients[id].Z, clients[id].Yaw, clients[id].Pitch)
	SendToAllClients(clients[id].ID, w)

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
			protocol.WriteMessage(w, 126, line)
			w.WriteToSocket(clients[id].Socket)
		}

		// Send a blank line at the end if it wasn't already sent

		log.Println(len(lines[len(lines)-1]))

		if len(lines[len(lines)-1]) != 0 {
			protocol.WriteMessage(w, 126, "")
			w.WriteToSocket(clients[id].Socket)
		}
	}

	protocol.WriteMessage(w, 0xff, username+" joined the game")
	SendToAllClients(0xff, w) // Send join message

	for i := 0; i < len(clients); i++ {
		if i == int(clients[id].ID) || clients[i] == NULL_CLIENT {
			continue
		}

		protocol.WriteSpawnPlayer(w, clients[i].Username, byte(i), clients[i].X, clients[i].Y, clients[i].Z, clients[i].Yaw, clients[i].Pitch)
		w.WriteToSocket(clients[id].Socket)
	}
}

func HandleMessage(r *packet.PacketReader, w *packet.PacketWriter, id byte) {
	if r.ReadByte() == byte(0x00) && r.ReadByte() != byte(0x00) {
		r.Reset()
		SendInitialData(r, w, id)
		return
	}

	/*if buffer[0] == byte(0x05) {
		// TODO: reimplement the anti-cheat code for this

		x := serialization.DecodeShort(buffer, 1)
		y := serialization.DecodeShort(buffer, 3)
		z := serialization.DecodeShort(buffer, 5)

		if serverLevel.IsOOB(x, y, z) {
			return
		}

		block_type := buffer[8]

		if buffer[7] != 0x01 {
			block_type = blocks.BLOCK_AIR
		}

		if block_type > blocks.BLOCK_OBSIDIAN {
			conn.Write(protocol.Disconnect("Invalid block!"))
			conn.Close()
			return
		}

		if block_type == blocks.BLOCK_DIRT && serverLevel.GetBlock(x, y+1, z) == blocks.BLOCK_AIR {
			serverLevel.SetBlockPlayer(x, y, z, blocks.BLOCK_GRASS, username)
			SendToAllClients(0xff, protocol.SetBlock(x, y, z, blocks.BLOCK_GRASS))
			return
		}

		serverLevel.SetBlockPlayer(x, y, z, block_type, username)
		SendToAllClients(0xff, protocol.SetBlock(x, y, z, block_type))

		return
	}

	if buffer[0] == byte(0x08) {
		x := serialization.DecodeShort(buffer, 2)
		y := serialization.DecodeShort(buffer, 4)
		z := serialization.DecodeShort(buffer, 6)

		clients[id].Yaw = buffer[8]
		clients[id].Pitch = buffer[9]

		SendToAllClients(id, protocol.PositionAndOrientationUpdate(id, clients[id].X, clients[id].Y, clients[id].Z, x, y, z, clients[id].Yaw, clients[id].Pitch))

		clients[id].X = x
		clients[id].Y = y
		clients[id].Z = z

		return
	}

	if buffer[0] == byte(0x0d) {
		message := serialization.DecodeString(buffer, 2)

		if len(message) == 0 {
			return
		}

		if message[0] == byte('/') {
			return
		}

		log.Println(username + ": " + message)
		SendToAllClients(0xff, protocol.Message(id, username+": "+message))
		return
	}*/
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

	w := packet.CreatePacketWriter()

	if slot_assigned == false {
		protocol.WriteDisconnect(&w, "The server is full!")
		w.WriteToSocket(conn)
		log.Println("Closed Connection:", conn.RemoteAddr())
		return
	}

	clients[client_index] = Client{"", client_index, 0, 0, 0, 0, 0, conn}

	for {
		buffer := make([]byte, 512)
		_, err := conn.Read(buffer)

		r := packet.CreatePacketReader(buffer)

		if err != nil {
			conn.Close()

			protocol.WriteDespawnPlayer(&w, client_index)
			SendToAllClients(0xff, &w)

			protocol.WriteMessage(&w, 0xff, clients[client_index].Username+" left the game")
			SendToAllClients(0xff, &w)

			clients[client_index] = NULL_CLIENT

			log.Println("Closed Connection:", conn.RemoteAddr())
			return
		}

		// respond

		/*if buffer[0] == byte(0x08) {
			packet_length := 1 + 1 + 2 + 2 + 2 + 1 + 1
			HandleMessage(conn, buffer, clients[client_index].Username, client_index)

			if buffer[packet_length] != byte(0x00) {
				HandleMessage(conn, buffer[packet_length:], clients[client_index].Username, client_index)
			}

			continue
		}

		if buffer[0] == byte(0x00) && buffer[1] != byte(0x00) {
			clients[client_index].Username = serialization.DecodeString(buffer, 2)
		}*/

		HandleMessage(&r, &w, client_index)

		//conn.Close()
		//log.Println("Closed Connection:", conn.RemoteAddr())
		//return
	}

	conn.Close()
}
