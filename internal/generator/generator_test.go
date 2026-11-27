package generator

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	dockertest "github.com/fsouza/go-dockerclient/testing"
	"github.com/nginx-proxy/docker-gen/internal/config"
	"github.com/nginx-proxy/docker-gen/internal/context"
	"github.com/nginx-proxy/docker-gen/internal/dockerclient"
	"github.com/stretchr/testify/assert"
)

func newStartEvent() *docker.APIEvents {
	return &docker.APIEvents{Type: "container", Action: "start"}
}

func TestNewDebounceChannel(t *testing.T) {
	orig := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(orig) })

	t.Run("passes events through when Min is zero", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			input := make(chan *docker.APIEvents, 1)
			out := newDebounceChannel(input, &config.Wait{Min: 0, Max: 0})

			ev := newStartEvent()
			input <- ev
			synctest.Wait()

			select {
			case got := <-out:
				assert.Same(t, ev, got)
			default:
				t.Fatal("expected the event to pass straight through")
			}
		})
	})

	t.Run("coalesces a burst and fires Min after the last event", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			input := make(chan *docker.APIEvents)
			out := newDebounceChannel(input, &config.Wait{Min: 200 * time.Millisecond, Max: time.Second})

			start := time.Now()
			var fires []time.Duration
			done := make(chan struct{})
			go func() {
				for range out {
					fires = append(fires, time.Since(start))
				}
				close(done)
			}()

			input <- newStartEvent()           // t=0: minTimer->200ms, maxTimer->1000ms
			time.Sleep(150 * time.Millisecond) //
			input <- newStartEvent()           // t=150ms (gap 150ms < Min)
			time.Sleep(150 * time.Millisecond) //
			input <- newStartEvent()           // t=300ms (gap 150ms < Min)
			time.Sleep(time.Second)            // advance the fake clock so the pending timer fires
			synctest.Wait()

			close(input)
			<-done

			// One coalesced event, fired Min (200ms) after the last event (t=300ms).
			assert.Equal(t, []time.Duration{500 * time.Millisecond}, fires)
		})
	})

	t.Run("Max caps the wait when events keep arriving", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			input := make(chan *docker.APIEvents)
			out := newDebounceChannel(input, &config.Wait{Min: 200 * time.Millisecond, Max: 250 * time.Millisecond})

			start := time.Now()
			var fires []time.Duration
			done := make(chan struct{})
			go func() {
				for range out {
					fires = append(fires, time.Since(start))
				}
				close(done)
			}()

			input <- newStartEvent()           // t=0: minTimer->200ms, maxTimer->250ms
			time.Sleep(150 * time.Millisecond) //
			input <- newStartEvent()           // t=150ms: minTimer reset->350ms, maxTimer still 250ms
			time.Sleep(150 * time.Millisecond) // maxTimer fires at 250ms -> first output
			input <- newStartEvent()           // t=300ms: new burst, minTimer->500ms
			time.Sleep(time.Second)            // advance the fake clock so the pending timer fires (500ms)
			synctest.Wait()

			close(input)
			<-done

			// First output capped by Max at 250ms; second is Min after the t=300ms event.
			assert.Equal(t, []time.Duration{250 * time.Millisecond, 500 * time.Millisecond}, fires)
		})
	})
}

func TestSortNetworks(t *testing.T) {
	for _, tc := range []struct {
		desc string
		in   []context.Network
		want []context.Network
	}{
		{
			desc: "multiple unsorted",
			in:   []context.Network{{Name: "frontend"}, {Name: "bridge"}, {Name: "app_net"}},
			want: []context.Network{{Name: "app_net"}, {Name: "bridge"}, {Name: "frontend"}},
		},
		{
			desc: "single element",
			in:   []context.Network{{Name: "bridge"}},
			want: []context.Network{{Name: "bridge"}},
		},
		{
			desc: "empty",
			in:   []context.Network{},
			want: []context.Network{},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			sortNetworks(tc.in)
			assert.Equal(t, tc.want, tc.in)
		})
	}
}

