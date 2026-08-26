package model

import "time"

// MechanismStatus 统计机制状态。
// 草稿 draft → 已验证 verified / 失效 invalid / 撤销 revoked。
type MechanismStatus string

const (
	MechDraft    MechanismStatus = "draft"    // 草稿
	MechVerified MechanismStatus = "verified" // 已验证
	MechInvalid  MechanismStatus = "invalid"  // 失效（参数非法）
	MechRevoked  MechanismStatus = "revoked"  // 撤销
)

// MechanismKind 差分隐私机制类型。
type MechanismKind string

const (
	MechLaplace    MechanismKind = "laplace"    // 拉普拉斯机制
	MechGaussian   MechanismKind = "gaussian"   // 高斯机制（支持 RDP 组合）
	MechExponential MechanismKind = "exponential" // 指数机制
	MechCount      MechanismKind = "count"      // 计数/稀疏向量类
)

// Mechanism 一个差分隐私统计机制，携带 ε/δ 与消费的若干数据集版本。
type Mechanism struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Kind        MechanismKind   `json:"kind"`
	Epsilon     float64         `json:"epsilon"`  // ε > 0
	Delta       float64         `json:"delta"`    // δ ∈ [0,1)
	DatasetIDs  []string        `json:"dataset_ids"` // 消费的数据集版本
	Status      MechanismStatus `json:"status"`
	ValidatedAt *time.Time      `json:"validated_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// IsLive 是否仍可作为有效预算来源（已验证且未撤销）。
func (m Mechanism) IsLive() bool {
	return m.Status == MechVerified
}

// Clone 返回浅拷贝。
func (m Mechanism) Clone() Mechanism {
	c := m
	c.DatasetIDs = append([]string(nil), m.DatasetIDs...)
	return c
}

// KnownMechanismKinds 返回受支持的全部机制类型。
func KnownMechanismKinds() []MechanismKind {
	return []MechanismKind{MechLaplace, MechGaussian, MechExponential, MechCount}
}

// DedupDatasetIDs 按首次出现顺序去重：同一数据集即使重复声明，
// 对该数据集的预算消耗也只计一次。
func DedupDatasetIDs(ids []string) []string {
	if len(ids) <= 1 {
		return append([]string(nil), ids...)
	}
	seen := make(map[string]bool, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
