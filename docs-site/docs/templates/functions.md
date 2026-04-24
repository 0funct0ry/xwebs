---
title: Template Functions
description: All built-in template functions in xwebs, grouped by category with examples.
---

# Template Functions

xwebs registers 100+ functions in its Go template engine. They're available in handler configs, CLI flags, REPL commands, and prompt templates.

---

## String Functions

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `upper` | `upper str` | Uppercase | `{{.Message \| upper}}` |
| `lower` | `lower str` | Lowercase | `{{"HELLO" \| lower}}` |
| `title` | `title str` | Title Case | `{{"hello world" \| title}}` |
| `trim` | `trim str` | Trim whitespace | `{{.Stdout \| trim}}` |
| `trimPrefix` | `trimPrefix prefix str` | Remove prefix | `{{trimPrefix "ws://" .URL}}` |
| `trimSuffix` | `trimSuffix suffix str` | Remove suffix | `{{trimSuffix "/" .Path}}` |
| `replace` | `replace old new str` | Replace all occurrences | `{{replace "old" "new" .Message}}` |
| `split` | `split sep str` | Split into slice | `{{index (split ":" .RemoteAddr) 0}}` |
| `join` | `join sep slice` | Join slice into string | `{{join "," .List}}` |
| `contains` | `contains substr str` | Substring test | `{{if contains "error" .Message}}...{{end}}` |
| `hasPrefix` | `hasPrefix prefix str` | Starts-with test | `{{hasPrefix "{" .Message}}` |
| `hasSuffix` | `hasSuffix suffix str` | Ends-with test | `{{hasSuffix "}" .Message}}` |
| `repeat` | `repeat n str` | Repeat string | `{{repeat 3 "ha"}}` |
| `truncate` | `truncate n str` | Truncate with ellipsis | `{{truncate 50 .Message}}` |
| `padLeft` | `padLeft n char str` | Left pad | `{{padLeft 10 "0" .ID}}` |
| `padRight` | `padRight n char str` | Right pad | `{{padRight 20 " " .Name}}` |
| `indent` | `indent n str` | Indent all lines | `{{indent 4 .Stdout}}` |
| `regexMatch` | `regexMatch pattern str` | Regex test | `{{if regexMatch "\\d+" .Message}}...{{end}}` |
| `regexFind` | `regexFind pattern str` | Extract first match | `{{regexFind "id=(\\w+)" .Message}}` |
| `regexReplace` | `regexReplace pattern repl str` | Regex replace | `{{regexReplace "\\s+" "_" .Message}}` |
| `shellEscape` | `shellEscape str` | Shell-safe quoting | `echo {{.Message \| shellEscape}}` |
| `urlEncode` | `urlEncode str` | URL percent-encode | `{{.Query \| urlEncode}}` |
| `urlDecode` | `urlDecode str` | URL percent-decode | `{{.Encoded \| urlDecode}}` |
| `quote` | `quote str` | JSON-safe double-quote | `{{.Message \| quote}}` |

---

## JSON Functions

| Function | Description | Example |
|----------|-------------|---------|
| `jq` | Run a jq query on a JSON string | `{{.Message \| jq ".data.items[0].name"}}` |
| `toJSON` | Marshal a value to JSON | `{{.Data \| toJSON}}` |
| `fromJSON` | Unmarshal JSON to a map | `{{(.Message \| fromJSON).key}}` |
| `prettyJSON` | Pretty-print JSON with indent | `{{.Message \| prettyJSON}}` |
| `compactJSON` | Minify JSON | `{{.Message \| compactJSON}}` |
| `mergeJSON` | Deep-merge two JSON objects | `{{mergeJSON .Defaults .Message}}` |
| `setJSON` | Set a field in a JSON string | `{{.Message \| setJSON "ts" (now \| formatTime "RFC3339")}}` |
| `deleteJSON` | Remove a field from JSON | `{{.Message \| deleteJSON "password"}}` |
| `jsonPath` | Run a JSONPath query | `{{.Message \| jsonPath "$.store.book[*].author"}}` |
| `isJSON` | Validate JSON (bool) | `{{if isJSON .Message}}...{{end}}` |

