package main

import (
	"io"
	"log"
	"net"
	"sync"
)

type Server struct {
	Listen string

	agentConn      net.Conn
	clientConn     map[string]net.Conn
	clientConnLock sync.RWMutex
}

func (s *Server) Run() {
	s.clientConn = make(map[string]net.Conn)

	serverConn, err := net.Listen("tcp", s.Listen)
	if err != nil {
		log.Fatalln("server listen err:", err)
	}

	for {
		client, err := serverConn.Accept()
		if err != nil {
			log.Println("server accept err:", err)
			continue
		}

		go s.handleClient(client)
	}
}

func (s *Server) handleClient(conn net.Conn) {
	if s.agentConn == nil { // wait for agent
		agentAddr := conn.RemoteAddr().String()
		for {
			pkt, err := ReadPacket(conn)

			if err == io.EOF {
				continue
			}

			if err != nil {
				log.Println("read agent conn err:", err, "agent addr:", agentAddr)
				continue
			}

			switch pkt.Cmd {
			case ExchangeCmdAgentRegister:
				log.Println("receive agent register, target addr:", string(pkt.Data), "agent addr:", conn.RemoteAddr())
				s.agentConn = conn
			case ExchangeCmdAgentHeartbeat:
				log.Println("receive heartbeat content:", string(pkt.Data), "agent:", agentAddr)
			case ExchangeCmdClientSendData:
				clientAddr, data := parseClientData(pkt.Data)

				s.clientConnLock.RLock()
				s.clientConn[clientAddr].Write(data)
				s.clientConnLock.RUnlock()

				log.Println("send data from agent to client, len:", len(data), "client:", clientAddr)
			}
		}

	} else {
		clientAddr := conn.RemoteAddr().String()

		log.Println("receive client connect, client addr:", clientAddr)
		s.clientConnLock.Lock()
		s.clientConn[clientAddr] = conn
		s.clientConnLock.Unlock()

		WritePacket(s.agentConn, &ExchangePacket{
			Cmd:  ExchangeCmdServerConnect,
			Data: []byte(clientAddr),
		})

		buf := make([]byte, 512)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				log.Println("read client conn err:", err, ".closing client:")
				s.clientConn[clientAddr].Close()

				s.clientConnLock.Lock()
				delete(s.clientConn, clientAddr)
				s.clientConnLock.Unlock()

				WritePacket(s.agentConn, &ExchangePacket{
					Cmd:  ExchangeCmdServerDisconnect,
					Data: []byte(clientAddr),
				})
				break
			}

			log.Println("send data from client to agent, len:", n, "client:", clientAddr)
			WritePacket(s.agentConn, &ExchangePacket{
				Cmd:  ExchangeCmdServerSendData,
				Data: buildClientData(clientAddr, buf[:n]),
			})
		}
	}
}
