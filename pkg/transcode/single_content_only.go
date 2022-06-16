package transcode

func ImageOnly(ctx *Context) File {
	temp := File{
		Dir:  ctx.TempDir,
		Name: "." + ctx.FilePath.Name + "_image_" + ctx.ID,
		Ext:  "." + ctx.Config.Get("image.target_ext").String(),
	}

	if isSkippable(ctx, "image", 0) {
		ffmpegEncodeAudioOnly(ctx.FilePath, temp, "-an -c:v copy", 0)
	} else {
		ffmpegEncodeVideoOnly(ctx.FilePath, temp, ctx.Config.Get("image.ffmpeg_param").String())
	}

	ctx.SwapFileToOriginal(temp)
	return temp
}

func AudioOnly(ctx *Context) File {
	temp := File{
		Dir:  ctx.TempDir,
		Name: "." + ctx.FilePath.Name + "_audio_" + ctx.ID,
		Ext:  "." + ctx.Config.Get("audio.target_ext").String(),
	}

	// select audio stream (not implemented)
	audio_stream := selectAudioStream(ctx)
	if isSkippable(ctx, "audio", audio_stream) {
		ffmpegEncodeAudioOnly(ctx.FilePath, temp, "-vn -c:a copy", audio_stream)
	} else {
		ffmpegEncodeAudioOnly(ctx.FilePath, temp, ctx.Config.Get("audio.ffmpeg_param").String(), audio_stream)
	}
	ctx.SwapFileToOriginal(temp)
	return temp
}

func VideoOnly(ctx *Context) File {
	temp := File{
		Dir:  ctx.TempDir,
		Name: "." + ctx.FilePath.Name + "_video_" + ctx.ID,
		Ext:  "." + ctx.Config.Get("video.target_ext").String(),
	}

	if isSkippable(ctx, "video", 0) {
		ffmpegEncodeAudioOnly(ctx.FilePath, temp, "-an -c:v copy", 0)
	} else {
		ffmpegEncodeVideoOnly(ctx.FilePath, temp, ctx.Config.Get("video.ffmpeg_param").String())
	}

	ctx.SwapFileToOriginal(temp)
	return temp
}
