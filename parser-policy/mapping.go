package parserpolicy

import (
	"context"
	"errors"

	"nexora.local/nli-v20/internal/fieldmapping"
)

func (e *ParserPackExecutor) executeMappingStage(ctx context.Context, request *ExecutionRequest, rule ParserRule, stage Stage, fields workingSet) (int, int, error) {
	plan := e.mappingPlans[rule.ID+"/"+stage.ID]
	if plan == nil {
		return 0, 0, fail("mapping_config", "mapping plan was not compiled")
	}
	if len(request.Input) > e.limitsFor(stage).MaxInputBytes {
		return 0, 0, fail("input_limit", "mapping input exceeds stage limit")
	}
	result, err := plan.Execute(ctx, request.Input)
	if err != nil {
		if errors.Is(err, fieldmapping.ErrIncompleteKV) {
			return 0, 0, fail("incomplete_structured_tail", err.Error())
		}
		return 0, 0, fail("mapping_validation", err.Error())
	}
	if err := plan.ValidateComplete(request.Input, result); err != nil {
		return 0, 0, fail("incomplete_structured_tail", err.Error())
	}
	dynamic := map[string]bool{}
	for _, name := range result.DynamicFields {
		dynamic[name] = true
	}
	captured := make(map[string]capturedField, len(result.Values))
	for name, value := range result.Values {
		evidence := result.Evidence[name]
		ruleID := ""
		if !evidence.Source {
			ruleID = "nfi:" + stage.ID
		}
		captured[name] = capturedField{Value: value, Source: evidence.Source, Start: evidence.Start, End: evidence.End, DerivedRule: ruleID, Dynamic: dynamic[name]}
	}
	return mergeCaptured(rule, fields, captured, false)
}
