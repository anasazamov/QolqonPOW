// qalqon — QalqonPoW uchun CLI: test vektorlari va benchmark.
//
//	go run ./cmd/qalqon vectors
//	go run ./cmd/qalqon bench -params mainnet -seconds 20
package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anasazamov/QolqonPOW/pow"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("foydalanish: qalqon vectors | qalqon bench [-params mainnet|test] [-threads N] [-seconds S]")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "vectors":
		vectors()
	case "bench":
		bench(os.Args[2:])
	default:
		fmt.Println("noma'lum buyruq:", os.Args[1])
		os.Exit(2)
	}
}

func vectors() {
	c := pow.NewCache(pow.H256("test-epoch-key"), pow.Test)
	hs := pow.NewHasher(pow.Test, pow.NewDataset(c, 0))
	type vector struct {
		Header string `json:"header"`
		Nonce  uint64 `json:"nonce"`
		Mix    string `json:"mix"`
		Pow    string `json:"pow"`
	}
	var vs []vector
	for i := uint64(0); i < 16; i++ {
		hdr := make([]byte, 80+i*3)
		for j := range hdr {
			hdr[j] = byte(j*31 + int(i))
		}
		nonce := i*0x9E3779B97F4A7C15 + i
		p, m := hs.Hash(hdr, nonce)
		vs = append(vs, vector{hex.EncodeToString(hdr), nonce, hex.EncodeToString(m[:]), hex.EncodeToString(p[:])})
	}
	os.MkdirAll("testdata", 0o755)
	out, _ := json.MarshalIndent(vs, "", "  ")
	path := filepath.Join("testdata", "vectors_test_params.json")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("%d ta test vektori yozildi: %s\n", len(vs), path)
}

type benchResult struct {
	Params             string  `json:"params"`
	Threads            int     `json:"threads"`
	CacheMiB           uint64  `json:"cache_mib"`
	DatasetMiB         uint64  `json:"dataset_mib"`
	CacheBuildSec      float64 `json:"cache_build_sec"`
	DatasetBuildSec    float64 `json:"dataset_build_sec"`
	HashesPerSec       float64 `json:"hashes_per_sec"`
	HashLatencyMs      float64 `json:"hash_latency_ms_single_thread"`
	PrefilterUs        float64 `json:"prefilter_us"`
	VerifyFastMs       float64 `json:"verify_fast_ms"`
	VerifyLightMs      float64 `json:"verify_light_ms"`
	ItemReadNs         float64 `json:"dataset_item_read_ns"`
	ItemComputeNs      float64 `json:"dataset_item_compute_ns"`
	LightEvalPenalty   float64 `json:"light_eval_penalty_x"`
	PrefilterVsFullX   float64 `json:"full_verify_vs_prefilter_x"`
}

