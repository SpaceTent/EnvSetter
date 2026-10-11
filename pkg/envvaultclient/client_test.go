package envvaultclient

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	vars, err := Parse([]byte("# comment\nA=one\nB=two=three\nEMPTY=\n"))
	if err != nil {
		t.Fatal(err)
	}
	if vars["A"] != "one" || vars["B"] != "two=three" || vars["EMPTY"] != "" {
		t.Fatalf("vars = %#v", vars)
	}
}

func TestParseErrorDoesNotRevealValue(t *testing.T) {
	_, err := Parse([]byte("GOOD=yes\nbad name=do-not-print-me\n"))
	if !errors.Is(err, ErrInvalidEnv) {
		t.Fatalf("error = %v", err)
	}
	if got := err.Error(); got != "envvault: decrypted file is not a valid ENV file (line 2)" {
		t.Fatalf("error = %q", got)
	}
}

func TestOptionsRejectCredentialInURL(t *testing.T) {
	err := (Options{URL: "https://user:pass@example.com", Token: "mvt_test", Key: "key", Vault: "vault", Secret: "secret"}).check()
	if err == nil {
		t.Fatal("expected URL validation error")
	}
}

func TestOptionsRejectPathInOrigin(t *testing.T) {
	err := (Options{URL: "https://example.com/base", Token: "mvt_test", Key: "key", Vault: "vault", Secret: "secret"}).check()
	if err == nil {
		t.Fatal("expected URL validation error")
	}
}

func TestOpenRejectsUnknownEnvelopeField(t *testing.T) {
	_, err := Open([]byte(`{"v":1,"alg":"AES-256-GCM","iv":"","ciphertext":"","aad":"envvault:v1:t:s","extra":true}`), "key", "t", "s")
	if !errors.Is(err, ErrEnvelope) {
		t.Fatalf("error = %v", err)
	}
}

func TestFetchSendsTokenAndDecryptsLocally(t *testing.T) {
	key, rawKey := testKey(t)
	token := "mvt_testtoken"
	teamID := "team_test"
	secret := "production-api"
	plain := "API_KEY=not-logged\n"
	envelope := testEnvelope(t, rawKey, teamID, secret, plain)

	client := handlerClient(t, func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.String() != "https://envvault.test/v1/api/my-vault/production-api" {
			t.Errorf("request = %s %s", request.Method, request.URL)
		}
		if request.Header.Get("Authorization") != "Bearer "+token {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		if strings.Contains(request.URL.String(), token) || strings.Contains(request.URL.String(), key) {
			t.Error("URL disclosed a credential")
		}
		w.Header().Set(TeamIDHeader, teamID)
		_, _ = w.Write(envelope)
	})

	got, err := Fetch(context.Background(), Options{
		URL:        "https://envvault.test",
		Token:      token,
		Key:        key,
		Vault:      "my-vault",
		Secret:     secret,
		HTTPClient: client,
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != plain {
		t.Fatalf("plaintext = %q", got)
	}
}

func TestFetchReturnsStructuredHTTPError(t *testing.T) {
	client := handlerClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthenticated"}}`))
	})
	_, err := Fetch(context.Background(), Options{
		URL:        "https://envvault.test",
		Token:      "mvt_testtoken",
		Key:        strings.Repeat("x", 43),
		Vault:      "my-vault",
		Secret:     "production-api",
		HTTPClient: client,
	})
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusUnauthorized || httpErr.Code != "unauthenticated" {
		t.Fatalf("error = %#v", err)
	}
}

func TestFetchEnvelopeDoesNotRequireDecryptionKey(t *testing.T) {
	client := handlerClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(TeamIDHeader, "team_test")
		_, _ = w.Write([]byte(`{"ciphertext":"still-encrypted"}`))
	})
	body, teamID, err := FetchEnvelope(context.Background(), Options{
		URL:        "https://envvault.test",
		Token:      "mvt_testtoken",
		Vault:      "my-vault",
		Secret:     "production-api",
		HTTPClient: client,
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"ciphertext":"still-encrypted"}` || teamID != "team_test" {
		t.Fatalf("body=%s teamID=%q", body, teamID)
	}
}

func TestOpenWrongKeyDoesNotRevealPlaintext(t *testing.T) {
	_, rawKey := testKey(t)
	otherKey, _ := testKey(t)
	plain := "SUPER_SECRET_VALUE=never-print-this\n"
	envelope := testEnvelope(t, rawKey, "team_test", "production-api", plain)
	_, err := Open(envelope, otherKey, "team_test", "production-api")
	if !errors.Is(err, ErrWrongKey) {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(err.Error(), "SUPER_SECRET_VALUE") || strings.Contains(err.Error(), otherKey) {
		t.Fatal("error disclosed secret material")
	}
}

func TestFromEnvKeepsExplicitValues(t *testing.T) {
	t.Setenv("ENVVAULT_URL", "https://from-env.test")
	t.Setenv("ENVVAULT_TOKEN", "mvt_from_env")
	t.Setenv("ENVVAULT_KEY", "key-from-env")
	t.Setenv("ENVVAULT_VAULT", "vault-from-env")
	t.Setenv("ENVVAULT_SECRET", "secret-from-env")
	options := FromEnv(Options{URL: "https://explicit.test", Vault: "explicit-vault"})
	if options.URL != "https://explicit.test" || options.Vault != "explicit-vault" || options.Token != "mvt_from_env" || options.Key != "key-from-env" || options.Secret != "secret-from-env" {
		t.Fatalf("options = %#v", options)
	}
}

func handlerClient(t *testing.T, handler http.HandlerFunc) *http.Client {
	t.Helper()
	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		return recorder.Result(), nil
	})}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func testKey(t *testing.T) (string, []byte) {
	t.Helper()
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), raw
}

func testEnvelope(t *testing.T, key []byte, teamID, secret, plain string) []byte {
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
	aad := AAD(teamID, secret)
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
