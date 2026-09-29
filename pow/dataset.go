package pow

import (
	"math/bits"
	"runtime"
	"sync"
	"unsafe"

	"lukechampine.com/blake3"
)

const lineBytes = 64

// Cache — epoch kalitidan hosil qilinadigan xotira. Light rejimda dataset
// elementlari shundan joyida hisoblanadi.
type Cache struct {
	Key   [32]byte
	p     Params
	mem   []byte
	base  unsafe.Pointer
	lines uint64
}

// NewCache keshni to'ldiradi: avval BLAKE3 XOF oqimi, keyin CachePasses marta
// ketma-ket, ma'lumotga bog'liq aralashtirish (scrypt ROMix / Argon2d uslubi).
// Har bir qator oldingi qatorga va oldingi qator ko'rsatgan tasodifiy qatorga
// bog'liq, shuning uchun uni parallel yoki kam xotira bilan arzon qurib bo'lmaydi.
func NewCache(key [32]byte, p Params) *Cache {
	c := &Cache{Key: key, p: p, lines: p.CacheBytes / lineBytes}
	c.mem = make([]byte, p.CacheBytes)
	c.base = unsafe.Pointer(&c.mem[0])
	h := blake3.New(32, nil)
	h.Write(tagCache)
	h.Write(key[:])
	h.XOF().Read(c.mem)

	var buf [2 * lineBytes]byte
	n := c.lines
	for pass := 0; pass < p.CachePasses; pass++ {
		for i := uint64(0); i < n; i++ {
			prev := c.mem[((i+n-1)%n)*lineBytes:][:lineBytes]
			j := le64(prev) % n
			copy(buf[:lineBytes], prev)
			copy(buf[lineBytes:], c.mem[j*lineBytes:][:lineBytes])
			d := blake3.Sum512(buf[:])
			line := c.mem[i*lineBytes:][:lineBytes]
			for k := range d {
				line[k] ^= d[k]
			}
		}
	}
	return c
}

// Dataset elementi aralashtirish konstantalari (toq sonlar, SplitMix/wyhash).
var (
	itemMul = [8]uint64{
		0x9E3779B97F4A7C15, 0xBF58476D1CE4E5B9, 0x94D049BB133111EB, 0xA0761D6478BD642F,
		0xE7037ED1A0B428DB, 0x8EBC6AF09C88C6E3, 0x589965CC75374CC3, 0x1D8E4E27C47D124F,
	}
	itemRot = [8]int{23, 29, 31, 37, 41, 43, 47, 53}
)

// item i-dataset elementini hisoblaydi (v0.2).
//
// Boshida va oxirida BLAKE3, o'rtada DatasetParents ta zanjirli kesh murojaati.
// Har bir keyingi manzil oldingi qatorni aralashtirish natijasiga bog'liq, shuning
// uchun murojaatlar ketma-ket bo'ladi va narx DRAM kechikishi bilan belgilanadi.
// Bu narxni maxsus hisoblash bloki bilan qisqartirib bo'lmaydi.
func (c *Cache) item(i uint64, out *[64]byte) {
	var in [9 + 8 + 32]byte
	copy(in[:], tagItem)
	put64(in[9:], i)
	copy(in[17:], c.Key[:])
	seed := blake3.Sum512(in[:])

	var s [8]uint64
	for w := range s {
		s[w] = le64(seed[8*w:])
	}
	x := s[0] ^ s[7]
	for p := 0; p < c.p.DatasetParents; p++ {
		hi, lo := bits.Mul64(x, 0x9E3779B97F4A7C15)
		off := ((hi ^ lo) % c.lines) * lineBytes
		var acc uint64
		for w := 0; w < 8; w++ {
			v := (s[w] ^ ld64(c.base, off+uint64(8*w))) * itemMul[w]
			v = bits.RotateLeft64(v, itemRot[w])
			s[w] = v
			acc += v
		}
		x = acc ^ (acc >> 29)
		for w := 0; w < 8; w++ {
			s[w] += x
		}
	}
	var buf [64]byte
	for w := range s {
		put64(buf[8*w:], s[w])
	}
	*out = blake3.Sum512(buf[:])
}

// ItemSource — Hasher dataset elementlarini shu interfeys orqali oladi.
type ItemSource interface {
	Items() uint64
	Item(i uint64, out *[64]byte)
	Prefetch(i uint64)
}

// Dataset — mining uchun to'liq, oldindan hisoblangan dataset (fast rejim).
type Dataset struct {
	mem  []byte
	base unsafe.Pointer
	n    uint64
}

// NewDataset keshdan to'liq datasetni threads ta goroutine'da quradi.
func NewDataset(c *Cache, threads int) *Dataset {
	n := c.p.DatasetBytes / lineBytes
	d := &Dataset{mem: make([]byte, n*lineBytes), n: n}
	d.base = unsafe.Pointer(&d.mem[0])
	if threads <= 0 {
		threads = runtime.NumCPU()
	}
	chunk := (n + uint64(threads) - 1) / uint64(threads)
	var wg sync.WaitGroup
	for t := 0; t < threads; t++ {
		lo := uint64(t) * chunk
		hi := min(lo+chunk, n)
		if lo >= hi {
			break
		}
		wg.Add(1)
		go func(lo, hi uint64) {
			defer wg.Done()
			var it [64]byte
			for i := lo; i < hi; i++ {
				c.item(i, &it)
				copy(d.mem[i*lineBytes:], it[:])
			}
		}(lo, hi)
	}
	wg.Wait()
	return d
}

func (d *Dataset) Items() uint64 { return d.n }

func (d *Dataset) Item(i uint64, out *[64]byte) {
	*out = *(*[64]byte)(unsafe.Add(d.base, i*lineBytes))
}

func (d *Dataset) Prefetch(i uint64) { prefetch(unsafe.Add(d.base, i*lineBytes)) }

// Light — datasetsiz tekshirish: har bir element keshdan joyida hisoblanadi.
type Light struct {
	c *Cache
	n uint64
}

func NewLight(c *Cache) *Light {
	return &Light{c: c, n: c.p.DatasetBytes / lineBytes}
}

func (l *Light) Items() uint64 { return l.n }

func (l *Light) Item(i uint64, out *[64]byte) { l.c.item(i, out) }

func (l *Light) Prefetch(uint64) {}
