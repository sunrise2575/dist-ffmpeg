package transcode

import (
	"github.com/sirupsen/logrus"
	"github.com/sunrise2575/VP9-parallel/pkg/ffprobe"
	"github.com/sunrise2575/VP9-parallel/pkg/util"
	"github.com/tidwall/gjson"
)

type File struct {
	Dir, Name, Ext string
}

func (fps *File) Fill(fp_in string) {
	fps.Dir, fps.Name, fps.Ext = util.PathSplit(fp_in)
}

func (fps *File) Join() string {
	return util.PathJoin(fps.Dir, fps.Name, fps.Ext)
}

type Context struct {
	// hashed ID
	ID string

	// file's original info.
	FilePath    File
	StreamInfo  []gjson.Result
	VideoFrame  int
	VideoLength float64

	// transcoding decision info.
	Config   gjson.Result
	FileType string
	//stream_selection map[string]int
	//stream_handling  map[string]string
	TempDir string
}

func (ctx *Context) Init(fp_in string, conf gjson.Result, temp_dir string) {
	ctx.FilePath.Fill(fp_in)

	ctx.ID = util.HashFNV64a(ctx.FilePath.Name)
	ctx.StreamInfo = ffprobe.StreamInfoJSON(ctx.FilePath.Join())

	ctx.FileType = ctx._DecideFileType()

	if ctx.FileType == "video" || ctx.FileType == "video_and_audio" {
		ctx.VideoLength = ffprobe.VideoTime(ctx.FilePath.Join())
	}

	ctx.Config = conf
	ctx.TempDir = temp_dir
}

func (ctx *Context) _DecideFileType() string {
	f_type := ""

	ext_image := map[string]bool{".jpg": true, ".png": true, ".gif": true, ".webp": true}
	ext_audio := map[string]bool{".m4a": true, ".mp3": true, ".ogg": true, ".opus": true, ".mka": true, ".wav": true, ".flac": true}
	ext_video := map[string]bool{".asf": true, ".avi": true, ".bik": true, ".flv": true, ".mkv": true, ".mov": true, ".mp4": true, ".mpeg": true, ".3gp": true, ".ts": true, ".webm": true, ".wmv": true}

	if ext_image[ctx.FilePath.Ext] {
		ctx.VideoFrame = ffprobe.VideoFrame(ctx.FilePath.Join())
		if ctx.VideoFrame > 1 {
			f_type = "image_animated"
		} else {
			f_type = "image"
		}
		return f_type
	}

	if ext_audio[ctx.FilePath.Ext] {
		f_type = "audio"
		return f_type
	}

	if ext_video[ctx.FilePath.Ext] {
		exist_video, exist_audio := false, false
		for _, v := range ctx.StreamInfo {
			switch v.Get("codec_type").String() {
			case "video":
				exist_video = true
			case "audio":
				exist_audio = true
			}
		}

		switch {
		case exist_video && exist_audio:
			f_type = "video_and_audio"
		case exist_video && !exist_audio:
			f_type = "video"
		case !exist_video && exist_audio:
			f_type = "audio"
		}

		return f_type
	}

	return ""
}

func (ctx *Context) SwapFileToOriginal(fp_new File) {
	temp := File{
		Dir:  ctx.FilePath.Dir,
		Name: "." + ctx.FilePath.Name,
		Ext:  ctx.FilePath.Ext,
	}

	e := util.PathMove(ctx.FilePath.Join(), temp.Join())
	if e != nil {
		logrus.Fatalf("File Move failed! %v->%v: %v", fp_new.Join(), temp.Join(), e)
	}
	logrus.Debugf("File Move %v->%v", ctx.FilePath.Join(), temp.Join())

	temp = File{
		Dir:  ctx.FilePath.Dir,
		Name: ctx.FilePath.Name,
		Ext:  fp_new.Ext,
	}

	e = util.PathMove(fp_new.Join(), temp.Join())
	if e != nil {
		logrus.Fatalf("File Move failed! %v->%v: %v", fp_new.Join(), temp.Join(), e)
	}
	logrus.Debugf("File Move %v->%v", fp_new.Join(), temp.Join())
}
