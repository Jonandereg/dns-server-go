package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
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

	buf := make([]byte, 512)

	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error receiving data:", err)
			break
		}

		receivedData := string(buf[:size])
		fmt.Printf("Received %d bytes from %s: %s\n", size, source, receivedData)

		dnsMessage := DNSMessage{}
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

type DNSMessage struct {
	header   []byte
	question []byte
	answer   []byte
	Flags
}
type Flags struct {
	qCount uint16
}

func (dm *DNSMessage) writeResponse() ([]byte, error) {
	if err := dm.writeQuestion(); err != nil {
		return nil, err
	}
	dm.writeHeader()
	res := make([]byte, 0, 32)
	res = append(res, dm.header...)
	res = append(res, dm.question...)
	return res, nil
}

func (dm *DNSMessage) writeHeader() {
	header := make([]byte, 12)
	id := uint16(1234)
	binary.BigEndian.PutUint16(header[0:2], id)
	flags := buildFlags()
	binary.BigEndian.PutUint16(header[2:4], flags)
	binary.BigEndian.PutUint16(header[4:6], dm.qCount)
	anCount := uint16(0)
	binary.BigEndian.PutUint16(header[6:8], anCount)
	nsCount := uint16(0)
	binary.BigEndian.PutUint16(header[8:10], nsCount)
	arCount := uint16(0)
	binary.BigEndian.PutUint16(header[10:12], arCount)
	dm.header = append(dm.header, header...)
}

func buildFlags() uint16 {
	flags := uint16(0)
	qr := uint16(1)
	flags |= qr << 15
	opcode := uint16(0)
	flags |= opcode << 11
	aa := uint16(0)
	flags |= aa << 10
	truncation := uint16(0)
	flags |= truncation << 9
	recursionDesired := uint16(0)
	flags |= recursionDesired << 8
	recursionAvailable := uint16(0)
	flags |= recursionAvailable << 7
	// 3 bits reserved
	rCode := uint16(0)
	flags |= rCode

	return flags
}

func (dm *DNSMessage) writeQuestion() error {
	buf := bytes.NewBuffer([]byte{})

	qname, err := buildQuestionName("codecrafter.io")
	if err != nil {
		return fmt.Errorf("failed to build question name: %v", err)
	}
	buf.Write(qname)

	recordType := uint16(1)
	if err := binary.Write(buf, binary.BigEndian, recordType); err != nil {
		return fmt.Errorf("failed to write recordType: %v", err)
	}

	recordClass := uint16(1)
	if err := binary.Write(buf, binary.BigEndian, recordClass); err != nil {
		return fmt.Errorf("failed to write recordClass: %v", err)
	}
	dm.qCount = uint16(1)
	dm.question = buf.Bytes()
	return nil
}

func buildQuestionName(d string) ([]byte, error) {
	b := bytes.NewBuffer([]byte{})
	labelsArray := strings.Split(d, ".")
	for _, label := range labelsArray {
		l := len(label)
		if l > 63 {
			return nil, fmt.Errorf("label too long")
		}
		b.WriteByte(byte(l))
		b.WriteString(label)
	}
	b.WriteByte(0)
	return b.Bytes(), nil
}
