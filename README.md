# CodeQuery

A minimal CLI tool that lets you ask questions about codebases using LLMs.

CodeQuery gives an LLM access to file system tools (`ls`, `cat`, `head`, `grep`, `find`, `tree`) so it can explore your code and answer questions accurately.

## Installation

### From Source (Recommended)

```bash
git clone https://github.com/user/codequery.git
cd codequery
go build -o codequery
sudo mv codequery /usr/local/bin/
```

### Pre-built Binary

Download the relevant binary from github releases

## Configuration

### API Key

Set your OpenAI API key (or compatible API key):

```bash
export OPENAI_API_KEY="sk-..."
```

Or create a config file at `~/.config/codequery/config.json`:

```json
{
  "base_url": "https://example-provider.ai/api/v1",
  "api_key": "sk-...",
  "model": "gpt-4o"
}
```

### Using with Other Providers

CodeQuery works with any OpenAI-compatible API:

```bash
# Ollama
export OPENAI_BASE_URL="http://localhost:11434/v1"
export CODEQUERY_MODEL="llama3.2"
export OPENAI_API_KEY="ollama"

# vLLM
export OPENAI_BASE_URL="http://localhost:8000/v1"
export CODEQUERY_MODEL="meta-llama/Llama-3-8b-chat-hf"
export OPENAI_API_KEY="token"

# OpenRouter
export OPENAI_BASE_URL="https://openrouter.ai/api/v1"
export OPENAI_API_KEY="sk-or-..."
export CODEQUERY_MODEL="anthropic/claude-3.5-sonnet"
```

## Usage

Navigate to any codebase and run:

```bash
cd /path/to/your/project
codequery
```

Then ask questions:

```
> Where is authentication handled?

[tool] tree -L 2 .
[tool] find "*auth*" .
[tool] cat src/auth/middleware.go

Authentication is handled in src/auth/middleware.go. The AuthMiddleware
function validates JWT tokens from the Authorization header...

> What about the login endpoint?

[tool] grep -r "login" .
[tool] cat src/handlers/auth.go

The login endpoint is at POST /api/login, defined in src/handlers/auth.go...
```

### Commands

- `exit` / `quit` - Exit the program
- `clear` / `reset` - Clear conversation history
- `help` - Show help

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OPENAI_API_KEY` | API key (required) | - |
| `OPENAI_BASE_URL` | API endpoint | `https://api.openai.com/v1` |
| `CODEQUERY_MODEL` | Model to use | `gpt-4o` |

## Available Tools

The LLM has access to these tools:

| Tool | Description |
|------|-------------|
| `ls` | List directory contents |
| `cat` | Read entire file |
| `head` | Read first N lines |
| `grep` | Search for patterns |
| `find` | Find files by name |
| `tree` | Show directory structure |

## License

APACHE
