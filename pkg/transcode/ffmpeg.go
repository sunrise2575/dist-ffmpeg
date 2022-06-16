package transcode

import (
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/sunrise2575/VP9-parallel/pkg/util"
)

const (
	FFMPEG_COMMON_INPUT_ARG  = " -hide_banner -loglevel warning -avoid_negative_ts 1 -analyzeduration 2147483647 -probesize 2147483647 -y "
	FFMPEG_COMMON_OUTPUT_ARG = " -max_muxing_queue_size 4096 "
)

func ffmpegEncodeAudioOnly(fp_in File, fp_out File, ffmpeg_param string, audio_stream_number int) {
	if !(audio_stream_number >= 0) {
		logrus.WithFields(
			logrus.Fields{
				"path_input": fp_in.Join(),
				"where":      util.GetCurrentFunctionInfo(),
			}).Fatalf("Should be audio_stream_number >= 0")
	}

	arg := strings.Fields(FFMPEG_COMMON_INPUT_ARG + "-i")
	arg = append(arg, fp_in.Join())
	arg = append(arg, strings.Fields(
		FFMPEG_COMMON_OUTPUT_ARG+
			ffmpeg_param+
			" -map 0:a:"+strconv.Itoa(audio_stream_number))...)
	arg = append(arg, fp_out.Join())

	out, e := exec.Command("ffmpeg", arg...).CombinedOutput()
	if e != nil {
		logrus.WithFields(
			logrus.Fields{
				"path_input":     fp_in.Join(),
				"subproc":        "ffmpeg",
				"subproc_param":  arg,
				"subproc_output": string(out),
				"error":          e,
				"where":          util.GetCurrentFunctionInfo(),
			}).Fatalf("Subprocess failed")
	}

	logrus.WithFields(
		logrus.Fields{
			"path_input":     fp_in.Join(),
			"subproc":        "ffmpeg",
			"subproc_param":  arg,
			"subproc_output": string(out),
			"where":          util.GetCurrentFunctionInfo(),
		}).Debugf("Subprocess success")
}

func ffmpegEncodeVideoOnly(fp_in File, fp_out File, ffmpeg_param string) {
	arg := strings.Fields(FFMPEG_COMMON_INPUT_ARG + "-threads 0 -i")
	arg = append(arg, fp_in.Join())
	arg = append(arg, strings.Fields(
		FFMPEG_COMMON_OUTPUT_ARG+
			ffmpeg_param+
			" -map 0:v:0")...)
	arg = append(arg, fp_out.Join())

	out, e := exec.Command("ffmpeg", arg...).CombinedOutput()
	if e != nil {
		logrus.WithFields(
			logrus.Fields{
				"path_input":     fp_in.Join(),
				"subproc":        "ffmpeg",
				"subproc_param":  arg,
				"subproc_output": string(out),
				"error":          e,
				"where":          util.GetCurrentFunctionInfo(),
			}).Fatalf("Subprocess failed")
	}

	logrus.WithFields(
		logrus.Fields{
			"path_input":     fp_in.Join(),
			"subproc":        "ffmpeg",
			"subproc_param":  arg,
			"subproc_output": string(out),
			"where":          util.GetCurrentFunctionInfo(),
		}).Debugf("Subprocess success")
}

func ffmpegSplitVideo(ctx *Context, expected_file_count int) []File {

	unit_time := int(math.Max(20, math.Ceil(ctx.VideoLength/float64(expected_file_count))))
	expected_file_count = int(math.Ceil(ctx.VideoLength / float64(unit_time)))

	arg := strings.Fields(FFMPEG_COMMON_INPUT_ARG + "-i")
	arg = append(arg, ctx.FilePath.Join())
	arg = append(arg, strings.Fields(FFMPEG_COMMON_OUTPUT_ARG+
		"-f segment -segment_time "+strconv.Itoa(unit_time)+" -reset_timestamps 1 -c:v copy -an -map 0:v:0")...)

	temp_f := File{
		Dir:  ctx.TempDir,
		Name: "." + ctx.FilePath.Name + "_video_" + ctx.ID + "_%d",
		Ext:  ctx.FilePath.Ext,
	}
	arg = append(arg, temp_f.Join())

	out, e := exec.Command("ffmpeg", arg...).CombinedOutput()
	if e != nil {
		logrus.WithFields(
			logrus.Fields{
				"path_input":     ctx.FilePath.Join(),
				"subproc":        "ffmpeg",
				"subproc_param":  arg,
				"subproc_output": string(out),
				"error":          e,
				"where":          util.GetCurrentFunctionInfo(),
			}).Fatalf("Subprocess failed")
	}

	logrus.WithFields(
		logrus.Fields{
			"path_input":     ctx.FilePath.Join(),
			"subproc":        "ffmpeg",
			"subproc_param":  arg,
			"subproc_output": string(out),
			"where":          util.GetCurrentFunctionInfo(),
		}).Debugf("Subprocess success")

	temp := []File{}
	result := []File{}

	// expected_file_count*2 is duct taping haha
	for i := 0; i < expected_file_count*2; i++ {
		temp = append(temp, File{
			Dir:  ctx.TempDir,
			Name: "." + ctx.FilePath.Name + "_video_" + ctx.ID + "_" + strconv.Itoa(i),
			Ext:  ctx.FilePath.Ext,
		})
	}

	for _, fp := range temp {
		if util.PathIsFile(fp.Join()) {
			result = append(result, fp)
		}
	}

	return result
}

