# xwebs Handler Configuration Schema

The `xwebs` CLI supports declarative, automated WebSocket message handling through YAML configuration files. You can pass a handlers file using the `--handlers` flag:

```bash
xwebs connect wss://echo.websocket.org --handlers handlers.yaml
```

This document details the schema of the handlers YAML configuration, common patterns, and examples.

---

## Root Structure

The root of the YAML file has three primary components:

```yaml
variables:    # Map of global variables (Optional)
handlers:     # List of message handlers (Required if no lifecycle events)
on_connect:   # Lifecycle event (Optional)
on_disconnect:# Lifecycle event (Optional)
on_error:     # Lifecycle event (Optional)
```

### `variables` Block
Global variables that are made available to all handlers under the `{{.Vars}}` template context.

```yaml
variables:
  temp_dir: "/tmp/xwebs"
  endpoint: "api/v1"
```

### Lifecycle Hooks
`on_connect`, `on_disconnect`, and `on_error` execute automatically when the client connects, disconnects, or encounters a connection error. They accept a list of basic actions (but not full pipelines).

```yaml
on_connect:
  - run: echo "Connected to {{.URL}}"
  - send: '{"type":"auth","token":"{{.Vars.token}}"}'
```

---

## The `handlers` Block

Each item in the `handlers` list defines how to filter incoming messages and what actions to take in response.

### Core Handler Fields

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | string | **(Required)** Unique identifier for the handler. |
| `match` | Matcher | **(Required)** Condition(s) to match the incoming message. |
| `priority` | int | Execution priority. Higher numbers run first. Default: 0. |
| `exclusive` | bool | If `true`, stops matching other handlers if this one matches. Default: false. |
| `concurrent` | bool | If `false`, messages for this handler are processed one at a time (serial execution). Default: true. |

### Execution Fields (Choose one format)

You can specify execution logic using **Shorthands**, **Pipelines**, or the legacy **Actions** list.

#### 1. Shorthands (Recommended)
Defines a quick, single-step execution flow: `run` -> `respond`.

| Field | Type | Description |
| :--- | :--- | :--- |
| `run` | string | Shell command to execute. The output is captured. |
| `respond`| string | Message to send back to the server. Can reference `{{.Stdout}}` from `run`. |
| `builtin`| string | Execute a built-in `xwebs` action. |
| `timeout`| string | Maximum wait time (e.g., `5s`). |

#### 2. Pipeline Execution
Run multiple shell commands sequentially and capture outputs for each step.

| Field      | Type       | Description                                                                      |
|------------|------------|----------------------------------------------------------------------------------|
| `pipeline` | list(Step) | List of steps to execute sequentially.                                           |
| `respond`  | string     | Executed after the pipeline. Access step outputs via `{{.Steps.<name>.Stdout}}`. |

**Step Object:**

| Field     | Type   | Description                                       |
|:----------|:-------|:--------------------------------------------------|
| `run`     | string | Shell command to execute.                         |
| `builtin` | string | Built-in command to execute.                      |
| `as`      | string | Key to store the result under `{{.Steps.<key>}}`. |
| `timeout` | string | Step-specific timeout.                            |

#### 3. Alternate Verbose Actions
For cases where shorthands or pipelines don't provide enough granularity, you can use a detailed list of specific `action` objects inside the `actions` field. This verbose syntax allows you to mix and match multiple shell commands, responses, and specific action types (like logging directly to a file).

| Field | Type | Description |
| :--- | :--- | :--- |
| `actions`| list | A list of explicit action objects. |

**Action Object:**

You can explicitly declare the action type, or let `xwebs` infer it from shorthand keys (`run`, `send`, `builtin`, `log`).

| Field | Type | Description |
| :--- | :--- | :--- |
| `action` | string | Explicitly define the type (`shell`, `send`, `log`, `builtin`). |
| `run` / `command` | string | The command to execute (for `shell` and `builtin`). |
| `send` / `message`| string | The payload to send back to the WebSocket server. |
| `log` / `message` | string | The text to print or log. |
| `target` | string | Output target for `log` actions (e.g., `stdout`, `stderr`, or a filepath). |
| `timeout`| string | Maximum wait time for `shell` or `builtin` actions (e.g., `10s`). |
| `env` | map | Custom environment variables injected during `shell` execution. |
| `silent` | bool | If `true`, suppresses implicit printing of stdout/stderr from `shell` commands. |

