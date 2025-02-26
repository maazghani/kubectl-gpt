package main

import (
    "bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
    "strconv"
	"strings"

	gpt "github.com/maazghani/kubectl-gpt/pkg/gpt"
)

const (
	DefaultMaxTokens   = 300
	DefaultTemperature = 0.2
	DefaultModel       = "gpt-3.5-turbo"
	systemMessage      = "Translate the given text to a kubectl command. Show only generated kubectl command without any description, code block."
)

var (
	version        string
	autoApproveFlag = flag.Bool("auto-approve", false, "Execute the generated command without asking for confirmation")
	helpFlag        = flag.Bool("help", false, "Show usage information")
	versionFlag     = flag.Bool("version", false, "Show the version")
)

func main() {
	// Parse flags
	flag.Parse()

	// Handle help and version flags
	if *helpFlag {
		printHelp()
		os.Exit(0)
	}

	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}

	apiKey, model, temperature, maxTokens := getOpenAIConfigFromEnv()
	if apiKey == "" {
		fmt.Println("Please set the environment variable: \"OPENAI_API_KEY\".")
		fmt.Println("You can add the following line to your .zshrc or .bashrc file:")
		fmt.Println("export OPENAI_API_KEY=<your-key>")
		fmt.Println()
		fmt.Println("If you don't have an OpenAI API Key, you can get one at this link: https://platform.openai.com/account/api-keys.")
		os.Exit(1)
	}

	query := strings.Join(flag.Args(), " ")
	query = strings.TrimSpace(query)
	if query == "" {
		fmt.Println("Please input a query.")
		fmt.Println("Usage: kubectl-gpt [OPTIONS] QUERY")
		os.Exit(1)
	}
	request := gpt.NewOpenAIRequest(model, temperature, maxTokens, systemMessage, query)

	response, err := gpt.RequestChatGptAPI(request, apiKey)
	if err != nil {
		fmt.Printf("Failed to call OpenAI API: %v\n", err)
		os.Exit(1)
	}

	kubectlCommand := extractCommand(response)
	fmt.Printf("\033[1;34m[Generated Command]\033[0m\n%s\n", kubectlCommand)

	if !*autoApproveFlag {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("\033[1;34m⎈ Press enter to execute the command or type 'abort' to cancel.\033[0m: ")
		text, _ := reader.ReadString('\n')
		text = strings.Replace(text, "\n", "", -1)

		if strings.ToLower(text) == "abort" {
			fmt.Println("Command execution aborted.")
			os.Exit(0)
		}
	}

	cmd := exec.Command("/bin/sh", "-c", kubectlCommand)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// getEnvironmentVariables retrieves OpenAI related variables from environment
func getOpenAIConfigFromEnv() (apiKey string, model string, temperature float64, maxTokens int) {
	apiKey = os.Getenv("OPENAI_API_KEY")

	model = os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = DefaultModel
	}

	temperatureStr := os.Getenv("OPENAI_TEMPERATURE")
	temperature = DefaultTemperature
	if temperatureStr != "" {
		var err error
		temperature, err = strconv.ParseFloat(temperatureStr, 64)
		if err != nil {
			fmt.Println("Failed to parse OPENAI_TEMPERATURE. Using default temperature.")
			temperature = DefaultTemperature
		}
	}

	maxTokensStr := os.Getenv("OPENAI_MAX_TOKENS")
	maxTokens = DefaultMaxTokens
	if maxTokensStr != "" {
		var err error
		maxTokens, err = strconv.Atoi(maxTokensStr)
		if err != nil {
			fmt.Println("Failed to parse OPENAI_MAX_TOKENS. Using default max tokens.")
			maxTokens = DefaultMaxTokens
		}
	}

	return apiKey, model, temperature, maxTokens
}

func printHelp() {
	fmt.Println("Usage: kubectl-gpt [OPTIONS] QUERY")
	fmt.Println("Translate the given query to a kubectl command using OpenAI GPT API.")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --auto-approve   Execute the generated command without asking for confirmation")
	fmt.Println("  --help           Show this message and exit")
	fmt.Println("  --version        Show the version")
	fmt.Println()
	fmt.Println("Environment variables:")
	fmt.Println("  OPENAI_API_KEY        OpenAI API Key")
	fmt.Println("  OPENAI_MODEL          OpenAI Model to use (default is gpt-3.5-turbo)")
	fmt.Println("  OPENAI_TEMPERATURE    Temperature for the OpenAI request (default is 0.2)")
	fmt.Println("  OPENAI_MAX_TOKENS     Max tokens for the OpenAI request (default is 300)")
	fmt.Println("  KUBEGPT_MODE          Set to 'auto' to automatically approve generated commands without prompting")
}

func extractCommand(response gpt.OpenAIResponse) string {
	s := strings.Trim(response.Choices[0].Message.Content, "`")
	return strings.TrimSpace(s)
}
