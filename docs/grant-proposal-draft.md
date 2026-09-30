# Grant proposal (draft)

> Draft template. Fill in the `[...]` fields, adjust the scope to each funder's focus, and
> check the funder's current rules and deadlines before applying.

## Title

Oblivious shares for modern pool protocols: open analysis, simulations and a
Stratum V2 compatibility study.

## Summary

Block withholding lets a pool member (or a rival pool) collect payouts while never submitting
blocks. Eyal (2015) showed that pools gain by attacking each other this way, and Fork After
Withholding (Kwon et al. 2017) makes it stronger. The main protocol-level defence,
*oblivious shares* (Rosenfeld 2011), was revisited on the bitcoin-dev list in 2024 (Towns),
where it ran into a design tension: Stratum V2 Job Declaration lets miners choose
transactions, while oblivious shares seemed to need the pool to know the template.

This project will produce an open, reproducible study of that tension. The work covers
formal payoff analysis, simulations of withholding, FAW and operator attacks, and a concrete
construction that binds a pool secret to a miner-declared job, or a clear argument for why
none exists. All code and write-ups will be MIT-licensed.

## Existing work (applicant's)

- Repository: https://github.com/anasazamov/QolqonPOW
- A reference design and Go implementation of oblivious shares, ASERT and a memory-hard PoW,
  with 11 tests and test vectors.
- Monte Carlo simulation that matches the closed-form payoff `(a − w·x)/(1 − w·x)`. It shows
  that any withholding strategy loses under oblivious shares, where classic PoW gives the
  attacker +1.85%.
- A prior-art review (PRIOR_ART.md) that credits Rosenfeld, Eyal & Sirer, Towns and others.

## Deliverables and milestones

| # | Deliverable | Output | Time |
|---|---|---|---|
| 1 | Survey and threat model | Paper section plus a public doc comparing oblivious shares, two-phase PoW, APoW, deposit schemes and statistical detection | Month 1–2 |
| 2 | Extended simulator | `cmd/sim` covering FAW, multi-pool games (Eyal 2015) and malicious-operator strategies, with reproducible configs | Month 2–3 |
| 3 | Stratum V2 compatibility study | Written design for binding the secret to Job Declaration jobs, and analysis of what the pool must learn | Month 3–5 |
| 4 | Prototype | Minimal pool/miner pair on a regtest-style chain that runs the chosen construction end to end | Month 5–6 |
| 5 | Write-up | IACR ePrint preprint and a Delving Bitcoin post that asks for review | Month 6 |

Each milestone ends with a public report in the repository.

## Relevance to [funder]

- **Bitcoin (OpenSats, Spiral, HRF BDF):** mining decentralisation. Non-KYC pools are the
  ones most exposed to withholding, as noted in the 2024 bitcoin-dev discussion.
  Deliverables 1–3 apply to Bitcoin directly, even though deploying oblivious shares there
  would need a hard fork.
- **Monero (CCS):** the TMTO analysis tooling for memory-hard PoW (open problem 4 in the
  README) could be scoped as a RandomX-applicable deliverable instead.

## Budget

| Item | Amount |
|---|---|
| Researcher/developer time: [N] months × [$ per month] | [$] |
| Compute for simulations | [$] |
| Total | [$] |

## Risks

- The Stratum V2 construction may turn out to be impossible without the pool seeing the
  template. That negative result is still useful, and the report will make the argument precise.
- The applicant has no prior grants. Mitigation: all intermediate work is public, and
  milestones are small and checkable.

## About the applicant

[Name], [location], [contact]. [Short background.]
