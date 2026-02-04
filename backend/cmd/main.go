package main

//  now we need to capture the packets, from the websockets and then convert that into object and display that
// 1 websocket capture

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// we need to create a udp server to listen for incoming packets parse it and then print that
// to achive first goal

type TCP_Packet struct {
	service_origin      [32]byte
	service_destination [32]byte
	portOrigin          uint32
	portDestination     uint32
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
		fmt.Printf("Raw bytes in hex: %x\n", raw_bytes)
		fmt.Printf("data", raw_bytes)
		err = binary.Read(raw_bytes, binary.LittleEndian, packet)
		fmt.Printf("asdasdasd", packet)
		// fmt.Printf("Puertos: %d -> %d\n", packet.portOrigin, packet.portDestination)

		fmt.Printf("It is working")

		time.Sleep(3 * time.Second)
	}

}
