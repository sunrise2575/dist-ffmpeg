package main

func processImageOnly(ctx *TranscodingContext) FilepathSplit {
	temp := FilepathSplit{
		dir:  ctx.temp_dir,
		name: "." + ctx.fp.name + "_image_" + ctx.id,
		ext:  "." + ctx.config.Get("image.target_ext").String(),
	}

	if checkSkip(ctx, "image", 0) {
		ffmpegEncodeAudioOnly(ctx.fp, temp, "-an -c:v copy", 0)
	} else {
		ffmpegEncodeVideoOnly(ctx.fp, temp, ctx.config.Get("image.ffmpeg_param").String())
	}
	ctx.SwapFileToOriginal(temp)

	return temp
}

func processAudioOnly(ctx *TranscodingContext) FilepathSplit {
	temp := FilepathSplit{
		dir:  ctx.temp_dir,
		name: "." + ctx.fp.name + "_audio_" + ctx.id,
		ext:  "." + ctx.config.Get("audio.target_ext").String(),
	}

	// select audio stream (not implemented)
	audio_stream := selectAudioStream(ctx)
	if checkSkip(ctx, "audio", audio_stream) {
		ffmpegEncodeAudioOnly(ctx.fp, temp, "-vn -c:a copy", audio_stream)
	} else {
		ffmpegEncodeAudioOnly(ctx.fp, temp, ctx.config.Get("audio.ffmpeg_param").String(), audio_stream)
	}
	ctx.SwapFileToOriginal(temp)

	return temp
}

func processVideoOnly(ctx *TranscodingContext) FilepathSplit {
	temp := FilepathSplit{
		dir:  ctx.temp_dir,
		name: "." + ctx.fp.name + "_video_" + ctx.id,
		ext:  "." + ctx.config.Get("video.target_ext").String(),
	}

	if checkSkip(ctx, "video", 0) {
		ffmpegEncodeAudioOnly(ctx.fp, temp, "-an -c:v copy", 0)
	} else {
		ffmpegEncodeVideoOnly(ctx.fp, temp, ctx.config.Get("video.ffmpeg_param").String())
	}
	ctx.SwapFileToOriginal(temp)

	return temp
}
