package compose

import (
	"fmt"
	"sort"

	"task256-privbudget/internal/model"
)

// Contribution 一个发布对一个预算单元的 ε/δ 贡献（含发布 ID 以便生成超限路径）。
type Contribution struct {
	ReleaseID string
	Epsilon   float64
	Delta     float64
}

// World 预算评估所需的当前世界状态快照。
type World struct {
	Datasets   []model.DatasetVersion
	Mechanisms []model.Mechanism
	Releases   []model.Release // 传入全部发布，内部仅统计 allowed
}

// Report 一次预算评估的结论。
type Report struct {
	Rule        model.CompositionRule
	Entries     []model.BudgetEntry
	Overlimited bool
	Roots       map[string]string // datasetID -> rootID，便于展示人口归属
}

// Evaluate 按给定组合规则计算各预算单元的消耗、上限与超限情况。
//
// 语义：
//   - sequential：同根人口内 ε、δ 直接相加。
//   - advanced：使用高级组合界压缩 ε。
//   - parallel：不向上传播，按各数据集自身上限独立计。
//   - rdp：仅对全部为高斯机制且 δ>0 的组合有效，否则返回错误。
func Evaluate(world World, rule model.CompositionRule) (*Report, error) {
	g, err := BuildGraph(world.Datasets)
	if err != nil {
		return nil, err
	}
	mechByID := make(map[string]model.Mechanism, len(world.Mechanisms))
	for _, m := range world.Mechanisms {
		mechByID[m.ID] = m
	}
	datasetByID := make(map[string]model.DatasetVersion, len(world.Datasets))
	for _, d := range world.Datasets {
		datasetByID[d.ID] = d
	}

	type unit struct {
		key      string
		scope    string
		epsCap   float64
		delCap   float64
		contribs []Contribution
	}
	units := map[string]*unit{}
	roots := map[string]string{}

	for _, r := range world.Releases {
		if r.Status != model.ReleaseAllowed {
			continue
		}
		m, ok := mechByID[r.MechanismID]
		if !ok || !m.IsLive() {
			continue
		}
		if rule == model.RuleRDP {
			if m.Kind != model.MechGaussian || m.Delta <= 0 {
				return nil, fmt.Errorf("%w: rdp rule requires gaussian mechanism with delta>0 (release %s)",
					model.ErrMechanismInvalid, r.ID)
			}
		}
		// 同一发布内对数据集去重：机制即便重复声明同一数据集，
		// 也只对该数据集计费一次，避免单次发布被重复扣减或误触超限。
		seen := make(map[string]bool, len(m.DatasetIDs))
		for _, did := range m.DatasetIDs {
			if seen[did] {
				continue
			}
			seen[did] = true
			d, ok := datasetByID[did]
			if !ok {
				continue
			}
			var ukey, scope string
			var capEps, capDel float64
			if rule == model.RuleParallel {
				ukey = did
				scope = "dataset"
				capEps = d.EpsilonCap
				capDel = d.DeltaCap
			} else {
				root := g.root(did)
				ukey = root
				scope = "root"
				rd := datasetByID[root]
				capEps = rd.EpsilonCap
				capDel = rd.DeltaCap
				roots[did] = root
			}
			u, ok := units[ukey]
			if !ok {
				u = &unit{key: ukey, scope: scope, epsCap: capEps, delCap: capDel}
				units[ukey] = u
			}
			u.contribs = append(u.contribs, Contribution{ReleaseID: r.ID, Epsilon: m.Epsilon, Delta: m.Delta})
		}
	}

	report := &Report{Rule: rule, Roots: roots}
	over := false
	for _, u := range units {
		eps := make([]float64, len(u.contribs))
		dels := make([]float64, len(u.contribs))
		for i, c := range u.contribs {
			eps[i] = c.Epsilon
			dels[i] = c.Delta
		}
		var eu, du float64
		switch rule {
		case model.RuleAdvanced:
			eu, du = AdvancedCompose(eps, dels, 1e-6)
		case model.RuleRDP:
			eu, du = rdpCompose(eps, dels, 1e-6)
		default: // sequential & parallel
			eu, du = SequentialCompose(eps, dels)
		}

		contribSorted := append([]Contribution(nil), u.contribs...)
		sort.Slice(contribSorted, func(i, j int) bool {
			return contribSorted[i].Epsilon > contribSorted[j].Epsilon
		})
		contribIDs := make([]string, len(contribSorted))
		for i, c := range contribSorted {
			contribIDs[i] = c.ReleaseID
		}

		isOver := eu > u.epsCap+1e-9 || du > u.delCap+1e-9
		if isOver {
			over = true
		}
		report.Entries = append(report.Entries, model.BudgetEntry{
			Key:          u.key,
			Scope:        u.scope,
			EpsilonUsed:  eu,
			DeltaUsed:    du,
			EpsilonCap:   u.epsCap,
			DeltaCap:     u.delCap,
			Overlimit:    isOver,
			Contributors: contribIDs,
		})
	}
	// 稳定排序，便于测试与展示
	sort.Slice(report.Entries, func(i, j int) bool {
		if report.Entries[i].Scope != report.Entries[j].Scope {
			return report.Entries[i].Scope < report.Entries[j].Scope
		}
		return report.Entries[i].Key < report.Entries[j].Key
	})
	report.Overlimited = over
	return report, nil
}

// OverlimitPathForRelease 给定评估结论与某个发布 ID，返回导致该发布所在单元超限的发布链
// （即该单元的全部贡献者，按 ε 降序）。若该发布所在单元未超限则返回空切片。
func OverlimitPathForRelease(rep *Report, releaseID string) []string {
	for _, e := range rep.Entries {
		if !e.Overlimit {
			continue
		}
		for _, c := range e.Contributors {
			if c == releaseID {
				return append([]string(nil), e.Contributors...)
			}
		}
	}
	return nil
}
