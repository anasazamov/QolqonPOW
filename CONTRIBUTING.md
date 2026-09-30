# Contributing

Thanks for looking at QalqonPoW. The most useful contributions right now are, in order:

1. **Attacks on the design.** TMTO shortcuts, ways to tell blocks from shares, spec/code
   mismatches. A written argument is enough; code is a bonus.
2. **Missing prior art.** If an idea here was published before, open an issue with the
   reference so that [PRIOR_ART.md](PRIOR_ART.md) can credit it.
3. **A second implementation** (C, Rust, ...) that reproduces `testdata/vectors_test_params.json`.
4. **Simulation improvements** in `cmd/sim`, for example FAW, selfish mining, or operator attacks.

## Ground rules

- Run `go vet ./...` and `go test ./pow/` before opening a pull request.
- Any change to hash output must regenerate the test vectors (`go run ./cmd/qalqon vectors`)
  and explain why the change is needed.
- Put numbers in `results/` together with the command that produced them.
- Keep claims measurable. "Resists ASICs" is not a claim; "recomputing an item costs 128× a
  read on CPU X" is.

## License of contributions

By submitting a contribution you agree that it is licensed under the MIT License of this repository.
