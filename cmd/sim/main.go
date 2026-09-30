// sim — QalqonPoW yechimlarini eski yechimlar bilan solishtiruvchi simulyatsiyalar.
//
//	go run ./cmd/sim
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"

	"github.com/anasazamov/QolqonPOW/pow"
)

func main() {
	os.MkdirAll("results", 0o755)
	out := map[string]any{
		"withholding": withholding(),
		"daa":         daa(),
		"timewarp":    timewarp(),
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	path := filepath.Join("results", "simulations.json")
	os.WriteFile(path, b, 0o644)
	fmt.Println("\nNatijalar saqlandi:", path)
}

// ======================= 1. Block withholding =======================
//
// Hujumchi (tarmoqning a=20%) quvvatining x qismini p=30% lik pool ichiga
// "infiltratsiya" qiladi. Klassik PoW'da u share blok ekanini ko'rib, uni
// yashiradi, lekin oddiy share'lar uchun haq oladi. Blind-share'da u blokni
// tanib ololmaydi: faqat tasodifiy w ulushini yashirishi mumkin.
// Qiyinlik moslashgani uchun daromad = hujumchi mukofoti / barcha mukofotlar.

type whRow struct {
	Mode          string  `json:"mode"`
	X             float64 `json:"infiltration_x"`
	W             float64 `json:"withhold_w"`
	AttackerShare float64 `json:"attacker_revenue_share"`
	Analytic      float64 `json:"analytic"`
	GainPct       float64 `json:"gain_vs_honest_pct"`
	PoolPerHash   float64 `json:"honest_pool_member_revenue_per_hash"`
	Blocks        int     `json:"blocks"`
}

func withholding() []whRow {
	const a, p = 0.20, 0.30
	const k = 8
	const n = 100_000_000
	secret := pow.H256("pool-secret")
	fmt.Println("=== 1. Block withholding hujumi (hujumchi a=20%, pool p=30%) ===")
	fmt.Printf("%-9s %6s %5s %12s %10s %9s %13s\n", "rejim", "x", "w", "hujumchi", "analitik", "foyda", "pool a'zosi")

	run := func(mode string, x, w float64, seed uint64) whRow {
		rng := rand.New(rand.NewPCG(seed, 99))
		var solo, pool, other, attSub, poolSub int
		var buf [32]byte
		isBlockClassic := func() bool { return rng.Uint64()>>(64-k) == 0 }
		blindBlock := func() bool {
			for i := 0; i < 4; i++ {
				put := rng.Uint64()
				for j := 0; j < 8; j++ {
					buf[i*8+j] = byte(put >> (8 * j))
				}
			}
			return pow.BlindOK(buf, secret, k)
		}
		for i := 0; i < n; i++ {
			u := rng.Float64()
			switch {
			case u < a-x: // hujumchining o'z (yakka) quvvati
				if isBlockClassic() {
					solo++
				}
			case u < a: // hujumchi pool ichida
				if mode == "klassik" {
					if isBlockClassic() {
						continue // blokni ko'rdi va yashirdi
					}
					attSub++
					poolSub++
				} else {
					block := blindBlock() // hujumchi buni bilmaydi
					if rng.Float64() < w {
						continue // ko'r-ko'rona yashirish
					}
					attSub++
					poolSub++
					if block {
						pool++
					}
				}
			case u < a+p: // halol pool a'zolari
				poolSub++
				if mode == "klassik" {
					if isBlockClassic() {
						pool++
					}
				} else if blindBlock() {
					pool++
				}
			default:
				if isBlockClassic() {
					other++
				}
			}
		}
		total := float64(solo + pool + other)
		attRev := float64(solo) + float64(pool)*float64(attSub)/float64(poolSub)
		honestPool := float64(pool) * float64(poolSub-attSub) / float64(poolSub)
		var analytic float64
		if mode == "klassik" {
			analytic = (a-x)/(1-x) + x/(p+x)*p/(1-x)
		} else {
			analytic = (a - w*x) / (1 - w*x)
		}
		r := whRow{
			Mode: mode, X: x, W: w,
			AttackerShare: attRev / total,
			Analytic:      analytic,
			GainPct:       (attRev/total/a - 1) * 100,
			PoolPerHash:   honestPool / total / p,
			Blocks:        int(total),
		}
		fmt.Printf("%-9s %6.3f %5.2f %11.4f%% %9.4f%% %+8.2f%% %12.3f\n",
			r.Mode, r.X, r.W, r.AttackerShare*100, r.Analytic*100, r.GainPct, r.PoolPerHash)
		return r
	}

	var rows []whRow
	for i, x := range []float64{0, 0.025, 0.05, 0.075, 0.10, 0.15} {
		rows = append(rows, run("klassik", x, 1, uint64(10+i)))
	}
	for i, x := range []float64{0.05, 0.10} {
		for j, w := range []float64{0, 0.5, 1} {
			rows = append(rows, run("blind", x, w, uint64(100+i*10+j)))
		}
	}
	fmt.Println("foyda = hujumchi daromadi halol qazishdagi 20% ga nisbatan; pool a'zosi = halol a'zoning 1 hash uchun daromadi (1.000 = adolatli)")
	return rows
}

// ======================= 2. Qiyinlik algoritmlari =======================
//
// 1 birlik doimiy miner + 3 birlik "hopper" (boshqa tangadan keladigan quvvat).
// Hopper qiyinlik past bo'lsa (D < D0) keladi, D > 1.1*D0 bo'lsa ketadi.

type daaRow struct {
	Name          string  `json:"name"`
	MeanBlock     float64 `json:"mean_block_s"`
	StdBlock      float64 `json:"std_block_s"`
	Over10Min     float64 `json:"blocks_over_10min_pct"`
	WorstHour100  float64 `json:"worst_100_block_avg_min"`
	HopperAdv     float64 `json:"hopper_reward_per_hash_vs_steady"`
	HopperOnShare float64 `json:"hopper_on_time_pct"`
}

func daa() []daaRow {
	const T = 120.0
	const H0, Hh = 1.0, 3.0
	const D0 = T * H0
	const blocks = 300_000
	fmt.Println("\n=== 2. Qiyinlik algoritmi va coin-hopping (1 doimiy + 3 hopper) ===")
	fmt.Printf("%-24s %9s %9s %12s %14s %13s %10s\n", "algoritm", "o'rtacha", "std", ">10 daq", "eng yomon 100", "hopper foyda", "hopper on")

	type algo struct {
		name string
		next func(h int, ts []float64, ds []float64) float64
	}
	asert := func(hl float64) func(int, []float64, []float64) float64 {
		return func(h int, ts, ds []float64) float64 {
			// langar: genesis (t=0, D0). h = keyingi blok indeksi.
			dev := ts[h-1] - T*float64(h)
			return D0 * math.Pow(2, -dev/hl)
		}
	}
	algos := []algo{
		{"Bitcoin (2016 blok)", func(h int, ts, ds []float64) float64 {
			if h%2016 != 0 || h < 2016 {
				return ds[h-1]
			}
			span := ts[h-1] - ts[h-2016]
			span = math.Max(math.Min(span, 4*2016*T), 2016*T/4)
			return ds[h-1] * 2016 * T / span
		}},
		{"SMA-144 (BCH 2017)", func(h int, ts, ds []float64) float64 {
			if h < 145 {
				return D0
			}
			var sum float64
			for i := h - 144; i < h; i++ {
				sum += ds[i]
			}
			span := ts[h-1] - ts[h-145]
			span = math.Max(math.Min(span, 288*T), 72*T)
			return sum * T / span
		}},
		{"ASERT 2 kun (BCH 2020)", asert(172800)},
		{"ASERT 1 kun (Qalqon)", asert(86400)},
	}

	var rows []daaRow
	for ai, al := range algos {
		rng := rand.New(rand.NewPCG(uint64(ai)+1, 5))
		ts := make([]float64, blocks)
		ds := make([]float64, blocks)
		bt := make([]float64, blocks)
		on := false
		var t, hopBlocks, steadyBlocks, hopHT, steadyHT, onTime float64
		for h := 0; h < blocks; h++ {
			d := D0
			if h > 0 {
				d = al.next(h, ts, ds)
			}
			ds[h] = d
			if d < D0 {
				on = true
			} else if d > 1.1*D0 {
				on = false
			}
			H := H0
			if on {
				H += Hh
			}
			dt := rng.ExpFloat64() * d / H
			t += dt
			ts[h] = t
			bt[h] = dt
			steadyBlocks += H0 / H
			steadyHT += H0 * dt
			if on {
				hopBlocks += Hh / H
				hopHT += Hh * dt
				onTime += dt
			}
		}
		var mean, sq, over float64
		for _, x := range bt {
			mean += x
			if x > 600 {
				over++
			}
		}
		mean /= blocks
		for _, x := range bt {
			sq += (x - mean) * (x - mean)
		}
		worst := 0.0
		var win float64
		for i := range bt {
			win += bt[i]
			if i >= 100 {
				win -= bt[i-100]
			}
			if i >= 99 {
				worst = math.Max(worst, win/100)
			}
		}
		adv := 0.0
		if hopHT > 0 {
			adv = (hopBlocks / hopHT) / (steadyBlocks / steadyHT)
		}
		r := daaRow{al.name, mean, math.Sqrt(sq / blocks), over / blocks * 100, worst / 60, adv, onTime / t * 100}
		fmt.Printf("%-24s %8.1fs %8.1fs %11.2f%% %11.1f daq %12.3fx %9.1f%%\n",
			r.Name, r.MeanBlock, r.StdBlock, r.Over10Min, r.WorstHour100, r.HopperAdv, r.HopperOnShare)
		rows = append(rows, r)
	}
	fmt.Println("ideal: o'rtacha 120 s, std ~120 s (eksponensial), >10 daq ~0.67%, hopper foyda 1.000x")
	return rows
}

// ======================= 3. Time-warp =======================
//
// Tarmoqning 100% quvvatini egallagan hujumchi 30 kun davomida timestampni
// qoidalar ruxsat bergan eng foydali qilib qo'yadi. Halol holatda 21600 blok.

type twRow struct {
	Name   string  `json:"name"`
	Blocks int     `json:"blocks_in_30_days"`
	Ratio  float64 `json:"vs_honest_x"`
	FinalD float64 `json:"final_difficulty_vs_start"`
}

func median(v []float64) float64 {
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	return s[len(s)/2]
}

func timewarp() []twRow {
	const T, D0, week = 120.0, 120.0, 30 * 86400.0
	fmt.Println("\n=== 3. Time-warp hujumi (hujumchi barcha bloklarni qazadi, 30 kun) ===")
	var rows []twRow
	report := func(name string, n int, d float64) {
		r := twRow{name, n, float64(n) / (week / T), d / D0}
		fmt.Printf("%-40s %9d blok  (halolga nisbatan %8.2fx), yakuniy qiyinlik %.4fx\n", r.Name, r.Blocks, r.Ratio, r.FinalD)
		rows = append(rows, r)
	}

	// Bitcoin uslubi: 2016 blokli davr; span = davrning birinchi va oxirgi bloki
	// orasida (klassik "off-by-one" xatosi). Timestamp: > MTP(11), <= vaqt+7200.
	{
		rng := rand.New(rand.NewPCG(7, 7))
		var ts []float64
		d, t := D0, 0.0
		for i := 0; i < 11; i++ {
			ts = append(ts, 0)
		}
		n := 0
		// Qiyinlik har davrda 4x tushadi, bloklar soni cheksizga intiladi:
		// 5 mln blokda to'xtatamiz.
		for t < week && n < 5_000_000 {
			h := n
			if h > 0 && h%2016 == 0 {
				span := ts[len(ts)-1] - ts[len(ts)-2016]
				span = math.Max(math.Min(span, 4*2016*T), 2016*T/4)
				d = d * 2016 * T / span
			}
			t += rng.ExpFloat64() * d
			var stamp float64
			if (h+1)%2016 == 0 {
				stamp = t + 7200 // davr oxiri: iloji boricha kelajakda
			} else {
				stamp = median(ts[len(ts)-11:]) + 1 // qolganlari: MTP+1
			}
			ts = append(ts, stamp)
			if len(ts) > 4096 {
				ts = append(ts[:0:0], ts[len(ts)-2100:]...)
			}
			n++
		}
		report(fmt.Sprintf("Bitcoin 2016-blok + time-warp (%.1f kunda)", t/86400), n, d)
	}

	// ASERT: foydali yagona yo'l — timestampni FTL chegarasida kelajakka qo'yish.
	for _, c := range []struct {
		name     string
		hl, ftl  float64
	}{
		{"ASERT, FTL=7200 s (Bitcoin FTL)", 86400, 7200},
		{"ASERT, FTL=360 s (Qalqon)", 86400, 360},
	} {
		rng := rand.New(rand.NewPCG(8, 8))
		t, n := 0.0, 0
		d := D0
		for t < week {
			if n > 0 {
				dev := (t + c.ftl) - T*float64(n) // ota blok timestampi = vaqt + FTL
				d = D0 * math.Pow(2, -dev/c.hl)
			}
			t += rng.ExpFloat64() * d
			n++
		}
		report(c.name, n, d)
	}
	return rows
}
