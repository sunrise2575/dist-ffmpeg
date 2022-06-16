package transcode

import (
	"os"

	"runtime"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/sunrise2575/VP9-parallel/pkg/util"
)

type job struct {
	index    int
	filepath File
}

func VideoAndAudio(ctx *Context) File {
	// find video stream index
	video_stream_idx := 0
	for i, v := range ctx.StreamInfo {
		if v.Get("codec_type").String() == "video" {
			video_stream_idx = i
			break
		}
	}

	audio_stream_idx := selectAudioStream(ctx)

	skip_audio := isSkippable(ctx, "audio", audio_stream_idx)
	skip_video := isSkippable(ctx, "video", video_stream_idx)

	// audio
	fp_audio := File{
		Dir:  ctx.TempDir,
		Name: "." + ctx.FilePath.Name + "_audio_" + ctx.ID,
		Ext:  "." + ctx.Config.Get("audio.target_ext").String(),
	}

	audio_complete := make(chan bool, 1)

	go func() {
		defer close(audio_complete)

		if skip_audio {
			ffmpegEncodeAudioOnly(ctx.FilePath, fp_audio, "-vn -c:a copy", audio_stream_idx)
		} else {
			ffmpegEncodeAudioOnly(ctx.FilePath, fp_audio, ctx.Config.Get("audio.ffmpeg_param").String(), audio_stream_idx)
		}

		audio_complete <- true
	}()

	fp_video_concat_out := File{
		Dir:  ctx.TempDir,
		Name: "." + ctx.FilePath.Name + "_videoconcat_" + ctx.ID,
		Ext:  "." + ctx.Config.Get("video.target_ext").String(),
	}

	if skip_video {
		ffmpegEncodeVideoOnly(ctx.FilePath, fp_video_concat_out, "-an -c:v copy")
	} else {
		var wg sync.WaitGroup

		// video split transcoding
		fps_video := ffmpegSplitVideo(ctx, runtime.NumCPU())
		fps_video_comp := make([]File, len(fps_video))

		job_q := make(chan job, 128)

		// video segment path feeder
		wg.Add(1)
		go func() {
			defer func() {
				close(job_q)
				wg.Done()
			}()
			for index, fp := range fps_video {
				job_q <- job{
					index:    index,
					filepath: fp,
				}
			}
		}()

		// video segment path consumer
		for worker_id := 0; worker_id < runtime.NumCPU()/5; worker_id++ {
			//for worker_id := 0; worker_id < 1; worker_id++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := range job_q {
					temp := File{
						Dir:  ctx.TempDir,
						Name: j.filepath.Name + "_converted",
						Ext:  "." + ctx.Config.Get("video.target_ext").String(),
					}

					ffmpegEncodeVideoOnly(j.filepath, temp, ctx.Config.Get("video.ffmpeg_param").String())
					os.RemoveAll(j.filepath.Join())
					fps_video_comp[j.index] = temp
				}
			}()
		}

		fp_text := File{
			Dir:  ctx.TempDir,
			Name: "." + ctx.FilePath.Name + "_concatlist_" + ctx.ID,
			Ext:  ".txt",
		}

		wg.Wait()

		ffmpegConcatFiles(fps_video_comp, fp_text, fp_video_concat_out)
	}

	<-audio_complete

	fp_mux_out := File{
		Dir:  ctx.TempDir,
		Name: "." + ctx.FilePath.Name + "_mux_" + ctx.ID,
		Ext:  "." + ctx.Config.Get("video.target_ext").String(),
	}

	ffmpegMuxVideoAudio(fp_video_concat_out, fp_audio, fp_mux_out)

	ctx.SwapFileToOriginal(fp_mux_out)

	e := os.RemoveAll(fp_video_concat_out.Join())
	if e != nil {
		logrus.WithFields(
			logrus.Fields{
				"path_target": fp_video_concat_out.Join(),
				"error":       e,
				"where":       util.GetCurrentFunctionInfo(),
			}).Warnf("Fail to remove a file")
	}
	e = os.Remove(fp_audio.Join())
	if e != nil {
		logrus.WithFields(
			logrus.Fields{
				"path_target": fp_audio.Join(),
				"error":       e,
				"where":       util.GetCurrentFunctionInfo(),
			}).Warnf("Fail to remove a file")
	}

	return fp_mux_out
}
