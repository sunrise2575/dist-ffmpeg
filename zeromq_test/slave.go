package main

import (
	"log"
	"time"

	zmq "github.com/pebbe/zmq4"
)

func main() {
	// worker
	zmq_ctx, _ := zmq.NewContext()

	sock_pull, _ := zmq_ctx.NewSocket(zmq.PULL)
	sock_push, _ := zmq_ctx.NewSocket(zmq.PUSH)
	sock_pull.Connect("tcp://localhost:5000")
	sock_push.Connect("tcp://localhost:6000")

	for {
		msg, _ := sock_pull.Recv(0)
		if msg == "stop" {
			sock_push.Send(msg, 0)
			continue
		}
		log.Println("[WORKER] processing message:", msg)
		time.Sleep(time.Millisecond * 1000)
		sock_push.Send(msg+" complete", 0)
	}
}
