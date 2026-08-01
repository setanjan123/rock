| Rock v0.01 |

────────────────────────────────────────
You › Hello
────────────────────────────────────────
Agent is thinking...

Agent ›
Hello! How can I help you today?

────────────────────────────────────────
You › I want you to read the contents of all the files in this directory and then summarise it
────────────────────────────────────────
Agent is thinking...

Agent ›
Based on the contents of the files, here is a comprehensive summary of the project.

### Project Summary: Rock CLI AI Agent

**Purpose:**
This repository contains "Rock," a basic Command Line Interface (CLI) based AI agent designed to interact with OpenAI-compatible inference servers. The primary goal of the developer was to learn how AI agents function by building a working example.

**Core Functionality (`main.go`):**
The `main.go` file is the heart of the application. It manages:
1.  **Initialization:** Loads environment variables from `.env`.
2.  **User Interaction Loop:** Presents a continuous user prompt (`You ›`) and takes input command by command, allowing the agent to function conversationally using a history of messages.
3.  **AI Communication:** Calls an external API (`call_ai`) using the provided chat completion endpoint (`API_BASE_URL`). It sends the current conversation context (system prompt + user inputs) and available tool definitions (`get_tools`).
4.  **Response Handling:** Processes the AI's response, which can either be a natural language reply or structured data indicating that a function/tool call is required (handled by `handle_response`).

**Technical Stack & Dependencies:**
*   **Language:** Go (Golang).
*   **Dependencies:** Uses `github.com/joho/godotenv` to load environment variables.
*   **API Compatibility:** Built to interact with models exposed via a standard OpenAI Chat Completion API schema.
*   **Execution Flow:** It reads the system context, prepares available tools, and then enters an infinite loop waiting for user input or tool execution cycles.

***

### File Breakdown Summary

| File | Role / Contents | Notes |
| :--- | :--- | :--- |
| **`main.go`** | **Core Logic.** Contains the main execution loop, API calling mechanism (`call_ai`), and control flow for user interaction. | The primary source code. |
| **`README.md`**| **Project Overview.** A brief description stating that Rock is a basic CLI AI agent built for learning purposes. | High-level documentation. |
| **`go.mod`** | **Module Definition.** Specifies the project module (`rockcli`) and its required Go version/dependencies (e.g., `godotenv`). | Used by a Go developer for dependency management. |
| **`.gitignore`** | Configuration file detailing files that should be ignored by Git (like `.env`, cache files). | Standard VCS configuration. |
| **`go.sum`** | Contains cryptographic checksums for the project's dependencies, ensuring integrity. | Dependency integrity file. |

────────────────────────────────────────
