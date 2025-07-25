package fms

import (
	"bufio"
	"io"
	"net/http"
	"os/exec"
)

func (f *FMS) runSystemCommand(w http.ResponseWriter, exe string, args ...string) error {
	flusher, flushAvailable := w.(http.Flusher)
	cmd := exec.Command(exe, args...)
	rPipe, wPipe := io.Pipe()
	cmd.Stdout = wPipe
	cmd.Stderr = wPipe
	cmd.Start()

	scanner := bufio.NewScanner(rPipe)
	scanner.Split(bufio.ScanLines)
	go func() {
		for scanner.Scan() {
			w.Write(scanner.Bytes())
			w.Write([]byte("\r\n"))
			if flushAvailable {
				flusher.Flush()
			}
		}
	}()
	err := cmd.Wait()
	w.Write([]byte("\r\n"))
	return err
}

func (f *FMS) invertTLMMap(m map[int]string) map[string]int {
	out := make(map[string]int)
	for k, v := range m {
		out[v] = k
	}
	return out
}
