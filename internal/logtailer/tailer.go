package logtailer

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/SC-Bridge/sc-companion/internal/events"
)

const pollInterval = 100 * time.Millisecond

// Tailer follows a Game.log file and emits parsed events.
type Tailer struct {
	path   string
	bus    *events.Bus
	parser *Parser
}

// New creates a Tailer for the given log file path.
func New(path string, bus *events.Bus) (*Tailer, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	return &Tailer{
		path:   path,
		bus:    bus,
		parser: NewParser(),
	}, nil
}

// Run starts tailing the log file. It seeks to the end and reads new lines.
// Blocks until ctx is cancelled.
func (t *Tailer) Run(ctx context.Context) error {
	f, err := os.Open(t.path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Seek to end — we only want new lines
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return err
	}

	slog.Info("tailing log", "path", t.path)

	reader := bufio.NewReader(f)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := t.readLines(reader); err != nil {
				return err
			}
		}
	}
}

// RunFromStart processes an entire log file from the beginning (for replay/analysis).
func (t *Tailer) RunFromStart(ctx context.Context) error {
	f, err := os.Open(t.path)
	if err != nil {
		return err
	}
	defer f.Close()

	slog.Info("processing log from start", "path", t.path)

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB line buffer
	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil
		}
		t.processLine(scanner.Text())
	}
	return scanner.Err()
}

func (t *Tailer) readLines(reader *bufio.Reader) error {
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			t.processLine(line)
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func (t *Tailer) processLine(line string) {
	evt, ok := t.parser.Parse(line)
	if ok {
		t.bus.Publish(evt)
	}
}