func TestGetContainersNilStructs(t *testing.T) {
	// A container inspected with nil Config/NetworkSettings/HostConfig must not panic (#227).
	log.SetOutput(io.Discard)
	containerID := "abc123def4567890"

	server, _ := dockertest.NewServer("127.0.0.1:0", nil, nil)
	server.CustomHandler("/info", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Containers":1,"Images":1,"NFd":11,"NGoroutines":21}`))
	}))
	server.CustomHandler("/version", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Version":"19.03.12","Os":"Linux","GoVersion":"go1.13.14","Arch":"amd64","ApiVersion":"1.40"}`))
	}))
	server.CustomHandler("/networks", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[]"))
	}))
	server.CustomHandler("/containers/json", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]docker.APIContainers{{ID: containerID, Names: []string{"/nil-test"}}})
	}))
	server.CustomHandler(fmt.Sprintf("/containers/%s/json", containerID), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(docker.Container{
			ID:              containerID,
			Name:            "/nil-test",
			Config:          nil,
			HostConfig:      nil,
			NetworkSettings: nil,
		})
	}))

	serverURL := fmt.Sprintf("tcp://%s", strings.TrimRight(strings.TrimPrefix(server.URL(), "http://"), "/"))
	client, err := dockerclient.NewDockerClient(serverURL, false, "", "", "")
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}
	client.SkipServerVersionCheck = true

	apiVersion, err := client.Version()
	if err != nil {
		t.Fatalf("failed to retrieve version: %s", err)
	}
	context.SetDockerEnv(apiVersion)

	g := &generator{Client: client, Endpoint: serverURL}
	containers, err := g.getContainers(config.Config{})
	assert.NoError(t, err)
	assert.Len(t, containers, 1)
	assert.Equal(t, containerID, containers[0].ID)
	assert.Equal(t, "nil-test", containers[0].Name)
	assert.Empty(t, containers[0].Hostname)
	assert.Empty(t, containers[0].NetworkMode)
	assert.Empty(t, containers[0].IP)
	assert.Empty(t, containers[0].Addresses)
	assert.Empty(t, containers[0].Networks)
}

func TestGetContainersDevices(t *testing.T) {
	orig := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(orig) })
	containerID := "dev123456789abcd"

	server, err := dockertest.NewServer("127.0.0.1:0", nil, nil)
	if err != nil {
		t.Fatalf("failed to create test server: %s", err)
	}
	t.Cleanup(server.Stop)
	server.CustomHandler("/info", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Containers":1,"Images":1,"NFd":11,"NGoroutines":21}`))
	}))
	server.CustomHandler("/version", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Version":"19.03.12","Os":"Linux","GoVersion":"go1.13.14","Arch":"amd64","ApiVersion":"1.40"}`))
	}))
	server.CustomHandler("/networks", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[]"))
	}))
	server.CustomHandler("/containers/json", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]docker.APIContainers{{ID: containerID, Names: []string{"/dev-test"}}})
	}))
	server.CustomHandler(fmt.Sprintf("/containers/%s/json", containerID), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(docker.Container{
			ID:   containerID,
			Name: "/dev-test",
			HostConfig: &docker.HostConfig{
				Devices: []docker.Device{
					{PathOnHost: "/dev/ttyACM0", PathInContainer: "/dev/ttyUSB0", CgroupPermissions: "rwm"},
				},
			},
		})
	}))

	serverURL := fmt.Sprintf("tcp://%s", strings.TrimRight(strings.TrimPrefix(server.URL(), "http://"), "/"))
	client, err := dockerclient.NewDockerClient(serverURL, false, "", "", "")
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}
	client.SkipServerVersionCheck = true

	apiVersion, err := client.Version()
	if err != nil {
		t.Fatalf("failed to retrieve version: %s", err)
	}
	context.SetDockerEnv(apiVersion)

	g := &generator{Client: client, Endpoint: serverURL}
	containers, err := g.getContainers(config.Config{})
	assert.NoError(t, err)
	assert.Len(t, containers, 1)
	assert.Equal(t, []context.Device{
		{PathOnHost: "/dev/ttyACM0", PathInContainer: "/dev/ttyUSB0", Permissions: "rwm"},
	}, containers[0].Devices)
}

