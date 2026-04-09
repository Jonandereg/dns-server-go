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

		dnsMessage := DNSMessage{
			rawRequest: buf[:size],
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

type DNSMessage struct {
	rawRequest []byte
	header     Header
	questions  []Question
	reader     *bytes.Reader
}
type Header struct {
	ID      uint16
	qCount  uint16
	anCount uint16
	Flags
}
type Flags struct {
	qr         uint16
	opCode     uint16
	rd         uint16
	aa         uint16
	truncation uint16
	rCode      uint16
}
type Question struct {
	Name  string
	Type  uint16
	Class uint16
}

func (dm *DNSMessage) writeResponse() ([]byte, error) {
	q, err := dm.writeQuestions()
	if err != nil {
		return nil, fmt.Errorf("failed to write questions: %v", err)
	}

	a, err := dm.writeAnswer()
	if err != nil {
		return nil, fmt.Errorf("failed to write answer: %v", err)
	}

	h := dm.writeHeader()
	res := make([]byte, 0, 32)
	res = append(res, h...)
	res = append(res, q...)
	res = append(res, a...)
	return res, nil
}

func (dm *DNSMessage) writeHeader() []byte {
	header := make([]byte, 12)

	binary.BigEndian.PutUint16(header[0:2], dm.header.ID)

	flags := writeFlags(dm.header.Flags)

	binary.BigEndian.PutUint16(header[2:4], flags)
	binary.BigEndian.PutUint16(header[4:6], dm.header.qCount)
	binary.BigEndian.PutUint16(header[6:8], dm.header.anCount)
	nsCount := uint16(0)
	binary.BigEndian.PutUint16(header[8:10], nsCount)
	arCount := uint16(0)
	binary.BigEndian.PutUint16(header[10:12], arCount)
	return header
}

func writeFlags(f Flags) uint16 {
	flags := uint16(0)
	qr := uint16(1)
	flags |= qr << 15
	opcode := f.opCode
	flags |= opcode << 11
	aa := f.aa
	flags |= aa << 10
	truncation := f.truncation
	flags |= truncation << 9
	recursionDesired := f.rd
	flags |= recursionDesired << 8
	recursionAvailable := uint16(0)
	flags |= recursionAvailable << 7
	// 3 bits reserved
	rCode := uint16(0)
	if opcode != 0 {
		rCode = uint16(4)
	}

	flags |= rCode

	return flags
}

func (dm *DNSMessage) writeQuestions() ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	for i := 0; i < int(dm.header.qCount); i++ {
		qname, err := writeQuestionName(dm.questions[i].Name)
		if err != nil {
			return nil, fmt.Errorf("failed to build questionMsg name: %v", err)
		}
		buf.Write(qname)

		recordType := dm.questions[i].Type
		if err := binary.Write(buf, binary.BigEndian, recordType); err != nil {
			return nil, fmt.Errorf("failed to write recordType: %v", err)
		}

		recordClass := dm.questions[i].Class
		if err := binary.Write(buf, binary.BigEndian, recordClass); err != nil {
			return nil, fmt.Errorf("failed to write recordClass: %v", err)
		}

	}

	return buf.Bytes(), nil
}

func writeQuestionName(d string) ([]byte, error) {
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

func (dm *DNSMessage) writeAnswer() ([]byte, error) {
	testIp := net.ParseIP("127.0.0.1").To4()
	buf := bytes.NewBuffer([]byte{})
	for i := 0; i < int(dm.header.qCount); i++ {
		qname, err := writeQuestionName(dm.questions[i].Name)
		if err != nil {
			return nil, fmt.Errorf("failed to build questionMsg name: %v", err)
		}
		buf.Write(qname)

		recordType := dm.questions[i].Type
		if err := binary.Write(buf, binary.BigEndian, recordType); err != nil {
			return nil, fmt.Errorf("failed to write recordType: %v", err)
		}

		recordClass := dm.questions[i].Class
		if err := binary.Write(buf, binary.BigEndian, recordClass); err != nil {
			return nil, fmt.Errorf("failed to write recordClass: %v", err)
		}
		ttl := uint32(60)
		if err := binary.Write(buf, binary.BigEndian, ttl); err != nil {
			return nil, fmt.Errorf("failed to write ttl: %v", err)
		}
		length := uint16(len(testIp))
		if err := binary.Write(buf, binary.BigEndian, length); err != nil {
			return nil, fmt.Errorf("failed to write length: %v", err)
		}
		buf.Write(testIp)
	}

	dm.header.anCount = dm.header.qCount

	return buf.Bytes(), nil
}

func (dm *DNSMessage) parseQuery() error {
	dm.reader = bytes.NewReader(dm.rawRequest)
	if err := dm.parseHeader(); err != nil {
		return fmt.Errorf("failed to parse header: %v", err)
	}
	if err := dm.parseQuestions(dm.rawRequest); err != nil {
		return fmt.Errorf("failed to parse questions: %v", err)
	}
	return nil
}

func (dm *DNSMessage) parseHeader() error {
	r := dm.reader
	rawHeader := make([]byte, 12)
	n, err := r.Read(rawHeader)
	if err != nil {
		return fmt.Errorf("failed to read header: %v", err)
	}
	if n != 12 {
		return fmt.Errorf("bad header")
	}

	id := binary.BigEndian.Uint16(rawHeader[0:2])
	flags := binary.BigEndian.Uint16(rawHeader[2:4])
	qCount := binary.BigEndian.Uint16(rawHeader[4:6])
	f := parseFlags(flags)
	dm.header.ID = id
	dm.header.Flags = f
	dm.header.qCount = qCount
	return nil
}

func parseFlags(f uint16) Flags {
	qr := (f >> 15) & 1
	opcode := (f >> 11) & 0b1111
	aa := (f >> 10) & 1
	truncation := (f >> 9) & 1
	rd := (f >> 8) & 1

	return Flags{
		qr:         qr,
		opCode:     opcode,
		rd:         rd,
		aa:         aa,
		truncation: truncation,
	}
}

func (dm *DNSMessage) parseQuestions(n []byte) error {
	offset := 12

	for i := uint16(0); i < dm.header.qCount; i++ {
		name, off, err := parseQuestionName(n, offset)
		if err != nil {
			return fmt.Errorf("failed to parse question name: %v", err)
		}
		offset = off
		qType := binary.BigEndian.Uint16(n[offset : offset+2])
		offset += 2
		qClass := binary.BigEndian.Uint16(n[offset : offset+2])
		offset += 2

		dm.questions = append(dm.questions, Question{
			Name:  name,
			Class: qClass,
			Type:  qType,
		})
	}

	return nil
}

func parseQuestionName(n []byte, off int) (string, int, error) {

	var name string

	for {
		b := n[off]
		if b == 0x00 {
			name = strings.TrimSuffix(name, ".")
			break
		}
		if b&0xC0 == 0xC0 {
			// take byte 1, strip the top 2 flag bits
			highBits := int(b & 0x3F)

			// Shift left 8 bits to make space for the other byte
			highBits <<= 8
			off++

			// Combine highBits with second pointer byte
			p := highBits | int(n[off])
			off++

			parsedQuestion, _, err := parseQuestionName(n, p)
			if err != nil {
				return "", off, fmt.Errorf("failed to parse questions name: %v", err)
			}
			name += parsedQuestion
			break
		}

		off++
		questionBytes := n[off : off+int(b)]
		off += int(b)
		name += string(questionBytes) + "."
	}

	return name, off, nil
}