```yaml
# Strip sensitive fields before forwarding
respond: '{{.Message | deleteJSON "token" | deleteJSON "password"}}'
```

```yaml
# Add a timestamp to any JSON message
respond: '{{.Message | setJSON "server_ts" (now | formatTime "RFC3339")}}'
```

---

## Encoding Functions

| Function | Description | Example |
|----------|-------------|---------|
| `base64Encode` | Base64 encode | `{{.Data \| base64Encode}}` |
| `base64Decode` | Base64 decode | `{{.Encoded \| base64Decode}}` |
| `hexEncode` | Hex encode | `{{.Bytes \| hexEncode}}` |
| `hexDecode` | Hex decode to bytes | `{{.Hex \| hexDecode}}` |
| `gzip` | Gzip compress | `{{.Large \| gzip \| base64Encode}}` |
| `gunzip` | Gzip decompress | `{{.Compressed \| base64Decode \| gunzip}}` |

```yaml
# Compress a large response
respond: '{"data":"{{.Stdout | gzip | base64Encode}}"}'
```

---

## Hash / Crypto Functions

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `md5` | `md5 str` | MD5 hex digest | `{{.Message \| md5}}` |
| `sha256` | `sha256 str` | SHA-256 hex digest | `{{.Message \| sha256}}` |
| `sha512` | `sha512 str` | SHA-512 hex digest | `{{.Message \| sha512}}` |
| `hmacSHA256` | `hmacSHA256 secret str` | HMAC-SHA256 hex | `{{hmacSHA256 (env "SECRET") .Message}}` |
| `jwt` | `jwt token` | Decode JWT (no verify) | `{{.Token \| jwt \| jq ".payload.sub"}}` |
| `randomBytes` | `randomBytes n` | Random bytes as hex | `{{randomBytes 16}}` |

```yaml
# Sign a webhook payload
run: |
  curl -X POST https://api.example.com/webhook \
    -H "X-Signature: {{hmacSHA256 (env "WEBHOOK_SECRET") .Message}}" \
    -d "{{.Message | shellEscape}}"
```

---

## Time Functions

| Function | Description | Example |
|----------|-------------|---------|
| `now` | Current time (time.Time) | `{{now}}` |
| `nowUnix` | Unix timestamp (seconds) | `{{nowUnix}}` |
| `nowUnixMilli` | Unix timestamp (milliseconds) | `{{nowUnixMilli}}` |
| `nowUnixNano` | Unix timestamp (nanoseconds) | `{{nowUnixNano}}` |
| `formatTime` | Format time with layout | `{{now \| formatTime "2006-01-02"}}` |
| `parseTime` | Parse time string | `{{parseTime "RFC3339" .Timestamp}}` |
| `duration` | Parse duration string | `{{duration "5m30s"}}` |
| `since` | Time elapsed since t | `{{since .StartTime}}` |
| `until` | Time until t | `{{until .Deadline}}` |
| `uptime` | Process uptime | `{{uptime}}` |

Common `formatTime` layouts: `"RFC3339"`, `"2006-01-02"`, `"15:04:05"`, `"Mon Jan 2 15:04:05 MST 2006"`.

```yaml
respond: '{"ts":"{{now | formatTime "RFC3339"}}","epoch":{{nowUnixMilli}}}'
```

---

## Numeric / Math Functions

| Function | Description | Example |
|----------|-------------|---------|
| `add` | Addition | `{{add .A .B}}` |
| `sub` | Subtraction | `{{sub .Total .Discount}}` |
| `mul` | Multiplication | `{{mul .Price .Quantity}}` |
| `div` | Division | `{{div .Total .Count}}` |
| `mod` | Modulo | `{{mod .Index 2}}` |
| `max` | Maximum of two values | `{{max .A .B}}` |
| `min` | Minimum of two values | `{{min .A .B}}` |
| `round` | Round to N decimal places | `{{round 2 .Float}}` |
| `seq` | Integer sequence | `{{range seq 1 10}}...{{end}}` |
| `toInt` | String to int | `{{.Count \| toInt}}` |
| `toFloat` | String to float64 | `{{.Price \| toFloat}}` |
| `atoi` | Shorthand for toInt | `{{atoi .N}}` |
| `random` | Random int in [0, n) | `{{random 100}}` |

