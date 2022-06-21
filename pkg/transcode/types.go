package transcode

import (
	"github.com/sunrise2575/dist-ffmpeg/pkg/ffprobe"
	"github.com/sunrise2575/dist-ffmpeg/pkg/util"
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

type Metadata struct {
	// hashed ID
	ID string

	// file's original info.
	FilePath   File
	StreamInfo []gjson.Result
	VideoFrame int

	// transcoding decision info.
	Config   gjson.Result
	FileType string
	TempDir  string
}

func (meta *Metadata) Init(fp_in string, conf gjson.Result, temp_dir string) error {
	meta.FilePath.Fill(fp_in)

	meta.ID = util.HashFNV64a(meta.FilePath.Name)
	var e error
	meta.StreamInfo, e = ffprobe.StreamInfoJSON(meta.FilePath.Join())
	if e != nil {
		return e
	}

	meta.FileType, e = meta._DecideFileType()
	if e != nil {
		return e
	}

	meta.Config = conf
	meta.TempDir = temp_dir

	return nil
}

func (meta *Metadata) _DecideFileType() (string, error) {
	f_type := ""

	ext_image := map[string]bool{".bmp": true, ".jpg": true, ".png": true, ".gif": true, ".webp": true}
	ext_audio := map[string]bool{".m4a": true, ".mp3": true, ".ogg": true, ".opus": true, ".mka": true, ".wav": true, ".flac": true, ".dtshd": true, ".tak": true}
	ext_video := map[string]bool{".asf": true, ".avi": true, ".bik": true, ".flv": true, ".mkv": true, ".mov": true, ".mp4": true, ".mpeg": true, ".3gp": true, ".ts": true, ".webm": true, ".wmv": true}

	if ext_image[meta.FilePath.Ext] {
		var e error
		meta.VideoFrame, e = ffprobe.VideoFrame(meta.FilePath.Join())
		if e != nil {
			return "", e
		}
		if meta.VideoFrame > 1 {
			f_type = "image_animated"
		} else {
			f_type = "image"
		}
		return f_type, nil
	}

	if ext_audio[meta.FilePath.Ext] {
		f_type = "audio"
		return f_type, nil
	}

	if ext_video[meta.FilePath.Ext] {
		exist_video, exist_audio := false, false
		for _, v := range meta.StreamInfo {
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

		return f_type, nil
	}

	return "", nil
}

func (meta *Metadata) SwapFileToOriginal(fp_new File) error {
	temp := File{
		Dir:  meta.FilePath.Dir,
		Name: "." + meta.FilePath.Name,
		Ext:  meta.FilePath.Ext,
	}
	if e := util.PathMove(meta.FilePath.Join(), temp.Join()); e != nil {
		return e
	}

	temp = File{
		Dir:  meta.FilePath.Dir,
		Name: meta.FilePath.Name,
		Ext:  fp_new.Ext,
	}
	return util.PathMove(fp_new.Join(), temp.Join())
}
