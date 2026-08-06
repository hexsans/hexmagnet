package classifier

type ifElseAction struct{}

const ifElseName = "if_else"

func (ifElseAction) name() string {
	return ifElseName
}

type ifElsePayload struct {
	Condition  any
	IfAction   any
	ElseAction any
}

var ifElsePayloadSpec = payloadSingleKeyValue[ifElsePayload]{
	key: ifElseName,
	valueSpec: payloadMustSucceed[ifElsePayload]{payloadStruct[ifElsePayload]{
		jsonSchema: map[string]any{
			schemaType: schemaTypeObject,
			schemaProperties: map[string]any{
				schemaKeyCondition: map[string]any{
					schemaRef: refCondition,
				},
				"if_action": map[string]any{
					schemaRef: refAction,
				},
				"else_action": map[string]any{
					schemaRef: refAction,
				},
			},
			"required":                 []string{"condition"},
			schemaAdditionalProperties: false,
		},
	}},
	description: "Execute an action based on a condition",
}

func (ifElseAction) compileAction(ctx compilerContext) (action, error) {
	p, decodeErr := ifElsePayloadSpec.Unmarshal(ctx)
	if decodeErr != nil {
		return action{}, ctx.error(decodeErr)
	}

	cond, cErr := ctx.compileCondition(ctx.child("condition", p.Condition))
	if cErr != nil {
		return action{}, ctx.error(cErr)
	}

	var ifAction, elseAction action

	if p.IfAction != nil {
		pIfAction, ifErr := ctx.compileAction(ctx.child("if_action", p.IfAction))
		if ifErr != nil {
			return action{}, ctx.error(ifErr)
		}

		ifAction = pIfAction
	}

	if p.ElseAction != nil {
		pElseAction, err := ctx.compileAction(ctx.child("else_action", p.ElseAction))
		if err != nil {
			return action{}, ctx.error(err)
		}

		elseAction = pElseAction
	}

	return action{
		run: func(ctx executionContext) (ClassificationResult, error) {
			if result, err := cond.check(ctx); err != nil {
				return ClassificationResult{}, err
			} else if result {
				if ifAction.run != nil {
					return ifAction.run(ctx)
				}
			} else {
				if elseAction.run != nil {
					return elseAction.run(ctx)
				}
			}

			return ctx.result, nil
		},
	}, nil
}

func (ifElseAction) JSONSchema() JSONSchema {
	return ifElsePayloadSpec.JSONSchema()
}
