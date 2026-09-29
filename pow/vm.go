package pow

import (
	"crypto/aes"
	"crypto/cipher"
	"math"
	"math/bits"
	"unsafe"
)

type opcode uint8

const (
	opIADD_RS opcode = iota
	opIADD_M
	opISUB_R
	opIMUL_R
	opIMULH_R
	opIMUL_M
	opIXOR_R
	opIXOR_M
	opIROR_R
	opISWAP_R
	opISTORE
	opFADD_R
	opFSUB_R
	opFMUL_R
	opFDIV_M
	opFSQRT_R
	opFSWAP_R
	opCBRANCH
	opFTOI
	numOps
)

// Har bir opcode 256 dan nechta qiymatni egallaydi (yig'indi 256).
// Aralashma CPU'ning butun son, FP, xotira va shartli o'tish bloklarini
// bir vaqtda band qiladigan qilib tanlangan.
var opFreq = [numOps]int{24, 7, 16, 28, 4, 4, 23, 5, 10, 4, 16, 16, 16, 32, 4, 6, 8, 25, 8}

var opTable [256]opcode

func init() {
	i := 0
	for op, n := range opFreq {
		for k := 0; k < n; k++ {
			opTable[i] = opcode(op)
			i++
		}
	}
	if i != 256 {
		panic("opFreq yig'indisi 256 emas")
	}
}

type instr struct {
	op     opcode
	dst    uint8
	src    uint8
	mod    uint8
	imm    uint64
	cmask  uint64
	target int
}

const instrBytes = 16

// genProgram holatdan AES-128-CTR oqimi bilan dastur yasaydi.
func genProgram(state [32]byte, prog []instr, buf []byte) {
	blk, err := aes.NewCipher(state[:16])
	if err != nil {
		panic(err)
	}
	clear(buf)
	cipher.NewCTR(blk, state[16:32]).XORKeyStream(buf, buf)
	for pc := range prog {
		b := buf[pc*instrBytes:]
		in := instr{
			op:  opTable[b[0]],
			dst: b[1] & 7,
			src: b[2] & 7,
			mod: b[3],
			imm: le64(b[8:16]),
		}
		in.target = int(uint16(b[4])|uint16(b[5])<<8) % (pc + 1)
		if in.op == opCBRANCH {
			shift := uint(b[6]) % 57
			in.cmask = 0xFF << shift
			in.imm |= 1 << shift // shart bitlari har safar o'zgaradi
		}
		prog[pc] = in
	}
}

// norm har qanday float64 ni oddiy (normal) diapazonga keltiradi: NaN, Inf,
// denormal va nol bo'lmaydi. Eksponenta [2^-24, 2^7] ichida qoladi.
func norm(x float64) float64 {
	b := math.Float64bits(x)
	e := (b >> 52) & 0x7FF
	e = 0x3FF - 24 + (e & 31)
	return math.Float64frombits(b&^(uint64(0x7FF)<<52) | e<<52)
}

// exec dasturni bir marta bajaradi (referens interpretator). FP natijalari har safar aniq float64
// ga o'giriladi, shuning uchun kompilyator FMA birlashtirishi mumkin emas.
func (hs *Hasher) exec(r *[8]uint64, f *[4]float64) {
	prog := hs.prog
	base := unsafe.Pointer(&hs.sp[0])
	m := &hs.masks
	budget := hs.p.BranchBudget
	for pc := 0; pc < len(prog); pc++ {
		in := &prog[pc]
		d, s := in.dst, in.src
		switch in.op {
		case opIADD_RS:
			r[d] += r[s] << (in.mod & 3)
		case opIADD_M:
			r[d] += ld64(base, (r[s]+in.imm)&m[in.mod&3])
		case opISUB_R:
			if s == d {
				r[d] -= in.imm
			} else {
				r[d] -= r[s]
			}
		case opIMUL_R:
			if s == d {
				r[d] *= in.imm | 1
			} else {
				r[d] *= r[s]
			}
		case opIMULH_R:
			hi, _ := bits.Mul64(r[d], r[s])
			r[d] = hi
		case opIMUL_M:
			r[d] *= ld64(base, (r[s]+in.imm)&m[in.mod&3])
		case opIXOR_R:
			if s == d {
				r[d] ^= in.imm
			} else {
				r[d] ^= r[s]
			}
		case opIXOR_M:
			r[d] ^= ld64(base, (r[s]+in.imm)&m[in.mod&3])
		case opIROR_R:
			if s == d {
				r[d] = bits.RotateLeft64(r[d], -int(in.imm&63))
			} else {
				r[d] = bits.RotateLeft64(r[d], -int(r[s]&63))
			}
		case opISWAP_R:
			r[d], r[s] = r[s], r[d]
		case opISTORE:
			st64(base, (r[d]+in.imm)&m[in.mod&3], r[s])
		case opFADD_R:
			f[d&3] = norm(float64(f[d&3] + f[s&3]))
		case opFSUB_R:
			f[d&3] = norm(float64(f[d&3] - f[s&3]))
		case opFMUL_R:
			f[d&3] = norm(float64(f[d&3] * f[s&3]))
		case opFDIV_M:
			div := norm(float64(int32(ld64(base, (r[s]+in.imm)&m[in.mod&3]))))
			f[d&3] = norm(float64(f[d&3] / div))
		case opFSQRT_R:
			f[d&3] = norm(math.Sqrt(math.Abs(f[d&3])))
		case opFSWAP_R:
			f[d&3], f[s&3] = f[s&3], f[d&3]
		case opCBRANCH:
			r[d] += in.imm
			if r[d]&in.cmask == 0 && budget > 0 {
				budget--
				pc = in.target - 1
			}
		case opFTOI:
			r[d] ^= math.Float64bits(f[s&3])
		}
	}
}
