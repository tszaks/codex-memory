package workflow

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

// codexUsage reads only completed-turn accounting events, never tool output.
// Output tokens already include reasoning. Dollar cost is not provided by the
// Codex CLI and remains absent, including for subscription-backed execution.
func codexUsage(stream string) map[string]any {
	var total map[string]any
	scanner := bufio.NewScanner(strings.NewReader(stream))
	scanner.Buffer(make([]byte, 4096), 4*1024*1024)
	for scanner.Scan() {
		var event struct {
			Type  string         `json:"type"`
			Usage map[string]any `json:"usage"`
		}
		if json.Unmarshal(scanner.Bytes(), &event) != nil || event.Type != "turn.completed" || event.Usage == nil {
			continue
		}
		if total == nil {
			total = map[string]any{"provider": "codex", "billing_basis": "tokens_only", "cost_status": "unknown"}
		}
		for _, key := range []string{"input_tokens", "cached_input_tokens", "output_tokens"} {
			if n, ok := event.Usage[key].(float64); ok && n >= 0 {
				prev, _ := total[key].(float64)
				total[key] = prev + n
			}
		}
	}
	return total
}

func writeCodexUsage(path, stream string) {
	if usage := codexUsage(stream); usage != nil {
		if raw, err := json.Marshal(usage); err == nil {
			_ = os.WriteFile(path, raw, 0o600)
		}
	}
}
