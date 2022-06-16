package transcode

func SingleStreamOnly(ctx *Context, file_type string) File {
	temp := File{
		Dir:  ctx.TempDir,
		Name: "." + ctx.ID,
		Ext:  "." + ctx.Config.Get(file_type).Get("target_ext").String(),
	}

	if file_type == "audio" {
		// audio
		audio_stream := selectAudioStream(ctx)
		if isSkippable(ctx, file_type, 0) {
			ffmpegEncodeAudioOnly(ctx.FilePath, temp, "-vn -c:a copy", audio_stream)
		} else {
			ffmpegEncodeAudioOnly(ctx.FilePath, temp, ctx.Config.Get(file_type).Get("ffmpeg_param").String(), audio_stream)
		}
	} else {
		// image, video
		if isSkippable(ctx, file_type, 0) {
			ffmpegEncodeVideoOnly(ctx.FilePath, temp, "-an -c:v copy")
		} else {
			ffmpegEncodeVideoOnly(ctx.FilePath, temp, ctx.Config.Get(file_type).Get("ffmpeg_param").String())
		}
	}

	ctx.SwapFileToOriginal(temp)
	return temp
}
