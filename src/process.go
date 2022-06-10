package main

func processImageOnly(ctx *TranscodingContext) FilepathSplit {
	temp := FilepathSplit{
		dir:  ctx.temp_dir,
		name: "." + ctx.fp.name + "_image_" + ctx.id,
		ext:  "." + ctx.config.Get("image.target_ext").String(),
	}

	ffmpegEncodeVideoOnly(ctx.fp, temp, ctx.config.Get("image.ffmpeg_param").String())
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
	ffmpegEncodeAudioOnly(ctx.fp, temp, ctx.config.Get("audio.ffmpeg_param").String(), audio_stream)
	ctx.SwapFileToOriginal(temp)

	return temp
}

func processVideoOnly(ctx *TranscodingContext) FilepathSplit {
	temp := FilepathSplit{
		dir:  ctx.temp_dir,
		name: "." + ctx.fp.name + "_video_" + ctx.id,
		ext:  "." + ctx.config.Get("video.target_ext").String(),
	}

	ffmpegEncodeVideoOnly(ctx.fp, temp, ctx.config.Get("video.ffmpeg_param").String())
	ctx.SwapFileToOriginal(temp)

	return temp
}
