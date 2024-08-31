# Performance optimization report

## Initial metrics
```
BenchmarkSlow-12              24          42754933 ns/op        20260408 B/op     182818 allocs/op
BenchmarkFast-12              25          43304436 ns/op        20316442 B/op     182827 allocs/op
```
## CPU bottle necks
* `regexp compile` is the first very obvious bottle neck, as regexp consumes 30% of total execution time. 
* `runtime mallocgc` during syntax parse in regexp also is expensive time-wise (14.79%)
* `json Unmarshal` takes 14.54% of total time execution
### Memory bottle necks
* The entiry programm consumes 934.52MB, which is insanely a lot (~480MB each search function)
* `regexp MatchString` takes 718.30MB in total. It is a sum of syntax parsing and compilation.
* `json Unmarshal` - parsing takes 73.51MB in total

## Possible Optimizations
* Instead of reading entire file at once, stream it line by line
* Allocate memory better. Reuse structures in loops
* Replace `json Unmarshal` with `easyjson`
* Replace compiled regexp with `strings.replaceAll` method
```
    r := regexp.MustCompile("@") // before
		email := r.ReplaceAllString(user["email"].(string), " [at] ")

		email := strings.ReplaceAll(user["email"].(string), "@", " [at] ") // after
```
* remove deprecated ioutil and replace with io in `io.ReadAll(file)`

## Post-optimization metrics
```
BenchmarkFast-12             528           2258169 ns/op          480265 B/op       6330 allocs/op
```
### Comparing to the best solution
* ~60% more efficient at operations ratio: 10422 allocs/op VS 6330 allocs/op
* ~16% more efficient at memory allocation: 559910 B/op VS 480265 B/op
* ~12% faster in execution: 2782432 ns/op VS 2258169 ns/op
