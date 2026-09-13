package classifier

import "context"

type strictLLMContextKey struct{}

// WithStrictLLM marks ctx so an LLM classify failure aborts the workflow
// instead of falling back to rules. Batch maintenance jobs use it to pause
// on LLM outages rather than silently degrading classification quality.
func WithStrictLLM(ctx context.Context) context.Context {
	return context.WithValue(ctx, strictLLMContextKey{}, true)
}

// StrictLLM reports whether ctx was marked by WithStrictLLM.
func StrictLLM(ctx context.Context) bool {
	strict, _ := ctx.Value(strictLLMContextKey{}).(bool)

	return strict
}

// LLMClassifyError wraps a failed LLM classification attempt. It is only
// produced in strict mode; regular classification falls back to rules.
type LLMClassifyError struct {
	Cause error
}

func (e *LLMClassifyError) Error() string {
	return "llm classify failed: " + e.Cause.Error()
}

func (e *LLMClassifyError) Unwrap() error {
	return e.Cause
}
