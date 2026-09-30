# QalqonPoW

[![CI](https://github.com/anasazamov/QolqonPOW/actions/workflows/ci.yml/badge.svg)](https://github.com/anasazamov/QolqonPOW/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Open research on proof-of-work defences: block withholding, time-warp, difficulty
oscillation and pool DoS. Includes a reference implementation and reproducible simulations.**

> **Status: research prototype. Not audited. Do not use in production.**
> Most components are known ideas; see [PRIOR_ART.md](PRIOR_ART.md) for what is new and what is not.

Uzbek version: [README.uz.md](README.uz.md)

## Why

A PoW chain faces several problems at the same time, and each one has a known but scattered fix:

| Problem | Known countermeasure | Where here |
|---|---|---|
| Block withholding inside pools (Eyal 2015, FAW: Kwon 2017) | Oblivious shares (Rosenfeld 2011) | `pow/consensus.go`: `BlindCommit`, `BlindOK` |
| Time-warp and hash-rate oscillation | ASERT (aserti3-2d) + tight future-time limit | `NextTarget`, `ValidTimestamp` |
| Pools flooded with invalid shares | Mix digest in the header, checked in ~1 µs (as in Ethash) | `Prefilter` |
| Heavy light-client sync | MMR header commitment for FlyClient | `MMR` |
| Cheap custom hardware | Memory-hard hash with random programs (Ethash/RandomX ideas) | `pow/dataset.go`, `pow/vm.go`, `pow/hash.go` |

This repository puts these countermeasures into one tested design and measures them.

## Results so far

All numbers come from `go run ./cmd/sim` and `go run ./cmd/qalqon bench`; raw data is in `results/`.

**Block withholding** (attacker 20% of the network, pool 30%, 100M shares, Monte Carlo vs. closed form):

| Mode | Infiltration x | Attacker gain vs. honest | Honest pool member, revenue per hash |
|---|---|---|---|
| Classic PoW | 5% | **+1.85%** | 0.904 |
| Oblivious shares, withhold all | 5% | **−20.7%** | 1.050 |
| Oblivious shares, withhold half | 5% | −9.9% | 1.025 |
| Oblivious shares, honest | 5% | 0.0% | 0.997 |

With oblivious shares the attacker's revenue is `(a − w·x)/(1 − w·x) ≤ a`, so any
withholding strategy loses. Simulation and formula agree within noise.

**Difficulty under coin-hopping** (1 unit steady + 3 units hopping hash rate, 300k blocks, 120 s target):

| Algorithm | Mean block | Blocks > 10 min | Hopper advantage |
|---|---|---|---|
| Bitcoin 2016-block | 204 s | 9.29% | 2.29× |
| SMA-144 (BCH 2017) | 120 s | 1.01% | 1.08× |
| ASERT, 1-day halflife | 120 s | 0.77% | 0.99× |

**Time-warp** (attacker mines every block for 30 days; honest result is 21,600 blocks):
Bitcoin's retarget gives 5,000,000 blocks in 16.2 days; ASERT with FTL 360 s gives 21,569 (1.00×).

**Hash core** (Intel Core Ultra 7 265KF, 20 threads, interpreter, mainnet parameters):
2,349 H/s. Full verification takes 5.8 ms, the prefilter 1.1 µs, and light verification (cache only) 91 ms.
Recomputing a dataset item instead of storing it costs 128× a read.

## Open problems (help wanted)

1. **Oblivious shares × Stratum V2 Job Declaration.** When the miner chooses the template,
   how does the pool secret bind to the job? (See Towns 2024 and the objections in [PRIOR_ART.md](PRIOR_ART.md).)
2. **Malicious operator and FAW** under oblivious shares.
3. **Decentralised pools** (P2Pool, Braidpool) with no single secret holder.
4. **TMTO/pebbling analysis** of the cache and dataset construction.
5. **Second implementation** (C or Rust) and differential fuzzing against the test vectors.
6. **JIT** for the VM, followed by an independent hardware-cost estimate.

## Layout

| Path | Contents |
|---|---|
| `pow/params.go` | Mainnet and test parameters |
| `pow/dataset.go` | Epoch cache (256 MiB, 3 data-dependent passes) and dataset (2 GiB, 32 chained parents) |
| `pow/vm.go` | Random-program VM: 19 opcodes covering integer, FP, memory and branches |
| `pow/hash.go` | Seed, hash, prefilter, share and block verification |
| `pow/consensus.go` | Oblivious shares, ASERT, timestamp rules, fork choice, MMR |
| `cmd/qalqon` | `vectors` and `bench` commands |
| `cmd/sim` | Withholding, difficulty and time-warp simulations |
| `testdata/` | Test vectors |

## Run

```
go test ./pow/ -v                          # 11 tests incl. test vectors
go run ./cmd/sim                           # results/simulations.json
go run ./cmd/qalqon bench -params mainnet  # needs ~2.5 GiB RAM
go run ./cmd/qalqon vectors                # regenerate testdata/
```

Requires Go 1.27+.

## Contributing and support

Issues, reviews and attacks on the design are welcome; see [CONTRIBUTING.md](CONTRIBUTING.md)
and [SECURITY.md](SECURITY.md).

## License

MIT, see [LICENSE](LICENSE).
