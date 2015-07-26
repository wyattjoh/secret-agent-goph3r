package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Game struct {
	Rooms     map[string]*Room
	RoomsLock sync.Mutex
	Port      int
}

func (g *Game) Play() {
	log.Printf("Now listening for connections on port %d\n", g.Port)

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", g.Port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %s\n", g.Port, err.Error())
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("Failed to accept the connection: %s\n", err.Error())
			continue
		}

		log.Printf("Accepted connection from %s\n", conn.RemoteAddr().String())

		go newClient(g, conn)
	}
}

// OpenRoom opens a created room or creates a new room based on the name and
// passes in the pointer to the room to the provided function
func (g *Game) OpenRoom(name string, f func(room *Room)) {
	// lock the rooms map
	g.RoomsLock.Lock()

	// try and lookup the room
	room, ok := g.Rooms[name]
	if !ok {
		// create the new room
		room = NewRoom(name)

		// ensure that the room is saved on the rooms
		g.Rooms[name] = room
	}

	// do stuff with room
	f(room)

	// unlock the rooms map
	g.RoomsLock.Unlock()
}

type Client struct {
	Conn     net.Conn
	Nickname string
	Ch       chan Message
}

func NewRoom(name string) *Room {
	return &Room{
		Name:    name,
		Clients: make([]*Client, 0),
	}
}

type Room struct {
	Name    string
	Clients []*Client
}

type Message struct {
	From Client
	Text string
}

func main() {
	portString := os.Getenv("PORT")

	if portString == "" {
		portString = "6000"
	}

	portNumber, err := strconv.Atoi(portString)
	if err != nil {
		log.Fatalf("An error occured parsing %s to integer: %s\n", portString, err.Error())
	}

	game := Game{
		Rooms: make(map[string]*Room),
		Port:  portNumber,
	}

	// start the game
	game.Play()
}

func prompt(reader *bufio.Reader, writer *bufio.Writer, question string) string {
	text := fmt.Sprintf("%s: ", question)

	if _, err := writer.WriteString(text); err != nil {
		log.Printf("An error occured writing: %s\n", err.Error())
	}

	if err := writer.Flush(); err != nil {
		log.Printf("An error occured flushing: %s\n", err.Error())
	}

	ans, _, err := reader.ReadLine()
	if err != nil {
		log.Printf("An error occured reading: %s\n", err.Error())
	}

	return string(ans)
}

func newClient(g *Game, c net.Conn) {
	reader := bufio.NewReader(c)
	writer := bufio.NewWriter(c)

	var roomName, clientName string

	for {
		roomName = prompt(reader, writer, "Room")

		roomName = strings.TrimSpace(roomName)

		if roomName == "" {
			if _, err := writer.WriteString("Invalid room name\n"); err != nil {
				log.Printf("An error occured writing: %s\n", err.Error())
			}

			if err := writer.Flush(); err != nil {
				log.Printf("An error occured flushing: %s\n", err.Error())
			}

		} else {

			// good room name
			break

		}
	}

	clientName = prompt(reader, writer, "Nickname")

	client := &Client{
		Nickname: clientName,
		Conn:     c,
		Ch:       make(chan Message),
	}

	g.OpenRoom(roomName, func(room *Room) {

		room.Clients = append(room.Clients, client)

		log.Printf("Added Client(%s)[%p] to Room(%s)[%p]\n", client.Nickname, client, room.Name, room)
	})

	log.Printf("A new client has joined room %s\n", roomName)

}
