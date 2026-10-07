// Package audit menulis log audit: satu baris JSON per kejadian.
package audit

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Kunci ditulis dengan urutan tetap: ts, level, event, msg, lalu data.
type entry struct {
	TS    string         `json:"ts"`
	Level string         `json:"level"`
	Event string         `json:"event"`
	Msg   string         `json:"msg"`
	Data  map[string]any `json:"data,omitempty"`
}

// Logger menulis kejadian ke file secara berurutan dan aman-goroutine.
type Logger struct {
	mu sync.Mutex
	f  *os.File
}

// Open membuka (atau membuat) file log untuk penulisan tambahan.
func Open(path string) (*Logger, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	return &Logger{f: f}, nil
}

func (l *Logger) write(level, event, msg string, data map[string]any) {
	e := entry{TS: time.Now().UTC().Format(time.RFC3339), Level: level, Event: event, Msg: msg, Data: data}
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	b = append(b, '\n')
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.f.Write(b)
}

// Info mencatat kejadian biasa.
func (l *Logger) Info(event, msg string, data map[string]any) {
	l.write("info", event, msg, data)
}

// Error mencatat kejadian gagal.
func (l *Logger) Error(event, msg string, data map[string]any) {
	l.write("error", event, msg, data)
}

// Close menutup file log.
func (l *Logger) Close() error {
	return l.f.Close()
}