func TestGetContainersSetsCurrentContainer(t *testing.T) {
	orig := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(orig) })

	currentID := "current123456789"

	server, err := dockertest.NewServer("127.0.0.1:0", nil, nil)
	if err != nil {
		t.Fatalf("failed to create test server: %s", err)
	}
	t.Cleanup(server.Stop)
	server.CustomHandler("/info", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Containers":0,"Images":0,"NFd":11,"NGoroutines":21}`))
	}))
	server.CustomHandler("/version", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Version":"19.03.12","Os":"Linux","GoVersion":"go1.13.14","Arch":"amd64","ApiVersion":"1.40"}`))
	}))
	server.CustomHandler("/networks", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[]"))
	}))
	server.CustomHandler("/containers/json", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[]"))
	}))
	server.CustomHandler(fmt.Sprintf("/containers/%s/json", currentID), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(docker.Container{ID: currentID, Name: "/self"})
	}))

	serverURL := fmt.Sprintf("tcp://%s", strings.TrimRight(strings.TrimPrefix(server.URL(), "http://"), "/"))
	client, err := dockerclient.NewDockerClient(serverURL, false, "", "", "")
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}
	client.SkipServerVersionCheck = true

	apiVersion, err := client.Version()
	if err != nil {
		t.Fatalf("failed to retrieve version: %s", err)
	}
	context.SetDockerEnv(apiVersion)

	g := &generator{
		Client:                client,
		Endpoint:              serverURL,
		getCurrentContainerID: func(...string) string { return currentID },
	}

	containers, err := g.getContainers(config.Config{})
	assert.NoError(t, err)
	assert.Empty(t, containers)

	var emptyCtx context.Context
	current := emptyCtx.CurrentContainer()
	assert.NotNil(t, current)
	assert.Equal(t, currentID, current.ID)
}

const notifiedContainerID = "notified123456789"

// newSwitchingGenerator returns a generator connected to a fake Docker server, which lists one container
// when withContainer returns true and no containers otherwise. Signals sent to the container with
// notifiedContainerID are handled by notify.
func newSwitchingGenerator(t *testing.T, withContainer func() bool, notify http.HandlerFunc) *generator {
	t.Helper()
	containerID := "switch1234567890"

	server, err := dockertest.NewServer("127.0.0.1:0", nil, nil)
	if err != nil {
		t.Fatalf("failed to create test server: %s", err)
	}
	t.Cleanup(server.Stop)
	server.CustomHandler("/version", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Version":"19.03.12","Os":"Linux","GoVersion":"go1.13.14","Arch":"amd64","ApiVersion":"1.40"}`))
	}))
	server.CustomHandler("/networks", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[]"))
	}))
	server.CustomHandler("/containers/json", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		containers := []docker.APIContainers{}
		if withContainer() {
			containers = append(containers, docker.APIContainers{ID: containerID, Names: []string{"/switch"}})
		}
		json.NewEncoder(w).Encode(containers)
	}))
	server.CustomHandler(fmt.Sprintf("/containers/%s/json", containerID), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(docker.Container{ID: containerID, Name: "/switch", Config: &docker.Config{}})
	}))
	server.CustomHandler(fmt.Sprintf("/containers/%s/kill", notifiedContainerID), notify)

	serverURL := fmt.Sprintf("tcp://%s", strings.TrimRight(strings.TrimPrefix(server.URL(), "http://"), "/"))
	client, err := dockerclient.NewDockerClient(serverURL, false, "", "", "")
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}
	client.SkipServerVersionCheck = true

	apiVersion, err := client.Version()
	if err != nil {
		t.Fatalf("failed to retrieve version: %s", err)
	}
	context.SetDockerEnv(apiVersion)

	return &generator{Client: client, Endpoint: serverURL}
}

