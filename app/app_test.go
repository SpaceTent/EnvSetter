package app

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/SpaceTent/EnvSetter/pkg/envvaultclient"
)

func TestPullWritesMode0600File(t *testing.T) {
	plain := "# application settings\nAPI_KEY=abc=def\nEMPTY=\n"
	serverURL, token, key := testServer(t, "my-vault", "production-api", plain)
	t.Setenv("ENVVAULT_URL", serverURL)
	t.Setenv("ENVVAULT_TOKEN", token)
	t.Setenv("ENVVAULT_KEY", key)

	output := filepath.Join(t.TempDir(), ".env")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"pull", "--vault", "my-vault", "--secret", "production-api", "-o", output}, &stdout, &stderr, "test")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != plain {
		t.Fatalf("file = %q", got)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
	if strings.Contains(stdout.String()+stderr.String(), token) || strings.Contains(stdout.String()+stderr.String(), key) || strings.Contains(stdout.String()+stderr.String(), "API_KEY") {
		t.Fatal("output disclosed secret material")
	}
}

func TestPullRefusesOverwriteWithoutForce(t *testing.T) {
	serverURL, token, key := testServer(t, "my-vault", "production-api", "NEW=value\n")
	t.Setenv("ENVVAULT_URL", serverURL)
	t.Setenv("ENVVAULT_TOKEN", token)
	t.Setenv("ENVVAULT_KEY", key)

	output := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(output, []byte("OLD=value\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := Run([]string{"pull", "-v", "my-vault", "-s", "production-api", "-o", output}, ioDiscard{}, &stderr, "test")
	if code != 1 || !strings.Contains(stderr.String(), "--force") {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	got, _ := os.ReadFile(output)
	if string(got) != "OLD=value\n" {
		t.Fatalf("existing file changed: %q", got)
	}
}

func TestLegacySyntaxAcceptsEnvVaultPrefix(t *testing.T) {
	serverURL, token, key := testServer(t, "my-vault", "production-api", "A=b\n")
	t.Setenv("ENVVAULT_URL", serverURL)
	t.Setenv("ENVVAULT_TOKEN", token)
	t.Setenv("ENVVAULT_KEY", key)

	output := filepath.Join(t.TempDir(), ".env")
	var stderr bytes.Buffer
	if code := Run([]string{"envvault:my-vault", "production-api", output}, ioDiscard{}, &stderr, "test"); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
}

func TestRunInjectsValuesAndStripsCredentials(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell assertion is Unix-specific")
	}
	serverURL, token, key := testServer(t, "my-vault", "production-api", "FROM_VAULT=yes\nEXISTING=replaced\n")
	t.Setenv("ENVVAULT_URL", serverURL)
	t.Setenv("ENVVAULT_TOKEN", token)
	t.Setenv("ENVVAULT_KEY", key)
	t.Setenv("EXISTING", "old")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"run", "-v", "my-vault", "-s", "production-api", "--", "sh", "-c", `printf '%s|%s|%s|%s' "$FROM_VAULT" "$EXISTING" "${ENVVAULT_TOKEN-unset}" "${ENVVAULT_KEY-unset}"`}, &stdout, &stderr, "test")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	if stdout.String() != "yes|replaced|unset|unset" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestCredentialFileMustBePrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte("mvt_test"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := credential(path, "vault token")
	if err == nil || !strings.Contains(err.Error(), "must not be accessible") {
		t.Fatalf("error = %v", err)
	}
}

func TestVersionAndUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"version"}, &stdout, &stderr, "v1.2.3"); code != 0 || stdout.String() != "v1.2.3\n" {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	if code := Run(nil, &stdout, &stderr, "test"); code != 2 || !strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

func testServer(t *testing.T, vault, secret, plain string) (string, string, string) {
	t.Helper()
	token := "mvt_" + strings.Repeat("t", 24)
	teamID := "team_test"
	rawKey := make([]byte, 32)
	if _, err := rand.Read(rawKey); err != nil {
		t.Fatal(err)
	}
	key := base64.RawURLEncoding.EncodeToString(rawKey)
	envelope := seal(t, rawKey, teamID, secret, plain)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/api/"+vault+"/"+secret {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+token {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if strings.Contains(r.URL.String(), token) || strings.Contains(r.URL.String(), key) {
			t.Error("request URL disclosed a credential")
		}
		w.Header().Set(envvaultclient.TeamIDHeader, teamID)
		_, _ = w.Write(envelope)
	})
	previous := transport
	transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		return recorder.Result(), nil
	})
	t.Cleanup(func() { transport = previous })
	return "https://envvault.test", token, key
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func seal(t *testing.T, key []byte, teamID, secret, plain string) []byte {
	t.Helper()
	iv := make([]byte, 12)
	if _, err := rand.Read(iv); err != nil {
		t.Fatal(err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	aad := envvaultclient.AAD(teamID, secret)
	ciphertext := gcm.Seal(nil, iv, []byte(plain), []byte(aad))
	body, err := json.Marshal(map[string]any{
		"v":          1,
		"alg":        "AES-256-GCM",
		"iv":         base64.RawURLEncoding.EncodeToString(iv),
		"ciphertext": base64.RawURLEncoding.EncodeToString(ciphertext),
		"aad":        aad,
	})
	if err != nil {
		t.Fatal(err)
	}
	return body
}
