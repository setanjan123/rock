# Rock

Rock is a small command-line coding agent written in Go. It was built as a learning project to explore how an LLM agent works internally: conversation history, tool definitions, tool calls, local execution, and the agent loop.

## What it can do

- Maintain an in-memory conversation with an LLM.
- Communicate with an OpenAI-compatible chat-completions endpoint.
- Load an optional system prompt from `system.md`.
- Discover the current directory.
- List directory contents.
- Read files.
- Create or overwrite files.
- Execute shell commands.
- Execute multiple requested tool calls concurrently while preserving their original order.
- Automatically compact conversation history when approaching the context limit (set via `CONTEXT_LIMIT` in `.env`).

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

Rock tracks estimated token usage with a chars/4 heuristic. When usage exceeds 80% of the configured limit (`CONTEXT_LIMIT` in `.env`, default 32k), it asks the LLM to summarize older messages and replaces them with the summary — keeping the system prompt and the last 10 messages intact. Compaction never interrupts an in-progress tool-call chain.

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

Create a `.env` file in the project directory.

```dotenv
API_KEY=your-api-key
API_BASE_URL=your-openai-compatible-chat-completions-url
MODEL=your-model-name
CONTEXT_LIMIT=32768
```

For a local Ollama setup, the URL is commonly similar to:

```dotenv
API_BASE_URL=http://localhost:11434/v1/chat/completions
```

Install dependencies and run Rock:

```bash
go mod tidy
go run .
```

Use `/exit` to leave the conversation.

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
| `util.go` | System-prompt loading and response handling. |
| `system.md` | Optional system instructions for the model. |

## Current scope

Rock is intentionally small and educational. Conversation history exists only in memory, and the project does not yet include persistent sessions or permission prompts.

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
