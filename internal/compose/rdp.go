package compose

import "math"

// GaussianRDPEpsilon 返回高斯机制在阶数 α 处的 RDP 上界（灵敏度 Δ=1）。
//
// 经典结论：高斯机制 M(x)=f(x)+N(0,σ²) 满足 (ε,δ)-DP 当
//
//	σ² ≥ Δ² · 2 ln(1.25/δ) / ε²   (Δ=1)
//
// 其 RDP 在 α 处上界为 α·Δ²/(2σ²)。由上式反解 σ² 后代入得：
//
//	RDP(α) = α · ε² / (4 · ln(1.25/δ))
//
// 该实现以 (ε,δ) 直接反推，自洽且与 Gaussian 机制一致。
func GaussianRDPEpsilon(epsilon, delta float64, alpha float64) float64 {
	if epsilon <= 0 || delta <= 0 || delta >= 1 || alpha <= 1 {
		return 0
	}
	denom := 4.0 * math.Log(1.25/delta)
	if denom <= 0 {
		return 0
	}
	return alpha * epsilon * epsilon / denom
}

// rdpCompose 将一组 (ε,δ) 高斯机制组合为 (ε', δ')：先对每个 α 累加 RDP 上界，
// 再经 α-RDP → (ε,δ) 转换取最小界。
func rdpCompose(eps, dels []float64, deltaPrime float64) (float64, float64) {
	if len(eps) == 0 {
		return 0, 0
	}
	if deltaPrime <= 0 {
		deltaPrime = 1e-6
	}
	bestEps := math.Inf(1)
	for alpha := 1.5; alpha <= 64; alpha += 0.5 {
		rho := 0.0
		for i := range eps {
			rho += GaussianRDPEpsilon(eps[i], dels[i], alpha)
		}
		e := alpha*rho + math.Log(1.0/deltaPrime)/(alpha-1)
		if e < bestEps {
			bestEps = e
		}
	}
	if math.IsInf(bestEps, 1) {
		bestEps = 0
	}
	return bestEps, deltaPrime
}
