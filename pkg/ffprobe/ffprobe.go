package ffprobe

import (
	"fmt"
	"os/exec"

	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/sunrise2575/VP9-parallel/pkg/util"

	"github.com/tidwall/gjson"
)

func StreamInfoJSON(fp_in string) ([]gjson.Result, error) {
	fp_in = util.PathSanitize(fp_in)
	arg := strings.Fields("-v quiet -print_format json -show_streams")
	arg = append(arg, fp_in)

	out, e := exec.Command("ffprobe", arg...).CombinedOutput()
	if e != nil {
		return nil, fmt.Errorf("error message: %v, ffprobe output: %v", e, string(out))
	}

	logrus.WithFields(
		logrus.Fields{
			"path_input":     fp_in,
			"subproc":        "ffprobe",
			"subproc_param":  arg,
			"subproc_output": string(out),
			"error":          e,
			"where":          util.GetCurrentFunctionInfo(),
		}).Tracef("Subprocess success")

	return gjson.Get(string(out), "streams").Array(), nil
}

func VideoTime(fp_in string) (float64, error) {
	fp_in = util.PathSanitize(fp_in)
	arg := strings.Fields("-v error -select_streams v:0 -show_entries format=duration -of default=noprint_wrappers=1:nokey=1")
	arg = append(arg, fp_in)

	out, e := exec.Command("ffprobe", arg...).CombinedOutput()
	if e != nil {
		return 0.0, fmt.Errorf("error message: %v, ffprobe output: %v", e, string(out))
	}

	logrus.WithFields(
		logrus.Fields{
			"path_input":     fp_in,
			"subproc":        "ffprobe",
			"subproc_param":  arg,
			"subproc_output": string(out),
			"error":          e,
			"where":          util.GetCurrentFunctionInfo(),
		}).Tracef("Subprocess success")

	result := strings.TrimRight(string(out), "\r\n")
	time, e := strconv.ParseFloat(result, 64)
	if e != nil {
		return 0.0, e
	}

	return time, nil
}

func VideoFrame(fp_in string) (int, error) {
	fp_in = util.PathSanitize(fp_in)
	arg := strings.Fields("-v error -select_streams v:0 -count_packets -show_entries stream=nb_read_packets -of csv=p=0")
	arg = append(arg, fp_in)

	out, e := exec.Command("ffprobe", arg...).CombinedOutput()
	if e != nil {
		return 0, fmt.Errorf("error message: %v, ffprobe output: %v", e, string(out))
	}

	logrus.WithFields(
		logrus.Fields{
			"path_input":     fp_in,
			"subproc":        "ffprobe",
			"subproc_param":  arg,
			"subproc_output": string(out),
			"error":          e,
			"where":          util.GetCurrentFunctionInfo(),
		}).Tracef("Subprocess success")

	result := strings.TrimRight(string(out), "\r\n")
	frames, e := strconv.Atoi(result)
	if e != nil {
		return 0, e
	}

	return frames, nil
}
