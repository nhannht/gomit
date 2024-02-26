package main

import (
	"fmt"
	"github.com/nhannht/gomit/function"
	"log"
	"os/exec"
)

func main() {
	diff, err := exec.Command("git", "--no-pager", "diff", "--minimal", "--no-color", "--staged").Output()
	if err != nil {
		log.Panic(err)
	}

	filesDiff := function.ParseDiff(string(diff))
	//for _, file := range filesDiff.Files {
	//	fmt.Println(file.FileName)
	//}
	fmt.Println(filesDiff.Files[1].Hunks)

}
