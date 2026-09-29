package pow

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"math"
	"math/big"
	"unsafe"
)

// Hasher — bitta goroutine uchun hash hisoblovchi (scratchpad va dastur
// buferlarini qayta ishlatadi).
type Hasher struct {
	p     Params
	src   ItemSource
	sp    []byte
	prog  []instr
	pbuf  []byte
	masks [4]uint64
}

func NewHasher(p Params, src ItemSource) *Hasher {
	hs := &Hasher{
		p:    p,
		src:  src,
		sp:   make([]byte, p.ScratchBytes),
		prog: make([]instr, p.ProgramSize),
		pbuf: make([]byte, p.ProgramSize*instrBytes),
	}
	l1 := min(uint64(16<<10), p.ScratchBytes)
	l2 := min(uint64(256<<10), p.ScratchBytes)
	l3 := p.ScratchBytes
	mask := func(l uint64) uint64 { return (l - 1) &^ 7 }
	hs.masks = [4]uint64{mask(l1), mask(l2), mask(l2), mask(l3)}
	return hs
}

// Seed: nonce birinchi turadi, keyin butun header. Natijaning 256 bitining
// hammasi keyingi bosqichlarga kiradi (ProgPoW'dagi 64-bitli seed xatosi yo'q).
func Seed(header []byte, nonce uint64) [32]byte {
	return h256(tagSeed, u64b(nonce), header)
}

// PowFromMix — arzon yakuniy bosqich. Seed ham qo'shiladi, shuning uchun bitta
// mix'ni boshqa header/nonce bilan qayta ishlatib bo'lmaydi.
func PowFromMix(seed, mix [32]byte) [32]byte {
	return h256(tagPow, seed[:], mix[:])
}

// Hash — header va nonce uchun (pow, mix_commit).
func (hs *Hasher) Hash(header []byte, nonce uint64) (pow, mix [32]byte) {
	return hs.HashSeed(Seed(header, nonce))
}

func serialize(r *[8]uint64, f *[4]float64, ma, mx uint64) []byte {
	b := make([]byte, 8*8+4*8+16)
	for i := 0; i < 8; i++ {
		put64(b[8*i:], r[i])
	}
	for i := 0; i < 4; i++ {
		put64(b[64+8*i:], math.Float64bits(f[i]))
	}
	put64(b[96:], ma)
	put64(b[104:], mx)
	return b
}

// HashSeed — asosiy xotira-og'ir ish.
func (hs *Hasher) HashSeed(seed [32]byte) (pow, mix [32]byte) {
	p := hs.p
	sp := hs.sp

	// 1. Scratchpad: AES-128-CTR (to'liq 10 raund) bilan to'ldiriladi.
	k := h256(tagSP, seed[:])
	blk, err := aes.NewCipher(k[:16])
	if err != nil {
		panic(err)
	}
	clear(sp)
	cipher.NewCTR(blk, k[16:32]).XORKeyStream(sp, sp)

	// 2. Registrlar.
	var r [8]uint64
	var f [4]float64
	init := xof(96, tagRegs, seed[:])
	for i := 0; i < 8; i++ {
		r[i] = le64(init[8*i:])
	}
	for i := 0; i < 4; i++ {
		f[i] = norm(math.Float64frombits(le64(init[64+8*i:])))
	}
	ma, mx := le64(seed[0:8]), le64(seed[8:16])

	nItems := hs.src.Items()
	lineMask := (p.ScratchBytes - 1) &^ 63
	base := unsafe.Pointer(&sp[0])
	var item [64]byte
	state := seed
	ma %= nItems
	hs.src.Prefetch(ma)

	// 3. Har bir nonce uchun yangi tasodifiy dasturlar.
	for prog := 0; prog < p.Programs; prog++ {
		genProgram(state, hs.prog, hs.pbuf)
		hs.compile()
		rA, rB, rC, rD := state[0]&7, state[1]&7, state[2]&7, state[3]&7
		spMix := r[rA] ^ r[rB]
		for it := 0; it < p.Iterations; it++ {
			a0 := spMix & lineMask
			a1 := (spMix >> 32) & lineMask
			for i := uint64(0); i < 8; i++ {
				r[i] ^= ld64(base, a0+8*i)
			}
			for i := uint64(0); i < 4; i++ {
				f[i] = norm(float64(f[i] + float64(int32(ld64(base, a1+8*i)))))
			}
			hs.execute(&r, &f)

			// Dataset (v0.2): bu iteratsiyada ma manzili o'qiladi — u oldingi
			// iteratsiyada hisoblangan va oldindan so'ralgan. Keyingi manzil
			// (mx) hozir hisoblanib, darhol prefetch qilinadi.
			mx ^= r[rC] ^ r[rD]
			next := mx % nItems
			hs.src.Prefetch(next)
			hs.src.Item(ma, &item)
			for i := 0; i < 8; i++ {
				r[i] ^= le64(item[8*i:])
			}
			ma, mx = next, ma

			for i := uint64(0); i < 8; i++ {
				st64(base, a1+8*i, r[i])
			}
			for i := uint64(0); i < 4; i++ {
				st64(base, a0+8*i, ld64(base, a0+8*i)^math.Float64bits(f[i]))
			}
			spMix = r[rA] ^ r[rB]
		}
		state = h256(tagProg, state[:], serialize(&r, &f, ma, mx))
	}

	// 4. Yakun: butun scratchpad va registrlar mix_commit ga kiradi.
	sph := h256(tagSPH, sp)
	mix = h256(tagMix, serialize(&r, &f, ma, mx), sph[:])
	pow = PowFromMix(seed, mix)
	return pow, mix
}

var (
	ErrPrefilter = errors.New("oldindan filtr: da'vo qilingan mix share targetga yetmaydi")
	ErrBlind     = errors.New("blind-share sharti yoki commit bajarilmadi")
	ErrMix       = errors.New("mix_commit qayta hisoblangan qiymatga mos emas")
)

// Prefilter — mikrosekundli tekshiruv: ikki BLAKE3 chaqiruvi.
func Prefilter(header []byte, nonce uint64, claimedMix [32]byte, shareTarget *big.Int) bool {
	return MeetsTarget(PowFromMix(Seed(header, nonce), claimedMix), shareTarget)
}

// VerifyBlock — spetsifikatsiyadagi tartibda: arzon tekshiruvlar oldin, qimmat
// qayta hisoblash oxirida.
func (hs *Hasher) VerifyBlock(header []byte, nonce uint64, claimedMix [32]byte,
	blockTarget *big.Int, k uint, secret, commit [32]byte) error {
	seed := Seed(header, nonce)
	pow := PowFromMix(seed, claimedMix)
	if !MeetsTarget(pow, ShareTarget(blockTarget, k)) {
		return ErrPrefilter
	}
	if BlindCommit(secret) != commit || !BlindOK(pow, secret, k) {
		return ErrBlind
	}
	if _, mix := hs.HashSeed(seed); mix != claimedMix {
		return ErrMix
	}
	return nil
}

// VerifyShare — pool uchun: prefiltr + to'liq qayta hisoblash.
func (hs *Hasher) VerifyShare(header []byte, nonce uint64, claimedMix [32]byte, shareTarget *big.Int) error {
	if !Prefilter(header, nonce, claimedMix, shareTarget) {
		return ErrPrefilter
	}
	if _, mix := hs.Hash(header, nonce); mix != claimedMix {
		return ErrMix
	}
	return nil
}
