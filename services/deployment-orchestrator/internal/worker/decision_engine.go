package worker

import "deployment-orchestrator/internal/model"

func applyStrategyDefaults(s *model.Strategy) {
	if s.CanaryPercent <= 0 {
		s.CanaryPercent = 10
	}
	if s.MaxFailRate <= 0 {
		s.MaxFailRate = 0.10
	}
	if s.AckWaitSec <= 0 {
		s.AckWaitSec = 5
	}
	if s.PollIntervalMs <= 0 {
		s.PollIntervalMs = 500
	}
	if s.MaxMLRisk <= 0 {
		s.MaxMLRisk = 0.8
	}
	if s.MaxTwinRisk <= 0 {
		s.MaxTwinRisk = 0.8
	}
	if s.CanaryTwinRisk <= 0 {
		s.CanaryTwinRisk = 0.5
	}
	if s.MaxLatencyThreshold <= 0 {
		s.MaxLatencyThreshold = 50
	}
	if s.MaxPacketLoss <= 0 {
		s.MaxPacketLoss = 0.05
	}
	if s.MaxOfflineRate <= 0 {
		s.MaxOfflineRate = 0.2
	}
}

func decidePreDeployment(
	strat model.Strategy,
	twinRisk *float64,
	mlRisk *float64,
) model.DecisionResult {
	reasons := []string{}

	if twinRisk != nil && *twinRisk > strat.MaxTwinRisk {
		reasons = append(reasons, "digital twin risk exceeded reject threshold")
		return model.DecisionResult{
			Action:  "reject",
			Reasons: reasons,
		}
	}

	if mlRisk != nil && *mlRisk > strat.MaxMLRisk {
		reasons = append(reasons, "ml risk exceeded reject threshold")
		return model.DecisionResult{
			Action:  "reject",
			Reasons: reasons,
		}
	}

	if twinRisk != nil && *twinRisk > strat.CanaryTwinRisk {
		reasons = append(reasons, "digital twin risk suggests canary rollout")
		return model.DecisionResult{
			Action:  "canary",
			Reasons: reasons,
		}
	}

	if mlRisk != nil && *mlRisk > 0.5 {
		reasons = append(reasons, "ml risk suggests canary rollout")
		return model.DecisionResult{
			Action:  "canary",
			Reasons: reasons,
		}
	}

	return model.DecisionResult{
		Action:  "full",
		Reasons: []string{"risk is acceptable"},
	}
}
