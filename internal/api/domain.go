package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/util/rand"
)

const DOMAIN_SUFFIX = "satusky.com"

func GenerateDomainName(projectName string) (string, error) {
	// Clean project name: lowercase and replace invalid chars with hyphens
	cleanName := strings.ToLower(projectName)
	cleanName = strings.ReplaceAll(cleanName, "_", "-")
	cleanName = strings.ReplaceAll(cleanName, ".", "-")

	// First try without suffix
	proposedDomain := fmt.Sprintf("%s.%s", cleanName, DOMAIN_SUFFIX)

	// Keep trying until we find an available domain
	for {
		ingress, err := GetIngressByDomainName(proposedDomain)
		if err != nil {
			return "", fmt.Errorf("failed to check domain existence: %w", err)
		}

		// If domain is available (not found), we can use it
		if ingress.IngressID == uuid.Nil {
			return proposedDomain, nil
		}

		// Domain exists, try with a new random suffix
		suffix := generateShortID()
		proposedDomain = fmt.Sprintf("%s-%s.%s", cleanName, suffix, DOMAIN_SUFFIX)
	}
}

func generateShortID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
