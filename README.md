# Rock

Rock is a small command-line coding agent written in Go. It was built as a learning project to explore how an LLM agent works internally: conversation history, tool definitions, tool calls, local execution, and the agent loop.

## What it can do

- Maintain a conversation with an LLM and persist it to SQLite.
- Save and resume conversations across sessions.
- Communicate with an OpenAI-compatible chat-completions endpoint.
- Load an optional system prompt from `system.md`.
- Discover the current directory.
- List directory contents.
- Read files.
- Create or overwrite files.
- Execute shell commands.
- Execute multiple requested tool calls concurrently while preserving their original order.
- Automatically compact conversation history when approaching the context limit (set via `CONTEXT_LIMIT`).

## How it works

```text
User input
    ↓
LLM request with conversation history and tool definitions
    ↓
Assistant response
    ├── normal text → display it
    └── tool calls  → execute tools → add results to history
                                      ↓
                                  ask the LLM again
```

The project uses the OpenAI-compatible message and tool-calling format, so it can work with a local or remote compatible backend.

### Context compaction

Rock tracks estimated token usage with a chars/4 heuristic. When usage exceeds 80% of the configured limit (`CONTEXT_LIMIT`, default 32k), it asks the LLM to summarize older messages and replaces them with the summary — keeping the system prompt and the last 10 messages intact. Compaction never interrupts an in-progress tool-call chain.

```text
... conversation continuing ...
    ↓
token estimate > 80% of CONTEXT_LIMIT?
    ↓ YES
summarize older history via LLM (no tools)
    ↓
replace with: [system prompt] [summary] [last 10 messages]
    ↓
... conversation continues with compacted history ...
```

## Setup

Configure the following environment variables. They can be set directly in your shell or placed in an optional `.env` file in the project directory. Exported variables take precedence over the `.env` file. The three required variables must be set or Rock exits with an error.

```dotenv
# Required
API_KEY=your-api-key
API_BASE_URL=your-openai-compatible-base-url
MODEL=your-model-name

# Optional
CONTEXT_LIMIT=32768
VERBOSE_TOOLS=1
```

`API_BASE_URL` is the base URL only, without a trailing slash and without the
API path. Rock appends `/v1/chat/completions` and `/v1/models` dynamically.

For a local Ollama setup, the URL is commonly similar to:

```dotenv
API_BASE_URL=http://localhost:11434
```

Install dependencies, build the binary, and run it:

```bash
go mod tidy
go build -o rock .
./rock
```

On Windows

```powershell
go build -o rock.exe .
.\rock.exe
```

### Sessions

Rock saves every completed turn to `rock.db` in the current directory. Each
conversation gets a short generated id that is printed on startup and on exit.

- Start a new conversation: `./rock` (or `.\rock.exe` on Windows)
- Resume a conversation: `./rock <id>` (the id is shown when starting and exiting)
- List past conversations during a session: `/history`
- List and switch models during a session: `/model` (or `/model <model-id>`)
- Leave and save: `/exit`

Set `VERBOSE_TOOLS=1` to print tool names, arguments, and results as they run
(and when replaying a resumed conversation). By default tool calls and results
are executed silently.

The system prompt is captured when a conversation starts. On resume, Rock loads
the full stored history instead of reading `system.md` again, so the resumed
conversation is an exact continuation.

The active model is also saved per conversation. New conversations start from
the `MODEL` environment variable; switching models with `/model` updates only
the current conversation, and resuming that conversation restores its chosen
model. Model listing requires the backend to expose `GET {base}/models` (Ollama
and most OpenAI-compatible servers do).

## Available tools

| Tool | Purpose |
| --- | --- |
| `current_directory` | Returns the directory where Rock is running. |
| `list_directory` | Lists the entries in a directory. |
| `read_file` | Returns the contents of a file. |
| `write_file` | Creates or overwrites a file. |
| `delete_file` | Deletes a file |
| `exec_command` | Executes a command with optional arguments. |

## Project structure

| File | Purpose |
| --- | --- |
| `main.go` | Application startup, user loop, and LLM communication. |
| `tools.go` | Tool definitions, dispatch, concurrency, and filesystem operations. |
| `compaction.go` | Token estimation, context compaction trigger, and LLM summarization. |
| `types.go` | Request, response, tool-call, and argument structs. |
| `storage.go` | SQLite persistence, conversation listing, and id generation. |
| `util.go` | System-prompt loading and response handling. |
| `system.md` | Optional system instructions for the model. |

## Current scope

Rock is intentionally small and educational. Conversations persist to SQLite, but
the project does not yet include permission prompts or full message-level browsing.

## A model-generated self-summary

I used Gemma 4 E4B running via Ollama and used Rock itself to generate its own summary:

~~~text
| Rock v0.01 |

────────────────────────────────────────
You › Summarise the project in the current directory.
────────────────────────────────────────
Agent is thinking...

Agent ›
This document provides a comprehensive summary of the "Rock" project repository, detailing its purpose, architecture, and the specific role of every significant file.

***

## ⚙️ Project Overview: Rock CLI AI Agent

**Purpose:**
The repository contains "Rock," a highly functional Command Line Interface (CLI) program designed to operate as an AI agent. Its primary goal is educational—to build a working proof-of-concept demonstrating how a conversational LLM model can be augmented with local system capabilities (tools) via structured API calls.

**Core Functionality:**
The agent runs in a continuous loop, maintaining conversation history. On each turn, it sends the context and definitions of its available **Tools** to an external AI API. The API responds either with a text answer or a request to execute one or more functions (tools). The `main.go` then catches this request, executes the local shell commands defined in `tools.go`, passes the results back to the LLM for final context integration, and continues the conversation.

***
~~~
