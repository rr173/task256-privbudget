package snapshot

import (
	"time"

	"task256-privbudget/internal/compose"
	"task256-privbudget/internal/model"
)

// Freeze 基于评估报告构造一个冻结快照（草稿状态），便于稍后发布。
func Freeze(id, name string, rep *compose.Report, ruleVersion string) model.BudgetSnapshot {
	now := time.Now().UTC()
	entries := append([]model.BudgetEntry(nil), rep.Entries...)
	return model.BudgetSnapshot{
		ID:          id,
		Name:        name,
		Rule:        rep.Rule,
		RuleVersion: ruleVersion,
		Status:      model.SnapDraft,
		Entries:     entries,
		Summary:     BuildSummary(rep),
		FrozenAt:    now,
		CreatedAt:   now,
	}
}
