package main

import (
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/tidwall/gjson"
)

func flattenJSONKey(json gjson.Result) map[string]bool {
	root := "@this"
	reserve := []string{root}
	complete := make(map[string]bool)

	//fmt.Printf("%v -> %v -> %v INIT\n", reserve, nil, complete)
	//fmt.Println(json)

	for {
		// pop
		current := reserve[0]
		reserve = reserve[1:]

		//fmt.Printf("%v -> %v -> %v PRE\n", reserve, current, complete)

		for k, v := range json.Get(current).Map() {
			newkey := strings.TrimPrefix(current+"."+k, "@this.")
			if v.Type == gjson.JSON {
				// push
				reserve = append(reserve, newkey)
				//fmt.Printf("%v -> %v -> %v JSON\n", reserve, current, complete)
			} else {
				complete[newkey] = true
				//fmt.Printf("%v -> %v -> %v ELSE\n", reserve, current, complete)
			}
		}

		if len(reserve) == 0 {
			break
		}
	}

	return complete
}

func matchRegexPCRE2(regex string, target string) bool {
	re, _ := regexp2.Compile(regex, 0)
	matched, _ := re.MatchString(target)
	return matched
}

func selectAudioStream(ctx *TranscodingContext) int {
	// check the number of audio streams
	scoreboard := map[int]uint{}
	for index, info := range ctx.stream_info {
		if info.Get("codec_type").String() == "audio" {
			scoreboard[index] = 0
		}
	}
	if len(scoreboard) == 1 {
		return 0
	}

	// check the selection_prefer query
	prefer := ctx.config.Get("audio.selection_prefer")
	if !prefer.Exists() {
		return 0
	}

	// check the selection_priority query
	priority := ctx.config.Get("audio.selection_priority")
	if !priority.Exists() {
		return 0
	}

	// check the key equivalence of two query
	prefer_keys := flattenJSONKey(priority)
	priority_keys := flattenJSONKey(priority)
	for key := range prefer_keys {
		if !priority_keys[key] {
			return 0
		}
	}

	// find best-fit
	for index := range scoreboard {
		target := ctx.stream_info[index]
		target_keys := flattenJSONKey(target)
		score_unit := 63
		for key := range priority_keys {
			if target_keys[key] {
				if matchRegexPCRE2(prefer.Get(key).String(), target.Get(key).String()) {
					scoreboard[index] += (1 << score_unit)
				}
			}
			score_unit--
			if score_unit < 0 {
				// max priority: 64
				break
			}
		}
	}

	// find best score and return the stream index
	max_index := 0
	max_score := uint(0)
	for index, score := range scoreboard {
		if score > max_score {
			max_score = score
			max_index = index
		}
	}

	return max_index
}

/*
func checkSkip(ctx *TranscodingContext) bool {
	// stream info, 파일 확장자 2개를 본다.
	// audio 파일, video 파일 서로 보는게 다르다
	for query_key, query_value := range query.Map() {
		if !matchRegexPCRE2(query_value.String(), stream_info.Get(query_key).String()) {
			// not matching
			return false
		}
	}

	return true
}
*/

/*
func streamWizard(fp_in, ext_out string, conf gjson.Result) map[string]StreamSelectionResult {
	// find best fit
	// find skippable thing -> (transcode, skip)
	// compare to ext -> (skip, copy)
}

func audioStreamSelection() {

}
*/
