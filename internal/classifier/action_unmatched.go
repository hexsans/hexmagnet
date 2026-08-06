package classifier

const unmatchedName = "unmatched"

type unmatchedAction struct{}

func (unmatchedAction) name() string {
	return unmatchedName
}

var unmatchedPayloadSpec = payloadLiteral[string]{
	literal:     unmatchedName,
	description: "Return a unmatched error for the current torrent",
}

func (unmatchedAction) compileAction(ctx compilerContext) (action, error) {
	if _, err := unmatchedPayloadSpec.Unmarshal(ctx); err != nil {
		return action{}, ctx.error(err)
	}

	path := ctx.path

	return action{
		run: func(ctx executionContext) (ClassificationResult, error) {
			return ctx.result, RuntimeError{Cause: ErrUnmatched, Path: path}
		},
	}, nil
}

func (unmatchedAction) JSONSchema() JSONSchema {
	return unmatchedPayloadSpec.JSONSchema()
}
