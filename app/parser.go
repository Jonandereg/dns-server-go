package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

func (dm *DNSMessage) parseQuery() error {
	if err := dm.parseHeader(); err != nil {
		return fmt.Errorf("failed to parse header: %v", err)
	}
	if err := dm.parseQuestions(); err != nil {
		return fmt.Errorf("failed to parse questions: %v", err)
	}
	return nil
}

func (dm *DNSMessage) parseHeader() error {
	r := bytes.NewReader(dm.rawRequest)
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

func (dm *DNSMessage) parseQuestions() error {
	n := dm.rawRequest
	offset := 12

	for i := uint16(0); i < dm.header.qCount; i++ {
		name, off, err := parseName(n, offset)
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

func parseName(n []byte, off int) (string, int, error) {

	var name string

	for {
		b := n[off]
		if b == 0x00 {
			name = strings.TrimSuffix(name, ".")
			off++
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

			parsedQuestion, _, err := parseName(n, p)
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
