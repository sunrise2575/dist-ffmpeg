package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	zmq "github.com/pebbe/zmq4"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func Map2JSON(input map[string]string) string {
	output := ""
	for k, v := range input {
		output, _ = sjson.Set(output, k, v)
	}
	return output
}

// only flat json works
func JSON2Map(input string) map[string]string {
	output := map[string]string{}
	for k, v := range gjson.Parse(input).Map() {
		output[k] = v.String()
	}
	return output
}

var (
	MY_HOSTNAME, MY_PID = "", ""
)

func SendRecv(sock *zmq.Socket, send_payload map[string]string) map[string]string {
	// Must Send
	send_payload["hostname"] = MY_HOSTNAME
	send_payload["pid"] = MY_PID
	sock.Send(Map2JSON(send_payload), 0)

	// Must Recv
	recv_json, _ := sock.Recv(0)
	return JSON2Map(recv_json)
}

func main() {
	MY_HOSTNAME, _ = os.Hostname()
	MY_PID = strconv.Itoa(os.Getpid())

	SERVER_IP, SERVER_PORT := "", ""
	flag.StringVar(&SERVER_IP, "ip", "172.0.0.12", "master ip")
	flag.StringVar(&SERVER_PORT, "port", "5000", "master port")

	ctx, _ := zmq.NewContext()
	sock, _ := ctx.NewSocket(zmq.REQ)
	sock.Connect("tcp://" + SERVER_IP + ":" + SERVER_PORT)

	current_fp := ""

	defer func() {
		if current_fp != "" {
			log.Println("INCOMPLETE JOB REMAIN!")
			SendRecv(sock, map[string]string{"req": "killed", "file_path_incomplete": current_fp})
			log.Println("JOB REPORETED TO MASTER!")
		}
	}()

	for {
		// Query to master server
		recv := SendRecv(sock, map[string]string{"req": "job_want"})

		if recv["res"] == "false" {
			log.Println("NO MORE JOB")
			return
		}

		current_fp = recv["file_path"]
		log.Printf("GET A JOB: %v", current_fp)

		time.Sleep(time.Millisecond * 1000)

		// Report to master server
		SendRecv(sock, map[string]string{"req": "job_done", "file_path": current_fp})
		current_fp = ""
	}
}
