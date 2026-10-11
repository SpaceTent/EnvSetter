package envvaultclient

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	// TokenPrefix marks a vault token issued by EnvVault.
	TokenPrefix = "mvt_"
	// DefaultURL is the public EnvVault service.
	DefaultURL = "https://www.envvault.net"
	// TeamIDHeader carries the team ID bound into the envelope's AAD.
	TeamIDHeader = "X-Envvault-Team-ID"
	// MaxEnvelopeBytes matches EnvVault's encrypted upload limit.
	MaxEnvelopeBytes = 262144
	// DefaultTimeout bounds a fetch when Options.Timeout is zero.
	DefaultTimeout = 15 * time.Second
)

var (
	// ErrWrongKey means the key is well formed but cannot open this envelope.
	ErrWrongKey = errors.New("envvault: that key does not open this file")
	// ErrInvalidKey means the key is not 32 bytes of unpadded base64url.
	ErrInvalidKey = errors.New("envvault: key must be 32 bytes, base64url without padding")
	// ErrEnvelope means the encrypted envelope or its authenticated metadata is invalid.
	ErrEnvelope = errors.New("envvault: envelope is invalid or belongs to a different secret")
	// ErrInvalidEnv means the decrypted document is not KEY=value lines.
	ErrInvalidEnv = errors.New("envvault: decrypted file is not a valid ENV file")
)

// Options identifies an EnvVault secret and supplies the credentials needed
// to fetch and decrypt it. The key is used locally and is never sent over HTTP.
type Options struct {
	URL    string
	Token  string
	Key    string
	Vault  string
	Secret string

	HTTPClient *http.Client
	Timeout    time.Duration
}

// FromEnv fills empty fields from ENVVAULT_URL, ENVVAULT_TOKEN, ENVVAULT_KEY,
// ENVVAULT_VAULT and ENVVAULT_SECRET.
func FromEnv(o Options) Options {
	set := func(value *string, name string) {
		if *value == "" {
			*value = os.Getenv(name)
		}
	}
	set(&o.URL, "ENVVAULT_URL")
	set(&o.Token, "ENVVAULT_TOKEN")
	set(&o.Key, "ENVVAULT_KEY")
	set(&o.Vault, "ENVVAULT_VAULT")
	set(&o.Secret, "ENVVAULT_SECRET")
	return o
}

// HTTPError is a non-200 response from EnvVault. Code is populated when the
// response contains EnvVault's JSON error envelope.
type HTTPError struct {
	StatusCode int
	Code       string
}

func (e *HTTPError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("envvault: fetch failed: HTTP %d (%s)", e.StatusCode, e.Code)
	}
	return fmt.Sprintf("envvault: fetch failed: HTTP %d", e.StatusCode)
}

// Fetch downloads, verifies and decrypts a secret, returning its plaintext
// .env document. Callers must keep the returned bytes out of logs.
func Fetch(ctx context.Context, o Options) ([]byte, error) {
	if o.Key == "" {
		return nil, errors.New("envvault: missing Key (ENVVAULT_KEY)")
	}
	raw, teamID, err := FetchEnvelope(ctx, o)
	if err != nil {
		return nil, err
	}
	return Open(raw, o.Key, teamID, o.Secret)
}

// Load fetches and decrypts the secret and parses it as environment variables.
func Load(ctx context.Context, o Options) (map[string]string, error) {
	plain, err := Fetch(ctx, o)
	if err != nil {
		return nil, err
	}
	return Parse(plain)
}

// Setenv loads a secret into the current process. Existing variables are kept
// unless overwrite is true. The key and token are removed afterwards so child
// processes do not inherit them.
func Setenv(ctx context.Context, o Options, overwrite bool) ([]string, error) {
	vars, err := Load(ctx, o)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(vars))
	for name, value := range vars {
		if _, exists := os.LookupEnv(name); exists && !overwrite {
			continue
		}
		if err := os.Setenv(name, value); err != nil {
			return names, fmt.Errorf("envvault: unable to set %s", name)
		}
		names = append(names, name)
	}
	_ = os.Unsetenv("ENVVAULT_KEY")
	_ = os.Unsetenv("ENVVAULT_TOKEN")
	return names, nil
}

