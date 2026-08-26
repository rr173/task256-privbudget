// Package compose 实现差分隐私预算组合的核心数学与图算法：
// 派生 DAG 的环检测、根人口传播、顺序/高级/RDP 组合以及超限评估。
// 该包只依赖 model，不触碰存储，便于独立测试与复用。
package compose

import (
	"task256-privbudget/internal/model"
)

// Graph 管理数据集版本的派生 DAG（parent 关系）。
type Graph struct {
	parents map[string][]string
	ids     map[string]bool
}

// BuildGraph 由数据集列表构建图并检测派生环，存在环则返回 ErrDatasetCycle。
func BuildGraph(datasets []model.DatasetVersion) (*Graph, error) {
	g := &Graph{
		parents: map[string][]string{},
		ids:     map[string]bool{},
	}
	for _, d := range datasets {
		g.ids[d.ID] = true
		g.parents[d.ID] = append([]string(nil), d.Parents...)
	}
	if err := g.detectCycle(); err != nil {
		return nil, err
	}
	return g, nil
}

// detectCycle 以三色 DFS 检测有向环。
func (g *Graph) detectCycle() error {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := map[string]int{}
	var visit func(id string) error
	visit = func(id string) error {
		color[id] = gray
		for _, p := range g.parents[id] {
			if !g.ids[p] {
				continue // 父不存在由上层校验，此处跳过避免崩溃
			}
			switch color[p] {
			case gray:
				return model.ErrDatasetCycle
			case white:
				if err := visit(p); err != nil {
					return err
				}
			}
		}
		color[id] = black
		return nil
	}
	for id := range g.ids {
		if color[id] == white {
			if err := visit(id); err != nil {
				return err
			}
		}
	}
	return nil
}

// root 返回数据集的人口根 ID：沿存在的父链向上传播。
// 规则：
//   - 无父或父不存在 → 自身即根；
//   - 单一父 → 继续向上；
//   - 多个存在的父（如连接/合并产生的新人口）→ 自身成为新根，不再上溯。
func (g *Graph) root(id string) string {
	cur := id
	seen := map[string]bool{}
	for {
		if seen[cur] {
			return cur
		}
		seen[cur] = true
		existing := g.parents[cur][:0:0]
		for _, p := range g.parents[cur] {
			if g.ids[p] {
				existing = append(existing, p)
			}
		}
		if len(existing) == 0 {
			return cur
		}
		if len(existing) > 1 {
			return cur
		}
		cur = existing[0]
	}
}

// RootOf 返回数据集的人口根 ID（导出包装，供编排层使用）。
func (g *Graph) RootOf(id string) string { return g.root(id) }
