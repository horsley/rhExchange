package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
)

const PROTOCOL_DEBUG = false

const (
	ExchangeCmdAgentRegister    = iota //data = target addr
	ExchangeCmdAgentHeartbeat          // data = meanless
	ExchangeCmdServerConnect           // data = clientAddr
	ExchangeCmdServerSendData          // data = [clientAddrLen] | clientAddr | data
	ExchangeCmdClientSendData          // data = [clientAddrLen] | clientAddr | data
	ExchangeCmdServerDisconnect        // data = clientAddr
)

type ExchangePacket struct {
	Cmd              uint32
	PacketFullLength uint32
	Data             []byte
}

func ReadPacket(reader io.Reader) (*ExchangePacket, error) {
	var pkt ExchangePacket

	header := make([]byte, 8)
	_, err := io.ReadFull(reader, header)
	if err != nil {
		return nil, err
	}

	pkt.Cmd = binary.BigEndian.Uint32(header[:4])
	pkt.PacketFullLength = binary.BigEndian.Uint32(header[4:])
	pkt.Data = make([]byte, pkt.PacketFullLength-8)

	if PROTOCOL_DEBUG {
		log.Println("cmd:", pkt.Cmd, "len:", pkt.PacketFullLength, header)
	}

	_, err = io.ReadFull(reader, pkt.Data)

	return &pkt, err
}

func WritePacket(writer io.Writer, pkt *ExchangePacket) error {
	pkt.PacketFullLength = uint32(len(pkt.Data) + 8)

	header := make([]byte, 8)
	binary.BigEndian.PutUint32(header, pkt.Cmd)
	binary.BigEndian.PutUint32(header[4:], pkt.PacketFullLength)
	_, err := writer.Write(header)
	if err != nil {
		return err
	}
	_, err = writer.Write(pkt.Data)
	return err
}

func buildClientData(clientAddr string, data []byte) []byte {
	var buf bytes.Buffer

	clientAddrLen := make([]byte, 4)

	binary.BigEndian.PutUint32(clientAddrLen, uint32(len(clientAddr)))
	buf.Write(clientAddrLen)
	buf.WriteString(clientAddr)
	buf.Write(data)

	ret := buf.Bytes()
	if PROTOCOL_DEBUG {
		log.Println("build client data", ret[:20])
	}
	return ret
}

func parseClientData(data []byte) (string, []byte) {
	clientAddrLen := int(binary.BigEndian.Uint32(data))
	if PROTOCOL_DEBUG {
		log.Println("parse client data, client addr len:", clientAddrLen, data[:20])
	}
	clientAddr := string(data[4 : 4+clientAddrLen])
	return clientAddr, data[4+clientAddrLen:]
}
