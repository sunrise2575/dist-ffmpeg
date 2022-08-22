package main

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/pebbe/zmq4"
	"github.com/sirupsen/logrus"

	"github.com/sunrise2575/dist-ffmpeg/pkg/util"
)

var (
	SERVER_PORT, DIRECTORY          string
	MY_HOSTNAME, MY_PID             string
	LOG_LEVEL, LOG_FILE, LOG_FORMAT string
)

func init() {
	MY_HOSTNAME, _ = os.Hostname()
	MY_PID = strconv.Itoa(os.Getpid())

	// log options
	flag.StringVar(&LOG_LEVEL, "loglevel", "info", "panic, fatal, error, warn, info, debug, trace")
	flag.StringVar(&LOG_FILE, "logfile", "", "log file location")
	flag.StringVar(&LOG_FORMAT, "logformat", "text", "text, json")

	// distributed processing options
	flag.StringVar(&SERVER_PORT, "port", "5000", "master port")
	flag.StringVar(&DIRECTORY, "dir", ".", "File root directory")

	flag.Parse()

	logrus.WithFields(logrus.Fields{"name": "hostname", "value": MY_HOSTNAME}).Debug("Process Info")
	logrus.WithFields(logrus.Fields{"name": "process_id", "value": MY_PID}).Debug("Process Info")

	logrus.WithFields(logrus.Fields{"name": "loglevel", "value": LOG_LEVEL}).Debug("Argument")
	logrus.WithFields(logrus.Fields{"name": "logfile", "value": LOG_FILE}).Debug("Argument")
	logrus.WithFields(logrus.Fields{"name": "logformat", "value": LOG_FORMAT}).Debug("Argument")
	logrus.WithFields(logrus.Fields{"name": "port", "value": SERVER_PORT}).Debug("Argument")
	logrus.WithFields(logrus.Fields{"name": "dir", "value": DIRECTORY}).Debug("Argument")

	util.InitLogrus(LOG_FILE, LOG_LEVEL, LOG_FORMAT)

	DIRECTORY = util.PathSanitize(DIRECTORY)
	if !util.PathIsDir(DIRECTORY) {
		logrus.WithFields(logrus.Fields{"path": DIRECTORY}).Panicf("Unable to find the directory")
	}
}

func main() {
	// create zeromq socket
	ENDPOINT := "tcp://*:" + SERVER_PORT
	ctx, e := zmq4.NewContext()
	if e != nil {
		logrus.WithFields(logrus.Fields{"error": e}).Panicf("Unable to create ZeroMQ context")
	}
	sock, e := ctx.NewSocket(zmq4.REP)
	if e != nil {
		logrus.WithFields(logrus.Fields{"error": e}).Panicf("Unable to create ZeroMQ socket")
	}
	e = sock.Bind("tcp://*:" + SERVER_PORT)
	if e != nil {
		logrus.WithFields(logrus.Fields{"error": e}).Panicf("Unable to bind ZeroMQ socket")
	}
	logrus.WithFields(logrus.Fields{"endpoint": ENDPOINT}).Debugf("Bind")

	// iterate files and transcode
	{
		chan_fp := make(chan string, 16)

		ext_exclude := util.Slice2Map([]string{".7z", ".rar", ".zip", ".tar", ".lzh", ".bin", ".cue", ".md5", ".mds", ".mdf", ".log", ".txt", ".lrc", ".exe", ".md", ".py", ".sample", ".go", ".mod", ".sum", ".json", ".sh", ".gitignore"})
		ext_subtitle := util.Slice2Map([]string{".smi", ".srt", ".vtt", ".ass"})
		ext_video := util.Slice2Map([]string{".webm"})
		ext_audio := util.Slice2Map([]string{".ogg"})
		ext_image := util.Slice2Map([]string{".png"})

		// search files recursively
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()

			logrus.WithFields(logrus.Fields{"path": DIRECTORY}).
				Infof("Start to seek files recursively in the directory")

			filepath.Walk(DIRECTORY, func(fp_in string, f_info os.FileInfo, err error) error {
				if f_info.IsDir() {
					return nil
				}
				// file path sanity check
				_, name, ext := util.PathSplit(fp_in)
				if len(ext) < 2 {
					return nil
				}

				if len(name) > 0 && name[0] == '.' {
					return nil
				}

				ext = strings.ToLower(ext)

				if ext_exclude[ext] || ext_subtitle[ext] {
					return nil
				}

				if ext_video[ext] || ext_audio[ext] || ext_image[ext] {
					return nil
				}

				chan_fp <- fp_in
				return nil
			})

			logrus.WithFields(logrus.Fields{"path": DIRECTORY}).
				Infof("Complete to seek files recursively in the directory")
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
						send_payload["path"] = fp
						logrus.WithFields(logrus.Fields{
							"hostname": recv["hostname"],
							"pid":      recv["pid"],
							"path":     fp,
						}).Infof("Start")
					default:
						send_payload["res"] = "false"
						logrus.WithFields(logrus.Fields{
							"hostname": recv["hostname"],
							"pid":      recv["pid"],
						}).Warnf("Got job request, but no more job")
					}

				case "job_done":
					logrus.WithFields(logrus.Fields{
						"hostname":     recv["hostname"],
						"pid":          recv["pid"],
						"path":         recv["path"],
						"elapsed_time": recv["elapsed_time"],
					}).Infof("Complete")

				case "job_fail":
					logrus.WithFields(logrus.Fields{
						"hostname":     recv["hostname"],
						"pid":          recv["pid"],
						"path":         recv["path"],
						"elapsed_time": recv["elapsed_time"],
					}).Warnf("Failed")

				case "job_skip":
					logrus.WithFields(logrus.Fields{
						"hostname":     recv["hostname"],
						"pid":          recv["pid"],
						"path":         recv["path"],
						"elapsed_time": recv["elapsed_time"],
					}).Warnf("Skipped")

				case "killed":
					logrus.WithFields(logrus.Fields{
						"hostname":     recv["hostname"],
						"pid":          recv["pid"],
						"path":         recv["path"],
						"elapsed_time": recv["elapsed_time"],
					}).Warnf("Incomplete")

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
