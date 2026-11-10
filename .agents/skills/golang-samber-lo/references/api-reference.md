# API Catalog

The `samber/lo` function surface by domain. Signatures evolve, so confirm a specific symbol with `godig symbol doc github.com/samber/lo <Symbol>` or [pkg.go.dev/github.com/samber/lo](https://pkg.go.dev/github.com/samber/lo); Context7 is the fallback when a symbol is missing there.

## Slice transformations

| Function | Purpose |
| --- | --- |
| `lo.Map(s, fn)` | Transform each element; `fn(item T, index int) R` |
| `lo.MapErr(s, fn)` | Map that stops at the first error |
| `lo.UniqMap(s, fn)` | Map and deduplicate results in a single pass |
| `lo.Filter(s, fn)` | Keep elements where the predicate is true |
| `lo.FilterErr(s, fn)` | Filter with error propagation |
| `lo.Reject(s, fn)` | Drop elements where the predicate is true (inverse of Filter) |
| `lo.RejectMap(s, fn)` | Reject and map in one pass |
| `lo.FilterReject(s, fn)` | Split into matching and non-matching slices |
| `lo.FlatMap(s, fn)` | Map then flatten one level |
| `lo.FlatMapErr(s, fn)` | FlatMap with error propagation |
| `lo.FilterMap(s, fn)` | Filter and map together; `fn(T) (R, bool)` |
| `lo.Reduce(s, fn, init)` | Fold left into an accumulator |
| `lo.ReduceRight(s, fn, init)` | Fold right into an accumulator |
| `lo.ForEach(s, fn)` | Iterate for side effects |
| `lo.ForEachWhile(s, fn)` | Iterate until `fn` returns false |
| `lo.Times(n, fn)` | Call `fn(index)` n times and collect results |
| `lo.Chunk(s, size)` | Split into batches of `size` |
| `lo.Window(s, size)` / `lo.Sliding(s, size)` | Sliding window of `size` |
| `lo.Flatten(s)` | Flatten `[][]T` one level |
| `lo.Concat(slices...)` | Join multiple slices |
| `lo.Interleave(slices...)` | Interleave elements from several slices |
| `lo.Repeat(n, val)` | n copies of one value |
| `lo.RepeatBy(n, fn)` | n values produced by `fn(index)` |
| `lo.Splice(s, i, elements...)` | Insert elements at index `i` |
| `lo.Fill(s, val)` | New slice filled with `val` |
| `lo.Reverse(s)` | Reversed copy |
| `lo.Shuffle(s)` | Randomly shuffled copy |
| `lo.Clone(s)` | Shallow copy |

Slice-to-map conversions:

| Function | Purpose |
| --- | --- |
| `lo.KeyBy(s, fn)` / `lo.Keyify(s, fn)` | Map keyed by `fn(T)` |
| `lo.SliceToMap(s, fn)` / `lo.Associate(s, fn)` | Map from `fn(T) (K, V)` pairs |
| `lo.FilterSliceToMap(s, fn)` | Filter plus conversion; `fn(T) (K, V, bool)` |
| `lo.GroupBy(s, fn)` | `map[K][]V` by key function |
| `lo.GroupByMap(s, fn)` | GroupBy producing `map[K]R` with a transform |

Almost every transform has an `Err` suffix — `MapErr`, `FlatMapErr`, `FilterErr`, `ReduceErr`, `ReduceRightErr`, `ForEachErr`, `GroupByErr`, `UniqByErr` — each stopping at the first error and returning `(result, error)`.

## Slice queries

| Function | Purpose |
| --- | --- |
| `lo.Find(s, fn)` | First match; returns `(T, bool)` |
| `lo.FindOrElse(s, fallback, fn)` | First match or a fallback |
| `lo.FindIndexOf(s, fn)` | First match with its index |
| `lo.FindLastIndexOf(s, fn)` | Last match with its index |
| `lo.FindKey(m, val)` | First map key holding the value |
| `lo.FindKeyBy(m, fn)` | First map key matching a predicate |
| `lo.IndexOf(s, val)` / `lo.LastIndexOf(s, val)` | First / last occurrence index (-1 if absent) |
| `lo.Contains(s, val)` | Member check; prefer `slices.Contains` (Go 1.21+) |
| `lo.ContainsBy(s, fn)` | Any element satisfying a predicate |
| `lo.Every(s, subset)` | Subset fully contained in `s` |
| `lo.EveryBy(s, fn)` | Every element satisfies the predicate |
| `lo.Some(s, subset)` | Subset partially contained |
| `lo.SomeBy(s, fn)` | Some element satisfies the predicate |
| `lo.None(s, subset)` | No subset element present |
| `lo.NoneBy(s, fn)` | No element satisfies the predicate |
| `lo.Count(s, val)` / `lo.CountBy(s, fn)` | Count by value / by predicate |
| `lo.CountValues(s)` | `map[T]int` frequency table |
| `lo.CountValuesBy(s, fn)` | Frequency table by key function |
| `lo.Min(s)` / `lo.Max(s)` | Extreme of a comparable slice |
| `lo.MinBy(s, fn)` / `lo.MaxBy(s, fn)` | Extreme by comparison function |
| `lo.MinIndex(s)` / `lo.MaxIndex(s)` | Index of the extremum |
| `lo.MinIndexBy(s, fn)` / `lo.MaxIndexBy(s, fn)` | Index of extremum by comparison |
| `lo.Earliest(vals...)` / `lo.EarliestBy(s, fn)` | Earliest `time.Time` |
| `lo.Latest(vals...)` / `lo.LatestBy(s, fn)` | Latest `time.Time` |
| `lo.First(s)` / `lo.Last(s)` | First / last element; `(T, bool)` |
| `lo.FirstOr(s, fallback)` / `lo.FirstOrEmpty(s)` | First element with fallback / zero default |
| `lo.LastOr(s, fallback)` / `lo.LastOrEmpty(s)` | Last element with fallback / zero default |
| `lo.Nth(s, n)` | Element at index `n` (negative allowed); `(T, error)` |
| `lo.NthOr(s, n, fallback)` / `lo.NthOrEmpty(s, n)` | Nth with fallback / zero default |
| `lo.Sample(s)` / `lo.SampleBy(s, fn)` | Random element, optionally constrained |
| `lo.Samples(s, n)` / `lo.SamplesBy(s, n, fn)` | n random elements |
| `lo.IsSorted(s)` / `lo.IsSortedBy(s, fn)` | Sorted-slice check |
| `lo.HasPrefix(s, prefix)` / `lo.HasSuffix(s, suffix)` | Start / end element match |

## Set operations

| Function | Purpose |
| --- | --- |
| `lo.Uniq(s)` / `lo.UniqBy(s, fn)` | Unique, preserving first occurrence |
| `lo.PartitionBy(s, fn)` | Consecutive groups by key |
| `lo.Compact(s)` | Drop zero-value elements |
| `lo.Without(s, vals...)` | Remove specific values |
| `lo.WithoutBy(s, fn)` | Remove elements matching a predicate |
| `lo.WithoutEmpty(s)` | Drop zero-value elements; Compact alias |
| `lo.WithoutNth(s, indices...)` | Remove elements at given indices |
| `lo.Union(slices...)` | Merge, deduplicated |
| `lo.Intersect(a, b)` / `lo.IntersectBy(a, b, fn)` | Elements present in both |
| `lo.Difference(a, b)` | Elements of `a` not in `b` |
| `lo.Replace(s, old, new, n)` / `lo.ReplaceAll(s, old, new)` | Substitution, bounded or total |
| `lo.FindDuplicates(s)` / `lo.FindDuplicatesBy(s, fn)` | Elements occurring more than once |
| `lo.FindUniques(s)` / `lo.FindUniquesBy(s, fn)` | Elements occurring exactly once |
| `lo.ElementsMatch(a, b)` / `lo.ElementsMatchBy(a, b, fn)` | Same elements regardless of order |
| `lo.Subset(s, offset, length)` | Sub-slice from an offset |
| `lo.Slice(s, start, end)` | Bounds-safe slice, no panic |

Slicing helpers:

| Function | Purpose |
| --- | --- |
| `lo.Take(s, n)` / `lo.TakeWhile(s, fn)` / `lo.TakeFilter(s, n, fn)` | Leading elements by count / predicate / both |
| `lo.Drop(s, n)` / `lo.DropRight(s, n)` | Skip leading / trailing elements |
| `lo.DropWhile(s, fn)` / `lo.DropRightWhile(s, fn)` | Skip while a predicate holds |
| `lo.DropByIndex(s, indices...)` | Skip elements at specific indices |
| `lo.Cut(s, start, end)` | Remove a bounded interior run |
| `lo.CutPrefix(s, prefix)` / `lo.CutSuffix(s, suffix)` | Strip a leading / trailing sequence |
| `lo.Trim(s, fn)` / `lo.TrimLeft(s, fn)` / `lo.TrimRight(s, fn)` | Trim both / one end while a predicate holds |
| `lo.TrimPrefix(s, prefix)` / `lo.TrimSuffix(s, suffix)` | Strip an exact prefix / suffix |

## Map operations

| Function | Purpose |
| --- | --- |
| `lo.Keys(m)` | All keys; for plain key listing prefer `maps`/`slices` on Go 1.23+ |
| `lo.UniqKeys(m)` | Unique keys (suited to multi-maps) |
| `lo.Values(m)` / `lo.UniqValues(m)` | All / unique values |
| `lo.HasKey(m, key)` | Key existence |
| `lo.ValueOr(m, key, fallback)` | Value or fallback when absent |
| `lo.PickBy(m, fn)` / `lo.OmitBy(m, fn)` | Keep / drop entries by predicate |
| `lo.PickByKeys(m, keys)` / `lo.OmitByKeys(m, keys)` | Keep / drop by key list |
| `lo.PickByValues(m, vals)` / `lo.OmitByValues(m, vals)` | Keep / drop by value list |
| `lo.FilterKeys(m, fn)` / `lo.FilterValues(m, fn)` | Keep by key / value predicate |
| `lo.MapKeys(m, fn)` / `lo.MapValues(m, fn)` | Transform keys / values |
| `lo.MapEntries(m, fn)` | Transform key and value |
| `lo.MapToSlice(m, fn)` | Entries to a slice |
| `lo.FilterMapToSlice(m, fn)` | Filter then entries to a slice |
| `lo.Entries(m)` / `lo.ToPairs(m)` | Map to `[]lo.Entry[K, V]` |
| `lo.FromEntries(entries)` / `lo.FromPairs(pairs)` | Entries back to a map |
| `lo.Invert(m)` | Swap keys and values |
| `lo.Assign(maps...)` | Merge maps; later maps win |
| `lo.ChunkEntries(m, size)` | Split entries into chunks |

## String operations

| Function | Purpose |
| --- | --- |
| `lo.Substring(s, offset, length)` | Bounds-safe, rune-aware substring |
| `lo.ChunkString(s, size)` | Chunk a string |
| `lo.RuneLength(s)` | Runecount, not byte count |
| `lo.PascalCase(s)` | `"hello world"` → `"HelloWorld"` |
| `lo.CamelCase(s)` | `"hello world"` → `"helloWorld"` |
| `lo.KebabCase(s)` | `"hello world"` → `"hello-world"` |
| `lo.SnakeCase(s)` | `"hello world"` → `"hello_world"` |
| `lo.Words(s)` | Split into words |
| `lo.Capitalize(s)` | Uppercase the leading rune |
| `lo.Ellipsis(s, maxLen)` | Truncate with an ellipsis |
| `lo.RandomString(n, charset)` | Random string from a charset |

## Math, ranges, conditionals

| Function | Purpose |
| --- | --- |
| `lo.Range(n)` | `[0 … n-1]` |
| `lo.RangeFrom(start, n)` | n values from `start` |
| `lo.RangeWithSteps(start, end, step)` | Ranged stepping |
| `lo.Clamp(val, min, max)` | Constrain to bounds |
| `lo.Sum(s)` / `lo.SumBy(s, fn)` | Sum by value / extractor |
| `lo.Product(s)` / `lo.ProductBy(s, fn)` | Product of elements |
| `lo.Mean(s)` / `lo.MeanBy(s, fn)` | Arithmetic mean |
| `lo.Mode(s)` | Most frequent element(s) |
| `lo.Ternary(cond, a, b)` | Inline branch; both arms evaluate |
| `lo.TernaryF(cond, fnA, fnB)` | Lazy branch; only the winner runs |
| `lo.If(cond, val).ElseIf(c2, v2).Else(v3)` | Chained conditional |
| `lo.IfF(cond, fn).ElseIfF(c2, fn2).ElseF(fn3)` | Lazy chained conditional |
| `lo.Switch[R](val).Case(v1, r1).Default(r3)` | Value pattern matching |

## Tuples

| Function | Purpose |
| --- | --- |
| `lo.T2(a, b)` … `lo.T9(...)` | Build a tuple |
| `lo.Unpack2(t)` … `lo.Unpack9(t)` | Destructure a tuple |
| `lo.Zip2(a, b)` … `lo.Zip9(...)` | Pair elements across slices |
| `lo.ZipBy2(a, b, fn)` … `lo.ZipBy9(...)` | Zip with a merge function |
| `lo.Unzip2(pairs)` … `lo.Unzip9(...)` | Split pairs back into slices |
| `lo.UnzipBy2(s, fn)` … `lo.UnzipBy9(...)` | Unzip with a split function |
| `lo.CrossJoin2(a, b)` … `lo.CrossJoin9(...)` | Cartesian product |
| `lo.CrossJoinBy2(a, b, fn)` … `lo.CrossJoinBy9(...)` | Product with a transform |

## Channels

| Function | Purpose |
| --- | --- |
| `lo.ChannelDispatcher(ch, count, strategy)` | Fan-out with `RoundRobin`, `Random`, `WeightedRandom`, `First`, `Least`, `Most` strategies |
| `lo.SliceToChannel(bufSize, s)` | Slice into a buffered channel |
| `lo.ChannelToSlice(ch)` | Channel drained to a slice |
| `lo.Generator(bufSize, fn)` | Channel fed by a generator |
| `lo.Buffer(ch, size)` / `lo.BufferWithContext(ctx, ch, size)` / `lo.BufferWithTimeout(ch, size, timeout)` | Batched reads |
| `lo.FanIn(channels...)` | Many channels to one |
| `lo.FanOut(ch, count)` | One channel to many consumers |

## Concurrency helpers

| Function | Purpose |
| --- | --- |
| `lo.Async(fn)` / `lo.Async0` … `lo.Async6` | Run in a goroutine, result via channel |
| `lo.Attempt(maxRetries, fn)` | Retry until success or budget spent |
| `lo.AttemptWithDelay(max, delay, fn)` | Retries with a fixed gap |
| `lo.AttemptWhile(fn)` / `lo.AttemptWhileWithDelay(delay, fn)` | Retry while a predicate holds |
| `lo.Debounce(duration, fn)` | Fire after a quiet period; returns trigger + cancel |
| `lo.DebounceBy(duration, fn)` | Per-key debounce |
| `lo.Throttle(duration, fn)` | At most one call per duration |
| `lo.ThrottleWithCount(duration, count, fn)` | Bounded calls per duration |
| `lo.ThrottleBy(duration, fn)` / `lo.ThrottleByWithCount(duration, count, fn)` | Per-key throttle |
| `lo.WaitFor(fn, timeout, heartbeat)` / `lo.WaitForWithContext(ctx, fn, ...)` | Poll until a condition or timeout |
| `lo.Synchronize(mutexes...)` | Lock within a wrapper |
| `lo.Transaction(fn)` | Run with rollback on error |

## Type manipulation

| Function | Purpose |
| --- | --- |
| `lo.ToPtr(v)` | Value to pointer |
| `lo.Nil[T]()` | Typed nil pointer |
| `lo.EmptyableToPtr(v)` | Pointer, but nil for the zero value |
| `lo.FromPtr(p)` / `lo.FromPtrOr(p, fallback)` | Pointer to value, zero or fallback on nil |
| `lo.ToSlicePtr(s)` | `[]T` → `[]*T` |
| `lo.FromSlicePtr(s)` / `lo.FromSlicePtrOr(s, fallback)` | `[]*T` → `[]T`, zero or fallback on nil |
| `lo.ToAnySlice(s)` | `[]T` → `[]any` |
| `lo.FromAnySlice[T](s)` | `[]any` → `([]T, bool)` |
| `lo.IsNil(v)` / `lo.IsNotNil(v)` | Nil-safe interface checks |
| `lo.Empty[T]()` | Zero value of `T` |
| `lo.IsEmpty(v)` / `lo.IsNotEmpty(v)` | Zero-value checks |
| `lo.Coalesce(vals...)` / `lo.CoalesceOrEmpty(vals...)` | First non-zero value |
| `lo.CoalesceSlice(slices...)` / `lo.CoalesceSliceOrEmpty(slices...)` | First non-empty slice |
| `lo.CoalesceMap(maps...)` / `lo.CoalesceMapOrEmpty(maps...)` | First non-empty map |

## Function helpers

| Function | Purpose |
| --- | --- |
| `lo.Partial(fn, arg)` | Bind the first argument |
| `lo.Partial2(fn, arg)` … `lo.Partial5(fn, arg)` | Partial application for 2–5 args |

## Duration helpers

| Function | Purpose |
| --- | --- |
| `lo.Duration(fn)` | Wall-clock time of a call |
| `lo.Duration0(fn)` … `lo.Duration10(fn)` | Timing for functions with 0–10 return values; `(time.Duration, ...)` |

## Error helpers

| Function | Purpose |
| --- | --- |
| `lo.Must(val, err)` / `lo.Must0(err)` … `lo.Must6(...)` | Panic on error; tests and init only |
| `lo.Try(fn)` / `lo.Try1(fn)` … `lo.Try6(fn)` | True if the function does not panic |
| `lo.TryOr(fn, fallback)` / `lo.TryOr1(fn, fallback)` … `lo.TryOr6(...)` | Fallback on panic |
| `lo.TryCatch(fn, catchFn)` | Recover into a catch callback |
| `lo.TryWithErrorValue(fn)` / `lo.TryCatchWithErrorValue(fn, catchFn)` | Recover the echoed panic value as an error |
| `lo.Validate(conditions...)` | First failing condition |
| `lo.ErrorsAs[T](err)` | Generic wrapper over `errors.As` |
| `lo.Assert[T](v)` / `lo.Assertf[T](v, format, args...)` | Type assertion with a panic message |