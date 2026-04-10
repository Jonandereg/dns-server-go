package main

import (
	"flag"
	"fmt"
	"net"
)

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:2053")
	if err != nil {
		fmt.Println("Failed to resolve UDP address:", err)
		return
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Failed to bind to address:", err)
		return
	}
	defer udpConn.Close()
	forwardAddress := flag.String("resolver", "", "forwarding server address")
	flag.Parse()
	buf := make([]byte, 512)

	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error receiving data:", err)
			break
		}

		dnsMessage := DNSMessage{
			rawRequest:     buf[:size],
			forwardAddress: *forwardAddress,
		}
		if err := dnsMessage.parseQuery(); err != nil {
			fmt.Println("Failed to parse DNS message:", err)
		}
		response, err := dnsMessage.writeResponse()
		if err != nil {
			fmt.Println("Error writing response:", err)
			break
		}

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
