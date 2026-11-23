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
	f, err := os.Open(path) // #nosec G304 -- user-provided Dockerfile path is intentional
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }() //nolint:errcheck

	stageUsers := map[string]string{} // stage name -> USER at the end of that stage
	stageName, user := "", ""
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
			user = stageUsers[strings.ToLower(args[0])] // "" for an external image
			stageName = ""
			if len(args) >= 3 && strings.EqualFold(args[1], "AS") {
				stageName = strings.ToLower(args[2])
			}
		case "USER":
			user = fields[1]
		}
		if stageName != "" {
			stageUsers[stageName] = user
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return user, nil
}

// CheckImageUserVerifiable returns an error when the Dockerfile's final USER is
// a name, such as "node" or "nginx", and no run_as_user is configured. The
// platform runs every container with runAsNonRoot, and the kubelet can only
// verify that against a numeric uid, so such a container never starts: the
// deployment stays in "reconciling" with CreateContainerConfigError (#135).
// Users set from build args (USER $APP_USER) are not checked.
func CheckImageUserVerifiable(dockerfilePath string, runAsUser *int64) error {
	if runAsUser != nil || dockerfilePath == "" {
		return nil
	}
	user, err := FinalStageUser(dockerfilePath)
	if err != nil || user == "" || strings.Contains(user, "$") {
		return nil
	}
	name := strings.SplitN(user, ":", 2)[0]
	uid, numErr := strconv.ParseInt(name, 10, 64)
	if name == "root" || (numErr == nil && uid == 0) {
		return utils.NewError(fmt.Sprintf(
			"%s runs as USER %s, which is root; the platform runs every container as non-root, so it would never start. "+
				"Use a non-root numeric user, for example USER 1000", dockerfilePath, user), nil)
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
		dockerfilePath, user, known, uidHint, uidHint), nil)
}
