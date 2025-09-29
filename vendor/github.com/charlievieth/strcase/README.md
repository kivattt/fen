[![Tests](https://github.com/charlievieth/strcase/actions/workflows/test.yml/badge.svg)](https://github.com/charlievieth/strcase/actions/workflows/test.yml)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/charlievieth/strcase@master)
[![codecov](https://codecov.io/github/charlievieth/strcase/branch/master/graph/badge.svg?token=ZJ0FTBECVA)](https://codecov.io/github/charlievieth/strcase)

# strcase

Package [strcase](.) and [bytcase](bytcase/README.md) are case-insensitive and
Unicode aware implementations of the Go standard library's
[`strings`](https://pkg.go.dev/strings) and [`bytes`](https://pkg.go.dev/bytes)
packages that are accurate, fast, and never allocate memory.

Simple Unicode case-folding is used for all comparisons. This matches the behavior
of [strings.EqualFold](https://pkg.go.dev/strings#EqualFold) and [regexp.Regexp](https://pkg.go.dev/regexp#Regexp)
(when the pattern is compiled with the case-insensitive flag `(?i)`) and is more
accurate (and significantly more efficient) than using [strings.ToLower](https://pkg.go.dev/strings#ToLower) or
[strings.ToUpper](https://pkg.go.dev/strings#ToUpper) to normalize the needle /
haystack before searching.

**Note:** The bytcase package is analogous to the strcase package; whatever
applies to strcase also applies to bytcase. For simplicity, the documentation
primarily refers to strcase. Unless otherwise noted, all comments apply to both
packages.

## Overview

* Drop-in replacement for the [`strings`](https://pkg.go.dev/strings) and
  [`bytes`](https://pkg.go.dev/bytes) packages that provides Unicode
  aware case-insensitive matching.
* Simple Unicode case-folding is used for all comparisons - making it more
  accurate than using [strings.ToLower](https://pkg.go.dev/strings#ToLower) or
  [strings.ToUpper](https://pkg.go.dev/strings#ToUpper) for case-insensitivity.
  <!-- TODO: note the equivalent using the regexp package `(?i)` -->
* Any string matched by strcase or bytcase will also match with
  [strings.EqualFold](https://pkg.go.dev/strings#EqualFold) or
  [bytes.EqualFold](https://pkg.go.dev/bytes#EqualFold)
* Fast and optimized for amd64 and arm64. For non-pathological inputs strcase
  is only 25-50% slower than the strings package.
* On average strcase/bytcase are 25x faster than using than using a case-insensitive
  [regexp.Regexp](https://pkg.go.dev/regexp#Regexp) (see the below
  [benchmarks](#benchmarks) section).

## Installation

strcase is available using the standard `go get` command.

Install by running:
```sh
# strcase
go get github.com/charlievieth/strcase

# bytcase
go get github.com/charlievieth/strcase/bytcase
```

## Features

- Fast: strcase is optimized for `amd64` and `arm64` and strives to be no worse
  than 2x slower than the strings package and is often only 30-50% slower.
  - The `IndexByte`, `IndexNonASCII` and `Count` functions are implemented in
    assembly and use SSE/AVX2 on `amd64` and NEON on `arm64`.
  - Instead of using the [unicode](https://pkg.go.dev/unicode) package for case
    conversions strcase uses its own multiplicative hash tables that are roughly
    10x faster (at the cost of package size).
- Accurate: Unicode simple folding is used to determine equality.
  - Any matched text would also match with
    [`strings.EqualFold`](https://pkg.go.dev/strings#EqualFold).
- Zero allocation: none of the strcase functions allocate memory.
- Thoroughly tested and fuzzed.

## Additional Features

strcase provides two functons for checking if a string contains non-ASCII
characters that are highly optimized for amd64/arm64:

- [strcase.IndexNonASCII](https://pkg.go.dev/github.com/charlievieth/strcase#IndexNonASCII):
  IndexNonASCII returns the index of first non-ASCII rune in s, or -1 if s consists
  only of ASCII characters.
- [strcase.ContainsNonASCII](https://pkg.go.dev/github.com/charlievieth/strcase#ContainsNonASCII):
  ContainsNonASCII returns true if s contains any non-ASCII characters.

## Caveats

<!--
From: regexp
https://pkg.go.dev/regexp

All characters are UTF-8-encoded code points. Following utf8.DecodeRune,
each byte of an invalid UTF-8 sequence is treated as if it encoded utf8.RuneError (U+FFFD).
-->

All invalid UTF-8 sequences are considered equal. This is because Go converts
invalid UTF-8 sequences to the Unicode replacement character `0xFFFD`
([unicode.ReplacementChar](https://pkg.go.dev/unicode#pkg-constants) /
[utf8.RuneError](https://pkg.go.dev/unicode/utf8#pkg-constants)).
This occurs both when [ranging](https://go.dev/ref/spec#For_statements) over
a string or using the [utf8](https://pkg.go.dev/utf8) package's `Decode*`
functions.

```go
strcase.Compare("\xff", string(utf8.RuneError)) // returns 0
strcase.Index("a\xff", string(utf8.RuneError))  // returns 1
```

<!-- TODO: do we need this / mention that this does not really matter but is still tested -->
Thus it is the callers responsibility to ensure strcase functions are called
with valid UTF-8 strings and not arbitrary binary data.

## Performance

strcase aims to be seriously fast and can beat or match the performance of the
strings package in some benchmarks (EqualFold and IndexRune). Overall, strcase
tends to be only 30-50% slower than the strings package for non-pathological inputs.

#### Optimizations

- Instead of using the standard library's Unicode package, which uses a binary
  search for its lookup tables, strcase uses multiplicative hashing for its
  lookup tables. This is 10x faster at the cost of larger tables.
- Searching for runes (IndexRune) is a big determinant of the strcase packages
  performance. Instead of searching for runes by their first byte (like the
  strings package) strcase searches for the second byte, which is more unique,
  then looks backwards/forwards to complete the match.
   * **NB:** [CL 539116](https://go-review.googlesource.com/c/go/+/539116) added
   this logic to Go's strings/bytes packages and is available starting with Go 1.23.
- Package strcase is optimized for amd64 and arm64 and includes assembly
  implementations of `IndexByte`, `CountByte` and `IndexNonASCII` that
  leverage the same SIMD technologies used in the standard library (SSE, AVX2,
  NEON).
- On armv7l (Raspberry Pi), which we do not optimize for, the average
  performance penalty is only ~30%.

## Notes:
- All invalid Unicode points and invalid multibyte UTF-8 sequences are
 considered equal.
 This is because the [utf8](https://pkg.go.dev/unicode/utf8) package converts
 invalid runes and multibyte UTF-8 sequences to
 [`utf8.RuneError`](https://pkg.go.dev/unicode/utf8#RuneError).
  * This matches the behavior of [strings.EqualFold](https://pkg.go.dev/strings#EqualFold).
- `İ` (LATIN CAPITAL LETTER I WITH DOT ABOVE) and `ı`: (LATIN SMALL LETTER DOTLESS
  I) doe not fold to ASCII `[iI]` (U+0069 / U+0049)
   * This matches the behavior of [strings.EqualFold](https://pkg.go.dev/strings#EqualFold)
   * See [unicode.org/UCD/CaseFolding.txt](https://www.unicode.org/Public/UCD/latest/ucd/CaseFolding.txt)
     for an explanation, basically this folding is normally ignored for non-Turkic languages
- Kelvin `K` (U+212A) matches ASCII `K` and `k`
- Latin small letter long S `ſ` matches ASCII `S` and `s`

## Contributing / Hacking

Contributions are welcome. Please, see the [CONTRIBUTING](./CONTRIBUTING.md)
document for details.

## Benchmarks

<!-- WARN: make sure this renders correctly on GitHub -->

The following benchmarks were created using
[internal/benchtest](https://github.com/charlievieth/strcase/tree/master/internal/benchtest).
Additional, processor specific benchmarks can be found in
[internal/benchtest/results](https://github.com/charlievieth/strcase/tree/master/internal/benchtest/results).

<details>
<summary>regexp vs. strcase</summary>

```
goos: darwin
goarch: arm64
pkg: github.com/charlievieth/strcase/internal/benchtest
cpu: Apple M1 Max
                                 │  regexp.10.txt   │             strcase.10.txt             │
                                 │      sec/op      │    sec/op     vs base                  │
IndexRune-10                          587.45n ±  2%   11.15n ±  1%  -98.10% (p=0.000 n=10)
IndexRuneLongString-10                584.15n ±  2%   12.67n ±  1%  -97.83% (p=0.000 n=10)
IndexRuneFastPath-10                 673.300n ±  6%   5.130n ±  3%  -99.24% (p=0.000 n=10)
Index-10                             671.900n ±  2%   4.982n ±  1%  -99.26% (p=0.000 n=10)
EqualFold/ASCII-10                  2920.500n ±  1%   9.418n ±  2%  -99.68% (p=0.000 n=10)
EqualFold/UnicodePrefix-10           3892.00n ±  1%   32.35n ±  2%  -99.17% (p=0.000 n=10)
EqualFold/UnicodeSuffix-10           3885.50n ±  2%   26.17n ±  1%  -99.33% (p=0.000 n=10)
IndexHard1-10                          334.1µ ±  0%   340.4µ ±  0%   +1.90% (p=0.000 n=10)
IndexHard2-10                          9.963m ±  6%   1.362m ±  1%  -86.33% (p=0.000 n=10)
IndexHard3-10                         10.727m ±  0%   1.369m ±  1%  -87.24% (p=0.000 n=10)
IndexHard4-10                         10.385m ±  0%   1.361m ±  1%  -86.89% (p=0.000 n=10)
CountHard1-10                          338.1µ ±  1%   336.4µ ±  1%        ~ (p=0.218 n=10)
CountHard2-10                          9.844m ±  1%   1.365m ±  1%  -86.13% (p=0.000 n=10)
CountHard3-10                         10.684m ±  0%   1.362m ±  1%  -87.25% (p=0.000 n=10)
IndexTorture-10                     27787.99µ ±  1%   18.04µ ±  2%  -99.94% (p=0.000 n=10)
CountTorture-10                     27631.25µ ±  1%   17.86µ ±  2%  -99.94% (p=0.000 n=10)
CountTortureOverlapping-10         14204.297m ±  2%   4.058m ±  1%  -99.97% (p=0.000 n=10)
CountByte/10-10                      570.100n ±  1%   7.684n ±  1%  -98.65% (p=0.000 n=10)
CountByte/32-10                      964.050n ±  2%   4.501n ±  1%  -99.53% (p=0.000 n=10)
CountByte/4K-10                      63123.0n ±  1%   103.1n ±  0%  -99.84% (p=0.000 n=10)
CountByte/4M-10                     87646.31µ ±  0%   96.02µ ±  7%  -99.89% (p=0.000 n=10)
CountByte/64M-10                    1433.190m ±  1%   1.574m ±  1%  -99.89% (p=0.000 n=10)
IndexAnyASCII/1:1-10                 742.800n ±  2%   5.513n ±  1%  -99.26% (p=0.000 n=10)
IndexAnyASCII/1:2-10                 915.150n ±  2%   7.462n ±  0%  -99.18% (p=0.000 n=10)
IndexAnyASCII/1:4-10                1031.500n ±  7%   7.453n ±  0%  -99.28% (p=0.000 n=10)
IndexAnyASCII/1:8-10                1217.000n ±  2%   7.397n ±  0%  -99.39% (p=0.000 n=10)
IndexAnyASCII/1:16-10               1802.000n ±  2%   7.710n ±  1%  -99.57% (p=0.000 n=10)
IndexAnyASCII/1:32-10               3100.000n ±  4%   7.711n ±  0%  -99.75% (p=0.000 n=10)
IndexAnyASCII/1:64-10               5183.500n ±  2%   8.037n ±  0%  -99.84% (p=0.000 n=10)
IndexAnyASCII/16:1-10                731.000n ±  4%   5.457n ±  0%  -99.25% (p=0.000 n=10)
IndexAnyASCII/16:2-10                1096.00n ±  2%   13.44n ±  1%  -98.77% (p=0.000 n=10)
IndexAnyASCII/16:4-10                1231.00n ±  4%   15.39n ±  3%  -98.75% (p=0.000 n=10)
IndexAnyASCII/16:8-10                1429.50n ±  2%   19.38n ±  1%  -98.64% (p=0.000 n=10)
IndexAnyASCII/16:16-10               2009.50n ±  3%   34.40n ±  1%  -98.29% (p=0.000 n=10)
IndexAnyASCII/16:32-10               3325.00n ±  1%   61.90n ±  0%  -98.14% (p=0.000 n=10)
IndexAnyASCII/16:64-10                5380.5n ±  1%   124.8n ±  3%  -97.68% (p=0.000 n=10)
IndexAnyASCII/256:1-10               727.600n ±  2%   8.571n ±  3%  -98.82% (p=0.000 n=10)
IndexAnyASCII/256:2-10                4038.5n ±  1%   153.7n ±  0%  -96.19% (p=0.000 n=10)
IndexAnyASCII/256:4-10                4167.5n ±  1%   156.2n ±  1%  -96.25% (p=0.000 n=10)
IndexAnyASCII/256:8-10                4356.0n ±  1%   161.6n ±  0%  -96.29% (p=0.000 n=10)
IndexAnyASCII/256:16-10               4953.0n ±  1%   176.3n ±  3%  -96.44% (p=0.000 n=10)
IndexAnyASCII/256:32-10               6819.5n ±  4%   207.5n ±  1%  -96.96% (p=0.000 n=10)
IndexAnyASCII/256:64-10               8949.5n ±  6%   271.9n ±  0%  -96.96% (p=0.000 n=10)
IndexAnyUTF8/1:16-10                1860.500n ±  1%   7.296n ±  0%  -99.61% (p=0.000 n=10)
IndexAnyUTF8/16:16-10                2100.50n ±  1%   86.67n ±  0%  -95.87% (p=0.000 n=10)
IndexAnyUTF8/256:16-10                5902.0n ±  1%   119.8n ±  1%  -97.97% (p=0.000 n=10)
IndexPeriodic/IndexPeriodic2-10      1451.35µ ±  1%   84.06µ ±  0%  -94.21% (p=0.000 n=10)
IndexPeriodic/IndexPeriodic4-10      1529.07µ ±  1%   83.80µ ±  0%  -94.52% (p=0.000 n=10)
IndexPeriodic/IndexPeriodic8-10      1349.62µ ± 18%   83.59µ ±  1%  -93.81% (p=0.000 n=10)
IndexPeriodic/IndexPeriodic16-10     1234.51µ ±  1%   64.12µ ±  1%  -94.81% (p=0.000 n=10)
IndexPeriodic/IndexPeriodic32-10     1196.96µ ±  1%   30.81µ ±  1%  -97.43% (p=0.000 n=10)
IndexPeriodic/IndexPeriodic64-10     1169.90µ ±  8%   14.68µ ±  0%  -98.75% (p=0.000 n=10)
IndexByte_Bytes/10-10                  4.517n ± 20%
IndexByte_Bytes/32-10                  4.518n ±  1%
IndexByte_Bytes/4K-10                  81.42n ±  2%
IndexByte_Bytes/4M-10                  74.80µ ±  1%
IndexByte_Bytes/64M-10                 1.237m ±  5%
IndexRune_Bytes/10-10                  12.56n ±  1%   12.32n ±  0%   -1.91% (p=0.000 n=10)
IndexRune_Bytes/32-10                  12.59n ±  1%   12.32n ±  0%   -2.10% (p=0.000 n=10)
IndexRune_Bytes/4K-10                  85.06n ±  1%   83.56n ±  0%   -1.76% (p=0.000 n=10)
IndexRune_Bytes/4M-10                  65.05µ ±  1%   64.33µ ±  0%   -1.11% (p=0.030 n=10)
IndexRune_Bytes/64M-10                 1.149m ±  4%   1.169m ±  5%        ~ (p=0.089 n=10)
IndexRuneASCII_Bytes/10-10             6.439n ±  1%   6.388n ±  0%   -0.79% (p=0.025 n=10)
IndexRuneASCII_Bytes/32-10             6.462n ±  1%   6.369n ±  1%   -1.43% (p=0.000 n=10)
IndexRuneASCII_Bytes/4K-10             83.87n ±  0%   83.49n ± 11%        ~ (p=0.210 n=10)
IndexRuneASCII_Bytes/4M-10             75.19µ ±  2%   74.01µ ±  1%   -1.57% (p=0.023 n=10)
IndexRuneASCII_Bytes/64M-10            1.256m ±  5%   1.300m ±  2%        ~ (p=0.218 n=10)
IndexNonASCII_Bytes/10-10              3.063n ±  0%   2.967n ±  0%   -3.13% (p=0.000 n=10)
IndexNonASCII_Bytes/32-10              3.835n ±  2%   3.759n ±  0%   -1.98% (p=0.027 n=10)
IndexNonASCII_Bytes/4K-10              80.74n ±  1%   79.59n ±  0%   -1.44% (p=0.016 n=10)
IndexNonASCII_Bytes/4M-10              74.47µ ±  1%   74.51µ ±  1%        ~ (p=0.529 n=10)
IndexNonASCII_Bytes/64M-10             1.271m ±  4%   1.307m ±  2%        ~ (p=0.089 n=10)
geomean                                12.78µ         595.2n        -96.26%                ¹
¹ benchmark set differs from baseline; geomeans may not be comparable

                                 │ regexp.10.txt │                 strcase.10.txt                 │
                                 │      B/s      │       B/s         vs base                      │
IndexRune-10                       27.60Mi ±  2%    1454.81Mi ±  1%    +5171.18% (p=0.000 n=10)
IndexRuneLongString-10             191.0Mi ±  2%     8802.1Mi ±  1%    +4508.17% (p=0.000 n=10)
IndexRuneFastPath-10               25.50Mi ±  5%    3346.27Mi ±  3%   +13024.42% (p=0.000 n=10)
Index-10                                              3.365Gi ±  1%
EqualFold/ASCII-10                 3.920Mi ±  1%   1215.096Mi ±  2%   +30900.49% (p=0.000 n=10)
EqualFold/UnicodePrefix-10         4.411Mi ±  1%    530.591Mi ±  2%   +11929.51% (p=0.000 n=10)
EqualFold/UnicodeSuffix-10         4.416Mi ±  2%    656.018Mi ±  1%   +14757.13% (p=0.000 n=10)
IndexHard1-10                                         2.869Gi ±  0%
IndexHard2-10                                         734.1Mi ±  1%
IndexHard3-10                                         730.6Mi ±  1%
IndexHard4-10                                         734.5Mi ±  1%
CountHard1-10                      2.888Gi ±  1%      2.903Gi ±  1%            ~ (p=0.218 n=10)
CountHard2-10                      101.6Mi ±  1%      732.6Mi ±  1%     +621.19% (p=0.000 n=10)
CountHard3-10                      93.59Mi ±  0%     734.10Mi ±  1%     +684.35% (p=0.000 n=10)
IndexTorture-10                                       325.0Mi ±  2%
CountTorture-10                    214.8Ki ±  0%   336108.4Ki ±  2%  +156343.18% (p=0.000 n=10)
CountTortureOverlapping-10         214.8Ki ±  5%   756987.3Ki ±  1%  +352243.18% (p=0.000 n=10)
CountByte/10-10                    16.73Mi ±  1%    1240.98Mi ±  1%    +7318.84% (p=0.000 n=10)
CountByte/32-10                    31.66Mi ±  2%    6780.62Mi ±  1%   +21318.89% (p=0.000 n=10)
CountByte/4K-10                    61.88Mi ±  1%   37908.26Mi ±  0%   +61157.04% (p=0.000 n=10)
CountByte/4M-10                    45.64Mi ±  0%   41657.45Mi ±  7%   +91177.83% (p=0.000 n=10)
CountByte/64M-10                   44.66Mi ±  1%   40660.56Mi ±  1%   +90953.26% (p=0.000 n=10)
IndexPeriodic/IndexPeriodic2-10                       743.5Mi ±  0%
IndexPeriodic/IndexPeriodic4-10                       745.8Mi ±  0%
IndexPeriodic/IndexPeriodic8-10                       747.7Mi ±  1%
IndexPeriodic/IndexPeriodic16-10                      974.8Mi ±  1%
IndexPeriodic/IndexPeriodic32-10                      1.981Gi ±  1%
IndexPeriodic/IndexPeriodic64-10                      4.159Gi ±  0%
IndexByte_Bytes/10-10              2.062Gi ± 17%
IndexByte_Bytes/32-10              6.598Gi ±  1%
IndexByte_Bytes/4K-10              46.85Gi ±  2%
IndexByte_Bytes/4M-10              52.23Gi ±  1%
IndexByte_Bytes/64M-10             50.52Gi ±  5%
IndexRune_Bytes/10-10              759.2Mi ±  1%      773.8Mi ±  0%       +1.93% (p=0.000 n=10)
IndexRune_Bytes/32-10              2.367Gi ±  1%      2.418Gi ±  0%       +2.17% (p=0.000 n=10)
IndexRune_Bytes/4K-10              44.85Gi ±  1%      45.65Gi ±  0%       +1.79% (p=0.000 n=10)
IndexRune_Bytes/4M-10              60.05Gi ±  1%      60.72Gi ±  0%       +1.12% (p=0.035 n=10)
IndexRune_Bytes/64M-10             54.38Gi ±  4%      53.46Gi ±  5%            ~ (p=0.089 n=10)
IndexRuneASCII_Bytes/10-10         1.446Gi ±  1%      1.458Gi ±  0%       +0.80% (p=0.023 n=10)
IndexRuneASCII_Bytes/32-10         4.612Gi ±  1%      4.679Gi ±  1%       +1.46% (p=0.000 n=10)
IndexRuneASCII_Bytes/4K-10         45.48Gi ±  0%      45.69Gi ± 10%            ~ (p=0.218 n=10)
IndexRuneASCII_Bytes/4M-10         51.95Gi ±  2%      52.78Gi ±  1%       +1.60% (p=0.023 n=10)
IndexRuneASCII_Bytes/64M-10        49.75Gi ±  5%      48.09Gi ±  3%            ~ (p=0.218 n=10)
IndexNonASCII_Bytes/10-10          3.040Gi ±  0%      3.138Gi ±  0%       +3.23% (p=0.000 n=10)
IndexNonASCII_Bytes/32-10          7.772Gi ±  2%      7.928Gi ±  0%       +2.01% (p=0.023 n=10)
IndexNonASCII_Bytes/4K-10          47.24Gi ±  1%      47.93Gi ±  0%       +1.46% (p=0.019 n=10)
IndexNonASCII_Bytes/4M-10          52.46Gi ±  1%      52.43Gi ±  1%            ~ (p=0.529 n=10)
IndexNonASCII_Bytes/64M-10         49.19Gi ±  4%      47.81Gi ±  2%            ~ (p=0.089 n=10)
geomean                            832.0Mi            3.764Gi          +1121.52%                ¹
¹ benchmark set differs from baseline; geomeans may not be comparable

                                 │  regexp.10.txt   │               strcase.10.txt                │
                                 │       B/op       │     B/op      vs base                       │
IndexRune-10                           811.0 ± 0%         0.0 ± 0%  -100.00% (p=0.000 n=10)
IndexRuneLongString-10                 811.0 ± 0%         0.0 ± 0%  -100.00% (p=0.000 n=10)
IndexRuneFastPath-10                   791.0 ± 0%         0.0 ± 0%  -100.00% (p=0.000 n=10)
Index-10                               791.0 ± 0%         0.0 ± 0%  -100.00% (p=0.000 n=10)
EqualFold/ASCII-10                   4.658Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
EqualFold/UnicodePrefix-10           6.480Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
EqualFold/UnicodeSuffix-10           6.480Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexHard1-10                          914.0 ± 0%         0.0 ± 0%  -100.00% (p=0.000 n=10)
IndexHard2-10                        1.433Ki ± 4%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexHard3-10                        3.665Ki ± 1%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexHard4-10                        6.977Ki ± 1%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
CountHard1-10                          914.0 ± 0%         0.0 ± 0%  -100.00% (p=0.000 n=10)
CountHard2-10                        1.427Ki ± 3%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
CountHard3-10                        3.668Ki ± 1%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexTorture-10                      647.6Ki ± 6%       0.0Ki ± 0%  -100.00% (p=0.000 n=10)
CountTorture-10                      647.6Ki ± 6%       0.0Ki ± 0%  -100.00% (p=0.000 n=10)
CountTortureOverlapping-10         4424.21Ki ± 0%     10.53Ki ± 1%   -99.76% (p=0.000 n=10)
CountByte/10-10                        765.0 ± 0%         0.0 ± 0%  -100.00% (p=0.000 n=10)
CountByte/32-10                       1021.0 ± 0%         0.0 ± 0%  -100.00% (p=0.000 n=10)
CountByte/4K-10                      19.36Ki ± 0%      0.00Ki ± 0%  -100.00% (p=0.000 n=10)
CountByte/4M-10                      27.79Mi ± 0%      0.00Mi ± 0%  -100.00% (p=0.000 n=10)
CountByte/64M-10                     421.2Mi ± 0%       0.0Mi ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/1:1-10                 1.255Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/1:2-10                 1.524Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/1:4-10                 1.556Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/1:8-10                 1.635Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/1:16-10                1.849Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/1:32-10                2.358Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/1:64-10                3.119Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/16:1-10                1.255Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/16:2-10                1.522Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/16:4-10                1.554Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/16:8-10                1.635Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/16:16-10               1.851Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/16:32-10               2.360Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/16:64-10               3.122Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/256:1-10               1.255Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/256:2-10               1.515Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/256:4-10               1.547Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/256:8-10               1.628Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/256:16-10              1.844Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/256:32-10              2.352Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/256:64-10              3.112Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyUTF8/1:16-10                 1.890Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyUTF8/16:16-10                1.889Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexAnyUTF8/256:16-10               1.877Ki ± 0%     0.000Ki ± 0%  -100.00% (p=0.000 n=10)
IndexPeriodic/IndexPeriodic2-10      984.000 ± 1%       4.000 ± 0%   -99.59% (p=0.000 n=10)
IndexPeriodic/IndexPeriodic4-10      987.500 ± 1%       4.000 ± 0%   -99.59% (p=0.000 n=10)
IndexPeriodic/IndexPeriodic8-10      975.500 ± 1%       4.000 ± 0%   -99.59% (p=0.000 n=10)
IndexPeriodic/IndexPeriodic16-10     971.000 ± 1%       3.000 ± 0%   -99.69% (p=0.000 n=10)
IndexPeriodic/IndexPeriodic32-10     963.000 ± 1%       1.000 ± 0%   -99.90% (p=0.000 n=10)
IndexPeriodic/IndexPeriodic64-10       967.0 ± 1%         0.0 ± 0%  -100.00% (p=0.000 n=10)
IndexByte_Bytes/10-10                  0.000 ± 0%
IndexByte_Bytes/32-10                  0.000 ± 0%
IndexByte_Bytes/4K-10                  0.000 ± 0%
IndexByte_Bytes/4M-10                  261.0 ± 1%
IndexByte_Bytes/64M-10               69.35Ki ± 5%
IndexRune_Bytes/10-10                  0.000 ± 0%       0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexRune_Bytes/32-10                  0.000 ± 0%       0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexRune_Bytes/4K-10                  0.000 ± 0%       0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexRune_Bytes/4M-10                  226.0 ± 1%       224.0 ± 0%    -0.88% (p=0.001 n=10)
IndexRune_Bytes/64M-10               64.47Ki ± 2%     64.75Ki ± 8%         ~ (p=1.000 n=10)
IndexRuneASCII_Bytes/10-10             0.000 ± 0%       0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexRuneASCII_Bytes/32-10             0.000 ± 0%       0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexRuneASCII_Bytes/4K-10             0.000 ± 0%       0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexRuneASCII_Bytes/4M-10             260.5 ± 2%       258.0 ± 1%    -0.96% (p=0.011 n=10)
IndexRuneASCII_Bytes/64M-10          73.51Ki ± 5%     73.26Ki ± 3%         ~ (p=0.896 n=10)
IndexNonASCII_Bytes/10-10              0.000 ± 0%       0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexNonASCII_Bytes/32-10              0.000 ± 0%       0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexNonASCII_Bytes/4K-10              0.000 ± 0%       0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexNonASCII_Bytes/4M-10              258.5 ± 1%       260.0 ± 1%         ~ (p=0.343 n=10)
IndexNonASCII_Bytes/64M-10           72.14Ki ± 4%     71.55Ki ± 4%         ~ (p=0.305 n=10)
geomean                                           ²                 ?                       ³ ² ⁴
¹ all samples are equal
² summaries must be >0 to compute geomean
³ benchmark set differs from baseline; geomeans may not be comparable
⁴ ratios must be >0 to compute geomean

                                 │ regexp.10.txt  │               strcase.10.txt               │
                                 │   allocs/op    │  allocs/op   vs base                       │
IndexRune-10                        13.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexRuneLongString-10              13.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexRuneFastPath-10                11.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
Index-10                            11.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
EqualFold/ASCII-10                  71.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
EqualFold/UnicodePrefix-10          82.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
EqualFold/UnicodeSuffix-10          82.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexHard1-10                       14.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexHard2-10                       18.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexHard3-10                       22.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexHard4-10                       24.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
CountHard1-10                       14.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
CountHard2-10                       18.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
CountHard3-10                       22.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexTorture-10                     255.0 ± 47%        0.0 ± 0%  -100.00% (p=0.000 n=10)
CountTorture-10                     255.0 ± 47%        0.0 ± 0%  -100.00% (p=0.000 n=10)
CountTortureOverlapping-10         3.141k ±  0%     0.000k ± 0%  -100.00% (p=0.000 n=10)
CountByte/10-10                     10.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
CountByte/32-10                     12.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
CountByte/4K-10                     202.0 ±  0%        0.0 ± 0%  -100.00% (p=0.000 n=10)
CountByte/4M-10                    190.7k ±  0%       0.0k ± 0%  -100.00% (p=0.000 n=10)
CountByte/64M-10                   3.051M ±  0%     0.000M ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/1:1-10                16.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/1:2-10                18.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/1:4-10                18.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/1:8-10                18.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/1:16-10               20.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/1:32-10               22.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/1:64-10               22.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/16:1-10               16.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/16:2-10               18.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/16:4-10               18.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/16:8-10               18.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/16:16-10              20.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/16:32-10              22.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/16:64-10              22.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/256:1-10              16.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/256:2-10              18.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/256:4-10              18.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/256:8-10              18.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/256:16-10             20.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/256:32-10             22.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyASCII/256:64-10             22.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyUTF8/1:16-10                22.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyUTF8/16:16-10               22.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexAnyUTF8/256:16-10              22.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexPeriodic/IndexPeriodic2-10     12.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexPeriodic/IndexPeriodic4-10     12.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexPeriodic/IndexPeriodic8-10     12.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexPeriodic/IndexPeriodic16-10    12.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexPeriodic/IndexPeriodic32-10    12.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexPeriodic/IndexPeriodic64-10    12.00 ±  0%       0.00 ± 0%  -100.00% (p=0.000 n=10)
IndexByte_Bytes/10-10               0.000 ±  0%
IndexByte_Bytes/32-10               0.000 ±  0%
IndexByte_Bytes/4K-10               0.000 ±  0%
IndexByte_Bytes/4M-10               0.000 ±  0%
IndexByte_Bytes/64M-10              0.000 ±  0%
IndexRune_Bytes/10-10               0.000 ±  0%      0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexRune_Bytes/32-10               0.000 ±  0%      0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexRune_Bytes/4K-10               0.000 ±  0%      0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexRune_Bytes/4M-10               0.000 ±  0%      0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexRune_Bytes/64M-10              0.000 ±  0%      0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexRuneASCII_Bytes/10-10          0.000 ±  0%      0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexRuneASCII_Bytes/32-10          0.000 ±  0%      0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexRuneASCII_Bytes/4K-10          0.000 ±  0%      0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexRuneASCII_Bytes/4M-10          0.000 ±  0%      0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexRuneASCII_Bytes/64M-10         0.000 ±  0%      0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexNonASCII_Bytes/10-10           0.000 ±  0%      0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexNonASCII_Bytes/32-10           0.000 ±  0%      0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexNonASCII_Bytes/4K-10           0.000 ±  0%      0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexNonASCII_Bytes/4M-10           0.000 ±  0%      0.000 ± 0%         ~ (p=1.000 n=10) ¹
IndexNonASCII_Bytes/64M-10          0.000 ±  0%      0.000 ± 0%         ~ (p=1.000 n=10) ¹
geomean                                         ²                ?                       ³ ² ⁴
¹ all samples are equal
² summaries must be >0 to compute geomean
³ benchmark set differs from baseline; geomeans may not be comparable
⁴ ratios must be >0 to compute geomean
```

</details>

<details>
<summary>arm64</summary>

```
goos: darwin
goarch: arm64
pkg: github.com/charlievieth/strcase/internal/benchtest
                                 │ stdlib.5.1681417522.txt │        strcase.5.1681417522.txt         │
                                 │         sec/op          │     sec/op      vs base                 │
IndexRune-10                                  12.11n ±  1%     11.67n ±  1%     -3.59% (p=0.002 n=6)
IndexRuneLongString-10                        13.44n ±  1%     13.02n ±  1%     -3.12% (p=0.002 n=6)
IndexRuneFastPath-10                          2.990n ±  1%     4.776n ±  1%    +59.75% (p=0.002 n=6)
Index-10                                      3.155n ±  0%     4.729n ±  2%    +49.87% (p=0.002 n=6)
LastIndex-10                                  3.511n ±  1%     6.615n ±  1%    +88.38% (p=0.002 n=6)
IndexByte-10                                  2.441n ±  1%     3.572n ±  0%    +46.33% (p=0.002 n=6)
EqualFold/ASCII-10                            9.619n ±  1%     9.287n ±  0%     -3.45% (p=0.002 n=6)
EqualFold/UnicodePrefix-10                    80.37n ±  1%     32.03n ±  2%    -60.14% (p=0.002 n=6)
EqualFold/UnicodeSuffix-10                    73.48n ±  1%     25.54n ±  2%    -65.25% (p=0.002 n=6)
IndexHard1-10                                 326.9µ ±  0%     331.7µ ±  0%     +1.47% (p=0.002 n=6)
IndexHard2-10                                 327.6µ ±  0%    2165.8µ ±  1%   +560.99% (p=0.002 n=6)
IndexHard3-10                                 357.0µ ±  1%    1854.5µ ±  3%   +419.51% (p=0.002 n=6)
IndexHard4-10                                 1.304m ±  0%     1.348m ±  2%     +3.35% (p=0.002 n=6)
LastIndexHard1-10                             1.305m ±  0%     1.419m ±  1%     +8.68% (p=0.002 n=6)
LastIndexHard2-10                             1.305m ±  1%     1.420m ±  1%     +8.76% (p=0.002 n=6)
LastIndexHard3-10                             1.311m ±  4%     1.420m ±  1%     +8.36% (p=0.002 n=6)
CountHard1-10                                 333.0µ ±  2%     335.8µ ±  1%          ~ (p=0.485 n=6)
CountHard2-10                                 331.3µ ±  2%    2166.3µ ±  1%   +553.98% (p=0.002 n=6)
CountHard3-10                                 357.6µ ±  1%    1851.0µ ±  1%   +417.59% (p=0.002 n=6)
IndexTorture-10                               9.834µ ±  3%    17.747µ ±  1%    +80.48% (p=0.002 n=6)
CountTorture-10                               10.09µ ±  0%     20.17µ ±  1%    +99.81% (p=0.002 n=6)
CountTortureOverlapping-10                    68.63µ ±  1%   4048.06µ ±  1%  +5798.77% (p=0.002 n=6)
CountByte/10-10                               6.731n ±  2%     7.590n ±  1%    +12.76% (p=0.002 n=6)
CountByte/32-10                               3.164n ±  1%     4.153n ±  1%    +31.24% (p=0.002 n=6)
CountByte/4096-10                             80.32n ±  0%    100.80n ±  1%    +25.50% (p=0.002 n=6)
CountByte/4194304-10                          82.84µ ±  2%     95.14µ ± 11%    +14.84% (p=0.002 n=6)
CountByte/67108864-10                         1.367m ±  2%     1.556m ±  5%    +13.84% (p=0.002 n=6)
IndexAnyASCII/1:1-10                          4.195n ±  1%     5.742n ±  1%    +36.88% (p=0.002 n=6)
IndexAnyASCII/1:2-10                          5.436n ±  0%     7.340n ±  1%    +35.05% (p=0.002 n=6)
IndexAnyASCII/1:4-10                          5.440n ±  0%     7.405n ±  3%    +36.13% (p=0.002 n=6)
IndexAnyASCII/1:8-10                          5.377n ±  1%     7.428n ±  1%    +38.14% (p=0.002 n=6)
IndexAnyASCII/1:16-10                         5.362n ±  1%     7.391n ±  0%    +37.84% (p=0.002 n=6)
IndexAnyASCII/1:32-10                         5.374n ±  0%     7.724n ±  1%    +43.72% (p=0.002 n=6)
IndexAnyASCII/1:64-10                         5.980n ±  1%     7.985n ±  1%    +33.52% (p=0.002 n=6)
IndexAnyASCII/16:1-10                         4.046n ±  2%     5.809n ±  2%    +43.57% (p=0.002 n=6)
IndexAnyASCII/16:2-10                         11.18n ±  1%     13.49n ±  1%    +20.62% (p=0.002 n=6)
IndexAnyASCII/16:4-10                         12.40n ±  0%     15.47n ±  2%    +24.81% (p=0.002 n=6)
IndexAnyASCII/16:8-10                         17.21n ±  1%     19.72n ±  3%    +14.58% (p=0.002 n=6)
IndexAnyASCII/16:16-10                        33.90n ±  0%     34.49n ±  1%     +1.74% (p=0.002 n=6)
IndexAnyASCII/16:32-10                        66.39n ±  0%     62.72n ±  1%     -5.54% (p=0.002 n=6)
IndexAnyASCII/16:64-10                        131.5n ±  0%     126.6n ±  2%     -3.73% (p=0.002 n=6)
IndexAnyASCII/256:1-10                        7.154n ±  0%     8.974n ±  1%    +25.44% (p=0.002 n=6)
IndexAnyASCII/256:2-10                        149.8n ±  0%     154.9n ±  1%     +3.44% (p=0.002 n=6)
IndexAnyASCII/256:4-10                        151.5n ±  1%     157.4n ±  1%     +3.89% (p=0.002 n=6)
IndexAnyASCII/256:8-10                        156.2n ±  0%     162.5n ±  1%     +3.97% (p=0.002 n=6)
IndexAnyASCII/256:16-10                       166.8n ±  0%     176.4n ±  1%     +5.72% (p=0.002 n=6)
IndexAnyASCII/256:32-10                       199.6n ±  0%     207.5n ±  1%     +3.98% (p=0.002 n=6)
IndexAnyASCII/256:64-10                       264.3n ±  0%     271.6n ±  1%     +2.76% (p=0.002 n=6)
IndexAnyUTF8/1:1-10                           3.105n ±  1%     3.177n ±  1%     +2.32% (p=0.002 n=6)
IndexAnyUTF8/1:2-10                           5.301n ±  1%     7.312n ±  0%    +37.94% (p=0.002 n=6)
IndexAnyUTF8/1:4-10                           5.315n ±  1%     7.340n ±  1%    +38.10% (p=0.002 n=6)
IndexAnyUTF8/1:8-10                           5.312n ±  1%     7.390n ±  1%    +39.11% (p=0.002 n=6)
IndexAnyUTF8/1:16-10                          5.331n ±  1%     7.913n ±  2%    +48.45% (p=0.002 n=6)
IndexAnyUTF8/1:32-10                          5.359n ±  0%     7.973n ±  1%    +48.78% (p=0.002 n=6)
IndexAnyUTF8/1:64-10                          5.998n ±  1%     8.233n ±  1%    +37.25% (p=0.002 n=6)
IndexAnyUTF8/16:1-10                          13.04n ±  0%     13.83n ±  1%     +6.06% (p=0.002 n=6)
IndexAnyUTF8/16:2-10                          63.08n ±  0%     35.06n ±  1%    -44.42% (p=0.002 n=6)
IndexAnyUTF8/16:4-10                          63.39n ±  1%     34.78n ±  0%    -45.14% (p=0.002 n=6)
IndexAnyUTF8/16:8-10                          63.30n ±  0%     90.81n ±  2%    +43.46% (p=0.002 n=6)
IndexAnyUTF8/16:16-10                         66.91n ±  2%     97.80n ±  2%    +46.16% (p=0.002 n=6)
IndexAnyUTF8/16:32-10                         66.80n ±  1%     97.09n ±  2%    +45.34% (p=0.002 n=6)
IndexAnyUTF8/16:64-10                         75.27n ±  1%    103.00n ±  1%    +36.85% (p=0.002 n=6)
IndexAnyUTF8/256:1-10                         168.9n ±  0%     179.2n ±  1%     +6.10% (p=0.002 n=6)
IndexAnyUTF8/256:2-10                         887.1n ±  1%     372.0n ±  1%    -58.07% (p=0.002 n=6)
IndexAnyUTF8/256:4-10                         892.0n ±  6%     201.2n ±  1%    -77.44% (p=0.002 n=6)
IndexAnyUTF8/256:8-10                         894.9n ±  0%     404.9n ±  1%    -54.75% (p=0.002 n=6)
IndexAnyUTF8/256:16-10                        940.5n ±  0%     123.3n ±  0%    -86.88% (p=0.002 n=6)
IndexAnyUTF8/256:32-10                        942.6n ±  2%     630.1n ±  1%    -33.16% (p=0.002 n=6)
IndexAnyUTF8/256:64-10                       1063.0n ±  0%     730.6n ±  2%    -31.27% (p=0.002 n=6)
LastIndexAnyASCII/1:1-10                      4.433n ±  1%     5.865n ±  1%    +32.33% (p=0.002 n=6)
LastIndexAnyASCII/1:2-10                      4.430n ±  0%     5.872n ±  1%    +32.55% (p=0.002 n=6)
LastIndexAnyASCII/1:4-10                      4.426n ±  1%     5.865n ±  0%    +32.53% (p=0.002 n=6)
LastIndexAnyASCII/1:8-10                      4.428n ±  0%     5.885n ±  2%    +32.92% (p=0.002 n=6)
LastIndexAnyASCII/1:16-10                     4.428n ±  0%     5.972n ±  1%    +34.86% (p=0.002 n=6)
LastIndexAnyASCII/1:32-10                     4.433n ±  0%     6.282n ±  3%    +41.71% (p=0.002 n=6)
LastIndexAnyASCII/1:64-10                     5.067n ±  1%     6.316n ±  1%    +24.65% (p=0.002 n=6)
LastIndexAnyASCII/16:1-10                     10.29n ±  3%     11.59n ±  1%    +12.58% (p=0.002 n=6)
LastIndexAnyASCII/16:2-10                     10.99n ±  0%     12.59n ±  1%    +14.51% (p=0.002 n=6)
LastIndexAnyASCII/16:4-10                     12.27n ± 11%     14.53n ±  1%    +18.42% (p=0.002 n=6)
LastIndexAnyASCII/16:8-10                     17.25n ±  1%     19.10n ±  1%    +10.72% (p=0.002 n=6)
LastIndexAnyASCII/16:16-10                    33.81n ±  0%     34.14n ±  1%     +0.96% (p=0.002 n=6)
LastIndexAnyASCII/16:32-10                    66.48n ±  0%     62.01n ±  1%     -6.72% (p=0.002 n=6)
LastIndexAnyASCII/16:64-10                    131.7n ±  0%     125.8n ±  1%     -4.48% (p=0.002 n=6)
LastIndexAnyASCII/256:1-10                    147.1n ±  0%     156.8n ±  1%     +6.56% (p=0.002 n=6)
LastIndexAnyASCII/256:2-10                    146.6n ±  1%     154.8n ±  1%     +5.63% (p=0.002 n=6)
LastIndexAnyASCII/256:4-10                    149.0n ±  0%     155.4n ±  1%     +4.23% (p=0.002 n=6)
LastIndexAnyASCII/256:8-10                    153.1n ±  0%     160.4n ±  1%     +4.84% (p=0.002 n=6)
LastIndexAnyASCII/256:16-10                   165.5n ±  0%     172.4n ±  1%     +4.17% (p=0.002 n=6)
LastIndexAnyASCII/256:32-10                   197.7n ±  0%     210.4n ±  1%     +6.42% (p=0.002 n=6)
LastIndexAnyASCII/256:64-10                   263.8n ±  0%     270.8n ±  0%     +2.65% (p=0.002 n=6)
LastIndexAnyUTF8/1:1-10                       4.398n ±  1%     5.692n ±  0%    +29.41% (p=0.002 n=6)
LastIndexAnyUTF8/1:2-10                       4.359n ±  1%     5.646n ±  0%    +29.52% (p=0.002 n=6)
LastIndexAnyUTF8/1:4-10                       4.364n ±  2%     5.668n ±  1%    +29.89% (p=0.002 n=6)
LastIndexAnyUTF8/1:8-10                       4.362n ±  1%     5.711n ±  0%    +30.91% (p=0.002 n=6)
LastIndexAnyUTF8/1:16-10                      4.429n ±  0%     6.025n ±  1%    +36.05% (p=0.002 n=6)
LastIndexAnyUTF8/1:32-10                      4.427n ±  0%     6.024n ±  0%    +36.09% (p=0.002 n=6)
LastIndexAnyUTF8/1:64-10                      5.060n ±  1%     6.315n ±  1%    +24.80% (p=0.002 n=6)
LastIndexAnyUTF8/16:1-10                      25.19n ±  1%     27.81n ±  1%    +10.42% (p=0.002 n=6)
LastIndexAnyUTF8/16:2-10                      74.92n ±  0%     97.78n ±  2%    +30.51% (p=0.002 n=6)
LastIndexAnyUTF8/16:4-10                      75.02n ±  1%     97.39n ±  1%    +29.83% (p=0.002 n=6)
LastIndexAnyUTF8/16:8-10                      75.01n ±  0%     97.67n ±  1%    +30.21% (p=0.002 n=6)
LastIndexAnyUTF8/16:16-10                     80.07n ±  0%    103.55n ±  1%    +29.32% (p=0.002 n=6)
LastIndexAnyUTF8/16:32-10                     80.05n ±  0%    103.60n ±  1%    +29.42% (p=0.002 n=6)
LastIndexAnyUTF8/16:64-10                     86.72n ±  0%    108.75n ±  1%    +25.40% (p=0.002 n=6)
LastIndexAnyUTF8/256:1-10                     553.9n ±  0%     556.2n ±  1%          ~ (p=0.102 n=6)
LastIndexAnyUTF8/256:2-10                     1.064µ ±  0%     1.393µ ±  1%    +30.98% (p=0.002 n=6)
LastIndexAnyUTF8/256:4-10                     1.062µ ±  0%     1.392µ ±  1%    +31.01% (p=0.002 n=6)
LastIndexAnyUTF8/256:8-10                     1.063µ ±  0%     1.388µ ±  1%    +30.57% (p=0.002 n=6)
LastIndexAnyUTF8/256:16-10                    1.144µ ±  0%     1.478µ ±  1%    +29.15% (p=0.002 n=6)
LastIndexAnyUTF8/256:32-10                    1.144µ ±  0%     1.483µ ±  1%    +29.59% (p=0.002 n=6)
LastIndexAnyUTF8/256:64-10                    1.237µ ±  0%     1.565µ ±  1%    +26.52% (p=0.002 n=6)
IndexPeriodic/IndexPeriodic2-10               20.49µ ±  0%     62.58µ ±  0%   +205.45% (p=0.002 n=6)
IndexPeriodic/IndexPeriodic4-10               20.52µ ±  0%     57.41µ ±  1%   +179.79% (p=0.002 n=6)
IndexPeriodic/IndexPeriodic8-10               20.53µ ±  0%     54.92µ ±  0%   +167.48% (p=0.002 n=6)
IndexPeriodic/IndexPeriodic16-10              55.62µ ±  1%     64.69µ ±  1%    +16.31% (p=0.002 n=6)
IndexPeriodic/IndexPeriodic32-10              27.72µ ±  1%     32.38µ ±  0%    +16.80% (p=0.002 n=6)
IndexPeriodic/IndexPeriodic64-10              15.21µ ±  0%     16.76µ ±  1%    +10.21% (p=0.002 n=6)
IndexByte_Bytes/10-10                         3.075n ±  0%     3.815n ±  0%    +24.07% (p=0.002 n=6)
IndexByte_Bytes/32-10                         2.844n ±  0%     3.510n ±  1%    +23.44% (p=0.002 n=6)
IndexByte_Bytes/4K-10                         71.52n ±  2%     80.02n ±  1%    +11.90% (p=0.002 n=6)
IndexByte_Bytes/4M-10                         62.95µ ±  0%     74.11µ ±  1%    +17.72% (p=0.002 n=6)
IndexByte_Bytes/64M-10                        1.110m ±  1%     1.211m ±  7%     +9.10% (p=0.002 n=6)
IndexRune_Bytes/10-10                         10.57n ±  0%     12.46n ±  1%    +17.83% (p=0.002 n=6)
IndexRune_Bytes/32-10                         11.81n ±  0%     12.42n ±  1%     +5.21% (p=0.002 n=6)
IndexRune_Bytes/4K-10                         83.05n ±  1%     85.73n ±  1%     +3.22% (p=0.002 n=6)
IndexRune_Bytes/4M-10                         63.88µ ±  0%     64.70µ ±  0%     +1.29% (p=0.002 n=6)
IndexRune_Bytes/64M-10                        1.110m ±  0%     1.152m ±  5%     +3.83% (p=0.002 n=6)
IndexRuneASCII_Bytes/10-10                    3.163n ±  0%     5.428n ±  0%    +71.62% (p=0.002 n=6)
IndexRuneASCII_Bytes/32-10                    3.166n ±  0%     5.432n ±  1%    +71.57% (p=0.002 n=6)
IndexRuneASCII_Bytes/4K-10                    71.84n ±  1%     83.24n ±  1%    +15.88% (p=0.002 n=6)
IndexRuneASCII_Bytes/4M-10                    64.09µ ±  0%     74.33µ ±  0%    +15.98% (p=0.002 n=6)
IndexRuneASCII_Bytes/64M-10                   1.112m ±  0%     1.245m ±  5%    +11.95% (p=0.002 n=6)
IndexNonASCII_Bytes/10-10                     4.673n ±  0%     3.018n ±  2%    -35.41% (p=0.002 n=6)
IndexNonASCII_Bytes/32-10                    11.535n ±  0%     2.872n ±  1%    -75.10% (p=0.002 n=6)
IndexNonASCII_Bytes/4K-10                   1287.00n ±  1%     79.36n ±  1%    -93.83% (p=0.002 n=6)
IndexNonASCII_Bytes/4M-10                   1323.89µ ±  0%     74.08µ ±  1%    -94.40% (p=0.002 n=6)
IndexNonASCII_Bytes/64M-10                   21.442m ±  1%     1.213m ±  3%    -94.34% (p=0.002 n=6)
geomean                                       198.3n           224.5n          +13.21%

                            │ stdlib.5.1681417522.txt │        strcase.5.1681417522.txt         │
                            │           B/s           │      B/s        vs base                 │
CountByte/10-10                          1.384Gi ± 2%    1.227Gi ±  1%    -11.32% (p=0.002 n=6)
CountByte/32-10                          9.418Gi ± 1%    7.176Gi ±  1%    -23.81% (p=0.002 n=6)
CountByte/4096-10                        47.49Gi ± 0%    37.84Gi ±  1%    -20.32% (p=0.002 n=6)
CountByte/4194304-10                     47.16Gi ± 2%    41.06Gi ± 10%    -12.93% (p=0.002 n=6)
CountByte/67108864-10                    45.72Gi ± 2%    40.16Gi ±  4%    -12.16% (p=0.002 n=6)
IndexByte_Bytes/10-10                    3.028Gi ± 0%    2.441Gi ±  0%    -19.39% (p=0.002 n=6)
IndexByte_Bytes/32-10                   10.479Gi ± 0%    8.490Gi ±  1%    -18.98% (p=0.002 n=6)
IndexByte_Bytes/4K-10                    53.34Gi ± 2%    47.67Gi ±  1%    -10.63% (p=0.002 n=6)
IndexByte_Bytes/4M-10                    62.05Gi ± 0%    52.71Gi ±  1%    -15.05% (p=0.002 n=6)
IndexByte_Bytes/64M-10                   56.32Gi ± 1%    51.62Gi ±  6%     -8.34% (p=0.002 n=6)
IndexRune_Bytes/10-10                    901.9Mi ± 0%    765.7Mi ±  1%    -15.11% (p=0.002 n=6)
IndexRune_Bytes/32-10                    2.525Gi ± 0%    2.400Gi ±  1%     -4.96% (p=0.002 n=6)
IndexRune_Bytes/4K-10                    45.93Gi ± 1%    44.50Gi ±  1%     -3.12% (p=0.002 n=6)
IndexRune_Bytes/4M-10                    61.15Gi ± 0%    60.37Gi ±  0%     -1.27% (p=0.002 n=6)
IndexRune_Bytes/64M-10                   56.31Gi ± 0%    54.24Gi ±  5%     -3.68% (p=0.002 n=6)
IndexRuneASCII_Bytes/10-10               2.945Gi ± 0%    1.716Gi ±  0%    -41.74% (p=0.002 n=6)
IndexRuneASCII_Bytes/32-10               9.413Gi ± 0%    5.486Gi ±  1%    -41.72% (p=0.002 n=6)
IndexRuneASCII_Bytes/4K-10               53.10Gi ± 1%    45.83Gi ±  1%    -13.70% (p=0.002 n=6)
IndexRuneASCII_Bytes/4M-10               60.95Gi ± 0%    52.55Gi ±  0%    -13.78% (p=0.002 n=6)
IndexRuneASCII_Bytes/64M-10              56.21Gi ± 0%    50.22Gi ±  5%    -10.67% (p=0.002 n=6)
IndexNonASCII_Bytes/10-10                1.993Gi ± 0%    3.086Gi ±  2%    +54.86% (p=0.002 n=6)
IndexNonASCII_Bytes/32-10                2.583Gi ± 0%   10.377Gi ±  1%   +301.71% (p=0.002 n=6)
IndexNonASCII_Bytes/4K-10                2.963Gi ± 1%   48.067Gi ±  1%  +1521.97% (p=0.002 n=6)
IndexNonASCII_Bytes/4M-10                2.951Gi ± 0%   52.731Gi ±  1%  +1687.12% (p=0.002 n=6)
IndexNonASCII_Bytes/64M-10               2.915Gi ± 1%   51.541Gi ±  3%  +1668.19% (p=0.002 n=6)
geomean                                  12.32Gi         16.24Gi          +31.83%
```

</details>

<details>
<summary>amd64</summary>

```
goos: linux
goarch: amd64
pkg: github.com/charlievieth/strcase/internal/benchtest
cpu: Intel(R) Core(TM) i9-9900K CPU @ 3.60GHz
                                │ stdlib.5.1681426916.txt │        strcase.5.1681426916.txt         │
                                │         sec/op          │     sec/op      vs base                 │
IndexRune-8                                  10.95n ±  1%     12.09n ±  0%    +10.41% (p=0.008 n=5)
IndexRuneLongString-8                        14.83n ±  0%     13.19n ±  1%    -11.06% (p=0.008 n=5)
IndexRuneFastPath-8                          3.726n ±  2%     6.431n ±  1%    +72.60% (p=0.008 n=5)
Index-8                                      4.206n ±  2%     6.593n ±  0%    +56.75% (p=0.008 n=5)
LastIndex-8                                  3.621n ±  1%     5.547n ±  0%    +53.19% (p=0.008 n=5)
IndexByte-8                                  2.835n ±  1%     4.564n ±  1%    +60.99% (p=0.008 n=5)
EqualFold/ASCII-8                            8.269n ±  1%     8.777n ±  1%     +6.14% (p=0.008 n=5)
EqualFold/UnicodePrefix-8                    84.69n ±  0%     37.79n ±  1%    -55.38% (p=0.008 n=5)
EqualFold/UnicodeSuffix-8                    75.96n ±  1%     28.75n ±  1%    -62.15% (p=0.008 n=5)
IndexHard1-8                                 79.80µ ±  1%     79.67µ ±  1%          ~ (p=0.421 n=5)
IndexHard2-8                                 111.8µ ±  0%    2217.6µ ±  1%  +1883.07% (p=0.008 n=5)
IndexHard3-8                                 450.1µ ±  1%    2070.0µ ±  1%   +359.95% (p=0.008 n=5)
IndexHard4-8                                 447.3µ ±  0%    1573.6µ ±  0%   +251.81% (p=0.008 n=5)
LastIndexHard1-8                             1.134m ±  1%     1.580m ±  1%    +39.28% (p=0.008 n=5)
LastIndexHard2-8                             1.133m ±  0%     1.579m ±  2%    +39.38% (p=0.008 n=5)
LastIndexHard3-8                             1.130m ±  0%     1.578m ±  1%    +39.60% (p=0.008 n=5)
CountHard1-8                                 79.66µ ±  1%     79.82µ ±  2%          ~ (p=0.286 n=5)
CountHard2-8                                 111.8µ ±  0%    2226.7µ ±  1%  +1890.99% (p=0.008 n=5)
CountHard3-8                                 447.7µ ±  1%    2071.9µ ±  2%   +362.77% (p=0.008 n=5)
IndexTorture-8                               8.706µ ±  0%    16.883µ ±  1%    +93.92% (p=0.008 n=5)
CountTorture-8                               8.693µ ±  1%    18.808µ ±  1%   +116.36% (p=0.008 n=5)
CountTortureOverlapping-8                    66.40µ ±  5%   3721.27µ ±  1%  +5504.57% (p=0.008 n=5)
CountByte/10-8                               3.587n ±  1%     5.299n ±  1%    +47.73% (p=0.008 n=5)
CountByte/32-8                               4.848n ±  0%     6.037n ±  0%    +24.53% (p=0.008 n=5)
CountByte/4K-8                               64.04n ±  0%     70.51n ±  1%    +10.10% (p=0.008 n=5)
CountByte/4M-8                               84.81µ ±  2%     91.86µ ±  3%     +8.31% (p=0.008 n=5)
CountByte/64M-8                              2.889m ±  8%     3.093m ±  4%          ~ (p=0.056 n=5)
IndexAnyASCII/1:1-8                          4.462n ±  1%     6.584n ±  1%    +47.56% (p=0.008 n=5)
IndexAnyASCII/1:2-8                          5.435n ±  1%     8.856n ±  0%    +62.94% (p=0.008 n=5)
IndexAnyASCII/1:4-8                          5.415n ±  1%     8.823n ±  0%    +62.94% (p=0.008 n=5)
IndexAnyASCII/1:8-8                          5.417n ±  1%     8.855n ±  0%    +63.47% (p=0.008 n=5)
IndexAnyASCII/1:16-8                         5.350n ±  1%     8.147n ±  0%    +52.28% (p=0.008 n=5)
IndexAnyASCII/1:32-8                         5.912n ±  0%    10.700n ± 21%    +80.99% (p=0.008 n=5)
IndexAnyASCII/1:64-8                         6.232n ±  1%     9.131n ± 40%    +46.52% (p=0.008 n=5)
IndexAnyASCII/16:1-8                         4.331n ±  1%     6.431n ±  4%    +48.49% (p=0.008 n=5)
IndexAnyASCII/16:2-8                         14.35n ±  1%     17.73n ±  4%    +23.55% (p=0.008 n=5)
IndexAnyASCII/16:4-8                         16.12n ±  0%     18.97n ±  3%    +17.68% (p=0.008 n=5)
IndexAnyASCII/16:8-8                         20.57n ±  1%     22.62n ±  0%     +9.97% (p=0.008 n=5)
IndexAnyASCII/16:16-8                        27.11n ±  1%     32.80n ±  1%    +20.99% (p=0.008 n=5)
IndexAnyASCII/16:32-8                        45.47n ±  0%     57.04n ±  1%    +25.45% (p=0.008 n=5)
IndexAnyASCII/16:64-8                        72.05n ±  0%     95.18n ±  1%    +32.10% (p=0.008 n=5)
IndexAnyASCII/256:1-8                        8.445n ±  0%    10.690n ±  0%    +26.58% (p=0.008 n=5)
IndexAnyASCII/256:2-8                        131.6n ±  0%     134.5n ±  2%     +2.20% (p=0.008 n=5)
IndexAnyASCII/256:4-8                        134.0n ±  0%     136.9n ±  1%     +2.16% (p=0.008 n=5)
IndexAnyASCII/256:8-8                        138.4n ±  1%     141.1n ±  0%     +1.95% (p=0.008 n=5)
IndexAnyASCII/256:16-8                       143.6n ±  0%     152.1n ±  0%     +5.92% (p=0.008 n=5)
IndexAnyASCII/256:32-8                       160.9n ±  1%     180.4n ±  1%    +12.12% (p=0.008 n=5)
IndexAnyASCII/256:64-8                       185.8n ±  0%     218.9n ±  1%    +17.81% (p=0.008 n=5)
IndexAnyUTF8/1:1-8                           3.395n ±  1%     3.643n ±  0%     +7.30% (p=0.008 n=5)
IndexAnyUTF8/1:2-8                           5.426n ±  0%     8.939n ±  1%    +64.74% (p=0.008 n=5)
IndexAnyUTF8/1:4-8                           5.400n ±  2%     8.937n ±  1%    +65.50% (p=0.008 n=5)
IndexAnyUTF8/1:8-8                           5.405n ±  1%     8.906n ±  0%    +64.77% (p=0.008 n=5)
IndexAnyUTF8/1:16-8                          5.358n ±  1%     8.068n ±  1%    +50.58% (p=0.008 n=5)
IndexAnyUTF8/1:32-8                          5.900n ±  0%     8.354n ±  1%    +41.59% (p=0.008 n=5)
IndexAnyUTF8/1:64-8                          6.243n ±  0%     8.850n ±  1%    +41.76% (p=0.008 n=5)
IndexAnyUTF8/16:1-8                          13.28n ±  2%     13.59n ±  8%          ~ (p=0.056 n=5)
IndexAnyUTF8/16:2-8                          62.27n ±  0%     33.34n ±  1%    -46.46% (p=0.008 n=5)
IndexAnyUTF8/16:4-8                          62.27n ±  0%     34.74n ±  1%    -44.21% (p=0.008 n=5)
IndexAnyUTF8/16:8-8                          62.31n ±  0%    101.40n ±  1%    +62.73% (p=0.008 n=5)
IndexAnyUTF8/16:16-8                         63.48n ±  0%     97.99n ±  1%    +54.36% (p=0.008 n=5)
IndexAnyUTF8/16:32-8                         71.60n ±  0%    103.90n ±  1%    +45.11% (p=0.008 n=5)
IndexAnyUTF8/16:64-8                         76.67n ±  1%    111.30n ±  2%    +45.17% (p=0.008 n=5)
IndexAnyUTF8/256:1-8                         170.6n ±  0%     170.3n ±  1%          ~ (p=0.143 n=5)
IndexAnyUTF8/256:2-8                         877.8n ±  1%     354.3n ±  0%    -59.64% (p=0.008 n=5)
IndexAnyUTF8/256:4-8                         874.7n ±  1%     195.6n ±  1%    -77.64% (p=0.008 n=5)
IndexAnyUTF8/256:8-8                         876.9n ±  0%     389.7n ±  1%    -55.56% (p=0.008 n=5)
IndexAnyUTF8/256:16-8                        883.6n ±  0%     121.2n ±  1%    -86.28% (p=0.008 n=5)
IndexAnyUTF8/256:32-8                       1006.0n ±  0%     641.4n ±  1%    -36.24% (p=0.008 n=5)
IndexAnyUTF8/256:64-8                       1096.0n ±  0%     757.3n ±  1%    -30.90% (p=0.008 n=5)
LastIndexAnyASCII/1:1-8                      4.829n ±  1%     7.173n ±  1%    +48.54% (p=0.008 n=5)
LastIndexAnyASCII/1:2-8                      4.814n ±  1%     7.162n ±  1%    +48.77% (p=0.008 n=5)
LastIndexAnyASCII/1:4-8                      4.818n ±  1%     7.167n ±  2%    +48.75% (p=0.008 n=5)
LastIndexAnyASCII/1:8-8                      4.809n ±  1%     7.196n ±  1%    +49.64% (p=0.008 n=5)
LastIndexAnyASCII/1:16-8                     4.500n ±  0%     6.571n ±  1%    +46.02% (p=0.008 n=5)
LastIndexAnyASCII/1:32-8                     4.900n ±  1%     6.819n ±  1%    +39.16% (p=0.008 n=5)
LastIndexAnyASCII/1:64-8                     5.128n ±  1%     7.309n ±  0%    +42.53% (p=0.008 n=5)
LastIndexAnyASCII/16:1-8                     13.82n ±  1%     16.80n ±  1%    +21.56% (p=0.008 n=5)
LastIndexAnyASCII/16:2-8                     14.47n ±  1%     16.98n ±  1%    +17.35% (p=0.008 n=5)
LastIndexAnyASCII/16:4-8                     16.30n ±  0%     18.33n ±  1%    +12.45% (p=0.008 n=5)
LastIndexAnyASCII/16:8-8                     21.19n ±  1%     21.97n ±  1%     +3.68% (p=0.008 n=5)
LastIndexAnyASCII/16:16-8                    26.91n ±  0%     33.26n ±  3%    +23.60% (p=0.008 n=5)
LastIndexAnyASCII/16:32-8                    44.98n ±  0%     55.87n ±  1%    +24.21% (p=0.008 n=5)
LastIndexAnyASCII/16:64-8                    71.95n ±  0%     93.77n ±  1%    +30.33% (p=0.008 n=5)
LastIndexAnyASCII/256:1-8                    130.9n ±  1%     132.5n ±  1%     +1.22% (p=0.008 n=5)
LastIndexAnyASCII/256:2-8                    131.8n ±  1%     133.2n ±  1%     +1.06% (p=0.008 n=5)
LastIndexAnyASCII/256:4-8                    134.1n ±  0%     135.6n ±  1%     +1.12% (p=0.008 n=5)
LastIndexAnyASCII/256:8-8                    137.8n ±  1%     139.5n ±  1%     +1.23% (p=0.008 n=5)
LastIndexAnyASCII/256:16-8                   143.3n ±  0%     150.2n ±  1%     +4.82% (p=0.008 n=5)
LastIndexAnyASCII/256:32-8                   160.1n ±  0%     178.6n ±  1%    +11.56% (p=0.008 n=5)
LastIndexAnyASCII/256:64-8                   185.3n ±  1%     217.3n ±  1%    +17.27% (p=0.008 n=5)
LastIndexAnyUTF8/1:1-8                       4.841n ±  0%     7.195n ±  1%    +48.63% (p=0.008 n=5)
LastIndexAnyUTF8/1:2-8                       4.842n ±  1%     7.172n ±  0%    +48.12% (p=0.008 n=5)
LastIndexAnyUTF8/1:4-8                       4.840n ±  0%     7.161n ±  1%    +47.95% (p=0.008 n=5)
LastIndexAnyUTF8/1:8-8                       4.831n ±  0%     7.157n ±  0%    +48.15% (p=0.008 n=5)
LastIndexAnyUTF8/1:16-8                      4.485n ±  0%     6.563n ±  1%    +46.33% (p=0.008 n=5)
LastIndexAnyUTF8/1:32-8                      4.896n ±  0%     6.817n ±  1%    +39.24% (p=0.008 n=5)
LastIndexAnyUTF8/1:64-8                      5.145n ±  0%     7.308n ±  1%    +42.04% (p=0.008 n=5)
LastIndexAnyUTF8/16:1-8                      30.13n ±  1%     27.98n ±  2%     -7.14% (p=0.008 n=5)
LastIndexAnyUTF8/16:2-8                      80.13n ±  0%    111.70n ±  1%    +39.40% (p=0.008 n=5)
LastIndexAnyUTF8/16:4-8                      80.12n ±  0%    112.30n ±  1%    +40.16% (p=0.008 n=5)
LastIndexAnyUTF8/16:8-8                      80.21n ±  0%    111.60n ±  0%    +39.13% (p=0.008 n=5)
LastIndexAnyUTF8/16:16-8                     81.12n ±  0%    109.20n ±  1%    +34.62% (p=0.008 n=5)
LastIndexAnyUTF8/16:32-8                     89.54n ±  0%    119.00n ±  1%    +32.90% (p=0.008 n=5)
LastIndexAnyUTF8/16:64-8                     92.18n ±  1%    130.20n ±  1%    +41.25% (p=0.008 n=5)
LastIndexAnyUTF8/256:1-8                     443.5n ±  1%     391.9n ±  1%    -11.63% (p=0.008 n=5)
LastIndexAnyUTF8/256:2-8                     1.130µ ±  0%     1.591µ ±  2%    +40.80% (p=0.008 n=5)
LastIndexAnyUTF8/256:4-8                     1.130µ ±  1%     1.593µ ±  1%    +40.97% (p=0.008 n=5)
LastIndexAnyUTF8/256:8-8                     1.129µ ±  1%     1.590µ ±  1%    +40.83% (p=0.008 n=5)
LastIndexAnyUTF8/256:16-8                    1.156µ ±  0%     1.582µ ±  2%    +36.85% (p=0.008 n=5)
LastIndexAnyUTF8/256:32-8                    1.285µ ±  1%     1.749µ ±  1%    +36.11% (p=0.008 n=5)
LastIndexAnyUTF8/256:64-8                    1.331µ ±  1%     1.910µ ±  1%    +43.50% (p=0.008 n=5)
IndexPeriodic/IndexPeriodic2-8               5.042µ ±  0%    56.941µ ±  1%  +1029.33% (p=0.008 n=5)
IndexPeriodic/IndexPeriodic4-8               5.041µ ±  1%    55.882µ ±  0%  +1008.55% (p=0.008 n=5)
IndexPeriodic/IndexPeriodic8-8               58.03µ ±  1%     91.47µ ±  1%    +57.62% (p=0.008 n=5)
IndexPeriodic/IndexPeriodic16-8              28.46µ ±  4%     45.73µ ±  0%    +60.67% (p=0.008 n=5)
IndexPeriodic/IndexPeriodic32-8              14.74µ ±  4%     22.99µ ±  2%    +56.05% (p=0.008 n=5)
IndexPeriodic/IndexPeriodic64-8              8.678µ ±  9%    12.441µ ±  4%    +43.36% (p=0.008 n=5)
IndexByte_Bytes/10-8                         3.037n ±  0%     3.935n ±  1%    +29.57% (p=0.008 n=5)
IndexByte_Bytes/32-8                         3.851n ±  1%     4.669n ±  0%    +21.24% (p=0.008 n=5)
IndexByte_Bytes/4K-8                         61.10n ±  0%     75.10n ±  1%    +22.91% (p=0.008 n=5)
IndexByte_Bytes/4M-8                         89.35µ ±  1%     95.83µ ±  4%     +7.25% (p=0.008 n=5)
IndexByte_Bytes/64M-8                        2.855m ± 10%     3.024m ±  6%          ~ (p=0.151 n=5)
IndexRune_Bytes/10-8                         10.57n ±  1%     12.59n ±  0%    +19.11% (p=0.008 n=5)
IndexRune_Bytes/32-8                         11.88n ±  1%     13.36n ±  1%    +12.46% (p=0.008 n=5)
IndexRune_Bytes/4K-8                         81.27n ±  0%     80.24n ±  1%     -1.27% (p=0.008 n=5)
IndexRune_Bytes/4M-8                        102.29µ ±  2%     99.42µ ±  3%          ~ (p=0.310 n=5)
IndexRune_Bytes/64M-8                        3.016m ±  2%     3.004m ±  3%          ~ (p=0.421 n=5)
IndexRuneASCII_Bytes/10-8                    3.177n ±  0%     6.011n ±  1%    +89.20% (p=0.008 n=5)
IndexRuneASCII_Bytes/32-8                    3.882n ±  1%     6.901n ±  1%    +77.77% (p=0.008 n=5)
IndexRuneASCII_Bytes/4K-8                    61.08n ±  0%     74.61n ±  0%    +22.15% (p=0.008 n=5)
IndexRuneASCII_Bytes/4M-8                    92.13µ ±  5%     97.13µ ±  2%     +5.43% (p=0.008 n=5)
IndexRuneASCII_Bytes/64M-8                   2.873m ±  2%     3.142m ±  7%     +9.35% (p=0.032 n=5)
IndexNonASCII_Bytes/10-8                     3.401n ±  0%     2.988n ±  1%    -12.14% (p=0.008 n=5)
IndexNonASCII_Bytes/32-8                     8.846n ±  1%     3.620n ±  1%    -59.08% (p=0.008 n=5)
IndexNonASCII_Bytes/4K-8                    895.00n ±  1%     71.40n ±  1%    -92.02% (p=0.008 n=5)
IndexNonASCII_Bytes/4M-8                    908.49µ ±  2%     94.63µ ±  4%    -89.58% (p=0.008 n=5)
IndexNonASCII_Bytes/64M-8                   15.038m ±  3%     3.094m ±  7%    -79.43% (p=0.008 n=5)
geomean                                      188.6n           239.4n          +26.96%

                           │ stdlib.5.1681426916.txt │        strcase.5.1681426916.txt        │
                           │           B/s           │      B/s       vs base                 │
CountByte/10-8                          2.596Gi ± 1%    1.757Gi ± 1%    -32.31% (p=0.008 n=5)
CountByte/32-8                          6.148Gi ± 0%    4.937Gi ± 0%    -19.69% (p=0.008 n=5)
CountByte/4K-8                          59.56Gi ± 0%    54.10Gi ± 1%     -9.18% (p=0.008 n=5)
CountByte/4M-8                          46.06Gi ± 2%    42.52Gi ± 3%     -7.67% (p=0.008 n=5)
CountByte/64M-8                         21.63Gi ± 8%    20.20Gi ± 4%          ~ (p=0.056 n=5)
IndexByte_Bytes/10-8                    3.066Gi ± 0%    2.367Gi ± 1%    -22.81% (p=0.008 n=5)
IndexByte_Bytes/32-8                    7.739Gi ± 1%    6.384Gi ± 0%    -17.51% (p=0.008 n=5)
IndexByte_Bytes/4K-8                    62.44Gi ± 0%    50.79Gi ± 1%    -18.65% (p=0.008 n=5)
IndexByte_Bytes/4M-8                    43.72Gi ± 1%    40.76Gi ± 3%     -6.76% (p=0.008 n=5)
IndexByte_Bytes/64M-8                   21.89Gi ± 9%    20.67Gi ± 6%          ~ (p=0.151 n=5)
IndexRune_Bytes/10-8                    902.2Mi ± 1%    757.4Mi ± 0%    -16.05% (p=0.008 n=5)
IndexRune_Bytes/32-8                    2.509Gi ± 1%    2.230Gi ± 1%    -11.11% (p=0.008 n=5)
IndexRune_Bytes/4K-8                    46.94Gi ± 0%    47.54Gi ± 1%     +1.28% (p=0.008 n=5)
IndexRune_Bytes/4M-8                    38.19Gi ± 2%    39.29Gi ± 3%          ~ (p=0.310 n=5)
IndexRune_Bytes/64M-8                   20.72Gi ± 2%    20.81Gi ± 3%          ~ (p=0.421 n=5)
IndexRuneASCII_Bytes/10-8               2.931Gi ± 0%    1.549Gi ± 1%    -47.14% (p=0.008 n=5)
IndexRuneASCII_Bytes/32-8               7.677Gi ± 1%    4.319Gi ± 1%    -43.74% (p=0.008 n=5)
IndexRuneASCII_Bytes/4K-8               62.45Gi ± 0%    51.13Gi ± 0%    -18.13% (p=0.008 n=5)
IndexRuneASCII_Bytes/4M-8               42.40Gi ± 5%    40.22Gi ± 2%     -5.15% (p=0.008 n=5)
IndexRuneASCII_Bytes/64M-8              21.75Gi ± 2%    19.89Gi ± 8%     -8.55% (p=0.032 n=5)
IndexNonASCII_Bytes/10-8                2.738Gi ± 0%    3.117Gi ± 1%    +13.84% (p=0.008 n=5)
IndexNonASCII_Bytes/32-8                3.369Gi ± 1%    8.232Gi ± 1%   +144.33% (p=0.008 n=5)
IndexNonASCII_Bytes/4K-8                4.262Gi ± 1%   53.429Gi ± 1%  +1153.52% (p=0.008 n=5)
IndexNonASCII_Bytes/4M-8                4.300Gi ± 2%   41.278Gi ± 4%   +860.01% (p=0.008 n=5)
IndexNonASCII_Bytes/64M-8               4.156Gi ± 3%   20.200Gi ± 7%   +386.03% (p=0.008 n=5)
geomean                                 10.97Gi         12.85Gi         +17.13%
```

</details>

<details>
<summary>arm (pi)</summary>

```
goos: linux
goarch: arm
pkg: github.com/charlievieth/strcase/internal/benchtest
                                │ stdlib.5.1681426807.txt │        strcase.5.1681426807.txt        │
                                │         sec/op          │    sec/op      vs base                 │
IndexRune-4                                  120.0n ±  1%    112.3n ±  3%     -6.46% (p=0.002 n=6)
IndexRuneLongString-4                        298.4n ±  1%    286.7n ±  1%     -3.94% (p=0.002 n=6)
IndexRuneFastPath-4                          78.38n ± 18%    69.51n ±  3%          ~ (p=0.180 n=6)
Index-4                                      77.23n ± 30%    71.47n ±  1%          ~ (p=0.589 n=6)
LastIndex-4                                  22.52n ±  3%    34.44n ± 10%    +52.96% (p=0.002 n=6)
IndexByte-4                                  49.02n ±  9%    62.13n ±  1%    +26.76% (p=0.002 n=6)
EqualFold/ASCII-4                            64.32n ±  1%    69.51n ±  1%     +8.07% (p=0.002 n=6)
EqualFold/UnicodePrefix-4                    636.3n ±  0%    234.8n ±  0%    -63.09% (p=0.002 n=6)
EqualFold/UnicodeSuffix-4                    585.9n ±  0%    190.6n ±  0%    -67.47% (p=0.002 n=6)
IndexHard1-4                                 5.684m ±  1%    5.674m ±  0%     -0.16% (p=0.015 n=6)
IndexHard2-4                                 5.715m ±  0%    9.439m ±  0%    +65.15% (p=0.002 n=6)
IndexHard3-4                                 5.705m ±  0%    8.681m ±  2%    +52.16% (p=0.002 n=6)
IndexHard4-4                                 5.726m ±  3%    9.271m ±  0%    +61.92% (p=0.002 n=6)
LastIndexHard1-4                             4.631m ±  1%   10.070m ±  1%   +117.46% (p=0.002 n=6)
LastIndexHard2-4                             4.667m ±  3%   10.073m ±  1%   +115.86% (p=0.002 n=6)
LastIndexHard3-4                             4.634m ±  3%   10.115m ±  0%   +118.27% (p=0.002 n=6)
CountHard1-4                                 5.680m ±  0%    5.681m ±  4%          ~ (p=0.699 n=6)
CountHard2-4                                 5.698m ±  0%    9.433m ±  0%    +65.54% (p=0.002 n=6)
CountHard3-4                                 5.856m ±  3%    8.690m ±  5%    +48.39% (p=0.002 n=6)
IndexTorture-4                               43.41µ ±  1%   116.26µ ±  0%   +167.81% (p=0.002 n=6)
CountTorture-4                               43.44µ ±  0%   126.59µ ±  0%   +191.43% (p=0.002 n=6)
CountTortureOverlapping-4                    1.197m ±  1%   28.433m ±  1%  +2275.73% (p=0.002 n=6)
CountByte/10-4                               41.48n ±  3%    45.49n ±  3%     +9.65% (p=0.002 n=6)
CountByte/32-4                               70.84n ±  0%   121.95n ±  3%    +72.14% (p=0.002 n=6)
CountByte/4K-4                               5.543µ ±  3%   14.107µ ±  1%   +154.52% (p=0.002 n=6)
CountByte/4M-4                               5.782m ±  3%   14.391m ±  9%   +148.87% (p=0.002 n=6)
CountByte/64M-4                              125.8m ±  1%    287.1m ±  9%   +128.16% (p=0.002 n=6)
IndexAnyASCII/1:1-4                          68.25n ±  3%   108.35n ±  1%    +58.77% (p=0.002 n=6)
IndexAnyASCII/1:2-4                          57.10n ±  3%    98.97n ±  1%    +73.33% (p=0.002 n=6)
IndexAnyASCII/1:4-4                          60.79n ±  2%   103.20n ±  2%    +69.76% (p=0.002 n=6)
IndexAnyASCII/1:8-4                          65.47n ±  2%   108.40n ±  1%    +65.57% (p=0.002 n=6)
IndexAnyASCII/1:16-4                         92.55n ± 13%   132.75n ± 11%    +43.44% (p=0.002 n=6)
IndexAnyASCII/1:32-4                         114.8n ±  2%    161.9n ±  2%    +41.03% (p=0.002 n=6)
IndexAnyASCII/1:64-4                         167.5n ±  1%    212.1n ±  1%    +26.66% (p=0.002 n=6)
IndexAnyASCII/16:1-4                         71.73n ±  2%   112.80n ±  5%    +57.25% (p=0.002 n=6)
IndexAnyASCII/16:2-4                         101.6n ±  0%    132.7n ±  1%    +30.55% (p=0.002 n=6)
IndexAnyASCII/16:4-4                         110.0n ±  3%    146.6n ±  0%    +33.27% (p=0.002 n=6)
IndexAnyASCII/16:8-4                         128.2n ±  3%    171.8n ±  1%    +34.01% (p=0.002 n=6)
IndexAnyASCII/16:16-4                        165.4n ±  2%    246.4n ±  2%    +48.97% (p=0.002 n=6)
IndexAnyASCII/16:32-4                        259.4n ±  0%    450.6n ±  0%    +73.69% (p=0.002 n=6)
IndexAnyASCII/16:64-4                        411.8n ±  0%    752.1n ±  0%    +82.64% (p=0.002 n=6)
IndexAnyASCII/256:1-4                        473.2n ±  0%    516.0n ±  0%     +9.06% (p=0.002 n=6)
IndexAnyASCII/256:2-4                        917.6n ±  0%    948.1n ±  0%     +3.32% (p=0.002 n=6)
IndexAnyASCII/256:4-4                        924.1n ±  3%    960.1n ±  3%     +3.89% (p=0.002 n=6)
IndexAnyASCII/256:8-4                        942.3n ±  0%    985.9n ±  0%     +4.62% (p=0.002 n=6)
IndexAnyASCII/256:16-4                       982.3n ±  3%   1059.0n ±  1%     +7.81% (p=0.002 n=6)
IndexAnyASCII/256:32-4                       1.075µ ±  1%    1.777µ ±  3%    +65.30% (p=0.002 n=6)
IndexAnyASCII/256:64-4                       1.226µ ±  0%    2.076µ ±  0%    +69.40% (p=0.002 n=6)
IndexAnyUTF8/1:1-4                           21.38n ±  0%    18.42n ±  1%    -13.87% (p=0.002 n=6)
IndexAnyUTF8/1:2-4                           58.02n ±  5%    99.98n ±  1%    +72.32% (p=0.002 n=6)
IndexAnyUTF8/1:4-4                           60.71n ±  3%   102.95n ±  1%    +69.58% (p=0.002 n=6)
IndexAnyUTF8/1:8-4                           65.74n ±  3%   107.45n ±  1%    +63.43% (p=0.002 n=6)
IndexAnyUTF8/1:16-4                          81.01n ±  6%   122.05n ± 12%    +50.66% (p=0.002 n=6)
IndexAnyUTF8/1:32-4                          115.2n ±  1%    160.8n ±  1%    +39.57% (p=0.002 n=6)
IndexAnyUTF8/1:64-4                          167.2n ±  1%    212.3n ±  2%    +27.04% (p=0.002 n=6)
IndexAnyUTF8/16:1-4                          75.35n ± 13%    80.66n ± 13%          ~ (p=0.310 n=6)
IndexAnyUTF8/16:2-4                          571.6n ±  8%    243.7n ±  2%    -57.36% (p=0.002 n=6)
IndexAnyUTF8/16:4-4                          655.2n ±  6%    274.2n ±  1%    -58.15% (p=0.002 n=6)
IndexAnyUTF8/16:8-4                          746.5n ±  3%   1418.5n ±  2%    +90.02% (p=0.002 n=6)
IndexAnyUTF8/16:16-4                         922.1n ±  0%   1599.0n ± 11%    +73.41% (p=0.002 n=6)
IndexAnyUTF8/16:32-4                         1.570µ ±  2%    2.267µ ±  1%    +44.36% (p=0.002 n=6)
IndexAnyUTF8/16:64-4                         2.404µ ±  1%    3.065µ ±  1%    +27.50% (p=0.002 n=6)
IndexAnyUTF8/256:1-4                         882.8n ±  0%    879.2n ±  0%     -0.40% (p=0.002 n=6)
IndexAnyUTF8/256:2-4                         8.575µ ±  6%    1.865µ ±  1%    -78.25% (p=0.002 n=6)
IndexAnyUTF8/256:4-4                         9.643µ ±  5%    1.486µ ±  0%    -84.59% (p=0.002 n=6)
IndexAnyUTF8/256:8-4                        11.164µ ±  4%    2.926µ ±  0%    -73.80% (p=0.002 n=6)
IndexAnyUTF8/256:16-4                       13.915µ ±  0%    4.702µ ±  0%    -66.21% (p=0.002 n=6)
IndexAnyUTF8/256:32-4                        24.39µ ±  2%    14.45µ ±  3%    -40.74% (p=0.002 n=6)
IndexAnyUTF8/256:64-4                        37.62µ ±  1%    27.55µ ±  3%    -26.78% (p=0.002 n=6)
LastIndexAnyASCII/1:1-4                      69.11n ±  2%   111.15n ±  3%    +60.83% (p=0.002 n=6)
LastIndexAnyASCII/1:2-4                      53.51n ±  3%    94.23n ±  0%    +76.10% (p=0.002 n=6)
LastIndexAnyASCII/1:4-4                      56.18n ±  3%    97.55n ±  3%    +73.63% (p=0.002 n=6)
LastIndexAnyASCII/1:8-4                      60.98n ±  1%   102.20n ±  0%    +67.58% (p=0.002 n=6)
LastIndexAnyASCII/1:16-4                     72.90n ±  3%   112.95n ±  0%    +54.94% (p=0.002 n=6)
LastIndexAnyASCII/1:32-4                     111.6n ±  2%    153.7n ±  2%    +37.72% (p=0.002 n=6)
LastIndexAnyASCII/1:64-4                     163.2n ±  1%    201.8n ±  2%    +23.62% (p=0.002 n=6)
LastIndexAnyASCII/16:1-4                     113.8n ±  3%    142.8n ±  1%    +25.53% (p=0.002 n=6)
LastIndexAnyASCII/16:2-4                     119.7n ±  0%    152.5n ±  3%    +27.36% (p=0.002 n=6)
LastIndexAnyASCII/16:4-4                     129.2n ±  1%    165.3n ±  1%    +27.95% (p=0.002 n=6)
LastIndexAnyASCII/16:8-4                     151.5n ±  0%    192.8n ±  2%    +27.29% (p=0.002 n=6)
LastIndexAnyASCII/16:16-4                    187.7n ±  0%    265.7n ±  2%    +41.56% (p=0.002 n=6)
LastIndexAnyASCII/16:32-4                    278.2n ±  0%    474.2n ±  1%    +70.47% (p=0.002 n=6)
LastIndexAnyASCII/16:64-4                    424.4n ±  1%    775.4n ±  1%    +82.70% (p=0.002 n=6)
LastIndexAnyASCII/256:1-4                    996.0n ±  0%   1119.5n ±  8%    +12.39% (p=0.002 n=6)
LastIndexAnyASCII/256:2-4                    1.003µ ±  1%    1.041µ ±  2%     +3.79% (p=0.002 n=6)
LastIndexAnyASCII/256:4-4                    1.014µ ±  0%    1.049µ ±  2%     +3.45% (p=0.002 n=6)
LastIndexAnyASCII/256:8-4                    1.033µ ±  1%    1.073µ ±  0%     +3.87% (p=0.002 n=6)
LastIndexAnyASCII/256:16-4                   1.069µ ±  0%    1.150µ ±  0%     +7.63% (p=0.002 n=6)
LastIndexAnyASCII/256:32-4                   1.159µ ±  0%    1.871µ ±  3%    +61.39% (p=0.002 n=6)
LastIndexAnyASCII/256:64-4                   1.310µ ±  1%    2.174µ ±  2%    +65.85% (p=0.002 n=6)
LastIndexAnyUTF8/1:1-4                       69.47n ±  2%   110.90n ±  1%    +59.64% (p=0.002 n=6)
LastIndexAnyUTF8/1:2-4                       53.42n ±  0%    94.05n ±  2%    +76.07% (p=0.002 n=6)
LastIndexAnyUTF8/1:4-4                       55.72n ±  4%    96.50n ±  1%    +73.19% (p=0.002 n=6)
LastIndexAnyUTF8/1:8-4                       61.43n ±  2%   102.30n ±  1%    +66.54% (p=0.002 n=6)
LastIndexAnyUTF8/1:16-4                      72.81n ±  0%   113.95n ±  6%    +56.50% (p=0.002 n=6)
LastIndexAnyUTF8/1:32-4                      111.6n ±  0%    155.5n ±  3%    +39.44% (p=0.002 n=6)
LastIndexAnyUTF8/1:64-4                      163.5n ±  1%    203.1n ±  2%    +24.18% (p=0.002 n=6)
LastIndexAnyUTF8/16:1-4                      212.7n ±  0%    246.6n ±  8%    +15.96% (p=0.002 n=6)
LastIndexAnyUTF8/16:2-4                      685.7n ±  3%   1366.5n ±  6%    +99.29% (p=0.002 n=6)
LastIndexAnyUTF8/16:4-4                      728.8n ±  2%   1431.0n ±  2%    +96.36% (p=0.002 n=6)
LastIndexAnyUTF8/16:8-4                      813.9n ±  2%   1504.5n ±  4%    +84.86% (p=0.002 n=6)
LastIndexAnyUTF8/16:16-4                     1.192µ ±  7%    1.734µ ±  2%    +45.47% (p=0.002 n=6)
LastIndexAnyUTF8/16:32-4                     1.669µ ±  2%    2.391µ ±  3%    +43.26% (p=0.002 n=6)
LastIndexAnyUTF8/16:64-4                     2.491µ ±  2%    3.198µ ±  1%    +28.38% (p=0.002 n=6)
LastIndexAnyUTF8/256:1-4                     2.526µ ±  0%    2.593µ ±  1%     +2.63% (p=0.002 n=6)
LastIndexAnyUTF8/256:2-4                     10.14µ ±  4%    20.74µ ±  2%   +104.44% (p=0.002 n=6)
LastIndexAnyUTF8/256:4-4                     10.84µ ±  1%    21.42µ ±  1%    +97.56% (p=0.002 n=6)
LastIndexAnyUTF8/256:8-4                     12.21µ ±  3%    22.76µ ±  5%    +86.46% (p=0.002 n=6)
LastIndexAnyUTF8/256:16-4                    16.35µ ± 19%    25.97µ ±  7%    +58.83% (p=0.002 n=6)
LastIndexAnyUTF8/256:32-4                    25.82µ ±  1%    38.08µ ± 10%    +47.51% (p=0.002 n=6)
LastIndexAnyUTF8/256:64-4                    39.09µ ±  2%    50.23µ ±  9%    +28.50% (p=0.002 n=6)
IndexPeriodic/IndexPeriodic2-4               351.4µ ±  0%    336.9µ ± 16%          ~ (p=0.394 n=6)
IndexPeriodic/IndexPeriodic4-4               351.4µ ±  1%    336.8µ ± 13%          ~ (p=0.065 n=6)
IndexPeriodic/IndexPeriodic8-4               351.2µ ±  1%    492.1µ ±  0%    +40.13% (p=0.002 n=6)
IndexPeriodic/IndexPeriodic16-4              167.9µ ±  0%    334.8µ ±  1%    +99.44% (p=0.002 n=6)
IndexPeriodic/IndexPeriodic32-4              166.1µ ±  1%    291.5µ ±  1%    +75.51% (p=0.002 n=6)
IndexPeriodic/IndexPeriodic64-4              132.6µ ±  0%    234.8µ ±  3%    +77.02% (p=0.002 n=6)
IndexByte_Bytes/10-4                         40.12n ±  3%    43.35n ±  3%     +8.06% (p=0.002 n=6)
IndexByte_Bytes/32-4                         89.12n ±  2%   117.65n ±  2%    +32.01% (p=0.002 n=6)
IndexByte_Bytes/4K-4                         6.342µ ±  5%   11.323µ ±  4%    +78.54% (p=0.002 n=6)
IndexByte_Bytes/4M-4                         6.742m ±  1%   11.580m ±  7%    +71.75% (p=0.002 n=6)
IndexByte_Bytes/64M-4                        126.9m ±  2%    212.5m ±  4%    +67.49% (p=0.002 n=6)
IndexRune_Bytes/10-4                         110.8n ±  1%    121.7n ±  2%     +9.84% (p=0.002 n=6)
IndexRune_Bytes/32-4                         159.9n ±  2%    171.5n ±  4%     +7.26% (p=0.002 n=6)
IndexRune_Bytes/4K-4                         6.685µ ±  1%    6.764µ ±  6%     +1.19% (p=0.004 n=6)
IndexRune_Bytes/4M-4                         6.760m ±  0%    6.806m ±  1%          ~ (p=0.065 n=6)
IndexRune_Bytes/64M-4                        128.1m ±  2%    130.6m ± 11%          ~ (p=0.589 n=6)
IndexRuneASCII_Bytes/10-4                    43.45n ±  1%    57.01n ±  7%    +31.21% (p=0.002 n=6)
IndexRuneASCII_Bytes/32-4                    92.28n ±  6%   130.85n ±  8%    +41.79% (p=0.002 n=6)
IndexRuneASCII_Bytes/4K-4                    6.613µ ±  0%   11.271µ ±  2%    +70.45% (p=0.002 n=6)
IndexRuneASCII_Bytes/4M-4                    6.751m ±  0%   11.502m ±  2%    +70.38% (p=0.002 n=6)
IndexRuneASCII_Bytes/64M-4                   129.2m ± 55%    208.2m ± 54%    +61.16% (p=0.002 n=6)
IndexNonASCII_Bytes/10-4                     24.71n ±  3%    26.28n ±  5%     +6.35% (p=0.002 n=6)
IndexNonASCII_Bytes/32-4                     78.75n ±  4%    75.23n ±  2%     -4.47% (p=0.009 n=6)
IndexNonASCII_Bytes/4K-4                     8.235µ ±  0%    8.874µ ±  7%     +7.77% (p=0.002 n=6)
IndexNonASCII_Bytes/4M-4                     8.449m ±  3%    9.721m ±  3%    +15.05% (p=0.002 n=6)
IndexNonASCII_Bytes/64M-4                    154.2m ±  2%    170.5m ± 15%    +10.58% (p=0.002 n=6)
geomean                                      2.636µ          3.464µ          +31.41%

                           │ stdlib.5.1681426807.txt │       strcase.5.1681426807.txt       │
                           │           B/s           │      B/s       vs base               │
CountByte/10-4                         229.9Mi ±  3%   209.7Mi ±  3%   -8.80% (p=0.002 n=6)
CountByte/32-4                         430.8Mi ±  0%   250.2Mi ±  3%  -41.91% (p=0.002 n=6)
CountByte/4K-4                         704.7Mi ±  3%   276.9Mi ±  1%  -60.71% (p=0.002 n=6)
CountByte/4M-4                         691.8Mi ±  3%   278.0Mi ± 10%  -59.82% (p=0.002 n=6)
CountByte/64M-4                        508.7Mi ±  1%   222.9Mi ± 10%  -56.17% (p=0.002 n=6)
IndexByte_Bytes/10-4                   237.7Mi ±  3%   220.0Mi ±  3%   -7.47% (p=0.002 n=6)
IndexByte_Bytes/32-4                   342.5Mi ±  2%   259.4Mi ±  2%  -24.26% (p=0.002 n=6)
IndexByte_Bytes/4K-4                   616.9Mi ±  4%   345.0Mi ±  4%  -44.08% (p=0.002 n=6)
IndexByte_Bytes/4M-4                   593.3Mi ±  1%   345.4Mi ±  7%  -41.78% (p=0.002 n=6)
IndexByte_Bytes/64M-4                  504.5Mi ±  2%   301.2Mi ±  4%  -40.29% (p=0.002 n=6)
IndexRune_Bytes/10-4                   86.10Mi ±  1%   78.39Mi ±  2%   -8.96% (p=0.002 n=6)
IndexRune_Bytes/32-4                   191.0Mi ±  2%   178.0Mi ±  4%   -6.77% (p=0.002 n=6)
IndexRune_Bytes/4K-4                   584.4Mi ±  1%   577.5Mi ±  5%   -1.17% (p=0.004 n=6)
IndexRune_Bytes/4M-4                   591.7Mi ±  0%   587.8Mi ±  1%        ~ (p=0.065 n=6)
IndexRune_Bytes/64M-4                  499.6Mi ±  2%   490.2Mi ± 10%        ~ (p=0.589 n=6)
IndexRuneASCII_Bytes/10-4              219.5Mi ±  1%   167.3Mi ±  7%  -23.79% (p=0.002 n=6)
IndexRuneASCII_Bytes/32-4              330.7Mi ±  6%   233.2Mi ±  7%  -29.47% (p=0.002 n=6)
IndexRuneASCII_Bytes/4K-4              590.8Mi ±  0%   346.6Mi ±  2%  -41.33% (p=0.002 n=6)
IndexRuneASCII_Bytes/4M-4              592.5Mi ±  0%   347.8Mi ±  2%  -41.30% (p=0.002 n=6)
IndexRuneASCII_Bytes/64M-4             495.4Mi ± 35%   307.4Mi ± 35%  -37.96% (p=0.002 n=6)
IndexNonASCII_Bytes/10-4               385.9Mi ±  3%   362.8Mi ±  5%   -5.99% (p=0.002 n=6)
IndexNonASCII_Bytes/32-4               387.5Mi ±  4%   405.7Mi ±  2%   +4.68% (p=0.009 n=6)
IndexNonASCII_Bytes/4K-4               474.4Mi ±  0%   440.2Mi ±  7%   -7.21% (p=0.002 n=6)
IndexNonASCII_Bytes/4M-4               473.4Mi ±  3%   411.5Mi ±  3%  -13.08% (p=0.002 n=6)
IndexNonASCII_Bytes/64M-4              415.1Mi ±  2%   375.5Mi ± 13%   -9.54% (p=0.002 n=6)
geomean                                407.9Mi         296.3Mi        -27.37%
```

</details>
