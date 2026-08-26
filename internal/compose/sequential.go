package compose

// SequentialCompose 顺序组合（基础组合定理）：同人口内 ε、δ 直接相加。
func SequentialCompose(eps []float64, dels []float64) (float64, float64) {
	var e, d float64
	for _, v := range eps {
		e += v
	}
	for _, v := range dels {
		d += v
	}
	return e, d
}
