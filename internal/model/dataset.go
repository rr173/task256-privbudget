// Package model 定义差分隐私统计预算组合验证服务的领域实体、枚举与错误。
//
// 业务域：隐私工程师验证多次统计发布组合后是否仍在允许的差分隐私预算内。
// 核心对象：数据集版本（人口与预算上限）、统计机制（ε/δ 参数）、
// 发布批次（消耗预算的发布）、预算快照（冻结的组合结论）。
package model

import "time"

// DatasetStatus 数据集版本状态。
// 登记 registered → 可发布 releasable → 预算紧张 tense / 超限 overlimit → 封存 sealed。
// tense 与 overlimit 由预算评估推导，sealed 由用户动作冻结且不可逆。
type DatasetStatus string

const (
	DatasetRegistered DatasetStatus = "registered" // 登记
	DatasetReleasable DatasetStatus = "releasable" // 可发布
	DatasetTense      DatasetStatus = "tense"      // 预算紧张
	DatasetOverlimit  DatasetStatus = "overlimit"  // 超限
	DatasetSealed     DatasetStatus = "sealed"     // 封存
)

// DatasetVersion 一个版本化的数据集（人口）。
// Parents 为派生 DAG 的父节点：若 A 的父节点为 B，则 A 的人口 ⊆ B 的人口，
// 对 A 的预算消耗会向上传播到根人口 B。
type DatasetVersion struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Version     string        `json:"version"`
	Status      DatasetStatus `json:"status"`
	PrivacyUnit string        `json:"privacy_unit"` // 隐私单元，如 "individual"
	Parents     []string      `json:"parents"`      // 派生 DAG 父节点 ID 列表
	EpsilonCap  float64       `json:"epsilon_cap"`  // ε 预算上限（根人口维度）
	DeltaCap    float64       `json:"delta_cap"`    // δ 预算上限
	CreatedAt   time.Time     `json:"created_at"`
	SealedAt    *time.Time    `json:"sealed_at,omitempty"`
}

// IsSealed 是否已封存（冻结，禁止修改）。
func (d DatasetVersion) IsSealed() bool { return d.Status == DatasetSealed }

// Clone 返回浅拷贝，避免调用方持有内部切片引用造成的并发污染。
func (d DatasetVersion) Clone() DatasetVersion {
	c := d
	c.Parents = append([]string(nil), d.Parents...)
	return c
}
