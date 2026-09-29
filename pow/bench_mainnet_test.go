package pow

import (
	"os"
	"sync"
	"testing"
)

// Mainnet parametrlarida profil olish uchun (QALQON_MAINNET=1 bo'lsa ishlaydi):
//
//	QALQON_MAINNET=1 go test ./pow -run x -bench Mainnet -cpuprofile cpu.out
var (
	mainOnce  sync.Once
	mainCache *Cache
	mainData  *Dataset
)

func mainnetSetup(b *testing.B) (*Cache, *Dataset) {
	if os.Getenv("QALQON_MAINNET") == "" {
		b.Skip("QALQON_MAINNET=1 o'rnatilmagan")
	}
	mainOnce.Do(func() {
		mainCache = NewCache(H256("bench-epoch-key"), Mainnet)
		mainData = NewDataset(mainCache, 0)
	})
	return mainCache, mainData
}

func BenchmarkHashMainnet(b *testing.B) {
	_, ds := mainnetSetup(b)
	hs := NewHasher(Mainnet, ds)
	hdr := make([]byte, 112)
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		hs.Hash(hdr, uint64(i))
	}
}

func BenchmarkLightItemMainnet(b *testing.B) {
	c, _ := mainnetSetup(b)
	l := NewLight(c)
	var it [64]byte
	b.ResetTimer()
	for i := uint64(0); b.Loop(); i++ {
		l.Item(i*0x9E3779B97F4A7C15%l.Items(), &it)
	}
}
