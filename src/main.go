package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sunrise2575/VP9-parallel/src/fsys"
	"github.com/tidwall/gjson"
)

func work(fp_in string, conf gjson.Result, temp_dir string) {
	// file path sanity check
	if len(filepath.Ext(fp_in)) < 2 {
		return
	}

	_, name, _ := fsys.Split(fp_in)
	if len(name) > 0 && name[0] == '.' {
		return
	}

	ctx := TranscodingContext{}
	ctx.Init(fp_in, conf, temp_dir)
	if ctx.file_type == "" {
		return
	}

	log.Printf("[START] %v", ctx.fp.Join())

	start := time.Now()
	fp_out := FilepathSplit{}

	// transcode
	switch ctx.file_type {
	case "image_animated":
		// skip
	case "image":
		fp_out = processImageOnly(&ctx)
	case "audio":
		fp_out = processAudioOnly(&ctx)
	case "video":
		fp_out = processVideoAndAudio(&ctx)
	case "video_only":
		fp_out = processVideoOnly(&ctx)
	}

	elapsed := time.Since(start)

	// report result
	if fp_out.Join() != "" {
		log.Printf("[DONE] %v, elapsed: %v (sec)", fp_out.Join(), elapsed.Seconds())
	} else {
		log.Printf("[SKIP] %v", ctx.fp.Join())
	}
}

func readJSONFile(fp string) (gjson.Result, error) {
	b, e := ioutil.ReadFile(fp)
	if e != nil {
		return gjson.Result{}, e
	}
	return gjson.ParseBytes(b), nil
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	var dp_in, config_path, temp_dir string

	flag.StringVar(&dp_in, "root", ".", "Root path for input files")
	flag.StringVar(&config_path, "conf", "./config.json", "Config path")
	flag.StringVar(&temp_dir, "temp_dir", "./temp/", "Temp Dir")
	flag.Parse()

	if !fsys.IsDir(dp_in) {
		log.Fatalf("[fatal] %v is not a directory, or doesn't exist\n", dp_in)
	}
	dp_in = fsys.Sanitize(dp_in)

	if !fsys.IsFile(config_path) {
		log.Fatalf("[fatal] %v is not a directory, or doesn't exist\n", config_path)
	}
	config_path = fsys.Sanitize(config_path)

	temp_dir = fsys.Sanitize(temp_dir)
	os.MkdirAll(temp_dir, 0755)

	// Read config file
	conf, e := readJSONFile(config_path)
	if e != nil {
		log.Fatalf("[fatal] %v\n", e.Error())
	}

	// iterate files and transcode
	{
		var wg sync.WaitGroup

		q_in := make(chan string, 128)

		// producer (file path feeder)
		wg.Add(1)
		go func() {
			defer func() {
				close(q_in)
				wg.Done()
			}()
			filepath.Walk(dp_in, func(fp_in string, f_info os.FileInfo, err error) error {
				if f_info.IsDir() {
					return nil
				}
				q_in <- fp_in
				return nil
			})
		}()

		// consumer (transcoder)
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for item := range q_in {
					// for catching panic(), cover codes by func(){}
					func() {
						defer func() {
							if r := recover(); r != nil {
								log.Println("Recovered", r)
							}
						}()
						work(item, conf, temp_dir)
					}()
				}
			}()
		}
		wg.Wait()
	}

}
