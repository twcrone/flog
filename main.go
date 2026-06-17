package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

var (
	volatileKeys = map[string]bool{
		"timestamp":            true,
		"beginTime":            true,
		"endTime":              true,
		"response":             true,
		"stack_trace":          true,
		"transId":              true,
		"masterFeatureContext": true,
		"applicationPermID":    true,
		"sessionId":            true,
		"userPermID":           true,
		"request_id":           true,
	}

	isoTimestampRE    = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})`)
	localTimestampRE  = regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d+)?`)
	durationMSRE      = regexp.MustCompile(`duration_ms=\d+(?:\.\d+)?`)
	elapsedTimeRE     = regexp.MustCompile(`elapsed_time"?: ?\d+`)
	objectIDRE        = regexp.MustCompile(`0x[0-9a-fA-F]+`)
	uuidRE            = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	draftIDRE         = regexp.MustCompile(`(?i)DRAFT-[0-9a-f-]+`)
	planIDRE          = regexp.MustCompile(`plan-[A-Za-z0-9]+`)
	httpsConnectionRE = regexp.MustCompile(`<urllib3\.connection\.HTTPSConnection object at <object_id>>`)
	tracebackRE       = regexp.MustCompile(`(?s)Traceback \(most recent call last\):.*$`)
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <logs-file>\n", filepath.Base(os.Args[0]))
		os.Exit(2)
	}

	inputPath := os.Args[1]
	outputPath := filepath.Join(filepath.Dir(inputPath), "filtered-"+filepath.Base(inputPath))

	rawCount, uniqueCount, err := filterUniqueErrors(inputPath, outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("raw_error_lines=%d unique_error_lines=%d output=%s\n", rawCount, uniqueCount, outputPath)
}

func filterUniqueErrors(inputPath, outputPath string) (int, int, error) {
	input, err := os.Open(inputPath)
	if err != nil {
		return 0, 0, err
	}
	defer input.Close()

	output, err := os.Create(outputPath)
	if err != nil {
		return 0, 0, err
	}
	defer output.Close()

	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 1024), 50*1024*1024)

	seen := make(map[string]bool)
	rawErrors := 0
	uniqueErrors := 0

	for scanner.Scan() {
		line := scanner.Bytes()
		var record map[string]any
		if err := json.Unmarshal(line, &record); err != nil {
			continue
		}
		if record["level"] != "ERROR" {
			continue
		}

		rawErrors++
		keyBytes, err := json.Marshal(normalize(record, ""))
		if err != nil {
			return rawErrors, uniqueErrors, err
		}
		key := string(keyBytes)
		if seen[key] {
			continue
		}

		seen[key] = true
		uniqueErrors++
		if _, err := output.Write(bytes.TrimRight(line, "\r\n")); err != nil {
			return rawErrors, uniqueErrors, err
		}
		if _, err := output.Write([]byte("\n")); err != nil {
			return rawErrors, uniqueErrors, err
		}
	}

	if err := scanner.Err(); err != nil {
		return rawErrors, uniqueErrors, err
	}
	return rawErrors, uniqueErrors, nil
}

func normalize(value any, key string) any {
	if volatileKeys[key] {
		return nil
	}

	switch typed := value.(type) {
	case string:
		return normalizeString(typed)
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			normalized := normalize(item, "")
			if normalized != nil {
				result = append(result, normalized)
			}
		}
		return result
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for childKey := range typed {
			keys = append(keys, childKey)
		}
		sort.Strings(keys)

		result := make(map[string]any, len(typed))
		for _, childKey := range keys {
			normalized := normalize(typed[childKey], childKey)
			if normalized != nil {
				result[childKey] = normalized
			}
		}
		return result
	default:
		return typed
	}
}

func normalizeString(value string) string {
	value = isoTimestampRE.ReplaceAllString(value, "<timestamp>")
	value = localTimestampRE.ReplaceAllString(value, "<timestamp>")
	value = durationMSRE.ReplaceAllString(value, "duration_ms=<duration>")
	value = elapsedTimeRE.ReplaceAllString(value, "elapsed_time:<duration>")
	value = objectIDRE.ReplaceAllString(value, "<object_id>")
	value = uuidRE.ReplaceAllString(value, "<uuid>")
	value = draftIDRE.ReplaceAllString(value, "DRAFT-<uuid>")
	value = planIDRE.ReplaceAllString(value, "plan-<id>")
	value = httpsConnectionRE.ReplaceAllString(value, "<HTTPSConnection object>")
	value = tracebackRE.ReplaceAllString(value, "<stack_trace>")
	return value
}
