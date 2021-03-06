// Copyright 2016 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package p256

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"math"
	"testing"
)

const (
	// openssl ecparam -name prime256v1 -genkey -out p256-key.pem
	privKey = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIGbhE2+z8d5lHzb0gmkS78d86gm5gHUtXCpXveFbK3pcoAoGCCqGSM49
AwEHoUQDQgAEUxX42oxJ5voiNfbjoz8UgsGqh1bD1NXK9m8VivPmQSoYUdVFgNav
csFaQhohkiCEthY51Ga6Xa+ggn+eTZtf9Q==
-----END EC PRIVATE KEY-----`
	// openssl ec -in p256-key.pem -pubout -out p256-pubkey.pem
	pubKey = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEUxX42oxJ5voiNfbjoz8UgsGqh1bD
1NXK9m8VivPmQSoYUdVFgNavcsFaQhohkiCEthY51Ga6Xa+ggn+eTZtf9Q==
-----END PUBLIC KEY-----`
)

func TestH1(t *testing.T) {
	for i := 0; i < 10000; i++ {
		m := make([]byte, 100)
		if _, err := rand.Read(m); err != nil {
			t.Fatalf("Failed generating random message: %v", err)
		}
		x, y := H1(m)
		if x == nil {
			t.Errorf("H1(%v)=%v, want curve point", m, x)
		}
		if got := curve.Params().IsOnCurve(x, y); !got {
			t.Errorf("H1(%v)=[%v, %v], is not on curve", m, x, y)
		}
	}
}

func TestH2(t *testing.T) {
	l := 32
	for i := 0; i < 10000; i++ {
		m := make([]byte, 100)
		if _, err := rand.Read(m); err != nil {
			t.Fatalf("Failed generating random message: %v", err)
		}
		x := H2(m)
		if got := len(x.Bytes()); got < 1 || got > l {
			t.Errorf("len(h2(%v)) = %v, want: 1 <= %v <= %v", m, got, got, l)
		}
	}
}

func TestVRF(t *testing.T) {
	k, pk := GenerateKey()

	m1 := []byte("data1")
	m2 := []byte("data2")
	m3 := []byte("data2")
	index1, proof1 := k.Evaluate(m1)
	index2, proof2 := k.Evaluate(m2)
	index3, proof3 := k.Evaluate(m3)
	for _, tc := range []struct {
		m     []byte
		index [32]byte
		proof []byte
		valid bool
	}{
		{m1, index1, proof1, true},
		{m2, index2, proof2, true},
		{m3, index3, proof3, true},
		{m3, index3, proof2, true},
		{m3, index3, proof1, false},
	} {
		if valid := pk.Verify(tc.m, tc.proof); valid != tc.valid {
			t.Errorf("Verify(%s, %x): %v, want %v", tc.m, tc.proof, valid, tc.valid)
		}
	}
}

func TestVerify(t *testing.T) {
	pk, err := NewVRFVerifierFromPEM([]byte(pubKey))
	if err != nil {
		t.Errorf("NewVRFSigner failure: %v", err)
	}

	for _, tc := range []struct {
		m     []byte
		index [32]byte
		proof []byte
	}{
		{
			m:     []byte("data1"),
			index: h2i("6ed5469b409c4ac0e48151d6db6250b28f6776af0f6eb05aaeb3970f3b72e022"),
			proof: h2b("ceaccb3cfc61954004948f131de6cd689555b3834480221ab9ef103a40a63f7a9b47fe8155512531bc0acf9b2314837c2fc43d24b4b9d98f13aff09b2a7ae8810423835a97b337a06769a47e05e4c0b68bcd499d35e7cf7606283d74e41d59a4bbc5f4af2da3b83b7c7ab76598aecbf495714815eae51016410e961f6153a6c5ea"),
		},
		{
			m:     []byte("data2"),
			index: h2i("ff6743a082fe6ed66dc04e6e11775070ebbe1088b0a378bf97fd84e960c9ed89"),
			proof: h2b("0c39c84e152596e81df4281c5459957b893a7fde2492e0358cc1c8ab891c9a00c74f36c349306e039a3c0f1fcc9e9523ee8d8f29398b68e6c02ddb70b3406f9e0447d0f7c330343720da2ae0959cfd2c3bda9083af475203efb07bcb2e18d12b99abf1a10001d355ae3f9a34c53052a70ff3af03024ad3ada1d188949a707376e6"),
		},
		{
			m:     []byte("data2"),
			index: h2i("ff6743a082fe6ed66dc04e6e11775070ebbe1088b0a378bf97fd84e960c9ed89"),
			proof: h2b("a907df20dcd190c10ab217db1c752ccf12817a221e43e99e6187e3d3848b803b991b7e474c120af45a46698724136a5691c189afdf73ab00033eb491849b44600447d0f7c330343720da2ae0959cfd2c3bda9083af475203efb07bcb2e18d12b99abf1a10001d355ae3f9a34c53052a70ff3af03024ad3ada1d188949a707376e6"),
		},
	} {
		if !pk.Verify(tc.m, tc.proof) {
			t.Errorf("Verify(%s, %x): %v, want %v", tc.m, tc.proof, false, true)
		}
	}
}

