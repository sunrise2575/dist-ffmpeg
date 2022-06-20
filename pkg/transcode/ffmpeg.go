package transcode

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/sunrise2575/dist-ffmpeg/pkg/ffprobe"
	"github.com/sunrise2575/dist-ffmpeg/pkg/util"
)

const (
	FFMPEG_COMMON_INPUT_ARG  = " -hide_banner -loglevel warning -avoid_negative_ts 1 -analyzeduration 2147483647 -probesize 2147483647 -y "
	FFMPEG_COMMON_OUTPUT_ARG = " -max_muxing_queue_size 4096 "
)

func ffmpegEncodeAudioOnly(ctx context.Context, fp_in File, fp_out File, ffmpeg_param string, audio_stream_number int) error {
	if !(audio_stream_number >= 0) {
		return fmt.Errorf("should be audio_stream_number >= 0")
	}

	arg := strings.Fields(FFMPEG_COMMON_INPUT_ARG + "-i")
	arg = append(arg, fp_in.Join())
	arg = append(arg, strings.Fields(
		FFMPEG_COMMON_OUTPUT_ARG+
			ffmpeg_param+
			" -map 0:a:"+strconv.Itoa(audio_stream_number))...)
	arg = append(arg, fp_out.Join())

	out, e := exec.CommandContext(ctx, "ffmpeg", arg...).CombinedOutput()
	if e != nil {
		return fmt.Errorf("error message: %v, ffmpeg output: %v", e, string(out))
	}

	logrus.WithFields(
		logrus.Fields{
			"path_input":     fp_in.Join(),
			"subproc":        "ffmpeg",
			"subproc_param":  arg,
			"subproc_output": string(out),
			"where":          util.GetCurrentFunctionInfo(),
		}).Debugf("Subprocess success")

	return nil
}

func ffmpegEncodeVideoOnly(ctx context.Context, fp_in File, fp_out File, ffmpeg_param string, video_stream_number int) error {
	arg := strings.Fields(FFMPEG_COMMON_INPUT_ARG + "-threads 0 -i")
	arg = append(arg, fp_in.Join())
	arg = append(arg, strings.Fields(
		FFMPEG_COMMON_OUTPUT_ARG+
			ffmpeg_param+
			" -map 0:v:"+strconv.Itoa(video_stream_number))...)
	arg = append(arg, fp_out.Join())

	out, e := exec.CommandContext(ctx, "ffmpeg", arg...).CombinedOutput()
	if e != nil {
		return fmt.Errorf("error message: %v, ffmpeg output: %v", e, string(out))
	}

	logrus.WithFields(
		logrus.Fields{
			"path_input":     fp_in.Join(),
			"subproc":        "ffmpeg",
			"subproc_param":  arg,
			"subproc_output": string(out),
			"where":          util.GetCurrentFunctionInfo(),
		}).Debugf("Subprocess success")

	return nil
}

func ffmpegSplitVideo(ctx context.Context, fp_in File, dp_out string, splited_filename_rule File, video_stream_number int, expected_file_count int) ([]File, error) {
	video_length, e := ffprobe.VideoTime(fp_in.Join())
	if e != nil {
		return nil, e
	}

	unit_time := int(math.Max(16, math.Ceil(video_length/float64(expected_file_count))))
	expected_file_count = int(math.Ceil(video_length / float64(unit_time)))

	arg := strings.Fields(FFMPEG_COMMON_INPUT_ARG + "-i")
	arg = append(arg, fp_in.Join())
	arg = append(arg, strings.Fields(FFMPEG_COMMON_OUTPUT_ARG+
		"-f segment -segment_time "+strconv.Itoa(unit_time)+
		" -reset_timestamps 1 -c:v copy -an -map 0:v:"+strconv.Itoa(video_stream_number))...)
	arg = append(arg, splited_filename_rule.Join())

	out, e := exec.CommandContext(ctx, "ffmpeg", arg...).CombinedOutput()
	if e != nil {
		return nil, fmt.Errorf("error message: %v, ffmpeg output: %v", e, string(out))
	}

	logrus.WithFields(
		logrus.Fields{
			"path_input":     fp_in.Join(),
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
			Dir:  dp_out,
			Name: fmt.Sprintf(splited_filename_rule.Name, i),
			Ext:  fp_in.Ext,
		})
		logrus.Debugln(temp[len(temp)-1].Join())
	}

	for _, fp := range temp {
		if util.PathIsFile(fp.Join()) {
			result = append(result, fp)
		}
	}

	return result, nil
}

func ffmpegConcatFiles(ctx context.Context, fps_in []File, fp_text, fp_out File) error {
	if len(fps_in) == 0 {
		return fmt.Errorf("Length of input file list is 0")
	}

	// write text file for ffmpeg concat function
	f_text, e := os.OpenFile(fp_text.Join(), os.O_CREATE|os.O_WRONLY, 0644)
	if e != nil {
		return fmt.Errorf("Failed to create/open file")
	}
	defer f_text.Close()

	for _, fp := range fps_in {
		_, e := f_text.Write([]byte("file '" + fp.Join() + "'\n"))
		if e != nil {
			return fmt.Errorf("Failed to write text file")
		}
	}

	f_text.Sync()

	// ffmpeg concat
	arg := strings.Fields(FFMPEG_COMMON_INPUT_ARG + "-f concat -safe 0 -i")
	arg = append(arg, fp_text.Join())
	arg = append(arg, strings.Fields(FFMPEG_COMMON_OUTPUT_ARG+"-c:v copy")...)
	arg = append(arg, fp_out.Join())

	out, e := exec.CommandContext(ctx, "ffmpeg", arg...).CombinedOutput()
	if e != nil {
		return fmt.Errorf("error message: %v, ffmpeg output: %v", e, string(out))
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
			return fmt.Errorf("Fail to remove a file: %v", fp.Join())
		}
	}

	e = os.RemoveAll(fp_text.Join())
	if e != nil {
		return fmt.Errorf("Fail to remove a file: %v", fp_text.Join())
	}

	return nil
}

func ffmpegMuxVideoAudio(ctx context.Context, fp_in_video, fp_in_audio, fp_out File) error {
	arg := strings.Fields(FFMPEG_COMMON_INPUT_ARG)
	arg = append(arg, "-i", fp_in_video.Join())
	arg = append(arg, "-i", fp_in_audio.Join())
	arg = append(arg, strings.Fields(FFMPEG_COMMON_OUTPUT_ARG+"-c:v copy -c:a copy -map 0:v:0 -map 1:a:0")...)
	arg = append(arg, fp_out.Join())

	out, e := exec.CommandContext(ctx, "ffmpeg", arg...).CombinedOutput()
	if e != nil {
		return fmt.Errorf("error message: %v, ffmpeg output: %v", e, string(out))
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

	return nil
}
