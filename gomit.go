package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"runtime"
)

var ConfigPath string
var Config map[string]string

type Data struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type Response struct {
	Choices []Choice `json:"choices"`
}
type Choice struct {
	Message Message `json:"message"`
}

func Contain(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

func main() {
	args := os.Args
	help := `Usage:
gomit config [config_variable] [value] - Set a config variable
gomit - Generate commit message of curent file
gomit -h - Show this help
environment variables:
OPENAI_KEY - OpenAI API key
OPENAI_URL - OpenAI API URL
`

	if Contain(args, "-h") {
		log.Println(help)
		os.Exit(1)

	}

	goos := runtime.GOOS
	if goos == "windows" {
		log.Panic("Windows is not supported")

	}
	if goos == "darwin" {
		log.Panic("MacOS is not supported")

	}
	if goos == "linux" {

		ConfigPath = path.Join(os.Getenv("HOME"), "/.config/gomit.json")
		if len(args) == 4 && args[1] == "config" {

			// Write to ConfigPath
			configString, err := os.ReadFile(ConfigPath)
			if err != nil {
				log.Println("Config file not found, creating one")
				config := make(map[string]string)
				config[args[2]] = args[3]
				configJson, _ := json.Marshal(config)
				err = os.WriteFile(ConfigPath, configJson, 0644)
				if err != nil {
					log.Panic(err)
				}
				fmt.Printf("Set %s : %s to %s\n", args[2], args[3], ConfigPath)
				os.Exit(1)

			}

			err = json.Unmarshal(configString, &Config)
			if err != nil {
				log.Panic(err)
			}
			Config[args[2]] = args[3]
			configJson, _ := json.Marshal(Config)
			err = os.WriteFile(ConfigPath, configJson, 0644)
			if err != nil {
				log.Panic(err)

			}
			newConfigString, err := os.ReadFile(ConfigPath)
			if err != nil {
				log.Panic(err)
			}

			fmt.Printf("Current config is %s\n", newConfigString)
			os.Exit(1)
		}

		configString, err := os.ReadFile(ConfigPath)
		if err != nil {
			log.Panic(err)
		}
		err = json.Unmarshal(configString, &Config)

		_, err = exec.LookPath("git")
		if err != nil {
			log.Panic(err)
		}

		gitExist, err := os.Stat(".git")
		if err != nil {
			log.Panic(err)
		}
		if gitExist == nil {
			log.Panic("No .git directory found")
		}

		diff, err := exec.Command("git", "--no-pager", "diff").Output()
		if err != nil {
			log.Panic(err)
		}
		if len(diff) == 0 {
			log.Println("No changes to commit")
			return
		}
		data := Data{
			Model: "gpt-3.5-turbo",
			Messages: []Message{
				{
					Role: "system",
					Content: "The input from user is the result of git diff" +
						"You behave like commit generator, only give the commit result, nothing else\n" +
						"Your response always include necessary information, nothing else, no need to give any personal response\n" +
						"Notice that the commit message should be short, in form of fix|feat|refactor|style|test|docs|chore: short description. " +
						"After 2 new lines add long description with multi lines",
				},
				{
					Role:    "user",
					Content: string(diff),
				},
			},
		}
		body, err := json.Marshal(data)

		req, err := http.NewRequest("POST",
			Config["OPENAI_URL"],
			bytes.NewBuffer(body),
		)

		if err != nil {
			log.Panic(err)
		}
		req.Header.Add("Authorization", "Bearer "+Config["OPENAI_KEY"])
		req.Header.Add("Content-Type", "application/json")
		client := &http.Client{}
		response, err := client.Do(req)
		if err != nil {
			log.Panic(err)
		}
		defer response.Body.Close()
		var result Response
		err = json.NewDecoder(response.Body).Decode(&result)
		if err != nil {
			log.Panic(err)
		}
		//fmt.Println(result)
		fmt.Println(result.Choices[0].Message.Content)
	}

}
