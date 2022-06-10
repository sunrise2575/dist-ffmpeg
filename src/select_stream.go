package main

import (
	"sort"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/tidwall/gjson"
)

func flattenJSONKey(json gjson.Result) map[string]struct{} {
	root := "@this"
	reserve := []string{root}
	complete := make(map[string]struct{})

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
				complete[newkey] = struct{}{}
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

func selectStreamBestFit(codecType string, queryJSON gjson.Result, priorityInvertedIndex map[string]int, metaJSONArray []gjson.Result) int {

	// this function is my own "priority-aware sorting" algorithm

	type scoreType struct {
		index int
		score []int
	}

	scoreDimension := len(priorityInvertedIndex) + 1
	scoreBoard := make([]scoreType, len(metaJSONArray))

	queryKeys := flattenJSONKey(queryJSON)

	for i, streamMeta := range metaJSONArray {
		streamMetaKeys := flattenJSONKey(streamMeta)

		// alloc score section
		scoreBoard[i] = scoreType{
			index: int(streamMeta.Get("index").Int()),
			score: make([]int, scoreDimension)}

		// hash-join (match each line of flattened JSON metadata and JSON query)
		for queryKey := range queryKeys {
			if _, ok0 := streamMetaKeys[queryKey]; ok0 {
				if matchRegexPCRE2(queryJSON.Get(queryKey).String(), streamMeta.Get(queryKey).String()) {
					// if the queryKey is in "matching_priority"
					if scoreIndex, ok1 := priorityInvertedIndex[queryKey]; ok1 {
						scoreBoard[i].score[scoreIndex] += 1
					} else {
						scoreBoard[i].score[scoreDimension-1] += 1
					}
				}
			}
		}
	}

	sort.Slice(scoreBoard, func(i, j int) bool {
		// lower index in the score array == higher priority
		for s := 0; s < scoreDimension; s++ {
			if scoreBoard[i].score[s] == scoreBoard[j].score[s] {
				continue
			}
			return scoreBoard[i].score[s] > scoreBoard[j].score[s]
		}

		// if rank is not decided
		return scoreBoard[i].index < scoreBoard[j].index
	})

	return scoreBoard[0].index
}

/*
func selectStream(arg commonArgType, mediaMetaJSON gjson.Result) map[string]transcodingInfoType {

	// group-by existing stream in the media file
	groupBy := make(map[string][]gjson.Result)
	for _, metaJSON := range mediaMetaJSON.Array() {
		codecType := metaJSON.Get("codec_type").String()
		if value, ok := groupBy[codecType]; ok {
			groupBy[codecType] = append(value, metaJSON)
		} else {
			groupBy[codecType] = []gjson.Result{metaJSON}
		}
	}

	result := make(map[string]transcodingInfoType)

	// intersect stream type between groupBy and queryInvertedIndex
	// i.e. stream=["Video":["video0", "video1"], "subtitle0"] AND query=["Video", "Audio"] = [best_fit(["video0", "video1"])]
	for codecType, metaJSONArray := range groupBy {
		{
			if _, ok := arg.queryJSON[codecType]; !ok {
				continue
			}
		}

		// after this line, the stream type is exists both input media side and query side
		temp := -1

		currentJSON := arg.queryJSON[codecType]
		queryJSON := currentJSON.Get("select_prefer")

		if len(metaJSONArray) > 1 && queryJSON.Exists() {
			// the input media have multiple stream of same type and the user specifies the stream information
			// it must pick best-fit single stream from input media stream

			// preprocessing for best-fit
			priorityInvertedIndex := make(map[string]int)

			priorityJSON := currentJSON.Get("select_priority")
			if priorityJSON.Exists() {
				for index, key := range priorityJSON.Array() {
					priorityInvertedIndex[key.String()] = index
				}
			}

			// find best-fit
			temp = findStreamBestFit(codecType, queryJSON, priorityInvertedIndex, metaJSONArray)
		} else {
			// the input media have single stream of same type or the user doesn't specifies the stream information
			temp = int(metaJSONArray[0].Get("index").Int())
		}
		result[codecType] =
			transcodingInfoType{
				streamIndex:         temp,
				isTranscodeRequired: true,
				streamInfo:          mediaMetaJSON.Array()[temp]}
	}

	return result
}
*/

func checkSkip(stream_info gjson.Result, query gjson.Result) bool {
	for query_key, query_value := range query.Map() {
		if !matchRegexPCRE2(query_value.String(), stream_info.Get(query_key).String()) {
			// not matching
			return false
		}
	}

	return true
}

/*
func streamWizard(fp_in, ext_out string, conf gjson.Result) map[string]StreamSelectionResult {
	// find best fit
	// find skippable thing -> (transcode, skip)
	// compare to ext -> (skip, copy)
}

func audioStreamSelection() {

}
*/
