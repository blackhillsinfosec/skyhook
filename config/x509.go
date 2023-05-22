package config

import (
	"golang.org/x/exp/rand"
	"math/big"
	"time"
)

var (
	rando *rand.Rand
)

func init() {
	rando = rand.New(rand.NewSource(uint64(time.Now().Unix())))
}

// GenerateX509Config is the config structure used
// to generate X509 certificates.
type GenerateX509Config struct {
	Subject      *X509Subject
	Timeline     *X509Timeline
	SerialNumber *big.Int `yaml:"serial_number"`
}

func DefaultX509Config() *GenerateX509Config {
	return &GenerateX509Config{
		Subject:      DefaultX509Subject(),
		Timeline:     DefaultX509Timeline(),
		SerialNumber: big.NewInt(rando.Int63()),
	}
}

// X509Timeline represents the not before and not after
// elements of an X509 certificate.
type X509Timeline struct {
	NotBefore time.Time `yaml:"not_before"`
	NotAfter  time.Time `yaml:"not_after"`
}

func DefaultX509Timeline() *X509Timeline {
	return &X509Timeline{
		NotBefore: time.Time{},
		NotAfter:  time.Time{},
	}
}

// X509Subject is the subject field of an X509 certificate.
type X509Subject struct {
	Country            string
	Province           string
	Locality           string
	Organization       string
	OrganizationalUnit string `yaml:"organizational_unit"`
	CommonName         string `yaml:"common_name"`
}

// DefaultX509Subject generates the default X509 subject for
// generating certificates.
func DefaultX509Subject() *X509Subject {
	return &X509Subject{
		Country:            "US",
		Province:           "South Dakota",
		Locality:           "Spearphish",
		Organization:       "Rando Widgets",
		OrganizationalUnit: "R&D",
		CommonName:         "innocuous.domain.com",
	}
}
