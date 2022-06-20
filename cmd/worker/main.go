package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pebbe/zmq4"
	"github.com/sirupsen/logrus"
	"github.com/sunrise2575/VP9-parallel/pkg/transcode"
	"github.com/sunrise2575/VP9-parallel/pkg/util"
	"github.com/tidwall/gjson"
)

var (
	SERVER_IP, SERVER_PORT          string
	MY_HOSTNAME, MY_PID             string
	PATH_CONFIG, PATH_TEMP          string
	LOG_LEVEL, LOG_FILE, LOG_FORMAT string
)

func init() {
	MY_HOSTNAME, _ = os.Hostname()
	MY_PID = strconv.Itoa(os.Getpid())
	my_home, _ := os.UserHomeDir()

	// log options
	flag.StringVar(&LOG_LEVEL, "loglevel", "info", "panic, fatal, error, warning, info, debug, trace")
	flag.StringVar(&LOG_FILE, "logfile", "", "log file location")
	flag.StringVar(&LOG_FORMAT, "logformat", "text", "text, json")

	// transcoding options
	flag.StringVar(&PATH_CONFIG, "conf", "./config.json", "Config file")
	flag.StringVar(&PATH_TEMP, "temp", filepath.Join(my_home, ".temp/"), "Temporary directory for transcoding")

	// distributed processing options
	flag.StringVar(&SERVER_IP, "ip", "localhost", "master port")
	flag.StringVar(&SERVER_PORT, "port", "5000", "master port")

	flag.Parse()

	logrus.WithFields(logrus.Fields{"name": "hostname", "value": MY_HOSTNAME}).Debug("Process Info")
	logrus.WithFields(logrus.Fields{"name": "process_id", "value": MY_PID}).Debug("Process Info")

	logrus.WithFields(logrus.Fields{"name": "loglevel", "value": LOG_LEVEL}).Debug("Argument")
	logrus.WithFields(logrus.Fields{"name": "logfile", "value": LOG_FILE}).Debug("Argument")
	logrus.WithFields(logrus.Fields{"name": "logformat", "value": LOG_FORMAT}).Debug("Argument")
	logrus.WithFields(logrus.Fields{"name": "port", "value": SERVER_PORT}).Debug("Argument")
	logrus.WithFields(logrus.Fields{"name": "conf", "value": PATH_CONFIG}).Debug("Argument")
	logrus.WithFields(logrus.Fields{"name": "temp", "value": PATH_TEMP}).Debug("Argument")

	util.InitLogrus(LOG_FILE, LOG_LEVEL, LOG_FORMAT)

	PATH_CONFIG = util.PathSanitize(PATH_CONFIG)
	if !util.PathIsFile(PATH_CONFIG) {
		logrus.WithFields(logrus.Fields{"path": PATH_CONFIG}).Panicf("Unable to find the configure file")
	}

	PATH_TEMP = util.PathSanitize(PATH_TEMP)
	e := os.MkdirAll(PATH_TEMP, 0755)
	if e != nil {
		logrus.WithFields(logrus.Fields{"path": PATH_TEMP}).Panicf("Unable to create/open the temporary directory")
	}
}

func SendRecv(sock *zmq4.Socket, send_payload map[string]string) map[string]string {
	// Must Send
	send_payload["hostname"] = MY_HOSTNAME
	send_payload["pid"] = MY_PID
	sock.Send(util.Map2JSON(send_payload), 0)

	// Must Recv
	recv_json, _ := sock.Recv(0)
	return util.JSON2Map(recv_json)
}

func main() {
	ENDPOINT := "tcp://" + SERVER_IP + ":" + SERVER_PORT

	// Read config file
	conf, e := util.ReadJSONFile(PATH_CONFIG)
	if e != nil {
		logrus.WithFields(logrus.Fields{"path": PATH_CONFIG}).Panicf("Unable to parse the configure file")
	}

	ctx, e := zmq4.NewContext()
	if e != nil {
		logrus.WithFields(logrus.Fields{"error": e}).Panicf("Unable to create ZeroMQ context")
	}
	sock, e := ctx.NewSocket(zmq4.REQ)
	if e != nil {
		logrus.WithFields(logrus.Fields{"error": e}).Panicf("Unable to create ZeroMQ socket")
	}
	e = sock.Connect(ENDPOINT)
	if e != nil {
		logrus.WithFields(logrus.Fields{"error": e}).Panicf("Unable to connect ZeroMQ socket")
	}
	logrus.WithFields(logrus.Fields{"endpoint": ENDPOINT}).Debugf("Connect")

	current_fp := ""

	defer func() {
		if current_fp != "" {
			logrus.WithFields(logrus.Fields{"path": current_fp}).Warnf("Incomplete")
			SendRecv(sock, map[string]string{"req": "killed", "path": current_fp})
			logrus.WithFields(logrus.Fields{"path": current_fp}).Debugf("Report to master")
		}
	}()

	for {
		func() {
			defer func() {
				if p := recover(); p != nil {
					logrus.WithFields(logrus.Fields{"recover_msg": p}).Warnf("Recovered from panic")
				}
			}()

			// Query to master server
			recv := SendRecv(sock, map[string]string{"req": "job_want"})

			if recv["res"] == "false" {
				logrus.Warnf("No more avaialbe job. Bye.")
				os.Exit(0)
			}

			current_fp = recv["path"]
			logrus.WithFields(logrus.Fields{"path": current_fp}).Debugf("Received a job")

			start := time.Now()

			status, e := work(current_fp, conf, PATH_TEMP)

			elapsed := time.Since(start)

			// Report to master server
			switch status {
			case "success":
				logrus.WithFields(logrus.Fields{"path": current_fp}).Infof("Success")
				SendRecv(sock, map[string]string{
					"req":          "job_done",
					"path":         current_fp,
					"elapsed_time": util.Atof(elapsed.Seconds()),
				})
				logrus.WithFields(logrus.Fields{"path": current_fp}).Debugf("Report complete job")

			case "skip":
				logrus.WithFields(logrus.Fields{"path": current_fp}).Warnf("Skip")
				SendRecv(sock, map[string]string{
					"req":          "job_skip",
					"path":         current_fp,
					"elapsed_time": util.Atof(elapsed.Seconds()),
				})
				logrus.WithFields(logrus.Fields{"path": current_fp}).Debugf("Report skipped job")

			case "fail":
				//logrus.WithFields(logrus.Fields{"path": current_fp}).Warnf("Failed")
				logrus.WithFields(logrus.Fields{"path": current_fp}).Warnf("Failed")
				SendRecv(sock, map[string]string{
					"req":          "job_fail",
					"path":         current_fp,
					"elapsed_time": util.Atof(elapsed.Seconds()),
					"error":        e.Error(),
				})
				logrus.WithFields(logrus.Fields{"path": current_fp}).Debugf("Report failed job")
			}

			current_fp = ""
		}()
	}
}

type CtxKey string

func work(fp_in string, conf gjson.Result, temp_dir string) (string, error) {
	meta := transcode.Metadata{}
	meta.Init(fp_in, conf, temp_dir)
	if meta.FileType == "" {
		return "skip", nil
	}

	ctx := context.Background()

	var e error
	// transcode
	switch meta.FileType {
	case "image":
		fallthrough
	case "audio":
		fallthrough
	case "video":
		e = transcode.SingleStreamOnly(ctx, &meta)
	case "video_and_audio":
		e = transcode.VideoAndAudio(ctx, &meta)
	case "image_animated":
		fallthrough
	default:
		return "skip", nil
	}

	if e != nil {
		return "fail", e
	}

	return "success", nil
}
