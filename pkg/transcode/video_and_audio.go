package transcode

import (
	"context"
	"os"
	"sync"

	"runtime"

	"github.com/sirupsen/logrus"
)

type job struct {
	index    int
	filepath File
}

func encodeAudioPart(ctx context.Context, meta *Metadata, fp_audio_out File) chan error {
	c := make(chan error)
	go func() {
		defer close(c)
		c <- func() error {
			audio_stream_idx := selectAudioStream(meta)
			skip_audio := isSkippable(meta, audio_stream_idx)

			var e error

			if skip_audio {
				e = ffmpegEncodeAudioOnly(ctx, meta.FilePath, fp_audio_out, "-vn -c:a copy", audio_stream_idx)
			} else {
				e = ffmpegEncodeAudioOnly(ctx, meta.FilePath, fp_audio_out, meta.Config.Get("audio.ffmpeg_param").String(), audio_stream_idx)
			}

			if e != nil {
				return e
			}

			return nil
		}()
	}()
	return c
}

func videoSegmentFeeder(ctx context.Context, job_q chan<- job, fps_video []File) chan error {
	c := make(chan error)
	go func() {
		defer close(c)
		for index, fp := range fps_video {
			job_q <- job{
				index:    index,
				filepath: fp,
			}
		}
		c <- nil
	}()
	return c
}

func videoSegmentProcessor(ctx context.Context, meta *Metadata, job_q <-chan job, fps_video_comp []File, video_stream_idx int, worker_id int) chan error {
	c := make(chan error)
	go func(worker_id int) {
		c <- func() error {
			for j := range job_q {
				fp_video_temp := File{
					Dir:  meta.TempDir,
					Name: j.filepath.Name + "_converted",
					Ext:  "." + meta.Config.Get("video.target_ext").String(),
				}
				if e := ffmpegEncodeVideoOnly(
					ctx,
					j.filepath,
					fp_video_temp,
					meta.Config.Get("video.ffmpeg_param").String(),
					video_stream_idx); e != nil {
					return e
				}
				if e := os.RemoveAll(j.filepath.Join()); e != nil {
					return e
				}
				fps_video_comp[j.index] = fp_video_temp
			}
			return nil
		}()
	}(worker_id)
	return c
}

func encodeVideoPart(ctx context.Context, meta *Metadata, fp_video_out File) chan error {
	c := make(chan error)
	go func() {
		defer close(c)
		c <- func() error {
			video_stream_idx := 0
			{
				// find video stream index
				for i, v := range meta.StreamInfo {
					if v.Get("codec_type").String() == "video" {
						video_stream_idx = i
						break
					}
				}

				if isSkippable(meta, video_stream_idx) {
					return ffmpegEncodeVideoOnly(
						ctx,
						meta.FilePath,
						fp_video_out,
						"-an -c:v copy",
						video_stream_idx)
				}
			}

			workers := runtime.NumCPU() / 4

			split_file_rule := File{
				Dir:  meta.TempDir,
				Name: "." + meta.ID + "_video_%d", // must use %d
				Ext:  meta.FilePath.Ext,
			}
			fps_video, e := ffmpegSplitVideo(
				ctx,
				meta.FilePath,
				meta.TempDir,
				split_file_rule,
				video_stream_idx,
				workers)
			if e != nil {
				return e
			}

			fps_video_comp := make([]File, len(fps_video))
			{
				var wg sync.WaitGroup

				ctx, cancel := context.WithCancel(ctx)
				job_q := make(chan job, 64)

				wg.Add(1)
				go func() {
					defer func() {
						close(job_q)
						wg.Done()
					}()
					select {
					case <-videoSegmentFeeder(ctx, job_q, fps_video):
						if e != nil {
							cancel()
						}
					case <-ctx.Done():
						// canceled
					}
				}()

				// video segment path consumer
				for worker_id := 0; worker_id < workers; worker_id++ {
					wg.Add(1)
					go func(worker_id int) {
						defer wg.Done()
						select {
						case e := <-videoSegmentProcessor(ctx, meta, job_q, fps_video_comp, video_stream_idx, worker_id):
							if e != nil {
								cancel()
							}
							// nothing
						case <-ctx.Done():
							// canceled
						}
					}(worker_id)
				}

				wg.Wait()

				if ctx.Err() != nil {
					return ctx.Err()
				}
			}

			// concat

			fp_text := File{
				Dir:  meta.TempDir,
				Name: "." + meta.ID + "_concatlist",
				Ext:  ".txt",
			}

			return ffmpegConcatFiles(ctx, fps_video_comp, fp_text, fp_video_out)
		}()
	}()
	return c
}

func VideoAndAudio(ctx context.Context, meta *Metadata) error {
	ctx, cancel := context.WithCancel(ctx)

	fp_audio := File{
		Dir:  meta.TempDir,
		Name: "." + meta.ID + "_audio",
		Ext:  "." + meta.Config.Get("audio.target_ext").String(),
	}

	fp_video := File{
		Dir:  meta.TempDir,
		Name: "." + meta.ID + "_videoconcat",
		Ext:  "." + meta.Config.Get("video.target_ext").String(),
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case e := <-encodeAudioPart(ctx, meta, fp_audio):
			if e != nil {
				cancel()
			}
		case <-ctx.Done():
			logrus.Debugf("encodeAudioPart() cancelled")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case e := <-encodeVideoPart(ctx, meta, fp_video):
			if e != nil {
				cancel()
			}
		case <-ctx.Done():
			logrus.Debugf("encodeAudioPart() cancelled")
		}
	}()

	wg.Wait()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	fp_mux_out := File{
		Dir:  meta.TempDir,
		Name: "." + meta.ID + "_mux",
		Ext:  "." + meta.Config.Get("video.target_ext").String(),
	}

	if e := ffmpegMuxVideoAudio(ctx, fp_video, fp_audio, fp_mux_out); e != nil {
		return e
	}

	if e := meta.SwapFileToOriginal(fp_mux_out); e != nil {
		return e
	}

	if e := os.RemoveAll(fp_video.Join()); e != nil {
		return e
	}

	if e := os.RemoveAll(fp_audio.Join()); e != nil {
		return e
	}

	return nil
}
