# DNS Server in Go

A fully functional DNS server built from scratch in Go as part of the [CodeCrafters "Build Your Own DNS Server" challenge](https://app.codecrafters.io/courses/dns-server/overview).

## What it does

- Parses and constructs DNS packets at the byte level (no DNS libraries)
- Handles DNS header and question section parsing, including label compression (pointer resolution)
- Responds to multiple questions in a single query
- Forwards queries to an upstream resolver, splitting multi-question packets into individual requests and merging the responses

## What I learned

- **DNS wire format**: how headers, flags, questions, and answer records are encoded as raw bytes with big-endian ordering
- **Label compression**: recursive pointer resolution where domain names reference earlier parts of the packet to save space
- **Bit manipulation in Go**: packing and unpacking flag fields from a uint16 using shifts and masks
- **UDP networking in Go**: using `net.ListenUDP` and `net.Dial("udp", ...)` for connectionless communication
- **Incremental protocol parsing**: moving from a `bytes.Reader` approach to raw byte slices with offset tracking, which was necessary to support compression pointers that jump to arbitrary positions

## Project structure

```
app/
  main.go       - UDP server loop and flag parsing
  models.go     - Data structures (DNSMessage, Header, Flags, Question)
  parser.go     - Parsing logic: header, questions, name decompression
  writer.go     - Serialization: header, flags, questions, answer records
  forwarder.go  - Query forwarding with request splitting and response merging
```

## What I'd improve next

- Add bounds checking on packet parsing to handle malformed input gracefully
- Tighten up error handling in the forwarding path
- Add read timeouts on upstream resolver connections
- Refactor `writeHeader`/`writeFlags` to cleanly separate query construction from response construction
- Parse answer records into a structured model instead of passing raw bytes

## Running

```sh
# Standard mode (responds with hardcoded answers)
./your_program.sh

# Forwarding mode (proxies to an upstream resolver)
./your_program.sh --resolver <ip>:<port>
```