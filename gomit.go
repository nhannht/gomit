package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"golang.design/x/clipboard"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
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

func SetupConfig(name string, value string) {

	// Write to ConfigPath
	configString, err := os.ReadFile(ConfigPath)
	if err != nil {
		log.Println("Config file not found, creating one")
		config := make(map[string]string)
		config[name] = value
		configJson, _ := json.Marshal(config)
		err = os.WriteFile(ConfigPath, configJson, 0644)
		if err != nil {
			log.Panic(err)
		}
		fmt.Printf("Set %s : %s to %s\n", name, value, ConfigPath)
		os.Exit(1)

	}

	err = json.Unmarshal(configString, &Config)
	if err != nil {
		log.Panic(err)
	}
	Config[name] = value
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

func SendDiffToGPT(diff []byte) Response {
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

	if *staged {
		data.Messages[0].Content = "The input from user is the result of git diff --staged, that mean this file already staged. \n" +
			"You behave like commit generator, only give the commit result, nothing else\n" +
			"Your response always include necessary information, nothing else, no need to give any personal response\n" +
			"Notice that the commit message should be short, in form of fix|feat|refactor|style|test|docs|chore: short description. " +
			"After 2 new lines add long description with multi lines"
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

	return result
}

func ReadConfig() {
	// read config
	configString, err := os.ReadFile(ConfigPath)
	if err != nil {
		log.Panic(err)
	}
	err = json.Unmarshal(configString, &Config)
	if err != nil {
		log.Panic(err)
	}

}

func HandleNonGitExist() {
	_, err := exec.LookPath("git")
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
}

var (
	help    = flag.Bool("h", false, "Show this help")
	config  = flag.String("config", "", "Set a config variable")
	value   = flag.String("value", "", "Set a config value")
	commit  = flag.Bool("commit", false, "Generate commit message of current file")
	gen     = flag.Bool("gen", false, "Generate commit message of current file")
	staged  = flag.Bool("stage", false, "Just generate for staged file")
	helpMsg = `Usage:
gomit -config [config_variable] -value [value] - Set a config variable
gomit -gen - Generate commit message of current stage of git project in current dir
gomit -gen -stage - Just generate for staged file
gomit -commit - Stage all files, open editor, copy the generated msg to clipboard
gomit -h - Show this help

Config variables:
OPENAI_KEY - OpenAI API key
OPENAI_URL - OpenAI API URL`
)

func usage() {
	flag.PrintDefaults()
	_, err := fmt.Fprintf(os.Stderr, helpMsg)
	if err != nil {
		return
	}
	os.Exit(1)

}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	flag.Usage = usage
	flag.Parse()
	if *help {
		fmt.Println(helpMsg)
		os.Exit(1)

	}

	err := clipboard.Init()
	if err != nil {
		log.Panic(err)
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Panic(err)
	}

	ConfigPath = path.Join(configDir, "gomit.json")

	if *config != "" && *value != "" {
		SetupConfig(*config, *value)
	}

	ReadConfig()

	// handle non git exist
	HandleNonGitExist()

	diff, err := exec.Command("git", "--no-pager", "diff").Output()

	if err != nil {
		log.Panic(err)
	}
	if *staged {
		diff, err = exec.Command("git", "--no-pager", "diff", "--staged").Output()
		if err != nil {
			log.Panic(err)
		}

	}
	if len(diff) == 0 {
		log.Println("No changes to commit")
		return
	}
	result := SendDiffToGPT(diff)

	if *gen {
		fmt.Println(result.Choices[0].Message.Content)
		os.Exit(1)
	}
	clipboard.Write(clipboard.FmtText, []byte(result.Choices[0].Message.Content))

	// copy the result to clipboard, run git commit and paste the result to editor
	if *commit {
		_, err = exec.Command("git", "add", ".").Output()
		if err != nil {
			log.Panic(err)

		}
		_, err = exec.Command("git", "commit").Output()
		if err != nil {
			log.Panic(err)
		}
		// paste the result to editor
		b := clipboard.Read(clipboard.FmtText)
		for len(b) > 0 {
			n, err := os.Stdout.Write(b)
			if err != nil {
				log.Panic(err)

			}
			b = b[n:]
		}

		os.Exit(1)

	}
	usage()

}
