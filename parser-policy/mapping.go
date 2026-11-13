package parserpolicy

import (
	"context"
	"errors"

	"nexora.local/nli-v20/internal/fieldmapping"
	"nexora.local/nli-v20/internal/mixedkv"
)

func (e *ParserPackExecutor) executeMappingStage(ctx context.Context, request *ExecutionRequest, rule ParserRule, stage Stage, fields workingSet, checkHeader bool) (int, int, map[string]fieldmapping.Evidence, error) {
	plan := e.mappingPlans[rule.ID+"/"+stage.ID]
	if plan == nil {
		return 0, 0, nil, fail("mapping_config", "mapping plan was not compiled")
	}
	if len(request.Input) > e.limitsFor(stage).MaxInputBytes {
		return 0, 0, nil, fail("input_limit", "mapping input exceeds stage limit")
	}
	result, err := plan.Execute(ctx, request.Input)
	if err != nil {
		if errors.Is(err, fieldmapping.ErrIncompleteKV) {
			return 0, 0, nil, fail("incomplete_structured_tail", err.Error())
		}
		return 0, 0, nil, fail("mapping_validation", err.Error())
	}
	if err := plan.ValidateComplete(request.Input, result); err != nil {
		if errors.Is(err, fieldmapping.ErrIncompleteHeader) {
			return 0, 0, nil, fail("incomplete_structured_header", err.Error())
		}
		return 0, 0, nil, fail("incomplete_structured_tail", err.Error())
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
		field := capturedField{Value: value, Source: evidence.Source, Start: evidence.Start, End: evidence.End, DerivedRule: ruleID, Dynamic: dynamic[name]}
		for _, span := range evidence.Spans {
			field.SourceSpans = append(field.SourceSpans, mixedkv.Span{Start: span.Start, End: span.End})
		}
		captured[name] = field
	}
	var omissions map[string]fieldmapping.Evidence
	if checkHeader {
		omissions = plan.HeaderOmissions(result)
	}
	added, modified, err := mergeCaptured(rule, fields, captured, false)
	return added, modified, omissions, err
}
