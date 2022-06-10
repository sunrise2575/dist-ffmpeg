package main

import (
	"log"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/sunrise2575/VP9-parallel/src/fsys"
)

const (
	FFMPEG_COMMON_INPUT_ARG  = " -hide_banner -loglevel warning -avoid_negative_ts 1 -analyzeduration 2147483647 -probesize 2147483647 -y "
	FFMPEG_COMMON_OUTPUT_ARG = " -max_muxing_queue_size 4096 "
)

func ffmpegEncodeAudioOnly(fp_in FilepathSplit, fp_out FilepathSplit, ffmpeg_param string, audio_stream_number int) {
	if !(audio_stream_number >= 0) {
		log.Panicf("[PANIC] it should be audio_stream_number >= 0")
	}

	arg := strings.Fields(FFMPEG_COMMON_INPUT_ARG + "-i")
	arg = append(arg, fp_in.Join())
	arg = append(arg, strings.Fields(
		FFMPEG_COMMON_OUTPUT_ARG+
			ffmpeg_param+
			" -map 0:a:"+strconv.Itoa(audio_stream_number))...)
	arg = append(arg, fp_out.Join())

	if out, e := exec.Command("ffmpeg", arg...).CombinedOutput(); e != nil {
		log.Println(string(out))
		log.Panicf("[PANIC] Failed to execute ffmpeg @ %v, error: %v", fp_in.Join(), e)
	} //else { log.Println(out) }
}

func ffmpegEncodeVideoOnly(fp_in FilepathSplit, fp_out FilepathSplit, ffmpeg_param string) {
	arg := strings.Fields(FFMPEG_COMMON_INPUT_ARG + "-threads 0 -i")
	arg = append(arg, fp_in.Join())
	arg = append(arg, strings.Fields(
		FFMPEG_COMMON_OUTPUT_ARG+
			ffmpeg_param+
			" -map 0:v:0")...)
	arg = append(arg, fp_out.Join())

	//log.Println(arg)

	if out, e := exec.Command("ffmpeg", arg...).CombinedOutput(); e != nil {
		log.Println(string(out))
		log.Panicf("[PANIC] Failed to execute ffmpeg @ %v, error: %v", fp_in.Join(), e)
	} //else { log.Println(out) }
}

func remove(s []FilepathSplit, i int) []FilepathSplit {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

func ffmpegSplitVideo(ctx *TranscodingContext, expected_file_count int) []FilepathSplit {
	unit_time := int(math.Max(20, math.Ceil(ctx.video_length/float64(expected_file_count))))
	expected_file_count = int(math.Ceil(ctx.video_length / float64(unit_time)))

	arg := strings.Fields(FFMPEG_COMMON_INPUT_ARG + "-i")
	arg = append(arg, ctx.fp.Join())
	arg = append(arg, strings.Fields(FFMPEG_COMMON_OUTPUT_ARG+
		"-f segment -segment_time "+strconv.Itoa(unit_time)+" -reset_timestamps 1 -c:v copy -an -map 0:v:0")...)

	temp := FilepathSplit{
		dir:  ctx.temp_dir,
		name: "." + ctx.fp.name + "_video_" + ctx.id + "_%d",
		ext:  ctx.fp.ext,
	}
	arg = append(arg, temp.Join())

	if _, e := exec.Command("ffmpeg", arg...).CombinedOutput(); e != nil {
		log.Panicf("[PANIC] Failed to execute ffmpeg, error: %v", e)
	} //else { log.Println(out) }

	result := []FilepathSplit{}
	for i := 0; i < expected_file_count; i++ {
		temp := FilepathSplit{
			dir:  ctx.temp_dir,
			name: "." + ctx.fp.name + "_video_" + ctx.id + "_" + strconv.Itoa(i),
			ext:  ctx.fp.ext,
		}
		result = append(result, temp)
	}

	{
		temp := []FilepathSplit{}
		copy(temp, result)
		for i, fp := range temp {
			if !fsys.IsFile(fp.Join()) {
				remove(result, i)
			}
		}
	}

	return result
}

func ffmpegConcatFiles(fps_in []FilepathSplit, fp_text, fp_out FilepathSplit) {
	if len(fps_in) == 0 {
		log.Panicf("[PANIC] Length of input file list is 0")
	}

	// write text file for ffmpeg concat function
	f_text, e := os.OpenFile(fp_text.Join(), os.O_CREATE|os.O_WRONLY, 0644)
	if e != nil {
		log.Panicf("[PANIC] Failed to create/open file %v, error: %v", f_text, e)
	}

	for _, fp := range fps_in {
		_, e := f_text.Write([]byte("file '" + fp.Join() + "'\n"))
		if e != nil {
			log.Panicf("[PANIC] Failed to create/open file %v, error: %v", f_text, e)
		}
	}
	f_text.Close()

	// ffmpeg concat
	arg := strings.Fields(FFMPEG_COMMON_INPUT_ARG + "-f concat -safe 0 -i")
	arg = append(arg, fp_text.Join())
	arg = append(arg, strings.Fields(FFMPEG_COMMON_OUTPUT_ARG+"-c:v copy")...)
	arg = append(arg, fp_out.Join())

	//log.Println(arg)

	if _, e := exec.Command("ffmpeg", arg...).CombinedOutput(); e != nil {
		log.Panicf("[PANIC] Failed to execute ffmpeg, error: %v", e)
	} //else { log.Println(out) }

	//os.RemoveAll(fp_text.Join())
}

func ffmpegMuxVideoAudio(fp_in_video, fp_in_audio, fp_out FilepathSplit) {
	arg := strings.Fields(FFMPEG_COMMON_INPUT_ARG)
	arg = append(arg, "-i", fp_in_video.Join())
	arg = append(arg, "-i", fp_in_audio.Join())
	arg = append(arg, strings.Fields(FFMPEG_COMMON_OUTPUT_ARG+"-c:v copy -c:a copy -map 0:v:0 -map 1:a:0")...)
	arg = append(arg, fp_out.Join())

	//log.Println(arg)

	if _, e := exec.Command("ffmpeg", arg...).CombinedOutput(); e != nil {
		log.Panicf("[PANIC] Failed to execute ffmpeg, error: %v", e)
	} //else { log.Println(out) }
}
