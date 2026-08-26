package model

import "time"

// ReleaseStatus 发布批次状态。
// 待评估 pending → 允许 allowed / 拒绝 rejected / 已撤销 revoked。
// 仅 allowed 状态的发布计入实时预算消耗。
type ReleaseStatus string

const (
	ReleasePending  ReleaseStatus = "pending"  // 待评估
	ReleaseAllowed  ReleaseStatus = "allowed"  // 允许（实时消耗预算）
	ReleaseRejected ReleaseStatus = "rejected" // 拒绝
	ReleaseRevoked  ReleaseStatus = "revoked"  // 已撤销（回退预算）
)

// CompositionRule 预算组合规则。
type CompositionRule string

const (
	// RuleSequential 顺序组合：同根人口内 ε、δ 直接相加。
	RuleSequential CompositionRule = "sequential"
	// RuleAdvanced 高级组合：使用 Dwork 高级组合界压缩 ε。
	RuleAdvanced CompositionRule = "advanced"
	// RuleParallel 并行组合：不向上传播，按各数据集自身上限独立计。
	RuleParallel CompositionRule = "parallel"
	// RuleRDP 基于 Rényi DP 的会计师组合（仅高斯机制有效）。
	RuleRDP CompositionRule = "rdp"
)

// SupportedCompositionRules 返回服务支持的全部组合规则，作为唯一的合法集合。
// 调用方据此判断请求的规则是否受支持，绝不可对不支持的规则静默改用其它规则。
func SupportedCompositionRules() []CompositionRule {
	return []CompositionRule{RuleSequential, RuleAdvanced, RuleParallel, RuleRDP}
}

// IsSupportedRule 判断组合规则是否在服务支持的合法集合内。
func IsSupportedRule(r CompositionRule) bool {
	for _, x := range SupportedCompositionRules() {
		if x == r {
			return true
		}
	}
	return false
}

// Release 一次统计发布批次，引用一个机制并采用一种组合规则。
type Release struct {
	ID             string           `json:"id"`
	Name           string           `json:"name"`
	MechanismID    string           `json:"mechanism_id"`
	Rule           CompositionRule  `json:"rule"`
	Status         ReleaseStatus    `json:"status"`
	EvaluatedAt    *time.Time       `json:"evaluated_at,omitempty"`
	OverlimitPath  []string         `json:"overlimit_path,omitempty"` // 触发超限的发布链
	CreatedAt      time.Time        `json:"created_at"`
}

// IsLive 是否计入实时预算。
func (r Release) IsLive() bool { return r.Status == ReleaseAllowed }

// Clone 返回浅拷贝。
func (r Release) Clone() Release {
	c := r
	c.OverlimitPath = append([]string(nil), r.OverlimitPath...)
	return c
}
