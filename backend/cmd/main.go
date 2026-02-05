package main

//  now we need to capture the packets, from the websockets and then convert that into object and display that
// 1 websocket capture

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

// we need to create a udp server to listen for incoming packets parse it and then print that
// to achive first goal

type TCP_Packet struct {
	// there are not array of characters just bytes supported in golang
	Service_origin      [32]byte
	Service_destination [32]byte
	PortOrigin          uint32
	PortDestination     uint32
}

type TCP_Packet2 struct {
	Service_origin      string
	Service_destination string
	PortOrigin          uint32
	PortDestination     uint32
}

func main() {
	addr, err := net.ResolveUDPAddr("udp", ":8083")
	if err != nil {
		panic(err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	for {
		fmt.Printf("first step")
		// create a buffer to store the bytes
		buffer_of_bytes := make([]byte, 72)
		// read the bytes
		n, _, err := conn.ReadFromUDP(buffer_of_bytes)
		if err != nil {
			fmt.Println("Error from the reading the socket", err)
			continue
		}
		var packet TCP_Packet
		raw_bytes := bytes.NewReader(buffer_of_bytes[:n])
		// fmt.Printf("this is the buffer of bytes", buffer_of_bytes)
		// fmt.Printf("Raw bytes in hex: %x\n", raw_bytes)
		// fmt.Printf("data", raw_bytes)
		// read raw bytes
		err = binary.Read(raw_bytes, binary.LittleEndian, &packet)
		var packet2show TCP_Packet2
		destination := string(packet.Service_destination[:])
		origin := string(packet.Service_origin[:])
		packet2show.Service_origin = strings.ReplaceAll(origin, "\u0000", "")
		packet2show.Service_destination = strings.ReplaceAll(destination, "\u0000", "")
		packet2show.PortDestination = packet.PortDestination
		packet2show.PortOrigin = packet.PortOrigin

		ans, err := json.Marshal(packet2show)
		if err != nil {
			break
		}
		fmt.Println(string(ans))

		// fmt.Print(packet2show)

		// fmt.Println(origin, destination)
		// fmt.Println(packet.Service_destination.decode('utf-8'))
		// jsonData, err := json.Marshal(packet)
		// if err != nil {
		// 	fmt.Printf("this is an error", err)
		// }
		// jsonString := string(jsonData)
		// fmt.Printf(jsonString)
		// fmt.Printf("Puertos: %d -> %d\n", packet.portOrigin, packet.portDestination)

		fmt.Printf("It is working")

		time.Sleep(3 * time.Second)
	}

}
