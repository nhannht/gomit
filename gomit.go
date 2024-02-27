package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/atotto/clipboard"
	"github.com/nhannht/gomit/function"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
)

var (
	ConfigPath string
	Config     map[string]string
	help       = flag.Bool("h", false, "Show this help")
	config     = flag.String("config", "", "Set a config variable")
	value      = flag.String("value", "", "Set a config value")
	commit     = flag.Bool("commit", false, "Generate commit message of current file")
	gen        = flag.Bool("gen", false, "Generate commit message of current file")
	staged     = flag.Bool("staged", false, "Just generate for staged file")
	helpMsg    = `Usage:
gomit -config [config_variable] -value [value] - Set a config variable
gomit -gen - Generate commit message of current stage of git project in current dir
gomit -gen -stage - Just generate for staged file
gomit -commit - Stage all files, open editor, copy the generated msg to clipboard
gomit -h - Show this help

Config variables:
OPENAI_KEY - OpenAI API key
OPENAI_URL - OpenAI API URL`
	result             Response
	GlobalConversation []Message
	scanner            = bufio.NewScanner(os.Stdin)
)

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

func SendMessageToGPT(messages []Message) Response {
	data := Data{
		Model:    "gpt-3.5-turbo",
		Messages: messages,
	}

	//messagesJson, _ := json.MarshalIndent(messages, "", "  ")
	//fmt.Println(string(messagesJson))

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

	//clipboard.Write(clipboard.FmtText, []byte(result.Choices[0].Message.Content))
	var message Message
	message.Role = "assistant"
	message.Content = result.Choices[0].Message.Content
	GlobalConversation = append(GlobalConversation, message)

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
	if *help {
		usage()
	}

	flag.Usage = usage
	flag.Parse()

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
	var initMessage Message
	initMessage.Role = "system"

	if !*staged {
		initMessage.Content = "The input from user are the description of all the change in current git project" +
			"You behave like commit generator, only give the commit result, nothing else\n" +
			"Your response always include necessary information, nothing else, no need to give any personal response\n" +
			"Notice that the commit message should be short, in form of fix|feat|refactor|style|test|docs|chore: short description. " +
			"After 2 new lines add long description with multi lines"

		GlobalConversation = append(GlobalConversation, initMessage)

	} else {
		initMessage.Content = "The input from user are the description of all the change in current git project, with staged file" +
			"You behave like commit generator, only give the commit result, nothing else\n" +
			"Notice that the commit message should be short, in form of fix|feat|refactor|style|test|docs|chore: short description. " +
			"After 2 new lines add long description with multi lines" +
			"Your response always include necessary information, nothing else, no need to give any personal response\n" +
			"You don't need to include the diff content in the commit."
		GlobalConversation = append(GlobalConversation, initMessage)
	}

	var diff []byte
	var diffMessage Message
	diffMessage.Role = "user"

	if !*staged {

		diff, err = exec.Command("git", "--no-pager", "diff", "--minimal", "--no-color").Output()

		if err != nil {
			log.Panic(err)
		}

	}

	if *staged {

		diff, err = exec.Command("git", "--no-pager", "diff", "--minimal", "--no-color", "--staged").Output()
		if err != nil {
			log.Panic(err)
		}

	}
	diffFiles := function.ParseDiff(string(diff))
	diffMessageParse := function.TokenizeFileDiffToSuitableString(diffFiles, initMessage.Content)
	diffMessage.Content = diffMessageParse
	GlobalConversation = append(GlobalConversation, diffMessage)

	if len(diff) == 0 {
		log.Println("No changes to commit")
		return
	}

	if *gen {

		var result Response
		for {
			result = SendMessageToGPT(GlobalConversation)
			fmt.Println(result.Choices[0].Message.Content)
			var accept string
			fmt.Println("Accept this commit message? (y/n)")
			_, err = fmt.Scanln(&accept)
			if err != nil {
				log.Panic(err)
			}
			if accept == "y" {
				clipboard.WriteAll(result.Choices[0].Message.Content)
				fmt.Println("The commit message has been copied to clipboard")
				os.Exit(1)
			} else {

				var previousMessage Message
				previousMessage.Role = "assistant"
				previousMessage.Content = result.Choices[0].Message.Content
				GlobalConversation = append(GlobalConversation, previousMessage)

				fmt.Println("Please tell the bot what you want to be edit? Example: Please add the change of file README.md, etc...")

				var edit string
				for scanner.Scan() {
					edit = scanner.Text()
					break
				}
				var systemEditMessage Message
				systemEditMessage.Role = "system"
				systemEditMessage.Content = `User want to edit the message, he will put what he want in the following message, then the bot will generate the message again,
	The process will be repeated until the user accept the message`
				GlobalConversation = append(GlobalConversation, systemEditMessage)

				var editMessage Message
				editMessage.Role = "user"
				editMessage.Content = edit
				GlobalConversation = append(GlobalConversation, editMessage)

			}
		}

		os.Exit(1)
	}

	// copy the result to clipboard, run git commit and paste the result to editor
	if *commit {
		result = SendMessageToGPT(GlobalConversation)

		_, err = exec.Command("git", "add", ".").Output()
		if err != nil {
			log.Panic(err)

		}
		_, err = exec.Command("git", "commit").Output()
		if err != nil {
			log.Panic(err)
		}
		// paste the result to editor
		b, err := clipboard.ReadAll()
		if err != nil {
			log.Panic(err)
		}
		fmt.Println(string(b))

		os.Exit(1)

	}

}
