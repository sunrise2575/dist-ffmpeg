package transcode

import "context"

func SingleStreamOnly(ctx context.Context, meta *Metadata) error {
	temp := File{
		Dir:  meta.TempDir,
		Name: "." + meta.ID,
		Ext:  "." + meta.Config.Get(meta.FileType).Get("target_ext").String(),
	}

	var e error

	if meta.FileType == "audio" {
		// audio
		audio_stream := selectAudioStream(meta)
		if isSkippable(meta, 0) {
			e = ffmpegEncodeAudioOnly(
				ctx,
				meta.FilePath,
				temp,
				"-vn -c:a copy",
				audio_stream)
		} else {
			e = ffmpegEncodeAudioOnly(
				ctx,
				meta.FilePath,
				temp,
				meta.Config.Get(meta.FileType).Get("ffmpeg_param").String(),
				audio_stream)
		}
	} else {
		// image, video
		if isSkippable(meta, 0) {
			e = ffmpegEncodeVideoOnly(
				ctx,
				meta.FilePath,
				temp,
				"-an -c:v copy",
				0)
		} else {
			e = ffmpegEncodeVideoOnly(
				ctx,
				meta.FilePath,
				temp,
				meta.Config.Get(meta.FileType).Get("ffmpeg_param").String(),
				0)
		}
	}

	if e != nil {
		return e
	}

	return meta.SwapFileToOriginal(temp)
}
