package function

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"
)

type FilesDiff struct {
	Files []FileDiff
}

type FileDiff struct {
	FileName string
	Hunks    []Hunk
}
type Hunk struct {
	FileName     string
	OldStartLine int
	OldLines     int
	NewStartLine int
	NewLines     int
	Content      string
}

var (
	fileDelimiterRegex, _ = regexp.Compile(`diff --git a/(.+) b/(.+)`)
	hunkDelimiterRegex, _ = regexp.Compile(`@@ -(\d+),(\d+) \+(\d+),(\d+) @@`)
	ignoreLineRegex, _    = regexp.Compile(`^(\-\-\-|\+\+\+|index|diff|new file|deleted file|similarity|rename|copy|old mode|new mode|deleted mode)`)
)

func ParseDiff(diff string) FilesDiff {
	// Read lines by lines, split file base on regex `diff --git a/(.+) b/(.+)`
	// and then split the content by `@@ -(\d+),(\d+) \+(\d+),(\d+) @@`
	scanner := bufio.NewScanner(strings.NewReader(diff))
	scanner.Split(bufio.ScanLines)
	var filesDiff FilesDiff
	var fileDiff FileDiff

	for scanner.Scan() {

		line := scanner.Text()
		//fmt.Println(line)
		if fileDelimiterRegex.MatchString(line) {
			fileName := fileDelimiterRegex.FindStringSubmatch(line)[2]
			if fileDiff.FileName != "" {
				filesDiff.Files = append(filesDiff.Files, fileDiff)

			}
			fileDiff.FileName = fileName

		} else if hunkDelimiterRegex.MatchString(line) {
			hunk := Hunk{}
			hunk.FileName = fileDiff.FileName
			hunk.OldStartLine, _ = strconv.Atoi(hunkDelimiterRegex.FindStringSubmatch(line)[1])
			hunk.OldLines, _ = strconv.Atoi(hunkDelimiterRegex.FindStringSubmatch(line)[2])
			hunk.NewStartLine, _ = strconv.Atoi(hunkDelimiterRegex.FindStringSubmatch(line)[3])
			hunk.NewLines, _ = strconv.Atoi(hunkDelimiterRegex.FindStringSubmatch(line)[4])

			fileDiff.Hunks = append(fileDiff.Hunks, hunk)

		} else if ignoreLineRegex.MatchString(line) {
			continue
		} else {
			if len(fileDiff.Hunks) > 0 {
				fileDiff.Hunks[len(fileDiff.Hunks)-1].Content += line + "\n"
			} else {
				fileDiff.Hunks[0].Content += line + "\n"
			}
		}
	}
	filesDiff.Files = append(filesDiff.Files, fileDiff)
	return filesDiff

}
