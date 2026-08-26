package release

import (
	"time"

	"task256-privbudget/internal/compose"
	"task256-privbudget/internal/model"
)

// Decide 依据评估报告决定发布状态并写入超限路径与评估时间。
//
// 设计：预算超限通过“数据集状态=overlimit”与评估报告反映，并不阻断发布——
// 隐私工程师据此决定是否撤销某次发布。只有当发布所引用的机制已不可用
// （非 verified/revoked 等）时才硬性拒绝。超限路径记录导致所在单元超限的发布链。
func Decide(r model.Release, m model.Mechanism, rep *compose.Report) model.Release {
	now := time.Now().UTC()
	r.EvaluatedAt = &now
	if !m.IsLive() {
		r.Status = model.ReleaseRejected
		return r
	}
	r.Status = model.ReleaseAllowed
	r.OverlimitPath = compose.OverlimitPathForRelease(rep, r.ID)
	return r
}
