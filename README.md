# flogs

`flogs` filters newline-delimited JSON log files down to unique error records.

The program reads a log file, keeps only records where `level` is `ERROR`, normalizes volatile values such as timestamps, IDs, durations, stack traces, request/session fields, and object addresses, then writes the first occurrence of each unique normalized error to a sibling output file named `filtered-<input-file>`.

## Requirements

- Go 1.22 or newer

## Usage

Run directly:

```sh
go run . /path/to/logs.jsonl
```

Build and run:

```sh
go build -o flogs .
./flogs /path/to/logs.jsonl
```

Example output:

```text
raw_error_lines=42 unique_error_lines=7 output=/path/to/filtered-logs.jsonl
```

## Input Format

Input should be newline-delimited JSON, with one log record per line. Lines that are not valid JSON are skipped. Non-error records are skipped.

Example:

```json lines
{"timestamp":"2026-06-17T12:00:00Z","level":"ERROR","message":"request failed","request_id":"abc"}
{"timestamp":"2026-06-17T12:00:01Z","level":"INFO","message":"request completed"}
```

## Output

For an input file named `errors.jsonl`, the output file is written as:

```text
filtered-errors.jsonl
```
