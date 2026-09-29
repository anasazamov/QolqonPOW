# QalqonPoW — referens implementatsiya

Yangi blockchain uchun proof-of-work algoritmi. Spetsifikatsiya: https://claude.ai/artifact/4a5shQAzBRYZioQSHcZ2jF

## Tuzilishi

| Fayl | Nima qiladi |
|---|---|
| `pow/params.go` | Mainnet va test parametrlari, dataset o'sishi, epoch kaliti |
| `pow/dataset.go` | Epoch keshi (256 MiB, ma'lumotga bog'liq 3 o'tish) va dataset (2 GiB, 32 ta zanjirli ota) |
| `pow/vm.go` | Tasodifiy dastur VM: 19 opcode, butun son, FP, xotira va shartli o'tish |
| `pow/hash.go` | Seed, bitta hash, prefiltr, share va blok tekshiruvi |
| `pow/consensus.go` | Blind-share, ASERT (aserti3-2d), timestamp qoidalari, fork tanlash, MMR |
| `pow/pow_test.go` | Testlar va test vektorlari tekshiruvi |
| `cmd/qalqon` | `vectors` va `bench` buyruqlari |
| `cmd/sim` | Uchta simulyatsiya: block withholding, qiyinlik algoritmlari, time-warp |

## Ishga tushirish

```
go test ./pow/ -v                               # 11 ta test
go run ./cmd/qalqon vectors                     # testdata/vectors_test_params.json
go run ./cmd/qalqon bench -params mainnet       # real parametrlar, ~2.5 GiB RAM
go run ./cmd/sim                                # results/simulations.json
```

## Natijalar (Intel Core Ultra 7 265KF, 20 oqim, Go 1.27)

### Benchmark (mainnet parametrlari)

| O'lchov | Qiymat |
|---|---|
| Kesh qurish (256 MiB) | 4.2 s |
| Dataset qurish (2 GiB) | 17.6 s |
| Mining tezligi | 1 831 H/s (bitta hash 10.9 ms) |
| Prefiltr | 1.2 µs |
| To'liq tekshiruv (dataset bilan) | 7.5 ms (prefiltrdan 6 226x qimmat) |
| Light tekshiruv (faqat kesh) | 192 ms |
| Dataset elementini o'qish / qayta hisoblash | 42 ns / 10 877 ns (saqlamaslik 258x qimmat) |

Mining tezligi interpretator bilan olingan. RandomX'dagi kabi JIT kompilyatsiya bilan bir necha barobar tezlashishi kutiladi.

### 1. Block withholding (hujumchi 20%, pool 30%, 100 mln share)

| Rejim | Infiltratsiya | Hujumchi foydasi | Halol pool a'zosi (1 hash uchun) |
|---|---|---|---|
| Klassik PoW | 5% | **+1.85%** | 0.904 (10% zarar) |
| Klassik PoW | 2.5% | +1.47% | 0.945 |
| Blind-share, hammasini yashiradi | 5% | **−20.7%** | 1.050 |
| Blind-share, yarmini yashiradi | 5% | −9.9% | 1.025 |
| Blind-share, halol | 5% | 0.0% | 0.997 |

Monte Carlo natijasi analitik formulaga mos keladi. Blind-share bilan hujumchi daromadi `(a − w·x)/(1 − w·x) ≤ a`, ya'ni har qanday yashirish strategiyasi faqat zarar keltiradi.

### 2. Qiyinlik algoritmlari (1 birlik doimiy + 3 birlik hopper, 300 ming blok)

| Algoritm | O'rtacha blok | >10 daqiqalik bloklar | Eng yomon 100 blok | Hopper ustunligi |
|---|---|---|---|---|
| Bitcoin (2016 blok) | 204 s | 9.29% | 11.0 daq | 2.29x |
| SMA-144 (BCH 2017) | 120 s | 1.01% | 2.9 daq | 1.08x |
| ASERT 1 kun (Qalqon) | 120 s | 0.77% | 3.1 daq | 0.99x |

Ideal holatda >10 daqiqalik bloklar 0.67% bo'ladi. Bu modelda ASERT SMA-144 dan biroz yaxshi, Bitcoin davr qoidasidan esa keskin yaxshi.

### 3. Time-warp (hujumchi barcha bloklarni qazadi, 30 kun, halol holatda 21 600 blok)

| Qoida | Natija |
|---|---|
| Bitcoin 2016-blok + time-warp | 16.2 kunda 5 000 000 blok, qiyinlik ~0 ga tushadi |
| ASERT, FTL=7200 s | 21 622 blok (1.00x) |
| ASERT, FTL=360 s (Qalqon) | 21 569 blok (1.00x) |

## Topilgan kamchiliklar

1. **Light tekshiruv qimmat: 192 ms.** Sababi `DATASET_PARENTS=32`. Bu light evaluation'ni 258 marta qimmatlashtiradi, lekin datasetsiz tekshiruvni ham sekinlashtiradi. To'liq node'lar va pool'lar 7.5 ms da tekshiradi. Variantlar: parents=16 (~95 ms) yoki yengil klientlar uchun faqat FlyClient.
2. **Mining interpretatorda ishlaydi.** Tijorat miner uchun x86-64 JIT kerak.
3. **Spetsifikatsiyadan chetlanishlar.** Kesh Argon2d emas, BLAKE3 bilan ROMix uslubida quriladi: Go'da Argon2d ichki xotirasi ochiq emas. AES-4R o'rniga to'liq AES-128-CTR ishlatiladi.
4. **Hali qilinmagan:** TMTO/pebbling tahlili, ASIC-narx modeli, mustaqil audit, ikkinchi implementatsiya, Stratum V2 pool.