// FetchEnvelope downloads the ciphertext and the team ID used in its AAD.
// It does not decrypt the response.
func FetchEnvelope(ctx context.Context, o Options) ([]byte, string, error) {
	if o.URL == "" {
		o.URL = DefaultURL
	}
	if err := o.check(); err != nil {
		return nil, "", err
	}

	endpoint := strings.TrimRight(o.URL, "/") + "/v1/api/" + url.PathEscape(o.Vault) + "/" + url.PathEscape(o.Secret)
	timeout := o.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", errors.New("envvault: invalid URL")
	}
	req.Header.Set("Authorization", "Bearer "+o.Token)
	req.Header.Set("Accept", "application/json")

	client := o.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("envvault: fetch failed: %w", redact(err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxEnvelopeBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("envvault: reading response: %w", redact(err))
	}
	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(body, &apiErr)
		return nil, "", &HTTPError{StatusCode: resp.StatusCode, Code: apiErr.Error.Code}
	}
	if len(body) > MaxEnvelopeBytes {
		return nil, "", errors.New("envvault: envelope too large")
	}
	teamID := resp.Header.Get(TeamIDHeader)
	if teamID == "" {
		return nil, "", errors.New("envvault: response missing " + TeamIDHeader)
	}
	return body, teamID, nil
}

func (o Options) check() error {
	var missing []string
	if o.Token == "" {
		missing = append(missing, "Token (ENVVAULT_TOKEN)")
	}
	if o.Vault == "" {
		missing = append(missing, "Vault (ENVVAULT_VAULT)")
	}
	if o.Secret == "" {
		missing = append(missing, "Secret (ENVVAULT_SECRET)")
	}
	if len(missing) > 0 {
		return errors.New("envvault: missing " + strings.Join(missing, ", "))
	}
	if !strings.HasPrefix(o.Token, TokenPrefix) {
		return errors.New("envvault: token must be the vault token (mvt_...) from the Account URL page")
	}
	u, err := url.Parse(o.URL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return errors.New("envvault: URL must be an http(s) origin, e.g. https://www.envvault.net")
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" || (u.Path != "" && u.Path != "/") {
		return errors.New("envvault: URL must be an origin without credentials, a path, query string or fragment")
	}
	return nil
}

func redact(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}

type envelope struct {
	V          int    `json:"v"`
	Alg        string `json:"alg"`
	IV         string `json:"iv"`
	Ciphertext string `json:"ciphertext"`
	AAD        string `json:"aad"`
}

// AAD returns the authenticated data bound to an EnvVault envelope.
func AAD(teamID, secret string) string {
	return "envvault:v1:" + teamID + ":" + secret
}

// Open validates and decrypts an EnvVault AES-256-GCM envelope.
func Open(raw []byte, key, teamID, secret string) ([]byte, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, ErrEnvelope
	}
	for name := range fields {
		switch name {
		case "v", "alg", "iv", "ciphertext", "aad":
		default:
			return nil, ErrEnvelope
		}
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, ErrEnvelope
	}
	if env.V != 1 || env.Alg != "AES-256-GCM" || env.AAD != AAD(teamID, secret) {
		return nil, ErrEnvelope
	}

	rawKey, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(key))
	if err != nil || len(rawKey) != 32 {
		return nil, ErrInvalidKey
	}
	iv, err := base64.RawURLEncoding.DecodeString(env.IV)
	if err != nil || len(iv) != 12 {
		return nil, ErrEnvelope
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(env.Ciphertext)
	if err != nil || len(ciphertext) < 16 {
		return nil, ErrEnvelope
	}

	block, err := aes.NewCipher(rawKey)
	if err != nil {
		return nil, ErrInvalidKey
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrInvalidKey
	}
	plain, err := gcm.Open(nil, iv, ciphertext, []byte(env.AAD))
	if err != nil {
		return nil, ErrWrongKey
	}
	return plain, nil
}

// Parse reads KEY=value lines. Blank lines and comments are ignored; the value
// is everything after the first equals sign. Errors reveal only the line number.
func Parse(plain []byte) (map[string]string, error) {
	out := map[string]string{}
	for i, line := range strings.Split(string(plain), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		if !ok || name == "" || strings.ContainsAny(name, " \t") {
			return nil, fmt.Errorf("%w (line %d)", ErrInvalidEnv, i+1)
		}
		out[name] = value
	}
	return out, nil
}
