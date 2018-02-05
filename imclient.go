// This program is a simple chat client.
package main

import (
	"os"
	"fmt"
	"log"
	"bufio"
	"net"
	"encoding/json"
	"messenger/imserver"
	"io"
)

var (
	userid string
)

//
// Connect to the chat server and login with username specified as first command argument.
//
func main() {

	if len(os.Args) != 2  {
		log.Fatal("Userid should be first argument")
	}
	userid := os.Args[1]
	// Create a channel for sending Packet structures.
	inputChannel := make(chan imserver.Packet)
	connect, err := net.Dial("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Chat client has started for %s", userid)

	rw := bufio.NewReadWriter(bufio.NewReader(connect), bufio.NewWriter(connect))

	// Make a longin packet
	packet := imserver.Packet{}
	packet.Action = "LOGIN"
	packet.Userid = userid

	writePacketToServer(rw, packet)

	// asynchronous receive messages
	go receiveHandler(rw)

	// Get message input from a goroutine provided through a channel.
	go inputHandler(userid, inputChannel)

	// Get input lines and send to server as a message packet.
	for {
		packet = <-inputChannel
		writePacketToServer(rw, packet)
		if packet.Action == "QUIT" {
			break;
		}
	}
}

// Read the input lines and for each pass back a packet with
// the proper action, userid, and text
//
func inputHandler(userid string, inputChannel chan imserver.Packet) {
	log.Print("Input chat messages, Enter q to quit")
	packet := imserver.Packet{}
	packet.Userid = userid
	scanner := bufio.NewScanner(os.Stdin)
	var text string
	for text != "q" {  // break the loop if text == "q"
		scanner.Scan()
		text = scanner.Text()
		if text != "q" {
			packet.Action = "MSG"
			packet.Data = text
			inputChannel <- packet
		}
	}
	packet.Action = "QUIT"
	inputChannel <- packet
}

//
// Write a packet to chat server. The packet is converted into JSON and
// then sent to  the server.
//
func writePacketToServer(rw *bufio.ReadWriter, packet imserver.Packet)  {

	stream, err := json.Marshal(packet)
    if ( err != nil ) {
    	log.Fatal("writePacketToServer marshal failed ", err)
	}
	s := fmt.Sprintf("%s\n", stream)
	_, err = rw.WriteString(s)
	rw.Flush()
	if err != nil {
		log.Fatal("writePacketToServer write failed ", err)
	}
}

//
// Receive raw text messages from the server and echo to the console
//
func receiveHandler(rw *bufio.ReadWriter) {

	for {
		response, err := rw.ReadString('\n')
		if err != nil  {
			if _, err := rw.Peek(1); err == io.EOF {
				log.Fatal("Chat server has unexpectedly disconnected")
			}
			log.Print(err)
			continue
		}

		packet := imserver.Packet{}
		err = json.Unmarshal([]byte(response), &packet)
		if err != nil {
			log.Printf("Unable to unmarshal package, err=%s", err)
			log.Fatal("Client is terminating\n")
		}
		s := packet.Userid + " -> " + packet.Data
		log.Print(s)
	}
}
