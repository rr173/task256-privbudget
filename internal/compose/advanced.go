package compose

import "math"

// AdvancedCompose 依据 Dwork 等人提出的高级组合定理（Advanced Composition），
// 将 k 个 (ε_i, δ_i) 机制组合为 (ε', Σδ_i + δ_target) 的高级组合界：
//
//	ε' = max_i ε_i · sqrt(2 · k · ln(1/δ_target))
//
// 其中 δ_target 为容错项（默认 1e-6）。高级组合在 k 较大时相比朴素顺序相加
// 显著压缩 ε，是差分隐私组合 accountant 的常用上界。
func AdvancedCompose(eps []float64, dels []float64, deltaTarget float64) (float64, float64) {
	if deltaTarget <= 0 {
		deltaTarget = 1e-6
	}
	k := len(eps)
	if k == 0 {
		return 0, 0
	}
	maxEps := 0.0
	for _, e := range eps {
		if e > maxEps {
			maxEps = e
		}
	}
	var sumDelta float64
	for _, d := range dels {
		sumDelta += d
	}
	epsPrime := maxEps * math.Sqrt(2.0*float64(k)*math.Log(1.0/deltaTarget))
	deltaPrime := sumDelta + deltaTarget
	if deltaPrime > 1 {
		deltaPrime = 1
	}
	return epsPrime, deltaPrime
}
