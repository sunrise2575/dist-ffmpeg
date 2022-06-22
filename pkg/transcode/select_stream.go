package transcode

import (
	"github.com/sunrise2575/dist-ffmpeg/pkg/util"
)

func selectAudioStream(meta *Metadata) int {
	// check the number of audio streams
	scoreboard := map[int]uint{}

	// { video, audio, audio } -> not {v:0 a:1 a:2} but {v:0 a:0 a:1}
	// we need to mark indices within audio stream only
	real_index_map := map[int]int{}
	real_index := 0
	for index, info := range meta.StreamInfo {
		if info.Get("codec_type").String() == "audio" {
			scoreboard[index] = 0
			real_index_map[index] = real_index
			real_index++
		}
	}
	if len(scoreboard) == 1 {
		return 0
	}

	// check the selection_prefer query
	prefer := meta.Config.Get("audio.selection_prefer")
	if !prefer.Exists() {
		return 0
	}

	if !meta.Config.Get("audio.selection_priority").Exists() {
		return 0
	}

	// check the selection_priority query
	priority_keys := map[string]bool{}
	for _, v := range meta.Config.Get("audio.selection_priority").Array() {
		priority_keys[v.String()] = true
	}

	// check the key equivalence of two query
	prefer_keys := util.FlattenJSONKey(prefer)
	for key := range prefer_keys {
		if !priority_keys[key] {
			return 0
		}
	}

	// find best-fit
	for index := range scoreboard {
		target := meta.StreamInfo[index]
		target_keys := util.FlattenJSONKey(target)
		score_unit := 63
		for key := range priority_keys {
			if target_keys[key] {
				if util.MatchRegexPCRE2(prefer.Get(key).String(), target.Get(key).String()) {
					scoreboard[index] += (1 << score_unit)
				}
			}
			score_unit--
			if score_unit < 0 {
				// max priority: 63
				break
			}
		}
	}

	// find best score and return the stream index
	max_index := 9999
	max_score := uint(0)
	for index, score := range scoreboard {
		switch {
		case score == max_score:
			if index < max_index {
				max_score = score
				max_index = index
			}
		case score > max_score:
			max_score = score
			max_index = index
		}
	}

	return real_index_map[max_index]
}

func isSkippable(meta *Metadata, stream_idx int) bool {

	target := meta.StreamInfo[stream_idx]
	target_keys := util.FlattenJSONKey(target)

	query := meta.Config.Get(meta.FileType).Get("skip_if")
	query_keys := util.FlattenJSONKey(query)

	if !(target.Exists() && query.Exists()) {
		return false
	}

	for query_key := range query_keys {
		if target_keys[query_key] {
			if !util.MatchRegexPCRE2(query.Get(query_key).String(), target.Get(query_key).String()) {
				// not matching
				return false
			}
		}
	}

	return true
}