// blockingNotify handles signals sent to a container by recording the start and the end of handling
// each of them, and blocking between them until release is called.
type blockingNotify struct {
	mu          sync.Mutex
	events      []string
	started     chan struct{}
	released    chan struct{}
	releaseOnce sync.Once
}

// newBlockingNotify returns a blockingNotify which is released at the latest when the test finishes.
func newBlockingNotify(t *testing.T) *blockingNotify {
	b := &blockingNotify{started: make(chan struct{}, 2), released: make(chan struct{})}
	t.Cleanup(b.release)
	return b
}

func (b *blockingNotify) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.record("start")
	b.started <- struct{}{}
	<-b.released
	b.record("end")
	w.WriteHeader(http.StatusNoContent)
}

func (b *blockingNotify) record(event string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, event)
}

// recorded returns the events recorded so far.
func (b *blockingNotify) recorded() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.events...)
}

// waitStarted waits until handling of a signal starts.
func (b *blockingNotify) waitStarted(t *testing.T) {
	t.Helper()
	select {
	case <-b.started:
	case <-time.After(5 * time.Second):
		t.Fatal("container was not signaled")
	}
}

// release unblocks all current and future handling of signals.
func (b *blockingNotify) release() {
	b.releaseOnce.Do(func() { close(b.released) })
}

func TestGenerateConfigSerializesNotifications(t *testing.T) {
	orig := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(orig) })

	notify := newBlockingNotify(t)
	g := newSwitchingGenerator(t, func() bool { return false }, notify.ServeHTTP)

	dir := t.TempDir()
	tmpl := filepath.Join(dir, "template")
	if err := os.WriteFile(tmpl, []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Template:         tmpl,
		Dest:             filepath.Join(dir, "dest"),
		NotifyContainers: map[string]int{notifiedContainerID: 1},
	}

	var wg sync.WaitGroup
	wg.Go(func() { assert.NoError(t, g.generateConfig(cfg, true)) })
	notify.waitStarted(t)
	wg.Go(func() { assert.NoError(t, g.generateConfig(cfg, true)) })
	// Without serialization the second notification would start while the first one is still blocked.
	time.Sleep(200 * time.Millisecond)
	notify.release()
	wg.Wait()

	assert.Equal(t, []string{"start", "end", "start", "end"}, notify.recorded())
}

func TestGenerateConfigListsContainersAfterLock(t *testing.T) {
	orig := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(orig) })

	var withContainer atomic.Bool
	notify := newBlockingNotify(t)
	g := newSwitchingGenerator(t, withContainer.Load, notify.ServeHTTP)

	dir := t.TempDir()
	tmpl := filepath.Join(dir, "template")
	if err := os.WriteFile(tmpl, []byte("{{ len . }}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "dest")
	cfg := config.Config{
		Template:         tmpl,
		Dest:             dest,
		NotifyContainers: map[string]int{notifiedContainerID: 1},
	}

	var wg sync.WaitGroup
	wg.Go(func() { assert.NoError(t, g.generateConfig(cfg, false)) })
	notify.waitStarted(t)
	// The second generation waits for the first one, whose notification has not finished yet.
	wg.Go(func() { assert.NoError(t, g.generateConfig(cfg, false)) })
	time.Sleep(200 * time.Millisecond)
	// The container appears while the second generation is waiting, so it has to list containers after waiting.
	withContainer.Store(true)
	notify.release()
	wg.Wait()

	contents, err := os.ReadFile(dest)
	assert.NoError(t, err)
	assert.Equal(t, "1\n", string(contents))
}
