package util

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/AdguardTeam/golibs/log"
	"golang.org/x/sys/cpu"
)

// LoadSystemRootCAs - load root CAs from the system
// Return the x509 certificate pool object
// Return nil if nothing has been found.
//  This means that Go.crypto will use its default algorithm to find system root CA list.
// https://github.com/AdguardTeam/AdGuardHome/internal/issues/1311
func LoadSystemRootCAs() *x509.CertPool {
	if runtime.GOOS != "linux" {
		return nil
	}

	// Directories with the system root certificates, that aren't supported by Go.crypto
	dirs := []string{
		"/opt/etc/ssl/certs", // Entware
	}
	roots := x509.NewCertPool()
	for _, dir := range dirs {
		fis, err := ioutil.ReadDir(dir)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				log.Error("opening directory: %q: %s", dir, err)
			}

			continue
		}

		rootsAdded := false
		for _, fi := range fis {
			var certData []byte
			certData, err = ioutil.ReadFile(dir + "/" + fi.Name())
			if err == nil && roots.AppendCertsFromPEM(certData) {
				rootsAdded = true
			}
		}

		if rootsAdded {
			return roots
		}
	}

	return nil
}

// InitTLSCiphers - the same as initDefaultCipherSuites() from src/crypto/tls/common.go
//  but with the difference that we don't use so many other default ciphers.
func InitTLSCiphers() []uint16 {
	var ciphers []uint16

	// Check the cpu flags for each platform that has optimized GCM implementations.
	// Worst case, these variables will just all be false.
	var (
		hasGCMAsmAMD64 = cpu.X86.HasAES && cpu.X86.HasPCLMULQDQ
		hasGCMAsmARM64 = cpu.ARM64.HasAES && cpu.ARM64.HasPMULL
		// Keep in sync with crypto/aes/cipher_s390x.go.
		hasGCMAsmS390X = cpu.S390X.HasAES && cpu.S390X.HasAESCBC && cpu.S390X.HasAESCTR && (cpu.S390X.HasGHASH || cpu.S390X.HasAESGCM)

		hasGCMAsm = hasGCMAsmAMD64 || hasGCMAsmARM64 || hasGCMAsmS390X
	)

	if hasGCMAsm {
		// If AES-GCM hardware is provided then prioritise AES-GCM
		// cipher suites.
		ciphers = []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		}
	} else {
		// Without AES-GCM hardware, we put the ChaCha20-Poly1305
		// cipher suites first.
		ciphers = []uint16{
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		}
	}

	otherCiphers := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
	}
	ciphers = append(ciphers, otherCiphers...)
	return ciphers
}