---

## System / Shell Functions

| Function | Description | Example |
|----------|-------------|---------|
| `env` | Get environment variable | `{{env "HOME"}}` |
| `shell` | Execute a shell command | `{{shell "hostname"}}` |
| `hostname` | Current hostname | `{{hostname}}` |
| `pid` | Current PID | `{{pid}}` |
| `cwd` | Working directory | `{{cwd}}` |
| `fileRead` | Read file contents | `{{fileRead "/etc/hostname"}}` |
| `fileExists` | Check if file exists | `{{if fileExists "config.yaml"}}...{{end}}` |
| `glob` | List matching file paths | `{{glob "/tmp/*.json"}}` |
| `tempFile` | Create temp file, return path | `{{tempFile "prefix"}}` |

`shell`, `env`, `fileRead`, and related functions are disabled when `--no-shell-func` is set. Use this flag in untrusted environments.

```yaml
# Inject current git commit into a response
respond: '{"version":"{{shell "git rev-parse --short HEAD"}}"}'
```

---

## Unique ID / Random Functions

| Function | Description | Example |
|----------|-------------|---------|
| `uuid` | UUID v4 | `{{uuid}}` |
| `ulid` | ULID (sortable, URL-safe) | `{{ulid}}` |
| `nanoid` | NanoID | `{{nanoid}}` |
| `shortid` | 8-char unique ID | `{{shortid}}` |
| `counter` | Auto-incrementing counter | `{{counter "requests"}}` |

`counter "name"` returns a monotonically increasing integer for the named counter. Counters are process-scoped.

```yaml
run: track-request.sh --id {{uuid}} --seq {{counter "req"}}
```

---

## Collection Functions

| Function | Description | Example |
|----------|-------------|---------|
| `default` | Return default if empty | `{{default "N/A" .Name}}` |
| `coalesce` | First non-empty value | `{{coalesce .Name .Fallback "unknown"}}` |
| `ternary` | Ternary expression | `{{ternary .IsAdmin "admin" "user"}}` |
| `dict` | Create map | `{{dict "key" "value" "k2" "v2"}}` |
| `list` | Create slice | `{{list "a" "b" "c"}}` |
| `keys` | Map keys | `{{.Headers \| keys}}` |
| `values` | Map values | `{{.Headers \| values}}` |
| `pick` | Select keys from map | `{{.Data \| pick "name" "email"}}` |
| `omit` | Remove keys from map | `{{.Data \| omit "password" "token"}}` |
| `chunk` | Split slice into chunks | `{{chunk 10 .Items}}` |
| `uniq` | Unique values | `{{.List \| uniq}}` |
| `sortAlpha` | Sort strings | `{{.Names \| sortAlpha}}` |
| `reverse` | Reverse slice | `{{.List \| reverse}}` |
| `first` | First element | `{{.List \| first}}` |
| `last` | Last element | `{{.List \| last}}` |
| `rest` | All but first | `{{.List \| rest}}` |
| `pluck` | Extract field from list of maps | `{{.Items \| pluck "name"}}` |

```yaml
# Strip sensitive fields from a JSON object
respond: '{{.Message | fromJSON | omit "password" "token" | toJSON}}'
```

---

## KV Store Function

| Function | Description | Example |
|----------|-------------|---------|
| `kv` | Read a value from the server KV store | `{{kv "maintenance_mode"}}` |

This function is server-mode only. It returns an empty string if the key doesn't exist.

```yaml
- name: check-maintenance
  match: "*"
  respond: |
    {{if eq (kv "maintenance_mode") "true"}}
    {"error":"service_unavailable","retry_after":60}
    {{else}}
    {"status":"ok"}
    {{end}}
```
