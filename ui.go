package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

var Version = "dev"

var (
	toolColor    = color.New(color.FgCyan, color.Faint)
	errorColor   = color.New(color.FgRed)
	successColor = color.New(color.FgGreen)
	dimColor     = color.New(color.Faint)
)

func PrintTool(name string, args string) {
	toolColor.Printf("[tool] %s %s\n", name, args)
}

func PrintDebug(label string, content string) {
	dimColor.Printf("  [%s] ", label)
	// Truncate long output
	if len(content) > 500 {
		content = content[:500] + "... (truncated)"
	}
	// Replace newlines for compact display
	content = strings.ReplaceAll(content, "\n", "\\n")
	dimColor.Printf("%s\n", content)
}

func PrintDebugJSON(label string, content string) {
	dimColor.Printf("  [%s] %s\n", label, content)
}

func PrintError(msg string) {
	errorColor.Printf("Error: %s\n", msg)
}

func PrintWelcome(model, baseURL string) {
	fmt.Println()
	successColor.Printf("CodeQuery %s\n", Version)
	dimColor.Printf("Model: %s | Provider: %s\n", model, baseURL)
	fmt.Println()
}

// Spinner provides a simple animated spinner
type Spinner struct {
	frames  []string
	stop    chan struct{}
	stopped chan struct{}
	mu      sync.Mutex
	running bool
}

func NewSpinner() *Spinner {
	return &Spinner{
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

func (s *Spinner) Start(msg string) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stop = make(chan struct{})
	s.stopped = make(chan struct{})
	s.mu.Unlock()

	go func() {
		defer close(s.stopped)
		i := 0
		for {
			select {
			case <-s.stop:
				fmt.Print("\r\033[K") // Clear line
				return
			default:
				dimColor.Printf("\r%s %s", s.frames[i%len(s.frames)], msg)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stop)
	<-s.stopped
}
