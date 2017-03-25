package main

import (
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type Agent struct {
	ServerAddr string
	TargetAddr string

	serverConn net.Conn
	targetConn map[string]net.Conn //map[clientAddr] => targetConn

	targetConnLock sync.RWMutex

	isBusy bool
}

func (a *Agent) Register() error {
	if a.serverConn != nil {
		a.serverConn.Close()
	}

	var err error
	a.serverConn, err = net.Dial("tcp", a.ServerAddr)
	if err != nil {
		return err
	}

	log.Println("register to server:", a.ServerAddr, "target:", a.TargetAddr)
	return WritePacket(a.serverConn, &ExchangePacket{
		Cmd:  ExchangeCmdAgentRegister,
		Data: []byte(a.TargetAddr),
	})
}

func (a *Agent) Heartbeat() {
	for {
		time.Sleep(10 * time.Second)

		if a.isBusy {
			a.isBusy = false
			continue
		}
		if a.serverConn != nil {
			log.Println("send heartbeat to server")
			WritePacket(a.serverConn, &ExchangePacket{
				Cmd:  ExchangeCmdAgentHeartbeat,
				Data: []byte(time.Now().String()),
			})
		}
	}
}

//agent state machine
func (a *Agent) Run() {
	a.targetConn = make(map[string]net.Conn)

	err := a.Register()
	if err != nil {
		log.Println("agent register err:", err)
	}

	go a.Heartbeat()

	for {
		pkt, err := ReadPacket(a.serverConn)
		if err == io.EOF {
			continue
		}

		a.isBusy = true

		if err != nil {
			log.Println("read pkt err:", err)
			continue
		}

		switch pkt.Cmd {
		case ExchangeCmdServerConnect:
			clientAddr := string(pkt.Data)

			a.targetConnLock.Lock()
			a.targetConn[clientAddr], err = net.Dial("tcp", a.TargetAddr)
			a.targetConnLock.Unlock()

			if err != nil {
				log.Println("make target conn err:", err, "client:", clientAddr)
			} else {
				log.Println("make target conn:", a.TargetAddr, "client:", clientAddr)
				go a.pipeTargetToServer(clientAddr)
			}

		case ExchangeCmdServerSendData: // client -> server -> agent -> target
			clientAddr, data := parseClientData(pkt.Data)
			log.Println("receive data from server to target, len:", len(data), "client:", clientAddr)

			a.targetConnLock.RLock()
			clientConn := a.targetConn[clientAddr]
			a.targetConnLock.RUnlock()

			if clientConn != nil {
				clientConn.Write(data)
			} else {
				log.Println("client conn not found!", clientAddr)
			}

		case ExchangeCmdServerDisconnect:
			clientAddr := string(pkt.Data)
			log.Println("receive server command to disconnect target, client:", clientAddr)

			a.targetConnLock.Lock()
			a.targetConn[clientAddr].Close()
			delete(a.targetConn, clientAddr)
			a.targetConnLock.Unlock()
		}
	}

}

// target -> agent -> server
func (a *Agent) pipeTargetToServer(clientAddr string) {
	a.targetConnLock.RLock()
	clientConn := a.targetConn[clientAddr]
	a.targetConnLock.RUnlock()

	buf := make([]byte, 512)
	for {
		n, err := clientConn.Read(buf)
		if err != nil {
			log.Println("read client conn for", clientAddr, "err:", err, "closing...")
			break
		}

		log.Println("send data from target to server, len:", n)
		WritePacket(a.serverConn, &ExchangePacket{
			Cmd:  ExchangeCmdClientSendData,
			Data: buildClientData(clientAddr, buf[:n]),
		})
	}
}
