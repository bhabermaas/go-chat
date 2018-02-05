// This is a simple chat server.
package imserver

import (
	"net"
	"log"
	"encoding/json"
	"fmt"
	"bufio"
	"io"
)

// Input packet structure from client(s)
type Packet struct {
	Action string
	Userid string
	Data string
}

// Output channel structure for a message
// Used when sending a message to the broadcaster
type Message struct {
	Userid string
	Data string
	Clinst  Instance
}

type client chan<- Message

// Instance structure for a chat client
type Instance struct {
	Userid string
	Channel client
	Connect net.Conn
	RW *bufio.ReadWriter
}

// Common channels and client list
var (
	broadcast = make(chan Message, 10)
	entering = make(chan Instance, 10)
	leaving  = make(chan Instance, 10)
	clients  = make(map[string]Instance)
)

//
// Start the server and accept connections. At each accepted connection a goroutine is
// started to handle input from its client.
//
func StartServer() {

	log.Print("Server started")
	listener, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}

	// Start broadcaster
	go broadcaster()

	// Loop around listening and accepting cloient connections.
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleConn(conn)
	}
}

//
// Handle a client connection. Receive JSON packets, handle actions, dispatch
// messages to broadcaster
//
func handleConn(conn net.Conn) {

	var userid string = "unknown"

	rw :=  bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	instance := Instance{}
	instance.Connect = conn
	instance.RW = rw
	ch := make(chan Message, 10)
	instance.Channel = ch

	for {
		response, err := rw.ReadString('\n')
		if err != nil  {
			log.Print(err)
			if _, err := rw.Peek(1); err == io.EOF {
				log.Printf("User %s has unexpectedly disconnected %s", userid, err)
				conn.Close()
				return
			}
			continue
		}

		packet := Packet{}
		err = json.Unmarshal([]byte(response), &packet)
		if err != nil {
			log.Printf("Unable to unmarshal package for %s, err=%s\n", userid, err)
			log.Fatal("Server is stopping\n")
		}

		msg := Message{}
		msg.Clinst = instance

		if packet.Action == "LOGIN" {

			userid = packet.Userid
			instance.Userid = userid

			msg.Userid = userid

			// Start client writer goroutine
			go clientWriter(rw, ch, userid)

			// Check if userid already registered
			if name, ok := clients[userid]; ok  {
				msg.Data = fmt.Sprintf("userid %s is already logged in", name)
				broadcast <- msg
				break
			}

			entering <-instance
			msg.Data = fmt.Sprintf("Entered chat (%s)", conn.RemoteAddr().String())
			broadcast <-msg
			continue
		}

		msg.Userid = userid
		if packet.Action == "QUIT" {
			msg.Data = fmt.Sprintf("Left chat")
			broadcast <- msg
			leaving <- instance
			return
		}
		msg.Data = packet.Data
		broadcast <- msg
	}
	conn.Close()
	close(instance.Channel)
}

//
// goroutine to broadcast messages to all chat clients. It also monitors entering and
// leaving channels to maintain the client list
//
func broadcaster() {
	log.Print("broadcaster running...")
	for {
		select {
		case msg := <-broadcast:

			// broadcast to all clients
			for _, instance := range clients {
				instance.Channel <- msg
			}

		case instance := <-entering:
			clients[instance.Userid]=instance

		case instance := <-leaving:
			delete(clients, instance.Userid)
			close(instance.Channel)
		}
	}
}

//
// goroutine to write a message to a specific client. There is one routine per client.
// This takes messages from the channel and writes them to the client.
//
func clientWriter(rw *bufio.ReadWriter, ch <- chan Message, userid string )  {
	log.Printf("clientWriter running.(%s) ...", userid)

	for msg := range ch {
		//msgText := fmt.Sprintf("%s -> %s", msg.Userid, msg.Data)
		//toMsgText := fmt.Sprintf("To %s -- %s", userid, msgText)
		//log.Print(toMsgText)

	 	if userid == msg.Userid {
			continue
		}
		packet := Packet{}
		packet.Userid = msg.Userid
		packet.Data = msg.Data
		packet.Action = "MSG"
		writePacketToClient(rw, packet)
	 }
	 log.Printf("clientWriter leaving (%s)", userid)
}

//
// Write a message packet to chat client. The message is converted into JSON and
// then sent to the client.
//
func writePacketToClient(rw *bufio.ReadWriter, packet Packet)  {

	stream, err := json.Marshal(packet)
	if ( err != nil ) {
		log.Print("writePacketToClient marshal failed ", err)
		return
	}
	s := fmt.Sprintf("%s\n", stream)
	_, err = rw.WriteString(s)
	rw.Flush()
	if err != nil {
		log.Fatal("writePacketToClient write failed ", err)
	}
}