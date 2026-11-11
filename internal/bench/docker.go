package bench

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
)

// Docker is the real Engine. Images are inspected, pulled and removed through
// the Docker SDK; builds go through `docker build`, so they use BuildKit with
// the same cache behaviour people get on their machines.
type Docker struct {
	cli *client.Client
	bin string
}

// NewDocker connects to the daemon from the environment (DOCKER_HOST etc.).
func NewDocker(ctx context.Context) (*Docker, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	if _, err := cli.Ping(ctx); err != nil {
		return nil, fmt.Errorf("can't reach the docker daemon: %w", err)
	}
	bin, err := exec.LookPath("docker")
	if err != nil {
		return nil, fmt.Errorf("the docker CLI is needed for builds: %w", err)
	}
	return &Docker{cli: cli, bin: bin}, nil
}

// Close releases the client.
func (d *Docker) Close() error { return d.cli.Close() }

// Pull makes sure an image is available locally.
func (d *Docker) Pull(ctx context.Context, ref string) error {
	if _, err := d.cli.ImageInspect(ctx, ref); err == nil {
		return nil // already there; pulling again would only measure the network
	}
	rc, err := d.cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return err
	}
	defer rc.Close()
	return jsonmessage.DisplayJSONMessagesStream(rc, io.Discard, 0, false, nil)
}

// Build runs docker build and returns the tail of the log on failure.
func (d *Docker) Build(ctx context.Context, contextDir, dockerfile, tag string, noCache bool) error {
	args := []string{"build", "--progress=plain", "-f", dockerfile, "-t", tag}
	if noCache {
		args = append(args, "--no-cache")
	}
	args = append(args, contextDir)

	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, d.bin, args...)
	cmd.Stdout, cmd.Stderr = &out, &out
	cmd.Env = append(cmd.Environ(), "DOCKER_BUILDKIT=1")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build failed: %w\n%s", err, tail(out.String(), 25))
	}
	return nil
}

// Inspect reads size and layer count.
func (d *Docker) Inspect(ctx context.Context, tag string) (ImageInfo, error) {
	resp, err := d.cli.ImageInspect(ctx, tag)
	if err != nil {
		return ImageInfo{}, err
	}
	return ImageInfo{Size: resp.Size, Layers: len(resp.RootFS.Layers)}, nil
}

// Remove deletes the image and its now untagged parents.
func (d *Docker) Remove(ctx context.Context, tag string) error {
	_, err := d.cli.ImageRemove(ctx, tag, image.RemoveOptions{Force: true, PruneChildren: true})
	return err
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