**Example:**
```yaml
handlers:
  - name: verbose-upload-handler
    match: '*"action":"upload"*'
    actions:
      # Action 1: Silent log to stderr
      - log: "Starting upload process... Size: {{.MessageLen}}"
        target: "stderr"
        
      # Action 2: Shell execution with custom environment
      - run: ./process_and_upload.sh "{{.Message | base64Encode}}"
        env:
          S3_BUCKET: "production-data"
          AWS_REGION: "us-west-2"
        timeout: "60s"
        
      # Action 3: Send the result
      - send: '{"status": "upload_complete", "code": {{.ExitCode}}}'
```

---

## Matchers (`match`)

Matchers determine which handler processes a message. `xwebs` offers highly flexible matching syntax.

### 1. Glob Shorthand
The simplest way to match a message containing a specific string using wildcards.
```yaml
match: "*ping*"
```

### 2. Strategy Shorthands
Expressive keys for specific matching engines.

| Shorthand Key | Used For | Example |
| :--- | :--- | :--- |
| `regex:` | Regular Expressions | `regex: "^\{.*\}$"` |
| `jq:` | JQ truthiness | `jq: ".status == 200"` |
| `json_path:` | JSON Path value extraction | `json_path: "$.type"` (Requires `equals:`) |
| `json_schema:` | Validating against a JSON Schema file | `json_schema: "schema.json"` |
| `template:` | Go Template truthiness | `template: '{{contains "error" .Message}}'` |
| `binary:` | Testing frame type | `binary: true` (or `false` for text) |

#### Path & Schema Examples:
```yaml
# JSON Path
match:
  json_path: "$.event"
  equals: "login"

# JSON Schema
match:
  json_schema: "user_schema.json"
```

### 3. Composite Matchers (AND / OR)
Combine multiple matchers using `all` (AND) and `any` (OR).

```yaml
match:
  all:
    - regex: '^{"type":'
    - jq: '.version >= 2'
  any:
    - glob: "*admin*"
    - glob: "*superuser*"
```

---

## Template Context (`{{.Variable}}`)

All execution blocks (`run`, `respond`, `send`, `log`) can use Go templates.

### Incoming Message Scope (`.`)
- `.Message`: Raw message text/content.
- `.MessageLen`: Size of the message in bytes.
- `.MessageType`: "text" or "binary".
- `.Timestamp`: Time message was received.
- `.ConnectionID`: Session identifier.

### Connection Scope
- `.URL`, `.Host`, `.Path`, `.Scheme`, `.Subprotocol`
- `.Headers`: HTTP Headers.
- `.Vars`: Global variables from the `variables:` block.
- `.Env`: System environment variables (e.g., `{{.Env.USER}}`).

### Execution Scope (Available in `respond` after `run` / pipelines)
- `.Stdout`: Standard output of the `run` command.
- `.Stderr`: Standard error.
- `.ExitCode`: Shell exit code (0 for success).
- `.DurationMs`: Execution time.
- `.Steps.<name>.Stdout`: Output of a specific pipeline step.

---

## Examples

You can find complete runnable examples in the `examples/handlers/` directory.

### 1. Simple Echo Response
*(See `examples/handlers/shell_test.yaml`)*
```yaml
handlers:
  - name: echo-handler
    match:
      jq: '.type == "echo"'
    run: echo "Hello, {{.Message | jq ".name"}}!"
    respond: '{"result": "{{.Stdout | trim}}"}'
```

### 2. Multi-Step Pipeline with Data Transformation
*(See `examples/handlers/pipeline_test.yaml`)*
```yaml
handlers:
  - name: process-data
    match: '*"action":"process"*'
    pipeline:
      - run: echo '{{.Message | jq ".payload"}}' | tr '[:lower:]' '[:upper:]'
        as: step_upper
      - run: printf '%s' '{{.Steps.step_upper.Stdout | trim}}' | wc -c
        as: step_count
    respond: >
      {
        "status":"success",
        "result":"{{.Steps.step_upper.Stdout | trim}}",
        "length":{{.Steps.step_count.Stdout | trim}}
      }
```

### 3. Concurrency Control and Composite Mapping
*(See `examples/handlers/composite_handlers.yaml`)*
```yaml
handlers:
  - name: serial-database-update
    concurrent: false      # Process one at a time to prevent race conditions
    exclusive: true        # Prevent other handlers from catching this
    match:
      all:
        - jq: '.critical == true'
        - binary: false
    run: ./update_db.sh '{{.Message | base64Encode}}'
    respond: '{"status": "saved", "code": {{.ExitCode}}}'
```
