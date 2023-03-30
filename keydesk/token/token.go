package token

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"

	"github.com/google/uuid"
)

const tokenSecretLen = 16 // secret len.

const tokenPurgePeriod = 300 * time.Second

// Config - jwt token.
type Config struct {
	jti    string
	exp    time.Time
	secret []byte
}

// Jti - jti getter.
func (tc *Config) Jti() string {
	return tc.jti
}

// Exp - exp getter.
func (tc *Config) Exp() time.Time {
	return tc.exp
}

// Secret - secret getter.
func (tc *Config) Secret() []byte {
	return bytes.Clone(tc.secret)
}

type tokenStorage struct {
	sync.Mutex
	m    map[string]*Config
	last time.Time // last purge
}

var tokens = &tokenStorage{
	m:    make(map[string]*Config),
	last: time.Now().Add(tokenPurgePeriod),
}

func (ts *tokenStorage) put(t *Config) {
	ts.Lock()
	defer ts.Unlock()

	for {
		jti := uuid.New().String()
		v, ok := ts.m[jti]
		if ok && v.exp.Before(time.Now()) {
			continue
		}

		if ok {
			delete(ts.m, jti)
		}

		t.jti = jti
		ts.m[jti] = t

		break
	}
}

func (ts *tokenStorage) get(jti string) *Config {
	ts.Lock()
	defer ts.Unlock()

	now := time.Now()

	ts._purge(now)

	v, ok := ts.m[jti]
	if ok && v.exp.Before(now) {
		delete(ts.m, jti)

		return nil
	}

	return v
}

func (ts *tokenStorage) _purge(now time.Time) {
	if ts.last.Before(now) {
		return
	}

	for jti, v := range ts.m {
		if v.exp.Before(now) {
			delete(ts.m, jti)
		}
	}

	ts.last = now.Add(tokenPurgePeriod)
}

// New - new jwt token
func New(ttl int) (*Config, error) {
	buf := make([]byte, tokenSecretLen)
	if _, err := rand.Reader.Read(buf); err != nil {
		return nil, err
	}

	secret := make([]byte, base64.StdEncoding.EncodedLen(tokenSecretLen))
	base64.StdEncoding.Encode(secret, buf)

	token := &Config{
		exp:    time.Now().Add(time.Second * time.Duration(ttl)),
		secret: secret,
	}

	tokens.put(token)

	return token, nil
}

// FetchSecret - fetch token fron storage.
func FetchSecret(jti string) []byte {
	token := tokens.get(jti)
	if token == nil {
		return []byte{}
	}

	return token.secret
}
