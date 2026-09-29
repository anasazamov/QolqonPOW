package pow

import (
	"encoding/hex"
	"encoding/json"
	"math"
	"math/big"
	"os"
	"sync"
	"testing"
)

var (
	setupOnce sync.Once
	tCache    *Cache
	tData     *Dataset
)

func setup(t testing.TB) (*Cache, *Dataset) {
	t.Helper()
	setupOnce.Do(func() {
		tCache = NewCache(H256("test-epoch-key"), Test)
		tData = NewDataset(tCache, 0)
	})
	return tCache, tData
}

func testHeader() []byte {
	h := make([]byte, 80)
	for i := range h {
		h[i] = byte(i * 7)
	}
	return h
}

// Bir xil kirish -> bir xil natija, ikki mustaqil Hasher'da ham.
func TestDeterministic(t *testing.T) {
	_, ds := setup(t)
	a := NewHasher(Test, ds)
	b := NewHasher(Test, ds)
	for n := uint64(0); n < 20; n++ {
		p1, m1 := a.Hash(testHeader(), n)
		p2, m2 := b.Hash(testHeader(), n)
		if p1 != p2 || m1 != m2 {
			t.Fatalf("nonce %d: natijalar farq qiladi", n)
		}
	}
}

// Light rejim (datasetsiz) va fast rejim bir xil natija beradi.
func TestLightEqualsFast(t *testing.T) {
	c, ds := setup(t)
	fast := NewHasher(Test, ds)
	light := NewHasher(Test, NewLight(c))
	for n := uint64(0); n < 10; n++ {
		p1, m1 := fast.Hash(testHeader(), n)
		p2, m2 := light.Hash(testHeader(), n)
		if p1 != p2 || m1 != m2 {
			t.Fatalf("nonce %d: light != fast", n)
		}
	}
}

// ProgPoW'dagi Kik xatosiga qarshi: header yoki nonce'ning HAR BIR biti
// xotira-og'ir ishni (mix_commit) o'zgartirishi kerak. Aks holda bitta mix'ni
// qayta ishlatib, arzon bosqichda saralash mumkin bo'lardi.
func TestEveryInputBitChangesMemoryWork(t *testing.T) {
	_, ds := setup(t)
	hs := NewHasher(Test, ds)
	hdr := testHeader()
	const nonce = 12345
	_, base := hs.Hash(hdr, nonce)
	for i := 0; i < len(hdr)*8; i++ {
		h2 := append([]byte(nil), hdr...)
		h2[i/8] ^= 1 << (i % 8)
		if _, m := hs.Hash(h2, nonce); m == base {
			t.Fatalf("header biti %d mix'ga ta'sir qilmadi", i)
		}
	}
	for i := 0; i < 64; i++ {
		if _, m := hs.Hash(hdr, nonce^(1<<i)); m == base {
			t.Fatalf("nonce biti %d mix'ga ta'sir qilmadi", i)
		}
	}
}

// Qalbaki mix: prefiltrdan o'tadigan mix'ni arzon usulda topish mumkin, lekin
// to'liq tekshiruv uni rad etadi.
func TestForgedMixRejected(t *testing.T) {
	_, ds := setup(t)
	hs := NewHasher(Test, ds)
	target := TargetFromBits(6)
	hdr := testHeader()
	var fake [32]byte
	var nonce uint64
	for ; ; nonce++ {
		fake = H256("fake", u64b(nonce))
		if Prefilter(hdr, nonce, fake, target) {
			break
		}
	}
	if err := hs.VerifyShare(hdr, nonce, fake, target); err != ErrMix {
		t.Fatalf("qalbaki mix qabul qilindi: %v", err)
	}
	var junk [32]byte
	if Prefilter(hdr, 0, junk, TargetFromBits(40)) {
		t.Fatal("tasodifiy mix 40-bit prefiltrdan o'tdi")
	}
}

// To'liq blok: mining -> VerifyBlock muvaffaqiyatli; noto'g'ri sir rad etiladi.
func TestMineAndVerifyBlock(t *testing.T) {
	_, ds := setup(t)
	hs := NewHasher(Test, ds)
	const k = 3
	blockTarget := TargetFromBits(8)
	secret := H256("pool-secret")
	commit := BlindCommit(secret)
	hdr := append(testHeader(), commit[:]...) // commit coinbase orqali header'ga bog'langan
	shareT := ShareTarget(blockTarget, k)
	for n := uint64(0); n < 1<<20; n++ {
		pow, mix := hs.Hash(hdr, n)
		if !MeetsTarget(pow, shareT) || !BlindOK(pow, secret, k) {
			continue
		}
		if err := hs.VerifyBlock(hdr, n, mix, blockTarget, k, secret, commit); err != nil {
			t.Fatalf("to'g'ri blok rad etildi: %v", err)
		}
		wrong := H256("boshqa-sir")
		if err := hs.VerifyBlock(hdr, n, mix, blockTarget, k, wrong, commit); err != ErrBlind {
			t.Fatalf("noto'g'ri sir qabul qilindi: %v", err)
		}
		return
	}
	t.Fatal("blok topilmadi")
}

