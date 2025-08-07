package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/TecharoHQ/anubis/internal"
	"github.com/facebookgo/flagenv"
)

var (
	dockerAnnotations = flag.String("docker-annotations", os.Getenv("DOCKER_METADATA_OUTPUT_ANNOTATIONS"), "Docker image annotations")
	dockerLabels      = flag.String("docker-labels", os.Getenv("DOCKER_METADATA_OUTPUT_LABELS"), "Docker image labels")
	slogLevel         = flag.String("slog-level", "INFO", "logging level (see https://pkg.go.dev/log/slog#hdr-Levels)")
)

func main() {
	flagenv.Parse()
	flag.Parse()

	internal.InitSlog(*slogLevel)

	koDockerRepo := "registry.hub.docker.com/vzgreposis"

	setOutput("docker_image", "vzgreposis/anubis")

	version, err := run("git describe --tags --always --dirty")
	if err != nil {
		log.Fatal(err)
	}

	commitTimestamp, err := run("git log -1 --format='%ct'")
	if err != nil {
		log.Fatal(err)
	}

	slog.Debug(
		"ko env",
		"KO_DOCKER_REPO", koDockerRepo,
		"SOURCE_DATE_EPOCH", commitTimestamp,
		"VERSION", version,
	)

	os.Setenv("KO_DOCKER_REPO", koDockerRepo)
	os.Setenv("SOURCE_DATE_EPOCH", commitTimestamp)
	os.Setenv("VERSION", version)

	setOutput("version", version)

	var tags = "main,latest"

	output, err := run(fmt.Sprintf("ko build --platform=all --base-import-paths --tags=%q --image-user=1000 --image-annotation=%q --image-label=%q ./cmd/anubis | tail -n1", tags, *dockerAnnotations, *dockerLabels))
	if err != nil {
		log.Fatalf("can't run ko build, check stderr: %v", err)
	}

	sp := strings.SplitN(output, "@", 2)

	setOutput("digest", sp[1])
}

// run executes a command and returns the trimmed output.
func run(command string) (string, error) {
	bin, err := exec.LookPath("sh")
	if err != nil {
		return "", err
	}
	slog.Debug("running command", "command", command)
	cmd := exec.Command(bin, "-c", command)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func setOutput(key, val string) {
	fmt.Printf("::set-output name=%s::%s\n", key, val)
}
