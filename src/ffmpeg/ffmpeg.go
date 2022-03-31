package ffmpeg

import (
	"log"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/sunrise2575/VP9-parallel/src/ffprobe"
	"github.com/sunrise2575/VP9-parallel/src/fsys"
)

const (
	FFMPEG_COMMON_INPUT_ARG  = " -hide_banner -loglevel warning -avoid_negative_ts 1 -analyzeduration 2147483647 -probesize 2147483647 -y "
	FFMPEG_COMMON_OUTPUT_ARG = " -max_muxing_queue_size 4096 "
)

func remove(s []string, i int) []string {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

func SplitVideo(fp_in string, expected_file_count int) []string {
	fp_in = fsys.Sanitize(fp_in)
	dir, name, ext := fsys.Split(fp_in)

	time := ffprobe.VideoTime(fp_in)
	unit_time := int(math.Max(20, math.Ceil(time/float64(expected_file_count))))
	expected_file_count = int(math.Ceil(time / float64(unit_time)))

	arg := strings.Fields(FFMPEG_COMMON_INPUT_ARG + "-i")
	arg = append(arg, fp_in)
	arg = append(arg, strings.Fields(FFMPEG_COMMON_OUTPUT_ARG)...)
	arg = append(arg, strings.Fields("-f segment -segment_time "+strconv.Itoa(unit_time)+" -reset_timestamps 1 -c:v copy -an -map 0:v:0")...)
	arg = append(arg, fsys.Join(dir, "."+name+"_video_%d", ext))

	//log.Println(arg)

	if _, e := exec.Command("ffmpeg", arg...).Output(); e != nil {
		log.Panicf("[PANIC] Failed to execute ffmpeg, error: %v", e)
	} //else { log.Println(out) }

	result := []string{}
	for i := 0; i < expected_file_count; i++ {
		result = append(result, fsys.Join(dir, "."+name+"_video_"+strconv.Itoa(i), ext))
	}

	temp := []string{}
	copy(temp, result)
	for i, fp := range temp {
		if !fsys.IsFile(fp) {
			remove(result, i)
		}
	}

	return result
}

func EncodeAudioOnly(fp_in string) string {
	fp_in = fsys.Sanitize(fp_in)
	dir, name, _ := fsys.Split(fp_in)
	outfile := fsys.Join(dir, "."+name+"_audio", ".ogg")

	arg := strings.Fields(FFMPEG_COMMON_INPUT_ARG + "-i")
	arg = append(arg, fp_in)
	arg = append(arg, strings.Fields(FFMPEG_COMMON_OUTPUT_ARG+"-vn -c:a libopus -b:a 128k -map 0:a:0?")...)
	arg = append(arg, outfile)

	//log.Println(arg)

	if _, e := exec.Command("ffmpeg", arg...).Output(); e != nil {
		log.Panicf("[PANIC] Failed to execute ffmpeg, error: %v", e)
	} //else { log.Println(out) }

	return outfile
}

func EncodeVideoOnly(fp_in string, ext_out string) string {
	dir, name, _ := fsys.Split(fp_in)
	outfile := fsys.Join(dir, name+"_transcoded", ext_out)

	arg := strings.Fields(FFMPEG_COMMON_INPUT_ARG + "-threads 0 -i")
	arg = append(arg, fp_in)
	arg = append(arg, strings.Fields(FFMPEG_COMMON_OUTPUT_ARG+"-c:v libvpx-vp9 -b:v 0 -pix_fmt:v yuv420p -cpu-used:v 4 -crf:v 27")...)
	arg = append(arg, outfile)

	//log.Println(arg)

	if _, e := exec.Command("ffmpeg", arg...).Output(); e != nil {
		log.Panicf("[PANIC] Failed to execute ffmpeg, error: %v", e)
	} //else { log.Println(out) }

	return outfile
}

func ConcatFiles(fps_in []string, ext_out string) string {
	if len(fps_in) == 0 {
		log.Panicf("[PANIC] Length of input file list is 0")
	}

	dir, common_name, ext := fsys.Split(fps_in[0])
	for _, fp := range fps_in {
		_dir, _, _ext := fsys.Split(fp)
		if !(_dir == dir && _ext == ext) {
			log.Panicf("[PANIC] Not the same directory (%v != %v) and extension (%v != %v)", _dir, dir, _ext, ext)
		}
	}

	re := regexp.MustCompile(`^(\..*)_\d*_transcoded$`)
	common_name = re.FindStringSubmatch(common_name)[1]

	fp_text := fsys.Join(dir, common_name, ".txt")
	fp_out := fsys.Join(dir, common_name, ext_out)

	// write text file for ffmpeg concat function
	f_text, e := os.OpenFile(fp_text, os.O_CREATE|os.O_WRONLY, 0644)
	if e != nil {
		log.Panicf("[PANIC] Failed to create/open file %v, error: %v", f_text, e)
	}
	for _, fp := range fps_in {
		_, e := f_text.Write([]byte("file '" + fp + "'\n"))
		if e != nil {
			log.Panicf("[PANIC] Failed to create/open file %v, error: %v", f_text, e)
		}
	}
	f_text.Close()

	// ffmpeg concat
	arg := strings.Fields(FFMPEG_COMMON_INPUT_ARG + "-f concat -safe 0 -i")
	arg = append(arg, fp_text)
	arg = append(arg, strings.Fields(FFMPEG_COMMON_OUTPUT_ARG+"-c:v copy")...)
	arg = append(arg, fp_out)

	//log.Println(arg)

	if _, e := exec.Command("ffmpeg", arg...).Output(); e != nil {
		log.Panicf("[PANIC] Failed to execute ffmpeg, error: %v", e)
	} //else { log.Println(out) }

	os.Remove(fp_text)
	return fp_out
}

func MuxVideoAudio(fp_in_video string, fp_in_audio string) string {
	video_dir, video_name, video_ext := fsys.Split(fp_in_video)
	audio_dir, _, _ := fsys.Split(fp_in_video)
	if !(video_dir == audio_dir) {
		log.Panicf("[PANIC] Failed to mux video %v, and audio %v. They should be on the same directory", fp_in_video, fp_in_audio)
	}

	re := regexp.MustCompile(`^\.(.*)_(video|audio)$`)
	video_name = re.FindStringSubmatch(video_name)[1]

	fp_out := fsys.Join(video_dir, video_name, video_ext)

	arg := strings.Fields(FFMPEG_COMMON_INPUT_ARG)
	arg = append(arg, "-i", fp_in_video)
	arg = append(arg, "-i", fp_in_audio)
	arg = append(arg, strings.Fields(FFMPEG_COMMON_OUTPUT_ARG+"-c:v copy -c:a copy -map 0:v:0 -map 1:a:0")...)
	arg = append(arg, fp_out)

	//log.Println(arg)

	if _, e := exec.Command("ffmpeg", arg...).Output(); e != nil {
		log.Panicf("[PANIC] Failed to execute ffmpeg, error: %v", e)
	} //else { log.Println(out) }

	return fp_out

}
