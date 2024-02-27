package function

import (
	"fmt"
	"github.com/pkoukk/tiktoken-go"
)

func TokenizeFileDiffToSuitableString(filesDiff FilesDiff, systemInstructMessage string) string {
	encoding := "gpt-3.5-turbo"
	tkm, err := tiktoken.EncodingForModel(encoding)
	if err != nil {
		return err.Error()
	}
	maxTokenNum := 4096

	numTokenOfInstruction := len(tkm.Encode(systemInstructMessage, nil, nil))
	remainTokenNum := maxTokenNum - numTokenOfInstruction
	resultMessage := ""
	for _, fileDiff := range filesDiff.Files {
		for _, hunk := range fileDiff.Hunks {
			formatHunk := fmt.Sprintf("```File: %s\nDiff\n%s\n```", fileDiff.FileName, hunk.Content)
			tokenizeFormatHunk := tkm.Encode(formatHunk, nil, nil)
			if remainTokenNum > len(tokenizeFormatHunk) {
				resultMessage += formatHunk
				remainTokenNum -= len(tokenizeFormatHunk)

			} else {
				remainTokenNumAfterExcludeFile := remainTokenNum - len(tkm.Encode(fmt.Sprintf("```File: %s\nDiff\n", fileDiff.FileName), nil, nil))
				if remainTokenNumAfterExcludeFile < 0 {

				} else {
					hunkTokenize := tkm.Encode(hunk.Content, nil, nil)
					lastHunkPart := tkm.Decode(hunkTokenize[:remainTokenNumAfterExcludeFile])
					resultMessage += fmt.Sprintf("```File: %s\nDiff\n%s\n```", fileDiff.FileName, lastHunkPart)
				}

			}
		}
	}
	return resultMessage
}
