package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/sunrise2575/VP9-parallel/src/ffmpeg"
	"github.com/sunrise2575/VP9-parallel/src/fsys"
)

func encodeVP9(fp_in string, audio_stream_number int, ext_out string) string {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ABORT] %v, capture a panic", fp_in)
			debug.PrintStack()
		}
	}()

	audio_res := func() chan string {
		res := make(chan string, 1)
		go func(fp_in string) {
			defer close(res)
			res <- ffmpeg.EncodeAudioOnly(fp_in, audio_stream_number)
		}(fp_in)
		return res
	}()

	fps_video_split := ffmpeg.SplitVideo(fp_in, runtime.NumCPU()/4)
	//log.Println(fps_video_split)

	var wg sync.WaitGroup
	fps_video_encode := make([]string, len(fps_video_split))
	for i, fp := range fps_video_split {
		wg.Add(1)
		go func(worker_id int, fp, ext_out string) {
			defer wg.Done()
			fps_video_encode[worker_id] = ffmpeg.EncodeVideoOnly(fp, ext_out)
			os.RemoveAll(fp)
		}(i, fp, ext_out)
	}
	wg.Wait()
	//log.Println(fps_video_encode)

	fp_video := ffmpeg.ConcatFiles(fps_video_encode, ext_out)
	//log.Println(fp_video)
	for _, fp := range fps_video_encode {
		os.RemoveAll(fp)
	}

	fp_audio := <-audio_res
	//log.Println(fp_audio)

	// change original filename
	{
		dir, name, ext := fsys.Split(fp_in)
		os.Rename(fp_in, fsys.Join(dir, "."+name, ext))
	}

	fp_out := ffmpeg.MuxVideoAudio(fp_video, fp_audio)
	//log.Println(fp_out)

	os.Remove(fp_video)
	os.Remove(fp_audio)

	return fp_out
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	dp_in := "C:/Users/heeyong/Desktop/test/"
	dp_in = fsys.Sanitize(dp_in)
	ext_out := ".webm"

	ext_candidate := map[string]struct{}{}
	for _, v := range []string{".mp4", ".mkv", ".avi"} {
		ext_candidate[v] = struct{}{}
	}

	filepath.Walk(dp_in, func(fp_in string, f_info os.FileInfo, err error) error {
		if f_info.IsDir() {
			return nil
		}
		if len(filepath.Ext(fp_in)) < 2 {
			return nil
		}
		_, name, ext := fsys.Split(fp_in)
		if !(len(name) > 1 && name[0] != '.') {
			return nil
		}
		if _, ok := ext_candidate[ext]; !ok {
			return nil
		}

		log.Printf("[START] %v", fp_in)
		start := time.Now()
		audio_stream_number := 0
		fp_out := encodeVP9(fp_in, audio_stream_number, ext_out)
		elapsed := time.Since(start)
		log.Printf("[DONE] %v, elapsed: %v (sec)", fp_out, elapsed.Seconds())

		return nil
	})
}
