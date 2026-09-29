// Package pow — QalqonPoW referens implementatsiyasi.
package pow

// Params algoritmning barcha o'lchamlari.
type Params struct {
	Name           string
	CacheBytes     uint64 // epoch keshi (light rejim va dataset manbai)
	CachePasses    int    // keshni ma'lumotga bog'liq aralashtirish o'tishlari
	DatasetBytes   uint64 // mining uchun to'liq dataset (boshlang'ich hajm)
	DatasetGrowth  uint64 // har yili qo'shiladigan hajm
	BlocksPerYear  uint64
	DatasetParents int    // bitta dataset elementi uchun zanjirli kesh murojaatlari
	ScratchBytes   uint64 // har bir hash uchun scratchpad (2 ning darajasi)
	Programs       int    // har bir hash uchun tasodifiy dasturlar soni
	Iterations     int    // har bir dastur necha marta aylanadi
	ProgramSize    int    // dasturdagi instruksiyalar
	BranchBudget   int    // bitta bajarishda ko'pi bilan nechta orqaga o'tish
	EpochLen       uint64
	KeyLag         uint64
}

// Mainnet — spetsifikatsiya v0.1 dagi qiymatlar.
var Mainnet = Params{
	Name:           "mainnet",
	CacheBytes:     256 << 20,
	CachePasses:    3,
	DatasetBytes:   2 << 30,
	DatasetGrowth:  128 << 20,
	BlocksPerYear:  262800,
	DatasetParents: 32,
	ScratchBytes:   2 << 20,
	Programs:       8,
	Iterations:     2048,
	ProgramSize:    256,
	BranchBudget:   16,
	EpochLen:       2048,
	KeyLag:         64,
}

// Test — tez testlar va test vektorlari uchun kichik parametrlar.
var Test = Params{
	Name:           "test",
	CacheBytes:     1 << 20,
	CachePasses:    3,
	DatasetBytes:   4 << 20,
	DatasetGrowth:  1 << 20,
	BlocksPerYear:  1000,
	DatasetParents: 32,
	ScratchBytes:   64 << 10,
	Programs:       2,
	Iterations:     64,
	ProgramSize:    64,
	BranchBudget:   16,
	EpochLen:       32,
	KeyLag:         4,
}

// AtHeight dataset hajmini blok balandligiga qarab o'stiradi.
func (p Params) AtHeight(height uint64) Params {
	p.DatasetBytes += (height / p.BlocksPerYear) * p.DatasetGrowth
	return p
}

// EpochKeyHeight — epoch kaliti olinadigan blok balandligi.
func (p Params) EpochKeyHeight(height uint64) uint64 {
	if height < p.KeyLag+p.EpochLen {
		return 0
	}
	return ((height - p.KeyLag) / p.EpochLen) * p.EpochLen
}
