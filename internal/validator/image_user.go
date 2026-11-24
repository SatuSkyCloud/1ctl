package validator

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"1ctl/internal/utils"
)

// wellKnownUIDs are the uids of named users that common base images declare.
// They only make the error message more useful; nothing depends on them.
var wellKnownUIDs = map[string]int64{
	"node":     1000, // official node images
	"nginx":    101,  // nginx and nginx-unprivileged
	"www-data": 33,   // debian-based php and apache images
	"nobody":   65534,
}

// FinalStageUser returns the USER that the image built from the Dockerfile at
// path runs as, as written in the Dockerfile. It follows multi-stage builds: a
// stage that starts FROM an earlier stage inherits that stage's USER. It
// returns "" when the final stage never sets USER, because the user then comes
// from the base image, which the CLI cannot see.
func FinalStageUser(path string) (string, error) {
	stage, err := parseFinalStage(path)
	return stage.user, err
}

// finalStage is what the CLI can tell about the image's final build stage.
type finalStage struct {
	user      string // last USER in effect, "" when none was set
	baseImage string // external image the final stage descends from, lowercased
}

func parseFinalStage(path string) (finalStage, error) {
	f, err := os.Open(path) // #nosec G304 -- user-provided Dockerfile path is intentional
	if err != nil {
		return finalStage{}, err
	}
	defer func() { _ = f.Close() }() //nolint:errcheck

	stages := map[string]finalStage{} // stage name -> state at the end of that stage
	stageName := ""
	var cur finalStage
	var pending strings.Builder
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), " \t")
		if strings.HasSuffix(line, "\\") {
			pending.WriteString(strings.TrimSuffix(line, "\\") + " ")
			continue
		}
		pending.WriteString(line)
		fields := strings.Fields(pending.String())
		pending.Reset()
		if len(fields) < 2 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		switch strings.ToUpper(fields[0]) {
		case "FROM":
			args := fields[1:]
			for len(args) > 0 && strings.HasPrefix(args[0], "--") {
				args = args[1:] // --platform=...
			}
			if len(args) == 0 {
				continue
			}
			from := strings.ToLower(args[0])
			if parent, ok := stages[from]; ok {
				cur = parent
			} else {
				cur = finalStage{baseImage: from}
			}
			stageName = ""
			if len(args) >= 3 && strings.EqualFold(args[1], "AS") {
				stageName = strings.ToLower(args[2])
			}
		case "USER":
			cur.user = fields[1]
		}
		if stageName != "" {
			stages[stageName] = cur
		}
	}
	if err := scanner.Err(); err != nil {
		return finalStage{}, err
	}
	return cur, nil
}

// nonRootBaseImage reports whether image is a base image known to default to a
// non-root numeric user, so a final stage without USER still starts. Unknown
// images are assumed to run as root, which is the Docker default.
func nonRootBaseImage(image string) bool {
	repo, tag := splitImageRef(image)
	switch {
	case strings.HasPrefix(repo, "gcr.io/distroless/"):
		return strings.Contains(tag, "nonroot")
	case strings.HasPrefix(repo, "cgr.dev/chainguard/"),
		strings.HasPrefix(repo, "bitnami/"),
		strings.HasPrefix(repo, "docker.io/bitnami/"),
		repo == "nginxinc/nginx-unprivileged",
		repo == "docker.io/nginxinc/nginx-unprivileged":
		return true
	}
	return false
}

// splitImageRef splits "registry/repo:tag@digest" into repo and tag.
func splitImageRef(image string) (repo, tag string) {
	repo = strings.SplitN(image, "@", 2)[0]
	if i := strings.LastIndex(repo, ":"); i > strings.LastIndex(repo, "/") {
		return repo[:i], repo[i+1:]
	}
	return repo, ""
}

// baseImageUserHint names the non-root user that well-known base images ship,
// for the error message only.
func baseImageUserHint(image string) string {
	repo, _ := splitImageRef(image)
	repo = strings.TrimPrefix(strings.TrimPrefix(repo, "docker.io/"), "library/")
	if repo == "node" {
		return " (the official node images include a non-root user node, uid 1000)"
	}
	return ""
}

// CheckImageUserVerifiable returns an error when the image built from the
// Dockerfile would run as root or as a user the platform cannot verify, and no
// run_as_user is configured. The platform runs every container with
// runAsNonRoot, and the kubelet can only verify that against a numeric uid, so
// such a container never starts: the deployment stays in "reconciling" with
// CreateContainerConfigError (#135). It blocks:
//   - a final USER that is a name, such as "node" or "nginx";
//   - a final USER of root or 0;
//   - no USER in the final stage, unless the base image is known to default to
//     a non-root user, because Docker then runs the container as root.
//
// Users and base images set from build args (USER $APP_USER, FROM $BASE) are
// not checked.
func CheckImageUserVerifiable(dockerfilePath string, runAsUser *int64) error {
	if runAsUser != nil || dockerfilePath == "" {
		return nil
	}
	stage, err := parseFinalStage(dockerfilePath)
	if err != nil || strings.Contains(stage.user, "$") {
		return nil
	}
	if stage.user == "" {
		if stage.baseImage == "" || strings.Contains(stage.baseImage, "$") || nonRootBaseImage(stage.baseImage) {
			return nil
		}
		return rootUserError(dockerfilePath, "no USER in the final stage", baseImageUserHint(stage.baseImage))
	}
	name := strings.SplitN(stage.user, ":", 2)[0]
	uid, numErr := strconv.ParseInt(name, 10, 64)
	if name == "root" || (numErr == nil && uid == 0) {
		return rootUserError(dockerfilePath, "USER "+stage.user, "")
	}
	if numErr == nil {
		return nil // numeric uid, verifiable
	}
	uidHint := "<uid>"
	known := ""
	if uid, ok := wellKnownUIDs[name]; ok {
		uidHint = strconv.FormatInt(uid, 10)
		known = fmt.Sprintf(" (the %s user is uid %d in the official images)", name, uid)
	}
	return utils.NewError(fmt.Sprintf(
		"%s runs as USER %s, a name the platform cannot verify as non-root, so the container would never start%s. "+
			"Fix it with either:\n"+
			"  - run_as_user = %s under [app] in satusky.toml, or\n"+
			"  - a numeric user in the Dockerfile, for example USER %s",
		dockerfilePath, stage.user, known, uidHint, uidHint), nil)
}

// rootUserError explains that the image runs as root. reason says why, for
// example "no USER in the final stage" or "USER root".
func rootUserError(dockerfilePath, reason, hint string) error {
	return utils.NewError(fmt.Sprintf(
		"%s runs as root (%s), and the platform only runs non-root containers, so the container would never start%s. "+
			"Fix it with either:\n"+
			"  - run_as_user = 1000 under [app] in satusky.toml, or\n"+
			"  - a non-root numeric user in the Dockerfile, for example USER 1000",
		dockerfilePath, reason, hint), nil)
}
