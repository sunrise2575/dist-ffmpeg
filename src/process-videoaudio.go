package main

import (
	"os"
	"runtime"
	"sync"
)

func processVideoAndAudio(ctx *TranscodingContext) FilepathSplit {
	// find video stream index
	video_stream_idx := 0
	for i, v := range ctx.stream_info {
		if v.Get("codec_type").String() == "video" {
			video_stream_idx = i
			break
		}
	}

	audio_stream_idx := selectAudioStream(ctx)

	skip_audio := checkSkip(ctx, "audio", audio_stream_idx)
	skip_video := checkSkip(ctx, "video", video_stream_idx)

	var wg sync.WaitGroup

	// audio
	fp_audio := FilepathSplit{
		dir:  ctx.temp_dir,
		name: "." + ctx.fp.name + "_audio_" + ctx.id,
		ext:  "." + ctx.config.Get("audio.target_ext").String(),
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		if skip_audio {
			ffmpegEncodeAudioOnly(ctx.fp, fp_audio, "-vn -c:a copy", audio_stream_idx)
		} else {
			ffmpegEncodeAudioOnly(ctx.fp, fp_audio, ctx.config.Get("audio.ffmpeg_param").String(), audio_stream_idx)
		}
	}()

	fp_video_concat_out := FilepathSplit{
		dir:  ctx.temp_dir,
		name: "." + ctx.fp.name + "_videoconcat_" + ctx.id,
		ext:  "." + ctx.config.Get("video.target_ext").String(),
	}

	if skip_video {
		ffmpegEncodeVideoOnly(ctx.fp, fp_video_concat_out, "-an -c:v copy")
	} else {
		// video split transcoding
		fps_video := ffmpegSplitVideo(ctx, runtime.NumCPU())
		fps_video_comp := make([]FilepathSplit, len(fps_video))

		type job struct {
			index int
			fp    FilepathSplit
		}

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
					index: index,
					fp:    fp,
				}
			}
		}()

		// video segment path consumer
		for worker_id := 0; worker_id < runtime.NumCPU()/8; worker_id++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := range job_q {
					temp := FilepathSplit{
						dir:  ctx.temp_dir,
						name: j.fp.name + "_converted",
						ext:  "." + ctx.config.Get("video.target_ext").String(),
					}

					ffmpegEncodeVideoOnly(j.fp, temp, ctx.config.Get("video.ffmpeg_param").String())
					os.RemoveAll(j.fp.Join())
					fps_video_comp[j.index] = temp
				}
			}()
		}

		fp_text := FilepathSplit{
			dir:  ctx.temp_dir,
			name: "." + ctx.fp.name + "_concatlist_" + ctx.id,
			ext:  ".txt",
		}

		ffmpegConcatFiles(fps_video_comp, fp_text, fp_video_concat_out)

		for _, fp := range fps_video_comp {
			os.RemoveAll(fp.Join())
		}
	}
	wg.Wait()

	fp_mux_out := FilepathSplit{
		dir:  ctx.temp_dir,
		name: "." + ctx.fp.name + "_mux_" + ctx.id,
		ext:  "." + ctx.config.Get("video.target_ext").String(),
	}

	ffmpegMuxVideoAudio(fp_video_concat_out, fp_audio, fp_mux_out)

	ctx.SwapFileToOriginal(fp_mux_out)

	os.Remove(fp_video_concat_out.Join())
	os.Remove(fp_audio.Join())

	return fp_mux_out
}
