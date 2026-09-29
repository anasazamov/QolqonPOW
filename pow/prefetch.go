package pow

import (
	"encoding/binary"
	"unsafe"
)

// prefetch keyingi iteratsiyada kerak bo'ladigan dataset qatorini keshga
// oldindan so'raydi (amd64: PREFETCHT0). Natijaga ta'sir qilmaydi.
//
//go:noescape
func prefetch(p unsafe.Pointer)

// Tezkor o'qish/yozish chegara tekshiruvisiz. Chaqiruvchi manzil ichkarida
// ekanini niqob (mask) orqali kafolatlaydi. Spetsifikatsiya little-endian,
// bu kod faqat little-endian protsessorlarda ishlaydi.
func ld64(base unsafe.Pointer, off uint64) uint64 {
	return *(*uint64)(unsafe.Add(base, off))
}

func st64(base unsafe.Pointer, off uint64, v uint64) {
	*(*uint64)(unsafe.Add(base, off)) = v
}

func init() {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], 1)
	if *(*uint64)(unsafe.Pointer(&b[0])) != 1 {
		panic("QalqonPoW referens kodi little-endian protsessor talab qiladi")
	}
}
