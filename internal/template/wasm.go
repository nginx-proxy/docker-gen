package template

import (
	"bytes"
	goctx "context"
	"log"
	"os"
	"path/filepath"

	"github.com/nginx-proxy/docker-gen/internal/context"
	"github.com/nginx-proxy/docker-gen/plugin"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

var WasmCacheDir = filepath.Join(os.TempDir(), "docker-gen-wasm-cache")

func executeWasm(wasmFile string, containers context.Context) []byte {
	outBuf := bytes.Buffer{}

	inBytes, err := pluginInputFromContext(containers).MarshalJSON()
	if err != nil {
		log.Fatalf("Unable to serialize plugin input: %s", err.Error())
	}
	inBuf := bytes.NewBuffer(inBytes)

	ctx := goctx.Background()

	wasmBytes, err := os.ReadFile(wasmFile)
	if err != nil {
		log.Fatalf("Unable to parse template: %s", err.Error())
	}

	wazeroCache, err := wazero.NewCompilationCacheWithDir(WasmCacheDir)
	if err != nil {
		log.Fatalf("wazero.NewCompilationCacheWithDir: %s", err.Error())
	}

	err = os.MkdirAll(WasmCacheDir, 0700)
	if err != nil {
		log.Fatalf("Unable to create WasmCacheDir: %s", err.Error())
	}
	config := wazero.NewRuntimeConfig().WithCompilationCache(wazeroCache)
	rt := wazero.NewRuntimeWithConfig(ctx, config)

	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		log.Fatalf("wasi_snapshot_preview1 instantiate: %s", err.Error())
	}

	// Compile the Wasm binary once so that we can skip the entire compilation
	// time during instantiation.
	code, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		log.Fatalf("compile module: %s", err.Error())
	}

	if _, err = rt.InstantiateModule(ctx,
		code,
		wazero.NewModuleConfig().
			WithStdin(inBuf).
			WithStdout(&outBuf).
			WithStderr(os.Stderr).
			WithSysWalltime().
			WithArgs("gen.wasm"),
	); err != nil {
		// Note: Most compilers do not exit the module after running "_start",
		// unless there was an error. This allows you to call exported functions.
		if exitErr, ok := err.(*sys.ExitError); ok && exitErr.ExitCode() != 0 {
			log.Panicf("exit_code: %d\n", exitErr.ExitCode())
		} else if !ok {
			log.Panicf("Failed to run %s", err)
		}
	}

	return outBuf.Bytes()
}

func pluginNetwork(in *context.Network) (ret plugin.Network) {
	ret.IP = in.IP
	ret.Name = in.Name
	ret.Gateway = in.Gateway
	ret.EndpointID = in.EndpointID
	ret.IPv6Gateway = in.IPv6Gateway
	ret.GlobalIPv6Address = in.GlobalIPv6Address
	ret.MacAddress = in.MacAddress
	ret.GlobalIPv6PrefixLen = in.GlobalIPv6PrefixLen
	ret.IPPrefixLen = in.IPPrefixLen
	ret.Internal = in.Internal
	return
}

func pluginVolume(in *context.Volume) (ret plugin.Volume) {
	ret.Path = in.Path
	ret.HostPath = in.HostPath
	ret.ReadWrite = in.ReadWrite
	return
}

func pluginState(in *context.State) (ret plugin.State) {
	ret.Running = in.Running
	ret.Health = pluginHealth(&in.Health)
	return
}

func pluginHealth(in *context.Health) (ret plugin.Health) {
	ret.Status = in.Status
	return
}

func pluginAddress(in *context.Address) (ret plugin.Address) {
	ret.IP = in.IP
	ret.IP6LinkLocal = in.IP6LinkLocal
	ret.IP6Global = in.IP6Global
	ret.Port = in.Port
	ret.HostPort = in.HostPort
	ret.Proto = in.Proto
	ret.HostIP = in.HostIP
	return
}

func pluginRuntimeContainer(in *context.RuntimeContainer) (ret *plugin.RuntimeContainer) {
	ret = new(plugin.RuntimeContainer)
	ret.ID = in.ID
	ret.Created = in.Created
	ret.Addresses = make([]plugin.Address, 0, len(in.Addresses))
	for _, v := range in.Addresses {
		ret.Addresses = append(ret.Addresses, pluginAddress(&v))
	}
	ret.Networks = make([]plugin.Network, 0, len(in.Networks))
	for _, v := range in.Networks {
		ret.Networks = append(ret.Networks, pluginNetwork(&v))
	}
	ret.Gateway = in.Gateway
	ret.Name = in.Name
	ret.Hostname = in.Hostname
	ret.NetworkMode = in.NetworkMode
	ret.Image = pluginDockerImage(&in.Image)
	ret.Env = in.Env
	ret.Volumes = make(map[string]plugin.Volume)
	for k, v := range in.Volumes {
		ret.Volumes[k] = pluginVolume(&v)
	}
	ret.Node = pluginSwarmNode(&in.Node)
	ret.Labels = in.Labels
	ret.IP = in.IP
	ret.IP6LinkLocal = in.IP6LinkLocal
	ret.IP6Global = in.IP6Global
	ret.Mounts = make([]plugin.Mount, 0, len(in.Mounts))
	for _, v := range in.Mounts {
		ret.Mounts = append(ret.Mounts, pluginMount(&v))
	}
	ret.State = pluginState(&in.State)
	return
}

func pluginDockerImage(in *context.DockerImage) (ret plugin.DockerImage) {
	ret.Registry = in.Registry
	ret.Repository = in.Repository
	ret.Tag = in.Tag
	return
}

func pluginSwarmNode(in *context.SwarmNode) (ret plugin.SwarmNode) {
	ret.ID = in.ID
	ret.Name = in.Name
	ret.Address = pluginAddress(&in.Address)
	return
}

func pluginMount(in *context.Mount) (ret plugin.Mount) {
	ret.Name = in.Name
	ret.Source = in.Source
	ret.Destination = in.Destination
	ret.Driver = in.Driver
	ret.Mode = in.Mode
	ret.RW = in.RW
	return
}

func pluginDocker(in *context.Docker) (ret plugin.Docker) {
	ret.Name = in.Name
	ret.NumContainers = in.NumContainers
	ret.NumImages = in.NumImages
	ret.Version = in.Version
	ret.ApiVersion = in.ApiVersion
	ret.GoVersion = in.GoVersion
	ret.OperatingSystem = in.OperatingSystem
	ret.Architecture = in.Architecture
	ret.CurrentContainerID = in.CurrentContainerID
	return
}

func pluginInputFromContext(ctx context.Context) *plugin.PluginContext {
	containers := make([]*plugin.RuntimeContainer, 0, len(ctx))
	for _, c := range ctx {
		containers = append(containers, pluginRuntimeContainer(c))
	}
	d := ctx.Docker()
	return &plugin.PluginContext{
		Containers: containers,
		Env:        ctx.Env(),
		Docker:     pluginDocker(&d),
	}
}
