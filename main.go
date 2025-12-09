package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
)

var debugMode bool

func main() {
	flag.BoolVar(&debugMode, "debug", false, "Enable debug output (show tool args and results)")
	flag.Parse()

	// Load ignore patterns
	LoadIgnorePatterns()

	// Load configuration
	cfg, err := LoadConfig()
	if err != nil {
		PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	// Validate configuration
	if cfg.APIKey == "" {
		PrintError("No API key found. Set OPENAI_API_KEY environment variable or add to config file.")
		fmt.Println("\nConfig file location: ~/.config/codequery/config.json")
		fmt.Println("Example config:")
		fmt.Println(`  {"api_key": "sk-...", "model": "gpt-4o"}`)
		os.Exit(1)
	}

	// Create client
	client := NewClient(cfg)

	// Print welcome
	PrintWelcome(cfg.Model, extractHost(cfg.BaseURL))

	// Setup readline
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "> ",
		HistoryFile:     getHistoryFile(),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		PrintError(fmt.Sprintf("Failed to initialize readline: %v", err))
		os.Exit(1)
	}
	defer rl.Close()

	spinner := NewSpinner()

	// REPL loop
	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				continue
			}
			if err == io.EOF {
				fmt.Println("\nGoodbye!")
				break
			}
			PrintError(fmt.Sprintf("readline error: %v", err))
			break
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		// Handle special commands
		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			break
		}
		if input == "clear" || input == "reset" {
			client.Reset()
			fmt.Println("Conversation cleared.")
			continue
		}
		if input == "help" {
			printHelp()
			continue
		}

		// Send to LLM
		if !debugMode {
			spinner.Start("Thinking...")
		}

		response, err := client.Chat(input, func(name, argsJSON, result string) {
			spinner.Stop()
			PrintTool(name, FormatToolCall(name, argsJSON))
			if debugMode {
				PrintDebugJSON("args", argsJSON)
				PrintDebug("result", result)
			}
			if !debugMode {
				spinner.Start("Thinking...")
			}
		})

		spinner.Stop()

		if err != nil {
			PrintError(err.Error())
			continue
		}

		fmt.Println()
		fmt.Println(response)
		fmt.Println()
	}
}

func extractHost(url string) string {
	// Extract host from URL for display
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	if idx := strings.Index(url, "/"); idx != -1 {
		url = url[:idx]
	}
	return url
}

func getHistoryFile() string {
	home, _ := os.UserHomeDir()
	return home + "/.codequery_history"
}

func printHelp() {
	fmt.Println(`
Commands:
  exit, quit  - Exit the program
  clear, reset - Clear conversation history
  help        - Show this help message

Flags:
  -debug      - Show tool arguments and results

Environment variables:
  OPENAI_API_KEY    - Your API key (required)
  OPENAI_BASE_URL   - API endpoint (default: https://api.openai.com/v1)
  CODEQUERY_MODEL   - Model to use (default: gpt-4o)

Config file: ~/.config/codequery/config.json
`)
}
