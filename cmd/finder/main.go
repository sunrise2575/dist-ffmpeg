package main

import (
	"bufio"
	"flag"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/sunrise2575/dist-ffmpeg/pkg/ffprobe"
	"github.com/sunrise2575/dist-ffmpeg/pkg/util"
)

var (
	DIRECTORY                       string
	LOG_LEVEL, LOG_FILE, LOG_FORMAT string
)

func init() {

	// log options
	flag.StringVar(&LOG_LEVEL, "loglevel", "info", "panic, fatal, error, warn, info, debug, trace")
	flag.StringVar(&LOG_FILE, "logfile", "", "log file location")
	flag.StringVar(&LOG_FORMAT, "logformat", "text", "text, json")

	// distributed processing options
	flag.StringVar(&DIRECTORY, "dir", ".", "File root directory")

	flag.Parse()

	logrus.WithFields(logrus.Fields{"name": "loglevel", "value": LOG_LEVEL}).Debug("Argument")
	logrus.WithFields(logrus.Fields{"name": "logfile", "value": LOG_FILE}).Debug("Argument")
	logrus.WithFields(logrus.Fields{"name": "logformat", "value": LOG_FORMAT}).Debug("Argument")
	logrus.WithFields(logrus.Fields{"name": "dir", "value": DIRECTORY}).Debug("Argument")

	util.InitLogrus(LOG_FILE, LOG_LEVEL, LOG_FORMAT)

	DIRECTORY = util.PathSanitize(DIRECTORY)
	if !util.PathIsDir(DIRECTORY) {
		logrus.WithFields(logrus.Fields{"path": DIRECTORY}).Panicf("Unable to find the directory")
	}
}

func main() {
	// iterate files and transcode
	workers := 1
	chan_fp := make(chan string, workers)

	// search files recursively
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fp_in := range chan_fp {
				info, e := ffprobe.StreamInfoJSON(fp_in)
				if e != nil {
					logrus.Warnf("%v", fp_in)
					continue
				}

				logrus.Info(fp_in)
				for i, stream := range info {
					if stream.Get("codec_type").String() == "audio" {
						if info[i].Get("tags.language").String() == "jpn" {
							logrus.Info(info[i].Get("tags").String())
						}
					}
				}
			}
		}()
	}

	/*
		ext_exclude := util.Slice2Map([]string{".7z", ".rar", ".zip", ".tar", ".lzh", ".bin", ".cue", ".md5", ".mds", ".mdf", ".log", ".txt", ".lrc", ".exe", ".md", ".py", ".sample", ".go", ".mod", ".sum", ".json", ".sh", ".gitignore"})
		ext_subtitle := util.Slice2Map([]string{".smi", ".srt", ".vtt", ".ass"})
		ext_video := util.Slice2Map([]string{".webm"})
		ext_audio := util.Slice2Map([]string{".ogg"})
		ext_image := util.Slice2Map([]string{".png"})
	*/

	func() {
		defer close(chan_fp)
		logrus.WithFields(logrus.Fields{"path": DIRECTORY}).
			Infof("Start to seek files recursively in the directory")

		file, err := os.Open("/fifi/dist-ffmpeg/double-jpn-stream.log")
		if err != nil {
			logrus.Fatal(err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		// optionally, resize scanner's capacity for lines over 64K, see next example
		for scanner.Scan() {
			chan_fp <- scanner.Text()
		}

		if err := scanner.Err(); err != nil {
			logrus.Fatal(err)
		}

		/*
			filepath.Walk(DIRECTORY, func(fp_in string, f_info os.FileInfo, err error) error {
				if f_info.IsDir() {
					return nil
				}
				// file path sanity check
				_, _, ext := util.PathSplit(fp_in)
				if len(ext) < 2 {
					return nil
				}

				//if len(name) > 0 && name[0] == '.' { return nil }

				ext = strings.ToLower(ext)

				if ext_exclude[ext] || ext_subtitle[ext] {
					return nil
				}

				if ext_video[ext] || ext_audio[ext] || ext_image[ext] {
					return nil
				}

				chan_fp <- fp_in
				return nil
			})
		*/

		logrus.WithFields(logrus.Fields{"path": DIRECTORY}).
			Infof("Complete to seek files recursively in the directory")
	}()

	wg.Wait()
}
