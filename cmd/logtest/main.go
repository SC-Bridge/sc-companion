// cmd/logtest: runs the parser against all log files in a directory and
// prints a per-event-type hit count plus any unmatched "Added notification" lines.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SC-Bridge/sc-companion/internal/logtailer"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: logtest <log-directory>")
		os.Exit(1)
	}
	dir := os.Args[1]

	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read dir:", err)
		os.Exit(1)
	}

	counts := map[string]int{}
	examples := map[string]string{}
	unmatchedNotifs := map[string]int{}
	fileCount := 0

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		fileCount++
		path := filepath.Join(dir, e.Name())
		processFile(path, counts, examples, unmatchedNotifs)
	}

	fmt.Printf("Processed %d log files.\n\n", fileCount)

	// --- Pattern hit summary ---
	fmt.Println("=== PATTERN HIT SUMMARY ===")
	types := make([]string, 0, len(counts))
	for t := range counts {
		types = append(types, t)
	}
	sort.Strings(types)
	for _, t := range types {
		fmt.Printf("  %-45s %d\n", t, counts[t])
		if ex, ok := examples[t]; ok {
			preview := ex
			if len(preview) > 120 {
				preview = preview[:120] + "..."
			}
			fmt.Printf("    ex: %s\n", preview)
		}
	}

	// --- Zero-hit patterns (by pattern name, not event type) ---
	fmt.Println("\n=== ZERO-HIT PATTERNS ===")
	parser := logtailer.NewParser()
	zeroHit := false
	for _, name := range parser.PatternNames() {
		if counts[name] == 0 {
			fmt.Printf("  %s\n", name)
			zeroHit = true
		}
	}
	if !zeroHit {
		fmt.Println("  (none — all patterns matched at least once)")
	}

	// --- Unmatched notifications ---
	fmt.Println("\n=== UNMATCHED 'Added notification' TYPES (top 40 by frequency) ===")
	type kv struct{ k string; v int }
	var sorted []kv
	for k, v := range unmatchedNotifs {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].v > sorted[j].v })
	limit := 40
	if len(sorted) < limit {
		limit = len(sorted)
	}
	for _, kv := range sorted[:limit] {
		fmt.Printf("  %-60s %d\n", kv.k, kv.v)
	}
	if len(sorted) == 0 {
		fmt.Println("  (none)")
	}
}

func processFile(path string, counts map[string]int, examples map[string]string, unmatchedNotifs map[string]int) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open:", err)
		return
	}
	defer f.Close()

	parser := logtailer.NewParser()
	const notifMarker = `Added notification "`

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		evt, ok := parser.Parse(line)
		if ok {
			counts[evt.Type]++
			if _, seen := examples[evt.Type]; !seen {
				examples[evt.Type] = line
			}
			continue
		}
		// Track unmatched notification lines
		idx := strings.Index(line, notifMarker)
		if idx < 0 {
			continue
		}
		after := line[idx+len(notifMarker):]
		key := after
		for i, ch := range after {
			if ch == ':' || ch == '"' {
				key = after[:i]
				break
			}
		}
		key = strings.TrimSpace(key)
		if key != "" {
			unmatchedNotifs[key]++
		}
	}
}
