package assembler

import (
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func ToDeliveryDispatchPlanView(plan model.DeliveryDispatchPlan) query.DeliveryDispatchPlanView {
	steps := make([]query.DeliveryDispatchStepView, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		steps = append(steps, query.DeliveryDispatchStepView{
			StepIndex: step.StepIndex,
			Kind:      string(step.Kind),
			Channel:   step.Channel,
			ChatID:    step.ChatID,
			Message:   step.Message,
			Image:     step.Image,
			File:      step.File,
		})
	}
	return query.DeliveryDispatchPlanView{
		EventID:    plan.EventID,
		Channel:    plan.Channel,
		ChatID:     plan.ChatID,
		StepCount:  plan.StepCount,
		Steps:      steps,
		Attributes: plan.Attributes,
	}
}

func ToDeliveryDispatchResultView(execution model.DeliveryDispatchExecution) query.DeliveryDispatchResultView {
	results := make([]query.DeliveryDispatchResultStepView, 0, len(execution.Results))
	for _, result := range execution.Results {
		results = append(results, query.DeliveryDispatchResultStepView{
			StepIndex:         result.StepIndex,
			Kind:              string(result.Kind),
			Channel:           result.Channel,
			ChatID:            result.ChatID,
			Status:            string(result.Status),
			Provider:          result.Provider,
			ProviderMessageID: result.ProviderMessageID,
			Attributes:        result.Attributes,
		})
	}
	return query.DeliveryDispatchResultView{
		EventID:    execution.EventID,
		StepCount:  execution.StepCount,
		Results:    results,
		Attributes: execution.Attributes,
	}
}
