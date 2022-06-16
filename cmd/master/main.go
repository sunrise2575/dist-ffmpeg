package main

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/pebbe/zmq4"
	"github.com/sirupsen/logrus"

	"github.com/sunrise2575/VP9-parallel/pkg/util"
)

var (
	SERVER_PORT, DIRECTORY          string
	MY_HOSTNAME, MY_PID             string
	LOG_LEVEL, LOG_FILE, LOG_FORMAT string
)

func init() {
	MY_HOSTNAME, _ = os.Hostname()
	MY_PID = strconv.Itoa(os.Getpid())

	logrus.Debugf("Hostname=%v, PID=%v", MY_HOSTNAME, MY_PID)

	// log options
	flag.StringVar(&LOG_LEVEL, "loglevel", "info", "panic, fatal, error, warn, info, debug, trace")
	flag.StringVar(&LOG_FILE, "logfile", "./master.log", "log file location")
	flag.StringVar(&LOG_FORMAT, "logformat", "text", "text, json")

	// distributed processing options
	flag.StringVar(&SERVER_PORT, "port", "5000", "master port")
	flag.StringVar(&DIRECTORY, "dir", "/fifi/.media/", "File root directory")

	flag.Parse()

	logrus.Debugf("Argument loglevel=%v", LOG_LEVEL)
	logrus.Debugf("Argument logfile=%v", LOG_FILE)
	logrus.Debugf("Argument logformat=%v", LOG_FORMAT)
	logrus.Debugf("Argument port=%v", SERVER_PORT)
	logrus.Debugf("Argument dir=%v", DIRECTORY)

	util.InitLogrus(LOG_FILE, LOG_LEVEL, LOG_FORMAT)

	DIRECTORY = util.PathSanitize(DIRECTORY)
	if !util.PathIsDir(DIRECTORY) {
		logrus.Fatalf("Unable to find the directory: %v (%v)", DIRECTORY)
	}
}

func main() {
	// create zeromq socket
	ENDPOINT := "tcp://*:" + SERVER_PORT
	ctx, e := zmq4.NewContext()
	if e != nil {
		logrus.Fatalf("Unable to create ZeroMQ context (%v)", e)
	}
	sock, e := ctx.NewSocket(zmq4.REP)
	if e != nil {
		logrus.Fatalf("Unable to create ZeroMQ socket (%v)", e)
	}
	e = sock.Bind("tcp://*:" + SERVER_PORT)
	if e != nil {
		logrus.Fatalf("Unable to bind ZeroMQ socket (%v)", e)
	}
	logrus.Debugf("Bind %v", ENDPOINT)

	// iterate files and transcode
	{
		chan_fp := make(chan string, 16)

		// search files recursively
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()

			logrus.Infof("Start to seek files recursively in the directory: %v", DIRECTORY)

			filepath.Walk(DIRECTORY, func(fp_in string, f_info os.FileInfo, err error) error {
				if f_info.IsDir() {
					return nil
				}
				// file path sanity check
				if len(filepath.Ext(fp_in)) < 2 {
					return nil
				}

				_, name, _ := util.PathSplit(fp_in)
				if len(name) > 0 && name[0] == '.' {
					return nil
				}
				chan_fp <- fp_in
				return nil
			})

			logrus.Infof("Complete to seek files recursively in the directory: %v", DIRECTORY)
		}()

		// request handling server
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				// Must Recv
				recv_json, _ := sock.Recv(0)
				recv := util.JSON2Map(recv_json)

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
						logrus.Infof("%v(%v) will process: %v",
							recv["hostname"], recv["pid"], fp)
					default:
						send_payload["res"] = "false"
						logrus.Warnf("%v(%v) requested, but no more job",
							recv["hostname"], recv["pid"])
					}

				case "job_done":
					logrus.Infof("%v(%v) completed (elapsed time: %v) %v",
						recv["hostname"], recv["pid"], recv["elapsed_time"], recv["file_path"])

				case "job_fail":
					logrus.Warnf("%v(%v) failed (elapsed time: %v): %v",
						recv["hostname"], recv["pid"], recv["elapsed_time"], recv["file_path"])

				case "killed":
					logrus.Warnf("%v(%v) aborted. Incomplete job: %v",
						recv["hostname"], recv["pid"], recv["file_path"])

				default:
					// ignore
					//send_payload["res"] = "wrong_req"
				}

				// Must Send
				sock.Send(util.Map2JSON(send_payload), 0)
			}
		}()
		wg.Wait()
	}
}
