// rhExchange project main.go
package main

import (
	"flag"
	"log"
)

var (
	Listen     = flag.String("l", ":17325", "server listen address (server only)")
	serverAddr = flag.String("s", "", "server address (client only)")
	targetAddr = flag.String("t", "", "target address (client only)")
)

func init() {
	flag.Parse()
}

func main() {
	if *targetAddr != "" && *serverAddr != "" {
		log.Println("run as agent, target:", *targetAddr)

		var agent Agent
		agent.TargetAddr = *targetAddr
		agent.ServerAddr = *serverAddr
		agent.Run()
	} else {
		log.Println("run as server, listen:", *Listen)

		var sever Server
		sever.Listen = *Listen
		sever.Run()
	}

}
