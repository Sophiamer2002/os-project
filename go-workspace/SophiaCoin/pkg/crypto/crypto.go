package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"errors"
	"os"
)

type Key struct {
	privateKey *ecdsa.PrivateKey
}

type PublicKey struct {
	publicKey *ecdsa.PublicKey
}

func NewKey() (*Key, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Key{privateKey: privateKey}, nil
}

func LoadKey(filename string) (*Key, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	b := make([]byte, 121)
	_, err = file.Read(b)
	if err != nil {
		return nil, err
	}
	key, err := x509.ParseECPrivateKey(b)
	return &Key{key}, err
}

func (key *Key) GetPublicKey() *PublicKey {
	return &PublicKey{publicKey: &key.privateKey.PublicKey}
}

func (key *Key) serialize() []byte {
	ret, _ := x509.MarshalECPrivateKey(key.privateKey)
	if len(ret) != 121 {
		panic("Key.Serialize: Invalid serialized key length")
	}
	return ret
}

func (key *Key) SaveKey(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(key.serialize())
	return err
}

func (key *Key) Sign(data []byte) []byte {
	ret, _ := ecdsa.SignASN1(rand.Reader, key.privateKey, data)
	return ret
}

func (pk *PublicKey) Verify(data []byte, signature []byte) bool {
	return ecdsa.VerifyASN1(pk.publicKey, data, signature)
}

func (pk *PublicKey) ToBytes() []byte {
	ret, _ := x509.MarshalPKIXPublicKey(pk.publicKey)
	if len(ret) != 91 {
		panic("PublicKey.Serialize: Invalid serialized public key length")
	}
	return ret
}

func FromBytes(data []byte) (*PublicKey, error) {
	if len(data) != 91 {
		return nil, errors.New("PublicKey.Deserialize: Invalid serialized public key length")
	}
	key, err := x509.ParsePKIXPublicKey(data)
	if err != nil {
		return nil, err
	}
	return &PublicKey{key.(*ecdsa.PublicKey)}, nil
}

func RandBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

func (pk *PublicKey) Equal(other *PublicKey) bool {
	return pk.publicKey.Equal(other.publicKey)
}
