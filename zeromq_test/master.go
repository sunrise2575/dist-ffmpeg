package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"

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

func RecvSend(sock *zmq.Socket, response func(map[string]string) map[string]string) {
}

func main() {
	MY_HOSTNAME, _ = os.Hostname()
	MY_PID = strconv.Itoa(os.Getpid())

	SERVER_PORT, DIRECTORY := "", ""
	flag.StringVar(&SERVER_PORT, "port", "5000", "master port")
	flag.StringVar(&DIRECTORY, "dir", "/fifi/.media/", "File root directory")

	ctx, _ := zmq.NewContext()
	sock, _ := ctx.NewSocket(zmq.REP)
	sock.Bind("tcp://*:" + SERVER_PORT)

	chan_fp := make(chan string, 16)

	// search files recursively
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
		}()

		files := 0

		filepath.Walk(DIRECTORY, func(fp_in string, f_info os.FileInfo, err error) error {
			if f_info.IsDir() {
				return nil
			}
			if files < 10 {
				chan_fp <- fp_in
				files++
			}
			return nil
		})
	}()

	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
		}()
		for {
			// Must Recv
			recv_json, _ := sock.Recv(0)
			recv := JSON2Map(recv_json)

			// Prepare response
			send_payload := map[string]string{}
			send_payload["hostname"] = MY_HOSTNAME
			send_payload["pid"] = MY_PID

			switch recv["req"] {
			case "job_want":
				select {
				case fp := <-chan_fp:
					send_payload["res"] = "true"
					send_payload["file_path"] = fp
					log.Println("GIVE_JOB", fp)
				default:
					send_payload["res"] = "false"
					log.Println("NO_JOB")
				}

			case "job_done":
				log.Println("COMPLETE", recv["file_path"])

			case "killed":
				// retry file
				log.Println("GOT RETRY FILE")
				wg.Add(1)
				go func() {
					defer wg.Done()
					chan_fp <- recv["file_path_incomplete"]
				}()

			default:
				// ignore
				//send_payload["res"] = "wrong_req"
			}

			// Must Send
			sock.Send(Map2JSON(send_payload), 0)
		}
	}()

	wg.Wait()
}