func ffmpegConcatFiles(fps_in []File, fp_text, fp_out File) {
	if len(fps_in) == 0 {
		logrus.WithFields(
			logrus.Fields{
				"path_output": fp_out.Join(),
				"where":       util.GetCurrentFunctionInfo(),
			}).Fatalf("Length of input file list is 0")
	}

	// write text file for ffmpeg concat function
	f_text, e := os.OpenFile(fp_text.Join(), os.O_CREATE|os.O_WRONLY, 0644)
	if e != nil {
		logrus.WithFields(
			logrus.Fields{
				"path_output": fp_text.Join(),
				"error":       e,
				"where":       util.GetCurrentFunctionInfo(),
			}).Fatalf("Failed to create/open file")
	}
	defer f_text.Close()

	for _, fp := range fps_in {
		_, e := f_text.Write([]byte("file '" + fp.Join() + "'\n"))
		if e != nil {
			logrus.WithFields(
				logrus.Fields{
					"path_output": fp_text.Join(),
					"error":       e,
					"where":       util.GetCurrentFunctionInfo(),
				}).Fatalf("Failed to write text file")
		}
	}

	f_text.Sync()

	// ffmpeg concat
	arg := strings.Fields(FFMPEG_COMMON_INPUT_ARG + "-f concat -safe 0 -i")
	arg = append(arg, fp_text.Join())
	arg = append(arg, strings.Fields(FFMPEG_COMMON_OUTPUT_ARG+"-c:v copy")...)
	arg = append(arg, fp_out.Join())

	out, e := exec.Command("ffmpeg", arg...).CombinedOutput()
	if e != nil {
		logrus.WithFields(
			logrus.Fields{
				"path_output":    fp_out.Join(),
				"subproc":        "ffmpeg",
				"subproc_param":  arg,
				"subproc_output": string(out),
				"error":          e,
				"where":          util.GetCurrentFunctionInfo(),
			}).Fatalf("Subprocess failed")
	}

	logrus.WithFields(
		logrus.Fields{
			"path_output":    fp_out.Join(),
			"subproc":        "ffmpeg",
			"subproc_param":  arg,
			"subproc_output": string(out),
			"error":          e,
			"where":          util.GetCurrentFunctionInfo(),
		}).Debugf("Subprocess success")

	for _, fp := range fps_in {
		e := os.RemoveAll(fp.Join())
		if e != nil {
			logrus.WithFields(
				logrus.Fields{
					"path_target": fp.Join(),
					"error":       e,
					"where":       util.GetCurrentFunctionInfo(),
				}).Warnf("Fail to remove a file")
		}
	}
	e = os.RemoveAll(fp_text.Join())
	if e != nil {
		logrus.WithFields(
			logrus.Fields{
				"path_target": fp_text.Join(),
				"error":       e,
				"where":       util.GetCurrentFunctionInfo(),
			}).Warnf("Fail to remove a file")
	}
}

func ffmpegMuxVideoAudio(fp_in_video, fp_in_audio, fp_out File) {
	arg := strings.Fields(FFMPEG_COMMON_INPUT_ARG)
	arg = append(arg, "-i", fp_in_video.Join())
	arg = append(arg, "-i", fp_in_audio.Join())
	arg = append(arg, strings.Fields(FFMPEG_COMMON_OUTPUT_ARG+"-c:v copy -c:a copy -map 0:v:0 -map 1:a:0")...)
	arg = append(arg, fp_out.Join())

	out, e := exec.Command("ffmpeg", arg...).CombinedOutput()
	if e != nil {
		logrus.WithFields(
			logrus.Fields{
				"path_output":    fp_out.Join(),
				"subproc":        "ffmpeg",
				"subproc_param":  arg,
				"subproc_output": string(out),
				"error":          e,
				"where":          util.GetCurrentFunctionInfo(),
			}).Fatalf("Subprocess failed")
	}

	logrus.WithFields(
		logrus.Fields{
			"path_output":    fp_out.Join(),
			"subproc":        "ffmpeg",
			"subproc_param":  arg,
			"subproc_output": string(out),
			"error":          e,
			"where":          util.GetCurrentFunctionInfo(),
		}).Debugf("Subprocess success")
}