// Blind-share umumiy qiyinlikni o'zgartirmaydi: BlindOK ehtimoli 2^-k.
func TestBlindProbability(t *testing.T) {
	const k, n = 4, 200000
	secret := H256("s")
	hits := 0
	for i := 0; i < n; i++ {
		if BlindOK(H256("p", u64b(uint64(i))), secret, k) {
			hits++
		}
	}
	got, want := float64(hits)/n, 1.0/16
	if math.Abs(got-want) > 0.004 {
		t.Fatalf("BlindOK ehtimoli %.4f, kutilgan %.4f", got, want)
	}
}

func TestASERT(t *testing.T) {
	anchor := new(big.Int).Lsh(big.NewInt(1), 220)
	const T, hl = 120, 86400
	limit := maxTarget
	// Jadval bo'yicha: target o'zgarmaydi.
	if got := NextTarget(anchor, T*100, 99, T, hl, limit); got.Cmp(anchor) != 0 {
		t.Fatalf("jadvalda target o'zgardi: %v", got)
	}
	// Bir halflife kechikish -> target 2 barobar (qiyinlik yarmi).
	want2 := new(big.Int).Lsh(anchor, 1)
	if got := NextTarget(anchor, T*100+hl, 99, T, hl, limit); got.Cmp(want2) != 0 {
		t.Fatalf("halflife kechikishda 2x emas: %v", got)
	}
	// Bir halflife oldinda -> target yarmi.
	want5 := new(big.Int).Rsh(anchor, 1)
	if got := NextTarget(anchor, T*100-hl, 99, T, hl, limit); got.Cmp(want5) != 0 {
		t.Fatalf("halflife oldinda 0.5x emas: %v", got)
	}
	// Kasr qismi haqiqiy 2^x ga yaqin (kubik yaqinlashuv xatosi < 0.02%).
	for _, dt := range []int64{-50000, -12345, 777, 30000, 60000} {
		got, _ := new(big.Float).SetInt(NextTarget(anchor, T*100+dt, 99, T, hl, limit)).Float64()
		a, _ := new(big.Float).SetInt(anchor).Float64()
		ideal := a * math.Pow(2, float64(dt)/hl)
		if math.Abs(got/ideal-1) > 2e-4 {
			t.Fatalf("dt=%d: xato %.6f", dt, got/ideal-1)
		}
	}
}

func TestTimestampRules(t *testing.T) {
	prev := []int64{100, 220, 340, 460, 580, 700, 820, 940, 1060, 1180, 1300}
	if ValidTimestamp(700, prev, 2000, 360) {
		t.Fatal("MTP dan kichik timestamp qabul qilindi")
	}
	if !ValidTimestamp(1420, prev, 1400, 360) {
		t.Fatal("to'g'ri timestamp rad etildi")
	}
	if ValidTimestamp(1800, prev, 1400, 360) {
		t.Fatal("FTL dan uzoq kelajak qabul qilindi")
	}
}

func TestTieBreakOrderIndependent(t *testing.T) {
	for i := 0; i < 100; i++ {
		a, b := H256("a", u64b(uint64(i))), H256("b", u64b(uint64(i)))
		if TieBreak(a, b) != TieBreak(b, a) {
			t.Fatal("tie-break tartibga bog'liq")
		}
	}
}

func TestMMR(t *testing.T) {
	var m1, m2 MMR
	for i := 0; i < 37; i++ {
		m1.Append(H256("h", u64b(uint64(i))))
		m2.Append(H256("h", u64b(uint64(i))))
	}
	if m1.Root() != m2.Root() {
		t.Fatal("MMR deterministik emas")
	}
	m2.Append(H256("x"))
	if m1.Root() == m2.Root() {
		t.Fatal("yangi header ildizni o'zgartirmadi")
	}
}

// Test vektorlari: boshqa implementatsiyalar (C, Rust, GPU miner) shu
// qiymatlarni aynan qaytarishi kerak.
type vector struct {
	Header string `json:"header"`
	Nonce  uint64 `json:"nonce"`
	Mix    string `json:"mix"`
	Pow    string `json:"pow"`
}

func TestVectors(t *testing.T) {
	raw, err := os.ReadFile("../testdata/vectors_test_params.json")
	if err != nil {
		t.Skip("test vektorlari hali yaratilmagan: go run ./cmd/qalqon vectors")
	}
	var vs []vector
	if err := json.Unmarshal(raw, &vs); err != nil {
		t.Fatal(err)
	}
	_, ds := setup(t)
	hs := NewHasher(Test, ds)
	for i, v := range vs {
		hdr, _ := hex.DecodeString(v.Header)
		pow, mix := hs.Hash(hdr, v.Nonce)
		if hex.EncodeToString(pow[:]) != v.Pow || hex.EncodeToString(mix[:]) != v.Mix {
			t.Fatalf("vektor %d mos emas", i)
		}
	}
}

func BenchmarkHashTest(b *testing.B) {
	_, ds := setup(b)
	hs := NewHasher(Test, ds)
	hdr := testHeader()
	for i := 0; b.Loop(); i++ {
		hs.Hash(hdr, uint64(i))
	}
}
