package main

type DNSMessage struct {
	rawRequest []byte
	header     Header
	questions  []Question
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
