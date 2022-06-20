package transcode

import (
	"github.com/sunrise2575/dist-ffmpeg/pkg/util"
)

func selectAudioStream(meta *Metadata) int {
	// check the number of audio streams
	scoreboard := map[int]uint{}
	for index, info := range meta.StreamInfo {
		if info.Get("codec_type").String() == "audio" {
			scoreboard[index] = 0
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

	// check the selection_priority query
	priority := meta.Config.Get("audio.selection_priority")
	if !priority.Exists() {
		return 0
	}

	// check the key equivalence of two query
	prefer_keys := util.FlattenJSONKey(priority)
	priority_keys := util.FlattenJSONKey(priority)
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
