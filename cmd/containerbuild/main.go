package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/TecharoHQ/anubis/internal"
	"github.com/facebookgo/flagenv"
)

var (
	dockerAnnotations = flag.String("docker-annotations", os.Getenv("DOCKER_METADATA_OUTPUT_ANNOTATIONS"), "Docker image annotations")
	dockerLabels      = flag.String("docker-labels", os.Getenv("DOCKER_METADATA_OUTPUT_LABELS"), "Docker image labels")
	dockerRepo        = flag.String("docker-repo", "vzgreposis/anubis", "Docker image repository for Anubis")
	dockerTags        = flag.String("docker-tags", os.Getenv("DOCKER_METADATA_OUTPUT_TAGS"), "newline separated docker tags including the registry name")
	slogLevel         = flag.String("slog-level", "INFO", "logging level (see https://pkg.go.dev/log/slog#hdr-Levels)")
)

func main() {
	flagenv.Parse()
	flag.Parse()

	internal.InitSlog(*slogLevel)

	koDockerRepo := strings.TrimSuffix(*dockerRepo, "/"+filepath.Base(*dockerRepo))

	setOutput("docker_image", strings.SplitN(*dockerTags, "\n", 2)[0])

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

	if *dockerTags == "" {
		log.Fatal("Must set --docker-tags or DOCKER_METADATA_OUTPUT_TAGS")
	}

	var tags = strings.Split(*dockerTags, " ")

	output, err := run(fmt.Sprintf("ko build --platform=all --base-import-paths --tags=%q --image-user=1000 --image-annotation=%q --image-label=%q ./cmd/anubis | tail -n1", strings.Join(tags, ","), *dockerAnnotations, *dockerLabels))
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
