package main

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	zmq "github.com/pebbe/zmq4"
)

const (
	STOP_MESSAGE = "!stop"
)

func main() {
	zmq_ctx, _ := zmq.NewContext()

	// scatter job
	sock_push, _ := zmq_ctx.NewSocket(zmq.PUSH)
	sock_push.Bind("tcp://*:5000")

	// gather job
	sock_pull, _ := zmq_ctx.NewSocket(zmq.PULL)
	sock_pull.Bind("tcp://*:6000")

	var wg sync.WaitGroup

	dp_in := "/fifi/.media"

	// job scatter
	wg.Add(1)
	go func() {
		defer wg.Done()
		filepath.Walk(dp_in, func(fp_in string, f_info os.FileInfo, err error) error {
			if f_info.IsDir() {
				return nil
			}
			sock_push.Send(fp_in, 0)
			log.Println("[VENT] send message:", fp_in)
			time.Sleep(time.Millisecond * 10) // for fair balancing
			return nil
		})
		sock_push.Send("stop", 0)
		sock_push.Close()
	}()

	// job gather
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			msg, _ := sock_pull.Recv(0)
			if msg == "stop" {
				break
			}
			log.Println("[SINK] received:", msg)
		}
	}()

	wg.Wait()
}
