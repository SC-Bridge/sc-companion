package logtailer

import (
	"bufio"
	"bytes"
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

// Run starts tailing the log file. It seeks back to the last player_login
// line, replays events from that point to the current EOF to restore session
// state, then continues tailing for new lines.
// Blocks until ctx is cancelled.
func (t *Tailer) Run(ctx context.Context) error {
	f, err := os.Open(t.path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := t.seekToLastSession(f); err != nil {
		slog.Warn("seekToLastSession failed, tailing from EOF", "error", err)
		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			return err
		}
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

// seekToLastSession scans backwards through the file to find the last
// player_login line (identified by the nickname=" marker) and seeks f to
// the start of that line. If no login is found, seeks to EOF so that only
// new lines will be read.
func (t *Tailer) seekToLastSession(f *os.File) error {
	size, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	if size == 0 {
		return nil
	}

	loginMarker := []byte(`nickname="`)
	const chunkSize = 64 * 1024 // 64 KB

	pos := size
	for pos > 0 {
		readSize := int64(chunkSize)
		if pos < readSize {
			readSize = pos
		}
		pos -= readSize

		if _, err := f.Seek(pos, io.SeekStart); err != nil {
			return err
		}

		buf := make([]byte, readSize)
		n, err := io.ReadFull(f, buf)
		if err != nil && err != io.ErrUnexpectedEOF {
			return err
		}
		buf = buf[:n]

		idx := bytes.LastIndex(buf, loginMarker)
		if idx < 0 {
			continue
		}

		// Find the newline before the match to locate the line start
		lineStart := bytes.LastIndexByte(buf[:idx], '\n')
		var lineOffset int64
		if lineStart < 0 {
			lineOffset = pos // match is on the first line of this chunk
		} else {
			lineOffset = pos + int64(lineStart+1)
		}

		_, err = f.Seek(lineOffset, io.SeekStart)
		return err
	}

	// No login found — tail from EOF only
	_, err = f.Seek(0, io.SeekEnd)
	return err
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
