package util

import (
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"runtime"
	"strconv"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func ReadJSONFile(fp string) (gjson.Result, error) {
	b, e := ioutil.ReadFile(fp)
	if e != nil {
		return gjson.Result{}, e
	}
	return gjson.ParseBytes(b), nil
}

func Map2JSON(input map[string]string) string {
	output := ""
	for k, v := range input {
		output, _ = sjson.Set(output, k, v)
	}

	return output
}

// only flat json works
func JSON2Map(input string) map[string]string {
	output := map[string]string{}
	for k, v := range gjson.Parse(input).Map() {
		output[k] = v.String()
	}
	return output
}

func FlattenJSONKey(json gjson.Result) map[string]bool {
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

func MatchRegexPCRE2(regex string, target string) bool {
	re, e := regexp2.Compile(regex, 0)
	if e != nil {
		logrus.WithFields(
			logrus.Fields{
				"regex_regex":     regex,
				"regex_candidate": target,
				"error":           e,
				"where":           GetCurrentFunctionInfo(),
			}).Fatalf("Failed to compile Regex")
	}

	matched, e := re.MatchString(target)
	if e != nil {
		logrus.WithFields(
			logrus.Fields{
				"regex_regex":     regex,
				"regex_candidate": target,
				"error":           e,
				"where":           GetCurrentFunctionInfo(),
			}).Fatalf("Faled to match Regex")
	}

	return matched
}

func HashFNV64a(text string) string {
	algorithm := fnv.New64a()
	algorithm.Write([]byte(text))
	return strconv.FormatUint(algorithm.Sum64(), 10)
}

func Atof(value float64) string {
	return strconv.FormatFloat(value, 'f', 3, 64)
}

func GetCurrentFunctionInfo() string {
	pc := make([]uintptr, 15)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	return fmt.Sprintf("%s:%s:%d", frame.File, frame.Function, frame.Line)
}
