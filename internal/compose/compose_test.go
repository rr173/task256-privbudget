package compose

import (
	"math"
	"testing"

	"task256-privbudget/internal/model"
)

func TestAdvancedComposeBounds(t *testing.T) {
	// 单组：高级组合应返回有限正值
	adv, _ := AdvancedCompose([]float64{0.3}, []float64{0}, 1e-6)
	if adv <= 0 || math.IsInf(adv, 1) {
		t.Fatalf("advanced should be positive finite, got %v", adv)
	}
	// 大数据量 k：高级组合 ε 应优于朴素顺序相加（√k 级 vs k 级）
	k := 200
	epsK := make([]float64, k)
	delsK := make([]float64, k)
	for i := range epsK {
		epsK[i] = 0.05
	}
	seq, _ := SequentialCompose(epsK, delsK)
	advK, _ := AdvancedCompose(epsK, delsK, 1e-6)
	if !(advK < seq) {
		t.Fatalf("for large k advanced %v should be < sequential %v", advK, seq)
	}
}

func TestRDPComposeMonotonic(t *testing.T) {
	// 单个高斯机制：RDP 组合应返回有限正值
	one, _ := rdpCompose([]float64{0.4}, []float64{1e-6}, 1e-6)
	if one <= 0 || math.IsInf(one, 1) {
		t.Fatalf("single rdp should be positive finite, got %v", one)
	}
	// 机制越多预算消耗不减（单调非降）
	two, _ := rdpCompose([]float64{0.4, 0.4}, []float64{1e-6, 1e-6}, 1e-6)
	if two < one {
		t.Fatalf("rdp should be non-decreasing in mechanisms: one=%v two=%v", one, two)
	}
	if math.IsInf(two, 1) {
		t.Fatalf("rdp should be finite, got %v", two)
	}
}

func TestGraphCycleDetection(t *testing.T) {
	ds := []model.DatasetVersion{
		{ID: "A", Parents: []string{"B"}},
		{ID: "B", Parents: []string{"A"}},
	}
	if _, err := BuildGraph(ds); err == nil {
		t.Fatal("expected cycle error, got nil")
	} else if err != model.ErrDatasetCycle {
		t.Fatalf("expected ErrDatasetCycle, got %v", err)
	}
}

func TestGraphRootPropagation(t *testing.T) {
	ds := []model.DatasetVersion{
		{ID: "D", EpsilonCap: 0.3},
		{ID: "C", Parents: []string{"D"}, EpsilonCap: 1.0},
	}
	g, err := BuildGraph(ds)
	if err != nil {
		t.Fatal(err)
	}
	if r := g.RootOf("C"); r != "D" {
		t.Fatalf("root of C should be D, got %s", r)
	}
}

func TestEvaluateParallelVsSequential(t *testing.T) {
	// D(根,cap0.3) ← C(cap1.0)；机制消费 C，ε=0.6。
	// 顺序组合向上传播到 D(cap0.3) → 超限；并行组合按 C(cap1.0) → 不超限。
	ds := []model.DatasetVersion{
		{ID: "D", EpsilonCap: 0.3, DeltaCap: 1},
		{ID: "C", Parents: []string{"D"}, EpsilonCap: 1.0, DeltaCap: 1},
	}
	ms := []model.Mechanism{
		{ID: "M", Kind: model.MechLaplace, Epsilon: 0.6, Delta: 0, DatasetIDs: []string{"C"}, Status: model.MechVerified},
	}
	rs := []model.Release{
		{ID: "R", MechanismID: "M", Rule: model.RuleSequential, Status: model.ReleaseAllowed},
	}
	seqRep, err := Evaluate(World{Datasets: ds, Mechanisms: ms, Releases: rs}, model.RuleSequential)
	if err != nil {
		t.Fatal(err)
	}
	if !seqRep.Overlimited {
		t.Fatalf("sequential should be overlimited, got %+v", seqRep.Entries)
	}
	parRep, err := Evaluate(World{Datasets: ds, Mechanisms: ms, Releases: rs}, model.RuleParallel)
	if err != nil {
		t.Fatal(err)
	}
	if parRep.Overlimited {
		t.Fatalf("parallel should NOT be overlimited, got %+v", parRep.Entries)
	}
}
