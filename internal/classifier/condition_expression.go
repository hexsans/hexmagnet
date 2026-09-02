package classifier

import (
	"errors"
	"fmt"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

const expressionName = "expression"

type expressionCondition struct{}

var exprProgramPayload = payloadTransformer[string, *vm.Program]{
	spec: payloadGeneric[string]{
		jsonSchema: JSONSchema{
			schemaType:    celTypeString,
			"minLength":   1,
			"description": "An expression describing a condition",
		},
	},
	transform: func(s string, ctx compilerContext) (*vm.Program, error) {
		prg, err := expr.Compile(s, expr.Env(ExprEnv{}), expr.AsBool())
		if err != nil {
			return nil, ctx.error(fmt.Errorf("type-check error: %w", err))
		}

		return prg, nil
	},
}

var expressionConditionPayload = payloadUnion[*vm.Program]{
	oneOf: []TypedPayload[*vm.Program]{
		payloadSingleKeyValue[*vm.Program]{
			key:       expressionName,
			valueSpec: payloadMustSucceed[*vm.Program]{exprProgramPayload},
		},
		payloadMustSucceed[*vm.Program]{exprProgramPayload},
	},
}

func (expressionCondition) name() string {
	return expressionName
}

func (expressionCondition) compileCondition(ctx compilerContext) (condition, error) {
	prg, err := expressionConditionPayload.Unmarshal(ctx)
	if err != nil {
		return condition{}, ctx.error(err)
	}

	return condition{
		check: func(ctx executionContext) (bool, error) {
			env := ctx.exprEnv
			env.Torrent = ctx.torrentExpr
			env.Result = ctx.resultExpr
			env.Flags = ctx.flags

			out, err := expr.Run(prg, env)
			if err != nil {
				return false, err
			}

			bl, ok := out.(bool)
			if !ok {
				return false, errors.New("not bool")
			}

			return bl, nil
		},
	}, nil
}

func (expressionCondition) JSONSchema() JSONSchema {
	return expressionConditionPayload.JSONSchema()
}
