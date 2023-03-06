package token

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"

	"github.com/google/uuid"
)

const secretLen = 16 // secret len.

const purgePeriod = 300 * time.Second

type tokenMeta struct {
	jti    string
	exp    time.Time
	secret []byte
}

type tokenStorage struct {
	sync.Mutex
	m    map[string]*tokenMeta
	last time.Time // last purge
}

var storage = &tokenStorage{
	m:    make(map[string]*tokenMeta),
	last: time.Now().Add(purgePeriod),
}

func (ts *tokenStorage) put(t *tokenMeta) {
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

func (ts *tokenStorage) get(jti string) *tokenMeta {
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

	ts.last = now.Add(purgePeriod)
}

func newToken(ttl int) (*tokenMeta, error) {
	buf := make([]byte, secretLen)
	if _, err := rand.Reader.Read(buf); err != nil {
		return nil, err
	}

	secret := make([]byte, base64.StdEncoding.EncodedLen(secretLen))
	base64.StdEncoding.Encode(secret, buf)

	token := &tokenMeta{
		exp:    time.Now().Add(time.Second * time.Duration(ttl)),
		secret: secret,
	}

	storage.put(token)

	return token, nil
}

func fetchSecret(jti string) []byte {
	token := storage.get(jti)
	if token == nil {
		return []byte{}
	}

	return token.secret
}
