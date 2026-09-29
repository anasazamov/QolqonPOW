//go:build !amd64

package pow

import "unsafe"

// prefetch amd64 bo'lmagan platformalarda hech narsa qilmaydi.
func prefetch(unsafe.Pointer) {}
