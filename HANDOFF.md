# QalqonPoW — loyiha holati va davom ettirish uchun prompt

Quyidagi matnni yangi sessiyaga to'liq nusxalab bering.

---

Men QalqonPoW nomli yangi proof-of-work algoritmini ishlab chiqyapman. Maqsad uni yangi L1 blockchain ishga tushirayotgan jamoalarga sotish (litsenziya, integratsiya, texnik yordam). O'zim mining qilmayman. Loyiha `D:\mining\qalqon` papkasida.

## Muhit
- Windows 11 Pro, Intel Core Ultra 7 265KF (20 yadro), 32 GB DDR5, RTX 5060 8 GB.
- Go 1.27 (modul `qalqon`, bog'liqlik `lukechampine.com/blake3`), Python 3.12 (`blake3`, `numpy`, `matplotlib`).
- Git repozitoriysi YO'Q. Avval zaxira nusxa oling va `git init` qiling.

## Qilingan tadqiqot (3 ta ko'p agentli deep-research)
1. RTX 5060 va 265KF uchun mining foydaliligi. O'zbekistonda jismoniy shaxsga uyda mining qilish qonuniy emas: NAPP faqat yuridik shaxslarga, quyosh energiyasi yoki 2x tarif bilan ruxsat beradi (napp.uz/en/pages/mining).
2. PoW qatlamlari (L1–L3) bo'yicha 21 ta tasdiqlangan da'vo:
   - ASIC ustunligi haqidagi dizayner baholari ishonchsiz: ProgPoW 1.1–1.2x deb baholagan, Equihash uchun esa Z9 ASIC chiqqan.
   - ProgPoW'dagi Kik bypass: seed 64-bitli bo'lgani uchun arzon bosqichni saralash mumkin edi, 0.9.4 da tuzatilgan.
   - Argon2i va Balloon'ga TMTO hujumlari topilgan.
   - RandomX: Trail of Bits auditi, 1R AES diffuziya zaif chiqqan, spetsifikatsiya bilan kod o'rtasida fork xavfi bor.
   - BCH'ning cw-144 qiyinlik algoritmi tebranish bergan, ASERT bilan tuzatilgan.
3. L4–L5 bo'yicha 23 ta tasdiqlangan da'vo:
   - FAW hujumi block withholding'dan kuchliroq (Kwon, CCS 2017), arzon yechimi yo'q.
   - Ergo nonoutsourceable jumboqlardan voz kechgan.
   - Firo 51% hujumidan keyin ChainLocks'ga o'tgan.
   - "Foydali ish" PoW xavfsizlik bermaydi: SoK 2025, Qubic Monero'da ~22% hashrate'ga erishgan.
   - FlyClient/MMR yengil klientlar uchun ishlaydi. Grover faqat kvadratik tezlashtirish beradi.
   - RandomX'ning 4 ta auditi bo'lgan, jamoa to'lagan uchtasi ~$118k.

Tasdiqlanmagan bo'shliqlar: Stratum V2 qabul qilinish holati, pool statistikasi, 51% hujumlar zararlari (ETC, BTG, Verge), selfish mining chegaralari, AsicBoost patentlari, Cuckoo va Equi-X.

## Spetsifikatsiya v0.1 (hujjat: https://claude.ai/artifact/4a5shQAzBRYZioQSHcZ2jF)
- BLAKE3-256, blok vaqti T=120 s, epoch 2048 blok, key lag 64.
- Kesh 256 MiB: BLAKE3 bilan ROMix uslubida, 3 o'tish.
- Dataset 2 GiB + 128 MiB/yil, har bir element 32 ta zanjirli ota elementdan.
- Scratchpad 2 MiB, AES-128-CTR bilan to'ldiriladi.
- 8 ta dastur × 2048 iteratsiya × 256 instruksiya, 19 ta opcode, har bir nonce uchun yangi tasodifiy dastur.
- Seed = BLAKE3(nonce || butun header). pow = BLAKE3(seed || mix_commit).
- Header'da `mix_commit` bor: mikrosekundli prefiltr uchun.
- Blind-share: block_valid = pow < T·2^k VA BLAKE3(pow || pool_secret) ning dastlabki k biti nol. Coinbase'da commit bor, k=16.
- ASERT (aserti3-2d, butun sonli), halflife 86400 s. Timestamp: > MTP(11) va <= mahalliy vaqt + 360 s.
- Fork tanlash eng katta kumulyativ ish bo'yicha, teng bo'lsa hash bo'yicha deterministik tie-break.
- Genesis'dan boshlab header'da MMR ildizi (FlyClient uchun). Tail emission.

## Kod (oxirgi ISHLAGAN holat, v0.1)
- `pow/params.go`: parametrlar.
- `pow/hashutil.go`: domen teglari.
- `pow/dataset.go`: kesh va dataset.
- `pow/vm.go`: VM interpretatori.
- `pow/hash.go`: hash, prefiltr, VerifyShare/VerifyBlock.
- `pow/consensus.go`: blind-share, ASERT, timestamp qoidalari, work, tie-break, MMR.
- `pow/pow_test.go`: 11 ta test (determinizm, light==fast, 704 bitning har biri xotira ishini o'zgartirishi, qalbaki mix, mining+verify, blind ehtimoli, ASERT, timestamp, tie-break, MMR, test vektorlari).
- `cmd/qalqon`: `vectors` va `bench` buyruqlari. `cmd/sim`: 3 ta simulyatsiya. `README.md`, `results/*.json`, `testdata/vectors_test_params.json`.

v0.1 natijalari:
- Benchmark: kesh 4.2 s, dataset 17.6 s, 1831 H/s, prefiltr 1.2 µs, to'liq tekshiruv 7.5 ms, light tekshiruv 192 ms, light-eval jarimasi 258x.
- Block withholding simulyatsiyasi: klassik PoW'da hujumchi +1.85%, blind-share'da har qanday strategiya zarar (−20.7% gacha).
- Time-warp, 30 kun: Bitcoin qoidasida 16 kunda 5 mln blok, ASERT'da 1.00x.
- Coin-hopping: >10 daqiqalik bloklar Bitcoin qoidasida 9.29%, SMA-144 da 1.01%, Qalqon'da 0.77%.

## HOZIRGI HOLAT: v0.2, kod ishlaydi (2026-09-29)
- Git: https://github.com/anasazamov/QolqonPOW (`main`).
- v0.1 dataset kodi saqlanmagan edi, shuning uchun v0.2 qoldirildi:
  - `pow/dataset.go`: dataset elementi `DatasetParents` ta zanjirli kesh murojaatidan hisoblanadi (DRAM kechikishiga bog'liq).
  - `pow/hash.go`: dataset manzili bir iteratsiya oldin hisoblanib, prefetch qilinadi.
- Yozilmay qolgan JIT'ga murojaatlar (`hs.jit`, `compile()`, `execute()`) olib tashlandi. VM faqat `exec` interpretatori bilan ishlaydi.
- `pow/prefetch_other.go`: amd64 bo'lmagan platformalar uchun bo'sh `prefetch`.
- `testdata/vectors_test_params.json` v0.2 uchun qayta yaratildi. `go vet` toza, 11/11 test o'tadi.
- v0.2 benchmark (mainnet, 20 oqim): dataset 8.2 s, 2349 H/s, prefiltr 1.1 µs, to'liq tekshiruv 5.8 ms, light tekshiruv 91 ms, light-eval jarimasi 128x. v0.1 raqamlari `results/*-v0.1.json` da. Simulyatsiyalar v0.1 bilan bayt-bayt bir xil (hash funksiyasiga bog'liq emas).
- Hali qilinmagan: spetsifikatsiya hujjatini v0.2 ga yangilash.

## Ochiq tadqiqot va grant yo'nalishi (2026-09-30)
- Loyiha "sotiladigan algoritm" emas, ochiq tadqiqot sifatida qayta rasmiylashtirildi. Litsenziya MIT, modul `github.com/anasazamov/QolqonPOW`.
- Adabiyot tekshiruvi (`PRIOR_ART.md`): blind-share bu Rosenfeld 2011 taklif qilgan "oblivious shares". Towns 2024-yilda bitcoin-dev ro'yxatida Stratum V2 bilan birga qayta ko'targan, Dashjr va Corallo e'tiroz bildirgan. Hash yadrosi Ethash va RandomX g'oyalariga, `mix_commit` Ethash `mixHash` ga asoslangan.
- Grant uchun asosiy tadqiqot savoli: oblivious shares va Stratum V2 Job Declaration'ni qanday birlashtirish mumkin. Qoralama: `docs/grant-proposal-draft.md`.
- Qo'shilgan fayllar: inglizcha `README.md` (o'zbekchasi `README.uz.md`), `CONTRIBUTING.md`, `SECURITY.md`, `.github/workflows/ci.yml`.

## Keyingi qadamlar
1. **Kodni tiklash.** v0.1 ga qaytaring: `hs.jit`, `compile()`, `execute()` ni olib tashlab, `exec` ni to'g'ridan-to'g'ri chaqiring, dataset va hash tartibini v0.1 ga qaytaring. Yoki v0.2 o'zgarishlarini qoldirib, spetsifikatsiyani yangilang va test vektorlarini qayta yarating. So'ng `go vet ./...`, `go test ./pow/ -v`, `go run ./cmd/sim` ishlating.
2. **`DATASET_PARENTS` bo'yicha qaror.** v0.2 da light tekshiruv 91 ms, jarima 128x. Uni light-eval jarimasi bilan murosada hal qiling va o'lchab tekshiring.
3. **TMTO/pebbling tahlili.** Kesh va dataset qurilishi uchun: xotira necha marta kamaysa, hisoblash necha marta oshadi.
4. **Blind-share'ning rasmiy tahlili.** Quyidagilarni ko'rib chiqing:
   - FAW va PAW;
   - pool operatori manipulyatsiyasi;
   - P2Pool/Braidpool bilan moslik;
   - sirni ochish vaqti va Stratum V2'ga integratsiya.
5. **Tadqiqotdagi bo'shliqlarni to'ldirish.** Yuqoridagi ro'yxat bo'yicha: Stratum V2, 51% hujumlar, selfish mining, AsicBoost patentlari.
6. **Mustaqil ASIC-narx modeli** (universitet yoki laboratoriya).
7. **Ikkinchi mustaqil implementatsiya** (C yoki Rust) va differensial fuzzing.
8. **Stratum V2 + blind-share pool prototipi, testnet.**
9. **Kamida 2 ta mustaqil audit** ($120k–250k).
10. **Oq qog'oz, xaridorlar uchun taqdimot, litsenziya modeli.** Model: ochiq kod va pullik integratsiya hamda texnik yordam.
