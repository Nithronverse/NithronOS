package hash

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Default Argon2id parameters
const (
	defaultTime    uint32 = 3
	defaultMemory  uint32 = 64 * 1024 // 64 MB in KiB
	defaultThreads uint8  = 1
	defaultSaltLen uint32 = 16
	defaultKeyLen  uint32 = 32
	phcAlg                = "argon2id"
	phcVersion            = 19
)

// HashPassword derives an Argon2id hash and returns a PHC-formatted string:
// $argon2id$v=19$m=65536,t=3,p=1$<saltB64>$<hashB64>
func HashPassword(plain string) (string, error) {
	salt := make([]byte, defaultSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	sum := argon2.IDKey([]byte(plain), salt, defaultTime, defaultMemory, defaultThreads, defaultKeyLen)
	// PHC with unpadded base64 (RawStdEncoding)
	p := fmt.Sprintf("$%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
		phcAlg, phcVersion, defaultMemory, defaultTime, defaultThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(sum),
	)
	return p, nil
}

// VerifyPassword parses the PHC string and verifies the supplied plain text.
// It accepts parameter changes encoded in the PHC (m, t, p) and performs
// a constant-time comparison of the derived hash.
func VerifyPassword(phc, plain string) bool {
	params, salt, sum, err := parsePHC(phc)
	if err != nil {
		return false
	}
	calc := argon2.IDKey([]byte(plain), salt, params.time, params.memory, params.threads, uint32(len(sum)))
	return subtle.ConstantTimeCompare(calc, sum) == 1
}

type phcParams struct {
	time    uint32
	memory  uint32
	threads uint8
}

func parsePHC(phc string) (phcParams, []byte, []byte, error) {
	// Expect 6 parts when splitting by '$': "", alg, v=19, params, salt, hash
	if !strings.HasPrefix(phc, "$") {
		return phcParams{}, nil, nil, errors.New("invalid phc: missing prefix")
	}
	parts := strings.Split(phc, "$")
	if len(parts) < 6 {
		return phcParams{}, nil, nil, errors.New("invalid phc: parts")
	}
	alg := parts[1]
	if alg != phcAlg {
		return phcParams{}, nil, nil, fmt.Errorf("unsupported alg: %s", alg)
	}
	// version part like v=19 (optional but expected)
	if parts[2] != "v=19" {
		if strings.HasPrefix(parts[2], "v=") {
			v, err := strconv.Atoi(strings.TrimPrefix(parts[2], "v="))
			if err != nil || v != phcVersion {
				return phcParams{}, nil, nil, fmt.Errorf("unsupported version: %s", parts[2])
			}
		} else {
			return phcParams{}, nil, nil, errors.New("invalid phc: version")
		}
	}
	// m=...,t=...,p=...
	var pp phcParams
	for _, kv := range strings.Split(parts[3], ",") {
		kvp := strings.SplitN(kv, "=", 2)
		if len(kvp) != 2 {
			continue
		}
		switch kvp[0] {
		case "m":
			if v, err := strconv.ParseUint(kvp[1], 10, 32); err == nil {
				pp.memory = uint32(v)
			}
		case "t":
			if v, err := strconv.ParseUint(kvp[1], 10, 32); err == nil {
				pp.time = uint32(v)
			}
		case "p":
			if v, err := strconv.ParseUint(kvp[1], 10, 8); err == nil {
				pp.threads = uint8(v)
			}
		}
	}
	if pp.memory == 0 || pp.time == 0 || pp.threads == 0 {
		return phcParams{}, nil, nil, errors.New("invalid phc: params")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return phcParams{}, nil, nil, errors.New("invalid phc: salt")
	}
	sum, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(sum) == 0 {
		return phcParams{}, nil, nil, errors.New("invalid phc: hash")
	}
	return pp, salt, sum, nil
}
