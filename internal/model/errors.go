package model

import "errors"

// 领域错误：调用方据此返回 4xx 业务错误。
var (
	// ErrNotFound 实体不存在。
	ErrNotFound = errors.New("entity not found")
	// ErrDatasetCycle 派生 DAG 出现环。
	ErrDatasetCycle = errors.New("dataset derivation cycle detected")
	// ErrDatasetSealed 数据集已封存，禁止修改。
	ErrDatasetSealed = errors.New("dataset is sealed and cannot be modified")
	// ErrMechanismInvalid 机制参数非法。
	ErrMechanismInvalid = errors.New("mechanism parameters invalid")
	// ErrReleaseConflict 发布状态冲突（如对非待评估发布重复评估）。
	ErrReleaseConflict = errors.New("release status conflict")
	// ErrRevokeMissing 撤销不存在的发布。
	ErrRevokeMissing = errors.New("cannot revoke a release that is not allowed")
	// ErrIllegalEpsilonDelta ε/δ 非法。
	ErrIllegalEpsilonDelta = errors.New("epsilon must be > 0 and delta in [0,1)")
	// ErrUnknownRule 未知组合规则。
	ErrUnknownRule = errors.New("unknown composition rule")
	// ErrUnknownKind 未知机制类型。
	ErrUnknownKind = errors.New("unknown mechanism kind")
	// ErrParentMissing 父数据集不存在。
	ErrParentMissing = errors.New("parent dataset version does not exist")
	// ErrDatasetMissing 引用的数据集不存在。
	ErrDatasetMissing = errors.New("referenced dataset version does not exist")
	// ErrMechanismMissing 引用的机制不存在。
	ErrMechanismMissing = errors.New("referenced mechanism does not exist")
	// ErrAlreadyExists 实体已存在（幂等冲突）。
	ErrAlreadyExists = errors.New("entity already exists")
	// ErrSnapshotSealed 快照已发布/替代，禁止修改。
	ErrSnapshotSealed = errors.New("snapshot is already published or superseded")
)
