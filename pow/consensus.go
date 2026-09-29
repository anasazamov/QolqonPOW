package pow

import (
	"bytes"
	"math/big"
	"math/bits"
	"sort"
)

var (
	two256    = new(big.Int).Lsh(big.NewInt(1), 256)
	maxTarget = new(big.Int).Sub(two256, big.NewInt(1))
)

// MeetsTarget: hash big-endian 256-bitli son sifatida target'dan kichikmi.
func MeetsTarget(h [32]byte, target *big.Int) bool {
	return new(big.Int).SetBytes(h[:]).Cmp(target) < 0
}

// TargetFromBits: 2^(256-zeroBits) — "zeroBits ta boshlang'ich nol bit" qiyinligi.
func TargetFromBits(zeroBits uint) *big.Int {
	return new(big.Int).Lsh(big.NewInt(1), 256-zeroBits)
}

// ---------------- Blind-share ----------------

// ShareTarget = blockTarget * 2^k (2^256-1 bilan cheklangan).
func ShareTarget(blockTarget *big.Int, k uint) *big.Int {
	t := new(big.Int).Lsh(blockTarget, k)
	if t.Cmp(maxTarget) > 0 {
		return new(big.Int).Set(maxTarget)
	}
	return t
}

func BlindCommit(secret [32]byte) [32]byte { return h256(tagBlind, secret[:]) }

// BlindOK: BLAKE3(luck || pow || secret) ning dastlabki k biti nolmi.
// Ehtimollik 2^-k; sirni bilmagan miner buni hisoblay olmaydi.
func BlindOK(pow, secret [32]byte, k uint) bool {
	h := h256(tagLuck, pow[:], secret[:])
	return leadingZeros(h) >= k
}

func leadingZeros(h [32]byte) uint {
	var n uint
	for _, b := range h {
		if b != 0 {
			return n + uint(bits.LeadingZeros8(b))
		}
		n += 8
	}
	return n
}

// ---------------- ASERT (aserti3-2d) ----------------

// NextTarget — BCHN aserti3-2d formulasi, faqat butun sonli arifmetika.
// timeDelta = ts_parent - ts_anchor_parent, heightDelta = h_parent - h_anchor.
func NextTarget(anchorTarget *big.Int, timeDelta, heightDelta, spacing, halflife int64, powLimit *big.Int) *big.Int {
	exponent := ((timeDelta - spacing*(heightDelta+1)) * 65536) / halflife
	shifts := exponent >> 16
	frac := new(big.Int).SetUint64(uint64(uint16(exponent)))

	// factor = 65536 + ((195766423245049*f + 971821376*f^2 + 5127*f^3 + 2^47) >> 48)
	f2 := new(big.Int).Mul(frac, frac)
	f3 := new(big.Int).Mul(f2, frac)
	poly := new(big.Int).Mul(big.NewInt(195766423245049), frac)
	poly.Add(poly, new(big.Int).Mul(big.NewInt(971821376), f2))
	poly.Add(poly, new(big.Int).Mul(big.NewInt(5127), f3))
	poly.Add(poly, new(big.Int).Lsh(big.NewInt(1), 47))
	poly.Rsh(poly, 48)
	factor := poly.Add(poly, big.NewInt(65536))

	next := new(big.Int).Mul(anchorTarget, factor)
	shifts -= 16
	if shifts <= 0 {
		next.Rsh(next, uint(-shifts))
	} else {
		next.Lsh(next, uint(shifts))
	}
	if next.Sign() == 0 {
		return big.NewInt(1)
	}
	if next.Cmp(powLimit) > 0 {
		return new(big.Int).Set(powLimit)
	}
	return next
}

// ValidTimestamp: ts > MTP(oxirgi 11 blok) va ts <= mahalliy vaqt + FTL.
func ValidTimestamp(ts int64, prev []int64, localTime, ftl int64) bool {
	if len(prev) > 11 {
		prev = prev[len(prev)-11:]
	}
	if len(prev) > 0 {
		s := append([]int64(nil), prev...)
		sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
		if ts <= s[len(s)/2] {
			return false
		}
	}
	return ts <= localTime+ftl
}

// ---------------- Fork tanlash ----------------

// Work = 2^256 / (target + 1).
func Work(target *big.Int) *big.Int {
	return new(big.Int).Div(two256, new(big.Int).Add(target, big.NewInt(1)))
}

// TieBreak: bir xil ishli ikki blokdan qaysi biri g'olib. Natija bloklar
// qaysi tartibda kelganiga bog'liq emas ("birinchi ko'rgan" qoidasi yo'q).
func TieBreak(a, b [32]byte) [32]byte {
	lo, hi := a, b
	if bytes.Compare(a[:], b[:]) > 0 {
		lo, hi = b, a
	}
	if h256(tagTie, lo[:], hi[:])[0]&1 == 0 {
		return lo
	}
	return hi
}

// ---------------- MMR (FlyClient) ----------------

// MMR — header hashlarining Merkle Mountain Range'i.
type MMR struct {
	peaks   [][32]byte
	heights []int
	size    uint64
}

func (m *MMR) Append(leaf [32]byte) {
	node := h256(tagMMRL, leaf[:])
	h := 0
	for n := len(m.peaks); n > 0 && m.heights[n-1] == h; n = len(m.peaks) {
		node = h256(tagMMRN, m.peaks[n-1][:], node[:])
		m.peaks, m.heights = m.peaks[:n-1], m.heights[:n-1]
		h++
	}
	m.peaks = append(m.peaks, node)
	m.heights = append(m.heights, h)
	m.size++
}

func (m *MMR) Root() [32]byte {
	if len(m.peaks) == 0 {
		return [32]byte{}
	}
	acc := m.peaks[len(m.peaks)-1]
	for i := len(m.peaks) - 2; i >= 0; i-- {
		acc = h256(tagMMRBag, m.peaks[i][:], acc[:])
	}
	return h256(tagMMRRt, u64b(m.size), acc[:])
}

func (m *MMR) Size() uint64 { return m.size }
