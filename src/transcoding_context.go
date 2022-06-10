package main

import (
	"hash/fnv"
	"os"
	"strconv"

	"github.com/sunrise2575/VP9-parallel/src/fsys"
	"github.com/tidwall/gjson"
)

type FilepathSplit struct {
	dir, name, ext string
}

func (fps *FilepathSplit) Fill(fp_in string) {
	fps.dir, fps.name, fps.ext = fsys.Split(fp_in)
}

func (fps *FilepathSplit) Join() string {
	return fsys.Join(fps.dir, fps.name, fps.ext)
}

type TranscodingContext struct {
	// hashed ID
	id string

	// file's original info.
	fp           FilepathSplit
	stream_info  []gjson.Result
	video_frame  int
	video_length float64

	// transcoding decision info.
	config    gjson.Result
	file_type string
	//stream_selection map[string]int
	//stream_handling  map[string]string
	temp_dir string
}

func (ctx *TranscodingContext) Init(fp_in string, conf gjson.Result, temp_dir string) {
	ctx.fp.Fill(fp_in)

	ctx.id = ctx._FNV64a(ctx.fp.name)
	ctx.stream_info = ffprobeStreamInfoJSON(ctx.fp.Join())

	ctx.file_type = ctx._DecideFileType()

	if ctx.file_type == "video" || ctx.file_type == "video_only" {
		ctx.video_length = ffprobeVideoTime(ctx.fp.Join())
	}

	ctx.config = conf
	ctx.temp_dir = temp_dir
}

func (ctx *TranscodingContext) _FNV64a(text string) string {
	algorithm := fnv.New64a()
	algorithm.Write([]byte(text))
	return strconv.FormatUint(algorithm.Sum64(), 10)
}

func (ctx *TranscodingContext) _DecideFileType() string {
	f_type := ""

	ext_image := map[string]bool{".jpg": true, ".png": true, ".gif": true, ".webp": true}
	ext_audio := map[string]bool{".m4a": true, ".mp3": true, ".ogg": true, ".opus": true, ".mka": true, ".wav": true, ".flac": true}
	ext_video := map[string]bool{".asf": true, ".avi": true, ".bik": true, ".flv": true, ".mkv": true, ".mov": true, ".mp4": true, ".mpeg": true, ".3gp": true, ".ts": true, ".webm": true, ".wmv": true}

	if ext_image[ctx.fp.ext] {
		ctx.video_frame = ffprobeVideoFrame(ctx.fp.Join())
		if ctx.video_frame > 1 {
			f_type = "image_animated"
		} else {
			f_type = "image"
		}
		return f_type
	}

	if ext_audio[ctx.fp.ext] {
		f_type = "audio"
		return f_type
	}

	if ext_video[ctx.fp.ext] {
		exist_video, exist_audio := false, false
		for _, v := range ctx.stream_info {
			switch v.Get("codec_type").String() {
			case "video":
				exist_video = true
			case "audio":
				exist_audio = true
			}
		}

		switch {
		case exist_video && exist_audio:
			f_type = "video"
		case exist_video && !exist_audio:
			f_type = "video_only"
		case !exist_video && exist_audio:
			f_type = "audio"
		}

		return f_type
	}

	return ""
}

func (ctx *TranscodingContext) SwapFileToOriginal(fp_new FilepathSplit) {
	temp := FilepathSplit{
		dir:  ctx.fp.dir,
		name: "." + ctx.fp.name,
		ext:  ctx.fp.ext,
	}

	os.Rename(ctx.fp.Join(), temp.Join())

	temp = FilepathSplit{
		dir:  ctx.fp.dir,
		name: ctx.fp.name,
		ext:  fp_new.ext,
	}

	os.Rename(fp_new.Join(), temp.Join())
}
