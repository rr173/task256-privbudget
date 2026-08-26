package mechanism

import "task256-privbudget/internal/model"

// validateParams 校验机制类型与 ε/δ 合法性。
func validateParams(m model.Mechanism) error {
	if !knownKind(m.Kind) {
		return model.ErrUnknownKind
	}
	if m.Epsilon <= 0 || m.Delta < 0 || m.Delta >= 1 {
		return model.ErrIllegalEpsilonDelta
	}
	return nil
}

// knownKind 判断机制类型是否受支持。
func knownKind(k model.MechanismKind) bool {
	for _, x := range model.KnownMechanismKinds() {
		if x == k {
			return true
		}
	}
	return false
}