func TestReadFromOpenSSL(t *testing.T) {
	for _, tc := range []struct {
		priv string
		pub  string
	}{
		{privKey, pubKey},
	} {
		// Private VRF Key
		signer, err := NewVRFSignerFromPEM([]byte(tc.priv))
		if err != nil {
			t.Errorf("NewVRFSigner failure: %v", err)
		}

		// Public VRF key
		verifier, err := NewVRFVerifierFromPEM([]byte(tc.pub))
		if err != nil {
			t.Errorf("NewVRFSigner failure: %v", err)
		}

		// Evaluate and verify.
		m := []byte("M")
		_, proof := signer.Evaluate(m)
		if !verifier.Verify(m, proof) {
			t.Errorf("Failed verifying VRF proof")
		}
	}
}

func TestRightTruncateProof(t *testing.T) {
	k, pk := GenerateKey()

	data := []byte("data")
	_, proof := k.Evaluate(data)
	proofLen := len(proof)
	for i := 0; i < proofLen; i++ {
		proof = proof[:len(proof)-1]
		if pk.Verify(data, proof) {
			t.Errorf("Verify unexpectedly succeeded after truncating %v bytes from the end of proof", i)
		}
	}
}

func TestLeftTruncateProof(t *testing.T) {
	k, pk := GenerateKey()

	data := []byte("data")
	_, proof := k.Evaluate(data)
	proofLen := len(proof)
	for i := 0; i < proofLen; i++ {
		proof = proof[1:]
		if pk.Verify(data, proof) {
			t.Errorf("Verify unexpectedly succeeded after truncating %v bytes from the beginning of proof", i)
		}
	}
}

func TestBitFlip(t *testing.T) {
	k, pk := GenerateKey()

	data := []byte("data")
	_, proof := k.Evaluate(data)
	for i := 0; i < len(proof)*8; i++ {
		// Flip bit in position i.
		if pk.Verify(data, flipBit(proof, i)) {
			t.Errorf("Verify unexpectedly succeeded after flipping bit %v of vrf", i)
		}
	}
}

func flipBit(a []byte, pos int) []byte {
	index := int(math.Floor(float64(pos) / 8))
	b := byte(a[index])
	b ^= (1 << uint(math.Mod(float64(pos), 8.0)))

	var buf bytes.Buffer
	buf.Write(a[:index])
	buf.Write([]byte{b})
	buf.Write(a[index+1:])
	return buf.Bytes()
}

func TestVectors(t *testing.T) {
	k, err := NewVRFSignerFromPEM([]byte(privKey))
	if err != nil {
		t.Errorf("NewVRFSigner failure: %v", err)
	}
	pk, err := NewVRFVerifierFromPEM([]byte(pubKey))
	if err != nil {
		t.Errorf("NewVRFSigner failure: %v", err)
	}
	for _, tc := range []struct {
		m     []byte
		index [32]byte
	}{
		{
			m:     []byte("test"),
			index: h2i("1af0a7e3d9a96a71be6257cf4ad1a0ffdec57e9959b2eafc4673a6c31241fc9f"),
		},
		{
			m:     nil,
			index: h2i("2ebac3669807f474f4d49891a1d0b2fba8e966f945ac01cbfffb3bb48627e67d"),
		},
	} {
		index, proof := k.Evaluate(tc.m)
		if got, want := index, tc.index; got != want {
			t.Errorf("Evaluate(%s).Index: %x, want %x", tc.m, got, want)
		}
		if !pk.Verify(tc.m, proof) {
			t.Errorf("Verify(%s): %v, want %v", tc.m, false, true)
		}
	}
}

func h2i(h string) [32]byte {
	b, err := hex.DecodeString(h)
	if err != nil {
		panic("Invalid hex")
	}
	var i [32]byte
	copy(i[:], b)
	return i
}

func h2b(h string) []byte {
	b, err := hex.DecodeString(h)
	if err != nil {
		panic("Invalid hex")
	}
	return b
}

func BenchmarkEvaluate(b *testing.B) {
	sk, _ := GenerateKey()
	alice := []byte("alice")
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		sk.Evaluate(alice)
	}
}

func BenchmarkVerify(b *testing.B) {
	sk, pk := GenerateKey()
	alice := []byte("alice")
	_, aliceProof := sk.Evaluate(alice)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		pk.Verify(alice, aliceProof)
	}
}
