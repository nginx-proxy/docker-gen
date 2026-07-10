package generator

import (
	"io"
	"log"
	"testing"
	"testing/synctest"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/nginx-proxy/docker-gen/internal/config"
	"github.com/nginx-proxy/docker-gen/internal/context"
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

			input <- newStartEvent() // t=0
			time.Sleep(150 * time.Millisecond)
			input <- newStartEvent() // t=150ms
			time.Sleep(150 * time.Millisecond)
			input <- newStartEvent() // t=300ms
			time.Sleep(time.Second)
			synctest.Wait()

			close(input)
			<-done

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

			input <- newStartEvent() // t=0
			time.Sleep(150 * time.Millisecond)
			input <- newStartEvent() // t=150ms, Max fires at 250ms
			time.Sleep(150 * time.Millisecond)
			input <- newStartEvent() // t=300ms
			time.Sleep(time.Second)
			synctest.Wait()

			close(input)
			<-done

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
