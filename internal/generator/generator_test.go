package generator

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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
	log.SetOutput(io.Discard)
	containerID := "dev123456789abcd"

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
