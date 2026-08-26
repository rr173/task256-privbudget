package model

import "time"

// SnapshotStatus 预算快照状态：草稿 draft → 发布 published / 替代 superseded。
type SnapshotStatus string

const (
	SnapDraft       SnapshotStatus = "draft"       // 草稿
	SnapPublished   SnapshotStatus = "published"   // 发布
	SnapSuperseded  SnapshotStatus = "superseded"  // 被替代
)

// BudgetEntry 快照中一个预算单元（根人口或单数据集）的冻结结论。
type BudgetEntry struct {
	Key         string  `json:"key"`          // 根人口代表 ID 或数据集 ID
	Scope       string  `json:"scope"`        // "root" 或 "dataset"
	EpsilonUsed float64 `json:"epsilon_used"` // 已消耗 ε
	DeltaUsed   float64 `json:"delta_used"`   // 已消耗 δ
	EpsilonCap  float64 `json:"epsilon_cap"`  // ε 上限
	DeltaCap    float64 `json:"delta_cap"`    // δ 上限
	Overlimit   bool    `json:"overlimit"`    // 是否超限
	Contributors []string `json:"contributors"` // 贡献预算的发布 ID
}

// BudgetSnapshot 冻结某一时刻的预算组合结论。
type BudgetSnapshot struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Rule          CompositionRule `json:"rule"`
	RuleVersion   string         `json:"rule_version"` // 组合规则版本（冻结）
	Status        SnapshotStatus `json:"status"`
	Entries       []BudgetEntry  `json:"entries"`
	Summary       string         `json:"summary"`     // 人类可读结论
	FrozenAt      time.Time      `json:"frozen_at"`
	CreatedAt     time.Time      `json:"created_at"`
}

// Clone 返回深拷贝。
func (s BudgetSnapshot) Clone() BudgetSnapshot {
	c := s
	c.Entries = append([]BudgetEntry(nil), s.Entries...)
	for i := range c.Entries {
		c.Entries[i].Contributors = append([]string(nil), s.Entries[i].Contributors...)
	}
	return c
}
