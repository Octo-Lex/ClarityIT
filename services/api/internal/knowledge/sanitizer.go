package knowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

// ─── Sanitizer ───
// Strips secrets, tokens, credentials, raw prompts, chain-of-thought,
// and storage identifiers from content before indexing.

var (
	// Authorization / token patterns
	reBearerToken   = regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9\-._~+/]+=*`)
	reAPIKey        = regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[:=]\s*\S+`)
	reAuthorization = regexp.MustCompile(`(?i)authorization\s*:\s*\S+`)
	rePassword      = regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*\S+`)
	reAccessToken   = regexp.MustCompile(`(?i)(access[_-]?token)\s*[:=]\s*\S+`)
	reRefreshToken  = regexp.MustCompile(`(?i)(refresh[_-]?token)\s*[:=]\s*\S+`)
	reSecret        = regexp.MustCompile(`(?i)(secret|client[_-]?secret)\s*[:=]\s*\S+`)
	reRecoveryCode  = regexp.MustCompile(`(?i)(recovery[_-]?code)\s*[:=]\s*\S+`)
	reTOTP          = regexp.MustCompile(`(?i)(totp[_-]?secret|otpauth://[^\s]+)`)

	// Storage identifiers
	reMinIOBucket   = regexp.MustCompile(`(?i)(bucket)\s*[:=]\s*\S+`)
	reMinIOObject   = regexp.MustCompile(`(?i)(object[_-]?key|storage[_-]?key)\s*[:=]\s*\S+`)
	reS3URL         = regexp.MustCompile(`(?i)s3://[^\s]+`)

	// Proxmox credentials
	reProxmoxCreds  = regexp.MustCompile(`(?i)(proxmox[_-]?(?:password|token|secret|api))\s*[:=]\s*\S+`)

	// Action target payloads (approval system)
	reActionTarget  = regexp.MustCompile(`(?i)"action_target"\s*:\s*\{[^}]*\}`)

	// Chain-of-thought / thinking / internal reasoning
	reChainOfThought = regexp.MustCompile(`(?i)"(chain_of_thought|thinking|internal_reasoning|reasoning_summary)"\s*:\s*"[^"]*"`)
	reChainOfThoughtObj = regexp.MustCompile(`(?i)"(chain_of_thought|thinking|internal_reasoning)"\s*:\s*\{[^}]*\}`)
)

// Dangerous metadata keys that must never be indexed
var dangerousMetaKeys = map[string]bool{
	"password": true, "passwd": true, "pwd": true,
	"token": true, "access_token": true, "refresh_token": true,
	"api_key": true, "apikey": true, "secret": true, "client_secret": true,
	"private_key": true, "session_id": true, "cookie": true,
	"totp_secret": true, "recovery_code": true,
	"authorization": true, "auth_header": true,
	"minio_bucket": true, "object_key": true, "storage_key": true,
	"proxmox_password": true, "proxmox_token": true,
	"prompt": true, "raw_prompt": true,
	"chain_of_thought": true, "thinking": true, "internal_reasoning": true,
}

// SanitizeContent strips dangerous patterns from text content.
func SanitizeContent(content string) string {
	s := content
	s = reBearerToken.ReplaceAllString(s, "[REDACTED]")
	s = reAPIKey.ReplaceAllString(s, "[REDACTED]")
	s = reAuthorization.ReplaceAllString(s, "[REDACTED]")
	s = rePassword.ReplaceAllString(s, "[REDACTED]")
	s = reAccessToken.ReplaceAllString(s, "[REDACTED]")
	s = reRefreshToken.ReplaceAllString(s, "[REDACTED]")
	s = reSecret.ReplaceAllString(s, "[REDACTED]")
	s = reRecoveryCode.ReplaceAllString(s, "[REDACTED]")
	s = reTOTP.ReplaceAllString(s, "[REDACTED]")
	s = reMinIOBucket.ReplaceAllString(s, "[REDACTED]")
	s = reMinIOObject.ReplaceAllString(s, "[REDACTED]")
	s = reS3URL.ReplaceAllString(s, "[REDACTED]")
	s = reProxmoxCreds.ReplaceAllString(s, "[REDACTED]")
	s = reActionTarget.ReplaceAllString(s, "[REDACTED]")
	s = reChainOfThought.ReplaceAllString(s, "")
	s = reChainOfThoughtObj.ReplaceAllString(s, "")
	return s
}

// SanitizeMetadata removes dangerous keys from a metadata map.
// Returns a clean map safe for indexing.
func SanitizeMetadata(meta map[string]any) map[string]any {
	clean := make(map[string]any)
	for k, v := range meta {
		lk := strings.ToLower(k)
		if dangerousMetaKeys[lk] {
			continue
		}
		// Deep sanitize string values
		if s, ok := v.(string); ok {
			clean[k] = SanitizeContent(s)
		} else {
			clean[k] = v
		}
	}
	return clean
}

// ComputeContentHash returns a deterministic SHA256 hex digest for content.
func ComputeContentHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// HasSecrets returns true if content contains patterns that look like secrets.
func HasSecrets(content string) bool {
	checks := []*regexp.Regexp{
		reBearerToken, reAPIKey, rePassword, reAccessToken,
		reRefreshToken, reSecret, reTOTP, reProxmoxCreds,
	}
	for _, re := range checks {
		if re.MatchString(content) {
			return true
		}
	}
	return false
}
