# Prior art and what QalqonPoW adds

QalqonPoW is mostly a **combination of known ideas**. This page lists, component by
component, where each idea comes from, so that reviewers can judge the actual
contribution. If you know of prior work that is missing here, please open an issue.

## Summary

| Component | Prior art | What is new here (if anything) |
|---|---|---|
| Blind-share (oblivious shares) | Rosenfeld 2011; Eyal & Sirer 2014; Towns 2024 | No new mechanism. A concrete spec, reference code, and a reproducible Monte Carlo check against the closed-form payoff |
| Memory-hard hash core (random programs, scratchpad, cache→dataset) | Ethash (cache/dataset), RandomX (random programs, scratchpad), scrypt ROMix (sequential memory-hard cache) | A latency-bound dataset item function and pipelined dataset reads (v0.2). **Not audited, and no evidence yet that it beats RandomX** |
| `mix_commit` in header for µs prefiltering | Ethash `mixHash` field; ProgPoW | None: included for pool DoS resistance |
| 256-bit seed from nonce + full header | ProgPoW 0.9.4 fix for the 64-bit seed bypass (Kik) | None: design constraint taken from that fix |
| ASERT difficulty adjustment | aserti3-2d, Bitcoin Cash (Nov 2020) | None: ported with integer arithmetic; halflife 1 day instead of 2 |
| Tight future-time limit against time-warp | Bitcoin time-warp analysis; Great Consensus Cleanup | Simulation that compares FTL 7200 s and 360 s under ASERT |
| MMR commitment in header for light clients | FlyClient (Bünz, Kiffer, Luu, Zamani); Zcash ZIP-221; Grin | None: in the header from genesis |
| Deterministic tie-break | Various proposals that replace the "first seen" rule | None |

## Block withholding and oblivious shares

- **Rosenfeld, "Analysis of Bitcoin Pooled Mining Reward Systems", 2011** (arXiv:1112.4980,
  section on block withholding). Proposes *oblivious shares*: the header commits to
  `ExtraHash = H(SecretSeed)`, and a block is valid only if `H(blockhash || SecretSeed)`
  also meets a condition. Miners cannot tell which shares are blocks. Needs a hard fork.
  **QalqonPoW's blind-share is this construction** (`BlindCommit`, `BlindOK` in `pow/consensus.go`).
- **Eyal & Sirer, "Two-phase proof of work", 2014.** A second puzzle over a signature made
  with the coinbase key, so that only the pool operator can complete a block.
- **Eyal, "The Miner's Dilemma", IEEE S&P 2015.** Pools gain by infiltrating each other;
  honest mining is not a Nash equilibrium.
- **Kwon et al., "Be Selfish and Avoid Dilemmas: Fork After Withholding (FAW) Attacks",
  ACM CCS 2017.** FAW dominates plain block withholding.
- **Lee & Kim, "Countering Block Withholding Attack Efficiently", 2018** (IACR ePrint 2018/1211).
- **Chang, "Share Withholding in Blockchain Mining", SecureComm 2020** (arXiv:2008.13317).
- **Towns, "Mining pools, stratumv2 and oblivious shares", bitcoin-dev mailing list,
  July 2024** (summarised in Bitcoin Optech, 9 Aug 2024). Revisits oblivious shares
  together with Stratum V2. Objections from Luke Dashjr and Matt Corallo: the pool must know
  the template, which pulls against miner-side template selection (Job Declaration).
- **Lerner, "APoW: Auditable Proof-of-Work Against Block Withholding Attacks", 2026**
  (arXiv:2601.02496). An auditing alternative that is also usable at pool level without a
  consensus change.

### Open questions that this project wants to study

1. **Template control.** How do oblivious shares combine with Stratum V2 Job Declaration,
   where the miner (not the pool) chooses transactions? Can the secret be bound to a job
   without the pool seeing the whole template?
2. **FAW and operator attacks.** Oblivious shares hide blocks from infiltrators, but the
   pool operator knows the secret. What attacks remain for a malicious operator?
3. **Decentralised pools.** Compatibility with P2Pool and Braidpool-style share chains,
   where there is no single secret holder.
4. **Secret reveal timing** and what light clients must check.

## Memory-hard hashing

- **Ethash / Dagger-Hashimoto**: small cache, large dataset derived from it, light
  verification. QalqonPoW follows the same cache/dataset split.
- **RandomX** (tevador et al., Monero 2019): random programs executed per nonce, scratchpad,
  AES. Four public audits (Trail of Bits, X41, Kudelski, QuarksLab). QalqonPoW's VM follows
  this idea, runs as an interpreter only, and has **no audit**.
- **scrypt ROMix** (Percival 2009), **Argon2d**: data-dependent, sequential memory-hard
  construction. The QalqonPoW cache uses a ROMix-style pass with BLAKE3.
- Known time-memory trade-off attacks on Argon2i and Balloon hashing are the reason a
  TMTO analysis is listed as open work.

## Difficulty adjustment and timestamps

- **aserti3-2d** (Mark Lundeberg, Jonathan Toomim and others, Bitcoin Cash, Nov 2020).
- **cw-144** oscillation in Bitcoin Cash (2017–2020), which ASERT replaced.
- **Time-warp** in Bitcoin's 2016-block retarget; mitigations in the Great Consensus Cleanup
  proposals.

## Light clients

- **FlyClient** (Bünz, Kiffer, Luu, Zamani, IEEE S&P 2020): MMR over headers and
  probabilistic sampling.
- **Zcash ZIP-221** (Heartwood): MMR commitment in the header.

## What is actually contributed

1. **An integrated, tested reference design.** Spec, Go code, 11 tests, and test vectors that
   put the countermeasures above into a single chain design.
2. **Reproducible simulations** (`cmd/sim`). They check the oblivious-shares payoff formula
   `(a − w·x)/(1 − w·x)` by Monte Carlo, compare four difficulty algorithms under
   coin-hopping, and replay time-warp under three rules.
3. **An honest list of open problems** (above and in the README) rather than claims.
