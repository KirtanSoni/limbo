package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"

	"bufio"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"golang.org/x/exp/rand"
)

// Global State
var (

	//go:embed frontend/dist/*
	reactApp embed.FS
	addr     = flag.String("addr", ":8080", "specify the port of the server")
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	allRooms = make(map[string]*Room)
	mu       = sync.RWMutex{}
)

// middleware
type Logger struct {
	handler http.Handler
}

func (l *Logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	l.handler.ServeHTTP(w, r)
	log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
}

func NewLogger(handlerToWrap http.Handler) *Logger {
	return &Logger{handlerToWrap}
}

func main() {
	flag.Parse()
	mux := http.NewServeMux()
	dist, err := fs.Sub(reactApp, "frontend/dist")

	if err != nil {
		log.Print("React Build Not found!!")
		panic(err)
	}

	// server react app
	frontend := http.FileServer(http.FS(dist))
	mux.Handle("GET /", frontend)

	// websocket chat endpoints
	mux.HandleFunc("POST /create-room", CreateRoomHandler)
	mux.HandleFunc("GET /join-room/{id}", JoinRoomHandler)

	// add middleware
	wrappedmux := NewLogger(mux)

	log.Println("Starting Server at :8080 ....")
	go startDebugCLI()
	http.ListenAndServe(*addr, wrappedmux)
}

// Handlers
func CreateRoomHandler(w http.ResponseWriter, r *http.Request) {

	const letterBytes = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 6)
	for i := range code {
		code[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	roomCode := string(code)
	fmt.Println(roomCode)
	err := CreateNewRoom(roomCode)
	if err != nil {
		http.Error(w, "Failed to create room", http.StatusInternalServerError)
		return
	}

	// Return the room code as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"code": roomCode,
	})
}

func JoinRoomHandler(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("id")
	if code == "" {
		http.Error(w, "No room code provided", http.StatusBadRequest)
		return
	}

	Room, ok := allRooms[code]
	if !ok {
		http.Error(w, "Room does not exist", http.StatusNotFound)
		return
	}

	serveWS(Room, w, r)
}

func serveWS(Room *Room, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading connection: %v", err)
		return
	}

	client := &Client{Room, conn, &sync.Mutex{}}
	Room.JoinClient(client)
	defer Room.KickClient(client)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			_, message, err := client.Conn.ReadMessage()
			msg := Message(message)
			Room.buffer <- BroadcastMessage{Message: &msg, Sender: client}
			if err != nil {
				log.Printf("Error reading message: %v", err)
				return
			}
			// client.WriteSafeData(Message(message))
		}
	}()
	wg.Wait()
}

// Structs
type Client struct {
	*Room
	*websocket.Conn
	*sync.Mutex
}

func (c *Client) WriteToClient(msg Message) {
	c.Lock()
	defer c.Unlock()
	c.Conn.WriteMessage(websocket.TextMessage, msg)
}

type Room struct {
	code    string
	clients map[*Client]bool
	safe    sync.RWMutex
	buffer  chan BroadcastMessage

	quit chan struct{}
}

type BroadcastMessage struct {
	Message *Message
	Sender  *Client
}

type Message []byte

// room threadsafe join and leave
func (r *Room) JoinClient(c *Client) {
	r.safe.Lock()
	r.clients[c] = true
	r.safe.Unlock()
}

func (r *Room) KickClient(c *Client) {

	log.Printf("Client %v leaving room %s", c.RemoteAddr(), r.code)

	r.safe.Lock()
	r.clients[c] = false
	c.Close()
	delete(r.clients, c)
	r.safe.Unlock()
}

func (r *Room) Run() {
	for {
		select {
		case msg := <-r.buffer:
			for c := range r.clients {
				if c == msg.Sender {
					continue
				}
				c.WriteToClient(*msg.Message)
			}
		case <-r.quit:
			DeleteRoom(r)
			log.Println("quitting room")
			return
		}
	}
}

func DeleteRoom(r *Room) {
	mu.Lock()
	defer mu.Unlock()

	close(r.quit)
	close(r.buffer)

	delete(allRooms, r.code)

}
func CreateNewRoom(code string) error {
	mu.Lock()
	defer mu.Unlock()

	_, ok := allRooms[code]
	if ok {
		return fmt.Errorf("room with code %s already exists", code)
	}

	allRooms[code] = &Room{
		code:    code,
		clients: make(map[*Client]bool),
		safe:    sync.RWMutex{},
		buffer:  make(chan BroadcastMessage, 10),
		quit:    make(chan struct{}),
	}
	go allRooms[code].Run()
	return nil
}

// debug CLI
func startDebugCLI() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Debug CLI started. Type 'help' for commands.")

	for scanner.Scan() {
		command := scanner.Text()
		parts := strings.Fields(command)

		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "list":
			if len(parts) != 2 {
				fmt.Println("Usage: list <room_code>")
				continue
			}
			printRoomClients(parts[1])

		case "rooms":
			printAllRooms()

		case "help":
			fmt.Println("Available commands:")
			fmt.Println("  list <room_code> - List all clients in a specific room")
			fmt.Println("  rooms           - List all active rooms")
			fmt.Println("  help            - Show this help message")
			fmt.Println("  exit            - Exit the debug CLI")

		case "exit":
			fmt.Println("Exiting debug CLI...")
			return

		default:
			fmt.Println("Unknown command. Type 'help' for available commands.")
		}
	}
}

func printRoomClients(roomCode string) {
	mu.RLock()
	room, exists := allRooms[roomCode]
	mu.RUnlock()

	if !exists {
		fmt.Printf("Room %s does not exist\n", roomCode)
		return
	}

	room.safe.RLock()
	defer room.safe.RUnlock()

	fmt.Printf("\nRoom: %s\n", roomCode)
	fmt.Printf("Number of clients: %d\n", len(room.clients))

	for client, active := range room.clients {
		remoteAddr := client.Conn.RemoteAddr()
		fmt.Printf("- Client %v (Active: %v)\n", remoteAddr, active)
	}
	fmt.Println()
}

func printAllRooms() {
	mu.RLock()
	defer mu.RUnlock()

	fmt.Printf("\nTotal active rooms: %d\n", len(allRooms))
	for code := range allRooms {
		fmt.Printf("- Room: %s\n", code)
	}
	fmt.Println()
}
