package main

import (
	"log"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/tidwall/gjson"
)

func ffmpeg(arg []string) {
	cmd := exec.Command("ffmpeg", arg...)

	stdoutStderr, e := cmd.CombinedOutput()

	if e != nil {
		log.Panicln(e.Error())
		log.Panicln(string(stdoutStderr))
	}

	if len(string(stdoutStderr)) > 0 {
		log.Println(string(stdoutStderr))
	}
}

func ffprobe(fpath string) []gjson.Result {
	result, e := exec.Command(
		"ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		fpath).Output()

	if e != nil {
		log.Panicln(e)
	}

	return gjson.ParseBytes(result).Get("streams").Array()
}

func encodeVP9(input, output string) {
	arg := []string{
		"-hide_banner",
		"-loglevel", "warning",
		"-y",
		"-i", input,
		"-c:v", "libvpx-vp9",
		"-threads:v", "8",
		"-row-mt:v", "1",
		"-cpu-used:v", "4",
		"-b:v", "0",
		"-pix_fmt:v", "yuv420p",
		"-crf:v", "27",
		"-c:a", "libopus",
		"-b:a", "128k",
		output,
	}

	ffmpeg(arg)
}

func getFramesFullScan(fpath string) int {
	result, e := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-count_packets",
		"-show_entries", "stream=nb_read_packets",
		"-of", "csv=p=0",
		fpath).Output()

	if e != nil {
		log.Panicln(e)
	}
	frames, _ := strconv.Atoi(strings.TrimRight(string(result), "\r\n"))
	return frames
}

func getFrames(input string) int {
	frames := int64(0)

	for _, v := range ffprobe(input) {
		if v.Get("codec_type").String() == "video" {
			switch {
			case v.Get("nb_frames").Exists():
				frames = v.Get("nb_frames").Int()
			case v.Get("tags.NUMBER_OF_FRAMES").Exists():
				frames = v.Get("tags.NUMBER_OF_FRAMES").Int()
			}

			break
		}
	}

	return int(frames)
}

func segment() {
	arg := []string{
		"-i", "03.mkv",
		"-f", "segment",
		"-segment_time", "10",
		"-reset_timestamps", "1",
		"-c:v", "copy",
		"-c:a", "copy",
		"out%d.mkv",
	}
}

func splitEncode(input string, frames int, workers int) {
	unit := int(math.Ceil(float64(frames) / float64(workers)))
	actualWorkers := math.Ceil(float64(frames) / float64(unit))
	log.Printf("%v/%v, %v\n", unit, frames, actualWorkers)

	var wg sync.WaitGroup

	for start, partID := 0, 0; start < frames; start += unit {
		if frames-start < unit {
			unit = frames - start
		}

		wg.Add(1)
		go func(partID, start, unit int) {
			defer wg.Done()
			log.Println(partID, start, unit)
			arg := []string{
				"-hide_banner",
				"-loglevel", "warning",
				"-y",
				"-start_number", strconv.Itoa(start),
				"-i", input,
				"-frames:v", strconv.Itoa(unit),
				"-c:v", "copy",
				"-c:a", "copy",
				//"-c:v", "libvpx-vp9",
				//"-b:v", "0",
				//"-pix_fmt:v", "yuv420p",
				//"-crf:v", "27",
				//"-threads:v", "8",
				//"-row-mt:v", "1",
				//"-cpu-used:v", "4",
				//"-c:a", "libopus",
				//"-b:a", "128k",
				input + strconv.Itoa(partID) + ".mp4",
			}

			ffmpeg(arg)
			log.Println(partID, start, unit, "done")
		}(partID, start, unit)

		partID++
	}
	wg.Wait()

	log.Println("finish")
}

func main() {
	input := `T:\encode-testset\03.mkv`
	//output := `T:\encode-testset\03.webm`
	//encodeVP9(input, output)
	frames := getFrames(input)
	splitEncode(input, frames, 1)
}
