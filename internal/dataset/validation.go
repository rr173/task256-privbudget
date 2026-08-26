package dataset

import (
	"context"

	"task256-privbudget/internal/compose"
	"task256-privbudget/internal/model"
)

// checkParents 校验父数据集全部存在且派生关系无环。
// 自环（指向自身）与缺失父均直接报错；其它环由 compose.BuildGraph 检出。
func (s *Service) checkParents(ctx context.Context, d model.DatasetVersion) error {
	if len(d.Parents) == 0 {
		return nil
	}
	all, err := s.store.DatasetList(ctx)
	if err != nil {
		return err
	}
	byID := make(map[string]model.DatasetVersion, len(all))
	for _, x := range all {
		byID[x.ID] = x
	}
	for _, p := range d.Parents {
		if p == d.ID {
			return model.ErrDatasetCycle
		}
		if _, ok := byID[p]; !ok {
			return model.ErrParentMissing
		}
	}
	all = append(all, d)
	if _, err := compose.BuildGraph(all); err != nil {
		return err
	}
	return nil
}
