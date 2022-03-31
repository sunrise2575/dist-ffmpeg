package ffprobe

import (
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/sunrise2575/VP9-parallel/src/fsys"
	"github.com/tidwall/gjson"
)

func StreamInfoJSON(fp_in string) []gjson.Result {
	fp_in = fsys.Sanitize(fp_in)
	arg := strings.Fields("-v quiet -print_format json -show_streams")
	arg = append(arg, fp_in)

	result, e := exec.Command("ffprobe", arg...).Output()
	if e != nil {
		log.Panicf("[PANIC] Failed to execute ffprobe, error: %v", e)
	}

	return gjson.Get(string(result), "streams").Array()
}

func VideoTime(fp_in string) float64 {
	fp_in = fsys.Sanitize(fp_in)
	arg := strings.Fields("-v error -select_streams v:0 -show_entries format=duration -of default=noprint_wrappers=1:nokey=1")
	arg = append(arg, fp_in)

	bresult, e := exec.Command("ffprobe", arg...).Output()
	if e != nil {
		log.Panicf("[PANIC] Failed to execute ffprobe, error: %v", e)
	}

	result := strings.TrimRight(string(bresult), "\r\n")
	time, e := strconv.ParseFloat(result, 64)
	if e != nil {
		log.Panicf("[PANIC] strconv.ParseFloat(%v, 64) failed, error: %v", result, e)
	}

	return time
}

func VideoFrame(fp_in string) int {
	fp_in = fsys.Sanitize(fp_in)
	arg := strings.Fields("-v error -select_streams v:0 -count_packets -show_entries stream=nb_read_packets -of csv=p=0")
	arg = append(arg, fp_in)

	bresult, e := exec.Command("ffprobe", arg...).Output()
	if e != nil {
		log.Panicf("[PANIC] Failed to execute ffprobe, error: %v", e)
	}

	result := strings.TrimRight(string(bresult), "\r\n")
	frames, e := strconv.Atoi(result)
	if e != nil {
		log.Panicf("[PANIC] strconv.Atoi(%v) failed, error: %v", result, e)
	}

	return frames
}
