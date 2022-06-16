package ffprobe

import (
	"os/exec"

	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/sunrise2575/VP9-parallel/pkg/util"

	"github.com/tidwall/gjson"
)

func StreamInfoJSON(fp_in string) []gjson.Result {
	fp_in = util.PathSanitize(fp_in)
	arg := strings.Fields("-v quiet -print_format json -show_streams")
	arg = append(arg, fp_in)

	result, e := exec.Command("ffprobe", arg...).CombinedOutput()
	if e != nil {
		logrus.WithFields(
			logrus.Fields{
				"filepath_input": fp_in,
				"subproc":        "ffprobe",
				"subproc_param":  arg,
				"subproc_output": string(result),
				"error":          e,
				"where":          util.GetCurrentFunctionInfo(),
			}).Fatalf("Subprocess failed")
	}
	logrus.WithFields(
		logrus.Fields{
			"filepath_input": fp_in,
			"subproc":        "ffprobe",
			"subproc_param":  arg,
			"subproc_output": string(result),
			"error":          e,
			"where":          util.GetCurrentFunctionInfo(),
		}).Tracef("Subprocess success")

	return gjson.Get(string(result), "streams").Array()
}

func VideoTime(fp_in string) float64 {
	fp_in = util.PathSanitize(fp_in)
	arg := strings.Fields("-v error -select_streams v:0 -show_entries format=duration -of default=noprint_wrappers=1:nokey=1")
	arg = append(arg, fp_in)

	bresult, e := exec.Command("ffprobe", arg...).CombinedOutput()
	if e != nil {
		logrus.WithFields(
			logrus.Fields{
				"filepath_input": fp_in,
				"subproc":        "ffprobe",
				"subproc_param":  arg,
				"subproc_output": string(bresult),
				"error":          e,
				"where":          util.GetCurrentFunctionInfo(),
			}).Fatalf("Subprocess failed")
	}

	logrus.WithFields(
		logrus.Fields{
			"filepath_input": fp_in,
			"subproc":        "ffprobe",
			"subproc_param":  arg,
			"subproc_output": string(bresult),
			"error":          e,
			"where":          util.GetCurrentFunctionInfo(),
		}).Tracef("Subprocess success")

	result := strings.TrimRight(string(bresult), "\r\n")
	time, e := strconv.ParseFloat(result, 64)
	if e != nil {
		logrus.WithFields(
			logrus.Fields{
				"input": result,
				"error": e,
				"where": util.GetCurrentFunctionInfo(),
			}).Fatalf("strconv.ParseFloat() failed")
	}

	return time
}

func VideoFrame(fp_in string) int {
	fp_in = util.PathSanitize(fp_in)
	arg := strings.Fields("-v error -select_streams v:0 -count_packets -show_entries stream=nb_read_packets -of csv=p=0")
	arg = append(arg, fp_in)

	bresult, e := exec.Command("ffprobe", arg...).CombinedOutput()
	if e != nil {
		logrus.WithFields(
			logrus.Fields{
				"filepath_input": fp_in,
				"subproc":        "ffprobe",
				"subproc_param":  arg,
				"subproc_output": string(bresult),
				"error":          e,
				"where":          util.GetCurrentFunctionInfo(),
			}).Fatalf("Subprocess failed")
	}

	logrus.WithFields(
		logrus.Fields{
			"filepath_input": fp_in,
			"subproc":        "ffprobe",
			"subproc_param":  arg,
			"subproc_output": string(bresult),
			"error":          e,
			"where":          util.GetCurrentFunctionInfo(),
		}).Tracef("Subprocess success")

	result := strings.TrimRight(string(bresult), "\r\n")
	frames, e := strconv.Atoi(result)
	if e != nil {
		logrus.WithFields(
			logrus.Fields{
				"input": result,
				"error": e,
				"where": util.GetCurrentFunctionInfo(),
			}).Fatalf("strconv.Atoi() failed")
	}

	return frames
}
