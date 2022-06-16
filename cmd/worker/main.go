package main

import (
	"flag"
	"os"
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

	logrus.Debugf("Hostname=%v, PID=%v", MY_HOSTNAME, MY_PID)

	// log options
	flag.StringVar(&LOG_LEVEL, "loglevel", "info", "panic, fatal, error, warn, info, debug, trace")
	flag.StringVar(&LOG_FILE, "logfile", "./worker.log", "log file location")
	flag.StringVar(&LOG_FORMAT, "logformat", "text", "text, json")

	// transcoding options
	flag.StringVar(&PATH_CONFIG, "conf", "./config.json", "Config file")
	flag.StringVar(&PATH_TEMP, "temp", "./.temp/", "Temporary directory for transcoding")

	// distributed processing options
	flag.StringVar(&SERVER_IP, "ip", "localhost", "master port")
	flag.StringVar(&SERVER_PORT, "port", "5000", "master port")

	flag.Parse()

	logrus.Debugf("Argument loglevel=%v", LOG_LEVEL)
	logrus.Debugf("Argument logfile=%v", LOG_FILE)
	logrus.Debugf("Argument logformat=%v", LOG_FORMAT)
	logrus.Debugf("Argument conf=%v", PATH_CONFIG)
	logrus.Debugf("Argument temp=%v", PATH_TEMP)
	logrus.Debugf("Argument port=%v", SERVER_PORT)

	util.InitLogrus(LOG_FILE, LOG_LEVEL, LOG_FORMAT)

	PATH_CONFIG = util.PathSanitize(PATH_CONFIG)
	if !util.PathIsFile(PATH_CONFIG) {
		logrus.Fatalf("Unable to find the configure file: %v (%v)", PATH_CONFIG)
	}

	PATH_TEMP = util.PathSanitize(PATH_TEMP)
	e := os.MkdirAll(PATH_TEMP, 0755)
	if e != nil {
		logrus.Fatalf("Unable to create/open the temporary directory: %v (%v)", PATH_TEMP, e)
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
		logrus.Fatalf("Unable to parse the configure file: %v (%v)", PATH_CONFIG, e.Error())
	}

	ctx, e := zmq4.NewContext()
	if e != nil {
		logrus.Panicf("Unable to create ZeroMQ context (%v)", e)
	}
	sock, e := ctx.NewSocket(zmq4.REQ)
	if e != nil {
		logrus.Panicf("Unable to create ZeroMQ socket (%v)", e)
	}
	e = sock.Connect(ENDPOINT)
	if e != nil {
		logrus.Panicf("Unable to connect ZeroMQ socket (%v)", e)
	}
	logrus.Debugf("Connect %v", ENDPOINT)

	current_fp := ""

	defer func() {
		if current_fp != "" {
			logrus.Warnf("Incomplete: %v", current_fp)
			SendRecv(sock, map[string]string{"req": "killed", "filepath_input": current_fp})
			logrus.Debugf("Report the incomplete job: %v", current_fp)
		}
	}()

	for {
		// Query to master server
		recv := SendRecv(sock, map[string]string{"req": "job_want"})

		if recv["res"] == "false" {
			logrus.Warnf("No more avaialbe job. Bye.")
			return
		}

		current_fp = recv["file_path"]
		logrus.Debugf("Received: %v", current_fp)

		start := time.Now()
		fp_out, status := work(current_fp, conf, PATH_TEMP)
		elapsed := time.Since(start)

		// Report to master server
		switch status {
		case "success":
			logrus.Infof("Success: %v", current_fp)
			SendRecv(sock, map[string]string{
				"req":             "job_done",
				"filepath_input":  current_fp,
				"filepath_output": fp_out,
				"elapsed_time":    util.Atof(elapsed.Seconds()),
			})
			logrus.Debugf("Report the complete job: %v", current_fp)
		case "skip":
			logrus.Warnf("Skip: %v", current_fp)
			SendRecv(sock, map[string]string{
				"req":             "job_skip",
				"filepath_input":  current_fp,
				"filepath_output": fp_out,
				"elapsed_time":    util.Atof(elapsed.Seconds()),
			})
			logrus.Debugf("Report the skipped job: %v", current_fp)
		case "fail":
			logrus.Warnf("Failed: %v", current_fp)
			SendRecv(sock, map[string]string{
				"req":             "job_fail",
				"filepath_input":  current_fp,
				"filepath_output": fp_out,
				"elapsed_time":    util.Atof(elapsed.Seconds()),
			})
			logrus.Debugf("Report the incomplete job: %v", current_fp)
		}

		current_fp = ""
	}
}

func work(fp_in string, conf gjson.Result, temp_dir string) (string, string) {
	fp_out := transcode.File{}

	ctx := transcode.Context{}
	ctx.Init(fp_in, conf, temp_dir)
	if ctx.FileType == "" {
		return fp_out.Join(), "skip"
	}

	e := true
	// transcode
	switch ctx.FileType {
	case "image":
		fp_out, e = transcode.ImageOnly(&ctx)
	case "audio":
		fp_out, e = transcode.AudioOnly(&ctx)
	case "video":
		fp_out, e = transcode.VideoAndAudio(&ctx)
	case "video_only":
		fp_out, e = transcode.VideoOnly(&ctx)
	case "image_animated":
		fallthrough
	default:
		return fp_out.Join(), "skip"
	}

	if !e {
		return fp_out.Join(), "fail"
	}

	return fp_out.Join(), "success"
}
