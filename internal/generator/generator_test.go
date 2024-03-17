package generator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	dockertest "github.com/fsouza/go-dockerclient/testing"
	"github.com/nginx-proxy/docker-gen/internal/config"
)

func TestGenerateFromEvents(t *testing.T) {
	log.SetOutput(io.Discard)
	containerID := "8dfafdbc3a40"
	var counter atomic.Int32

	eventsResponse := `
{"status":"start","id":"8dfafdbc3a40","from":"base:latest","time":1374067924}
{"status":"stop","id":"8dfafdbc3a40","from":"base:latest","time":1374067966}
{"status":"start","id":"8dfafdbc3a40","from":"base:latest","time":1374067970}
{"status":"destroy","id":"8dfafdbc3a40","from":"base:latest","time":1374067990}`
	infoResponse := `{"Containers":1,"Images":1,"Debug":0,"NFd":11,"NGoroutines":21,"MemoryLimit":1,"SwapLimit":0}`
	versionResponse := `{"Version":"1.8.0","Os":"Linux","KernelVersion":"3.18.5-tinycore64","GoVersion":"go1.4.1","GitCommit":"a8a31ef","Arch":"amd64","ApiVersion":"1.19"}`

	server, _ := dockertest.NewServer("127.0.0.1:0", nil, nil)
	server.CustomHandler("/events", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rsc := bufio.NewScanner(strings.NewReader(eventsResponse))
		for rsc.Scan() {
			w.Write([]byte(rsc.Text()))
			w.(http.Flusher).Flush()
			time.Sleep(150 * time.Millisecond)
		}
		time.Sleep(500 * time.Millisecond)
	}))
	server.CustomHandler("/info", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(infoResponse))
		w.(http.Flusher).Flush()
	}))
	server.CustomHandler("/version", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(versionResponse))
		w.(http.Flusher).Flush()
	}))
	server.CustomHandler("/containers/json", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := []docker.APIContainers{
			{
				ID:      containerID,
				Image:   "base:latest",
				Command: "/bin/sh",
				Created: time.Now().Unix(),
				Status:  "running",
				Ports:   []docker.APIPort{},
				Names:   []string{"/docker-gen-test"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}))
	server.CustomHandler(fmt.Sprintf("/containers/%s/json", containerID), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter := counter.Add(1)
		container := docker.Container{
			Name:    "docker-gen-test",
			ID:      containerID,
			Created: time.Now(),
			Path:    "/bin/sh",
			Args:    []string{},
			Config: &docker.Config{
				Hostname:     "docker-gen",
				AttachStdout: true,
				AttachStderr: true,
				Env:          []string{fmt.Sprintf("COUNTER=%d", counter)},
				Cmd:          []string{"/bin/sh"},
				Image:        "base:latest",
			},
			HostConfig: &docker.HostConfig{
				NetworkMode: "container:d246e2c9e3d465d96359c942e91de493f6d51a01ba33900d865180d64c34ee91",
			},
			State: docker.State{
				Running:   true,
				Pid:       400,
				ExitCode:  0,
				StartedAt: time.Now(),
				Health: docker.Health{
					Status:        "healthy",
					FailingStreak: 5,
					Log:           []docker.HealthCheck{},
				},
			},
			Image: "0ff407d5a7d9ed36acdf3e75de8cc127afecc9af234d05486be2981cdc01a38d",
			NetworkSettings: &docker.NetworkSettings{
				IPAddress:   "10.0.0.10",
				IPPrefixLen: 24,
				Gateway:     "10.0.0.1",
				Bridge:      "docker0",
				PortMapping: map[string]docker.PortMapping{},
				Ports:       map[docker.Port][]docker.PortBinding{},
			},
			ResolvConfPath: "/etc/resolv.conf",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(container)
	}))

	serverURL := fmt.Sprintf("tcp://%s", strings.TrimRight(strings.TrimPrefix(server.URL(), "http://"), "/"))

	tmplFile, err := os.CreateTemp(os.TempDir(), "docker-gen-tmpl")
	if err != nil {
		t.Errorf("Failed to create temp file: %v\n", err)
	}
	defer func() {
		tmplFile.Close()
		os.Remove(tmplFile.Name())
	}()
	err = os.WriteFile(tmplFile.Name(), []byte("{{range $key, $value := .}}{{$value.ID}}.{{$value.Env.COUNTER}}{{end}}"), 0644)
	if err != nil {
		t.Errorf("Failed to write to temp file: %v\n", err)
	}

	var destFiles []*os.File
	for i := 0; i < 4; i++ {
		destFile, err := os.CreateTemp(os.TempDir(), "docker-gen-out")
		if err != nil {
			t.Errorf("Failed to create temp file: %v\n", err)
		}
		destFiles = append(destFiles, destFile)
	}
	defer func() {
		for _, destFile := range destFiles {
			destFile.Close()
			os.Remove(destFile.Name())
		}
	}()

	generator, err := NewGenerator(GeneratorConfig{
		Endpoint:  serverURL,
		TLSVerify: false,
		ConfigFile: config.ConfigFile{
			Config: []config.Config{
				{
					Template: tmplFile.Name(),
					Dest:     destFiles[0].Name(),
					Watch:    false,
				},
				{
					Template: tmplFile.Name(),
					Dest:     destFiles[1].Name(),
					Watch:    true,
					Wait:     &config.Wait{Min: 0, Max: 0},
				},
				{
					Template: tmplFile.Name(),
					Dest:     destFiles[2].Name(),
					Watch:    true,
					Wait:     &config.Wait{Min: 200 * time.Millisecond, Max: 250 * time.Millisecond},
				},
				{
					Template: tmplFile.Name(),
					Dest:     destFiles[3].Name(),
					Watch:    true,
					Wait:     &config.Wait{Min: 250 * time.Millisecond, Max: 1 * time.Second},
				},
			},
		},
	})
	if err != nil {
		t.Errorf("Error creating generator: %v\n", err)
	}

	generator.retry = false

	generator.generateFromEvents()
	generator.wg.Wait()

	var (
		value    []byte
		expected string
	)

	// The counter is incremented in each output file in the following sequence:
	//
	//       init   150ms  200ms  250ms  300ms  350ms  400ms  450ms  500ms  550ms  600ms  650ms  700ms
	//       ├──────╫──────┼──────┼──────╫──────┼──────┼──────╫──────┼──────┼──────┼──────┼──────┤
	// File0 ├─ 1   ║                    ║                    ║
	// File1 ├─ 1   ╟─ 2                 ╟─ 3                 ╟─ 5
	// File2 ├─ 1   ╟───── max (250ms) ──║───────────> 4      ╟─────── min (200ms) ─────> 6
	// File3 └─ 1   ╟──────────────────> ╟──────────────────> ╟─────────── min (250ms) ────────> 7
	//          ┌───╨───┐            ┌───╨──┐             ┌───╨───┐
	//          │ start │            │ stop │             │ start │
	//          └───────┘            └──────┘             └───────┘

	expectedCounters := []int{1, 5, 6, 7}

	for i, counter := range expectedCounters {
		value, _ = os.ReadFile(destFiles[i].Name())
		expected = fmt.Sprintf("%s.%d", containerID, counter)
		if string(value) != expected {
			t.Errorf("expected: %s. got: %s", expected, value)
		}
	}
}