func bench(args []string) {
	fs := flag.NewFlagSet("bench", flag.ExitOnError)
	pname := fs.String("params", "mainnet", "mainnet yoki test")
	threads := fs.Int("threads", runtime.NumCPU(), "mining oqimlari")
	seconds := fs.Int("seconds", 20, "mining benchmark davomiyligi")
	fs.Parse(args)

	p := pow.Mainnet
	if *pname == "test" {
		p = pow.Test
	}
	res := benchResult{Params: p.Name, Threads: *threads, CacheMiB: p.CacheBytes >> 20, DatasetMiB: p.DatasetBytes >> 20}
	fmt.Printf("QalqonPoW benchmark — parametrlar: %s, oqimlar: %d\n\n", p.Name, *threads)

	t0 := time.Now()
	cache := pow.NewCache(pow.H256("bench-epoch-key"), p)
	res.CacheBuildSec = time.Since(t0).Seconds()
	fmt.Printf("Kesh (%d MiB) qurildi:       %.2f s\n", res.CacheMiB, res.CacheBuildSec)

	t0 = time.Now()
	ds := pow.NewDataset(cache, *threads)
	res.DatasetBuildSec = time.Since(t0).Seconds()
	fmt.Printf("Dataset (%d MiB) qurildi:    %.2f s\n", res.DatasetMiB, res.DatasetBuildSec)

	hdr := make([]byte, 112)
	for i := range hdr {
		hdr[i] = byte(i)
	}

	// Bitta oqim kechikishi.
	hs := pow.NewHasher(p, ds)
	const lat = 20
	t0 = time.Now()
	for i := uint64(0); i < lat; i++ {
		hs.Hash(hdr, 1_000_000+i)
	}
	res.HashLatencyMs = float64(time.Since(t0).Nanoseconds()) / 1e6 / lat

	// Ko'p oqimli mining.
	var count atomic.Uint64
	var wg sync.WaitGroup
	deadline := time.Now().Add(time.Duration(*seconds) * time.Second)
	t0 = time.Now()
	for t := 0; t < *threads; t++ {
		wg.Add(1)
		go func(t int) {
			defer wg.Done()
			h := pow.NewHasher(p, ds)
			n := uint64(t) << 40
			for time.Now().Before(deadline) {
				h.Hash(hdr, n)
				n++
				count.Add(1)
			}
		}(t)
	}
	wg.Wait()
	res.HashesPerSec = float64(count.Load()) / time.Since(t0).Seconds()
	fmt.Printf("Mining tezligi:              %.0f H/s (%d oqim), bitta hash %.2f ms\n", res.HashesPerSec, *threads, res.HashLatencyMs)

	// Tekshirish narxlari.
	_, mix := hs.Hash(hdr, 42)
	target := pow.TargetFromBits(0) // har qanday hash o'tadi: faqat tekshirish narxi o'lchanadi
	const pf = 200000
	t0 = time.Now()
	for i := 0; i < pf; i++ {
		pow.Prefilter(hdr, 42, mix, target)
	}
	res.PrefilterUs = float64(time.Since(t0).Nanoseconds()) / 1000 / pf

	const vf = 20
	t0 = time.Now()
	for i := 0; i < vf; i++ {
		hs.VerifyShare(hdr, 42, mix, target)
	}
	res.VerifyFastMs = float64(time.Since(t0).Nanoseconds()) / 1e6 / vf

	light := pow.NewHasher(p, pow.NewLight(cache))
	const vl = 5
	t0 = time.Now()
	for i := 0; i < vl; i++ {
		if err := light.VerifyShare(hdr, 42, mix, target); err != nil {
			panic(err)
		}
	}
	res.VerifyLightMs = float64(time.Since(t0).Nanoseconds()) / 1e6 / vl
	res.PrefilterVsFullX = res.VerifyFastMs * 1000 / res.PrefilterUs
	fmt.Printf("Prefiltr (arzon tekshiruv):  %.2f µs\n", res.PrefilterUs)
	fmt.Printf("To'liq tekshiruv (dataset):  %.2f ms  -> prefiltrdan %.0fx qimmat\n", res.VerifyFastMs, res.PrefilterVsFullX)
	fmt.Printf("Light tekshiruv (faqat kesh): %.2f ms\n", res.VerifyLightMs)

	// Light evaluation jarimasi: elementni saqlamasdan qayta hisoblash narxi.
	rng := rand.New(rand.NewPCG(1, 2))
	var it [64]byte
	lt := pow.NewLight(cache)
	const nr, nc = 2_000_000, 50_000
	t0 = time.Now()
	for i := 0; i < nr; i++ {
		ds.Item(rng.Uint64N(ds.Items()), &it)
	}
	res.ItemReadNs = float64(time.Since(t0).Nanoseconds()) / nr
	t0 = time.Now()
	for i := 0; i < nc; i++ {
		lt.Item(rng.Uint64N(lt.Items()), &it)
	}
	res.ItemComputeNs = float64(time.Since(t0).Nanoseconds()) / nc
	res.LightEvalPenalty = res.ItemComputeNs / res.ItemReadNs
	fmt.Printf("Dataset elementi: o'qish %.0f ns, qayta hisoblash %.0f ns -> saqlamaslik %.0fx qimmat\n",
		res.ItemReadNs, res.ItemComputeNs, res.LightEvalPenalty)

	os.MkdirAll("results", 0o755)
	out, _ := json.MarshalIndent(res, "", "  ")
	path := filepath.Join("results", "bench-"+p.Name+".json")
	os.WriteFile(path, out, 0o644)
	fmt.Println("\nNatija saqlandi:", path)
}
