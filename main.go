// gitlab.com/jjlabs-dev/clipper
package main

/*
#include "pasteboard.h"
#include <stdlib.h>
*/
import "C"
import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unsafe"
)

const (
	storeFile  = "/tmp/copypasta.bin"
	chooseCmd  = "/opt/homebrew/bin/choose"
	chooseArgs = "-n 30 -w 1000"
)

// Entry represents a clipboard entry with type info
type Entry struct {
	UTI  string
	Data []byte
}

func getPasteboard() *Entry {
	content := C.getPasteboardContent()
	defer C.freeContent(content)

	if content.data == nil || content.length == 0 {
		return nil
	}

	data := C.GoBytes(content.data, content.length)
	uti := C.GoString(content.uti)

	return &Entry{UTI: uti, Data: data}
}

func setPasteboard(e *Entry) {
	cuti := C.CString(e.UTI)
	defer C.free(unsafe.Pointer(cuti))

	C.setPasteboardData(unsafe.Pointer(&e.Data[0]), C.int(len(e.Data)), cuti)
}

func emitCopy() {
	C.emitCopy()
}

func emitPaste() {
	C.emitPaste()
}

// Storage format: base64(uti:data)\n per line
func encodeEntry(e *Entry) string {
	combined := e.UTI + ":" + base64.StdEncoding.EncodeToString(e.Data)
	return combined
}

func decodeEntry(line string) *Entry {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return nil
	}
	data, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	return &Entry{UTI: parts[0], Data: data}
}

func appendToStore(e *Entry) error {
	f, err := os.OpenFile(storeFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(encodeEntry(e) + "\n")
	return err
}

func readStore() ([]*Entry, error) {
	f, err := os.Open(storeFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []*Entry
	scanner := bufio.NewScanner(f)
	// Increase buffer for large entries
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		if e := decodeEntry(scanner.Text()); e != nil {
			entries = append(entries, e)
		}
	}
	return entries, scanner.Err()
}

func getLastEntry() *Entry {
	entries, err := readStore()
	if err != nil || len(entries) == 0 {
		return nil
	}
	return entries[len(entries)-1]
}

// For display in chooser - show preview of content
func entryPreview(e *Entry) string {
	if e.UTI == "public.utf8-plain-text" {
		s := string(e.Data)
		s = strings.ReplaceAll(s, "\n", "↵ ")
		s = strings.ReplaceAll(s, "\t", "→ ")
		if len(s) > 80 {
			s = s[:80] + "..."
		}
		return s
	}
	// Binary content - show type and size
	return fmt.Sprintf("[%s] %d bytes", filepath.Base(e.UTI), len(e.Data))
}

func runChooser(entries []*Entry) (int, error) {
	if len(entries) == 0 {
		return -1, fmt.Errorf("no entries")
	}

	// Build input for chooser (reversed - newest first)
	var input bytes.Buffer
	for i := len(entries) - 1; i >= 0; i-- {
		fmt.Fprintf(&input, "%d: %s\n", i, entryPreview(entries[i]))
	}

	args := strings.Fields(chooseArgs)
	cmd := exec.Command(chooseCmd, args...)
	cmd.Stdin = &input

	out, err := cmd.Output()
	if err != nil {
		return -1, err
	}

	// Parse selection
	selection := strings.TrimSpace(string(out))
	if selection == "" {
		return -1, fmt.Errorf("no selection")
	}

	var idx int
	fmt.Sscanf(selection, "%d:", &idx)
	return idx, nil
}

func doCopy() {
	// Emit Cmd+C
	emitCopy()

	// Wait for clipboard to be populated
	time.Sleep(50 * time.Millisecond)

	// Get and store
	if e := getPasteboard(); e != nil {
		if err := appendToStore(e); err != nil {
			fmt.Fprintf(os.Stderr, "store error: %v\n", err)
		}
	}
}

func doClipboardCopy() {
	// Just store current clipboard (no key sim)
	if e := getPasteboard(); e != nil {
		if err := appendToStore(e); err != nil {
			fmt.Fprintf(os.Stderr, "store error: %v\n", err)
		}
	}
}

func doPasteLast() {
	e := getLastEntry()
	if e == nil {
		return
	}
	setPasteboard(e)
	time.Sleep(100 * time.Millisecond)
	emitPaste()
}

func doPasteSelect() {
	entries, err := readStore()
	if err != nil || len(entries) == 0 {
		return
	}

	idx, err := runChooser(entries)
	if err != nil || idx < 0 || idx >= len(entries) {
		return
	}

	setPasteboard(entries[idx])
	time.Sleep(100 * time.Millisecond)
	emitPaste()
}

func doClear() {
	os.Remove(storeFile)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: clipper [c|C|v|P|x]")
		fmt.Println("  c  Copy (Cmd+C + store)")
		fmt.Println("  C  Store current clipboard")
		fmt.Println("  v  Paste last entry")
		fmt.Println("  P  Paste with selector")
		fmt.Println("  x  Clear store")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "c":
		doCopy()
	case "C":
		doClipboardCopy()
	case "v":
		doPasteLast()
	case "P":
		doPasteSelect()
	case "x":
		doClear()
	default:
		fmt.Fprintf(os.Stderr, "Unknown action: %s\n", os.Args[1])
		os.Exit(1)
	}
}
