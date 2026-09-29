package pow

import (
	"encoding/binary"

	"lukechampine.com/blake3"
)

// Domen ajratish teglari: har bir bosqich o'z prefiksiga ega.
var (
	tagCache  = []byte("QLQ1/cache")
	tagItem   = []byte("QLQ1/item")
	tagSeed   = []byte("QLQ1/seed")
	tagSP     = []byte("QLQ1/sp")
	tagRegs   = []byte("QLQ1/regs")
	tagProg   = []byte("QLQ1/prog")
	tagSPH    = []byte("QLQ1/sph")
	tagMix    = []byte("QLQ1/mix")
	tagPow    = []byte("QLQ1/pow")
	tagBlind  = []byte("QLQ1/blind")
	tagLuck   = []byte("QLQ1/luck")
	tagTie    = []byte("QLQ1/tie")
	tagMMRL   = []byte("QLQ1/mmr-leaf")
	tagMMRN   = []byte("QLQ1/mmr-node")
	tagMMRBag = []byte("QLQ1/mmr-bag")
	tagMMRRt  = []byte("QLQ1/mmr-root")
)

func h256(parts ...[]byte) [32]byte {
	h := blake3.New(32, nil)
	for _, p := range parts {
		h.Write(p)
	}
	var out [32]byte
	h.Sum(out[:0])
	return out
}

func xof(n int, parts ...[]byte) []byte {
	h := blake3.New(32, nil)
	for _, p := range parts {
		h.Write(p)
	}
	out := make([]byte, n)
	h.XOF().Read(out)
	return out
}

func le64(b []byte) uint64     { return binary.LittleEndian.Uint64(b) }
func put64(b []byte, v uint64) { binary.LittleEndian.PutUint64(b, v) }

func u64b(v uint64) []byte {
	var b [8]byte
	put64(b[:], v)
	return b[:]
}

// H256 — tashqi paketlar (simulyatsiya, CLI) uchun domen tegli BLAKE3.
func H256(tag string, parts ...[]byte) [32]byte {
	return h256(append([][]byte{[]byte(tag)}, parts...)...)
}
