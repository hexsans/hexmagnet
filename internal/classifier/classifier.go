package classifier

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hexsans/hexmagnet/internal/model"
)

type Compiler interface {
	Compile(source Source) (Runner, error)
}

type Runner interface {
	Run(ctx context.Context, workflow string, flags Flags, t model.Torrent) (ClassificationResult, error)
}

type compiler struct {
	options          []compilerOption
	dependencies     dependencies
	defaultLLMConfig LLMConfig
}

type compilerContext struct {
	features
	exprEnv       ExprEnv
	source        any
	path          []string
	workflowNames map[string]struct{}
}

type compilerOption func(Source, *compilerContext) error

type executionContext struct {
	context.Context
	dependencies
	flags       map[string]any
	workflows   map[string]action
	torrent     model.Torrent
	torrentExpr Torrent
	result      ClassificationResult
	resultExpr  Classification
	exprEnv     ExprEnv
}

func (c executionContext) withResult(result ClassificationResult) executionContext {
	c.result = result
	c.resultExpr = NewClassificationFromResult(result)

	return c
}

func (c compilerContext) child(pathPart string, source any) compilerContext {
	c.source = source
	newPath := make([]string, len(c.path), len(c.path)+1)
	copy(newPath, c.path)
	newPath = append(newPath, pathPart)
	c.path = newPath

	return c
}

func (c compilerContext) error(cause error) error {
	if asCompilerError(cause) != nil {
		return cause
	}

	return compilerError{c.path, cause}
}

func (c compilerContext) fatal(cause error) error {
	if asFatalCompilerError(cause) != nil {
		return cause
	}

	cErr := asCompilerError(cause)
	if cErr != nil {
		return fatalCompilerError{compilerError: *cErr}
	}

	return fatalCompilerError{compilerError{c.path, cause}}
}

func (c compiler) Compile(source Source) (Runner, error) {
	ctx := &compilerContext{
		source:        source,
		workflowNames: source.workflowNames(),
	}

	source, sourceErr := decode[Source](*ctx)
	if sourceErr != nil {
		return nil, ctx.fatal(sourceErr)
	}

	for _, opt := range c.options {
		if err := opt(source, ctx); err != nil {
			return nil, ctx.fatal(err)
		}
	}

	workflowsCtx := ctx.child("workflows", source.Workflows)
	workflows := make(map[string]action)

	for name, src := range source.Workflows {
		a, err := ctx.compileAction(workflowsCtx.child(name, src))
		if err != nil {
			return nil, ctx.fatal(err)
		}

		workflows[name] = a
	}

	llmCfg := c.defaultLLMConfig
	c.dependencies.llmClient = nil

	c.dependencies.llmEnabled = false
	if llmCfg.IsActive() {
		c.dependencies.llmClient = NewClient(llmCfg, c.dependencies.logger)
		c.dependencies.llmEnabled = true
	}

	if c.dependencies.logger != nil {
		if llmCfg.IsActive() {
			c.dependencies.logger.Infow("llm classifier configured",
				"endpoint", llmCfg.Endpoint,
				"model", llmCfg.Model,
			)
		} else {
			c.dependencies.logger.Info("llm classifier not configured, using rule-based classification only")
		}
	}

	defaultFlags := make(Flags, len(source.FlagDefinitions))

	for k, def := range source.FlagDefinitions {
		rawVal, ok := source.Flags[k]
		if !ok {
			return nil, ctx.fatal(fmt.Errorf("missing value for flag '%q'", k))
		}

		val, err := def.validate(rawVal)
		if err != nil {
			return nil, ctx.fatal(fmt.Errorf("invalid value for flag '%s': %w", k, err))
		}

		defaultFlags[k] = val
	}

	return runner{
		dependencies:    c.dependencies,
		flagDefinitions: source.FlagDefinitions,
		defaultFlags:    defaultFlags,
		exprEnv:         ctx.exprEnv,
		workflows:       workflows,
	}, nil
}

func decodeTo[T any](ctx compilerContext, target *T) error {
	decoder, decoderErr := newDecoder(target)
	if decoderErr != nil {
		return ctx.error(decoderErr)
	}

	return decoder.Decode(ctx.source)
}

func decode[T any](ctx compilerContext) (T, error) {
	var target T

	err := decodeTo(ctx, &target)

	return target, err
}

type compilerError struct {
	path  []string
	cause error
}

func (e compilerError) Error() string {
	return fmt.Sprintf("compiler error at path '%s': %s", strings.Join(e.path, "."), e.cause)
}

func (e compilerError) Unwrap() error {
	return e.cause
}

func asCompilerError(err error) *compilerError {
	if ue, ok := errors.AsType[*compilerError](err); ok {
		return ue
	}

	return nil
}

type fatalCompilerError struct {
	compilerError
}

func (e fatalCompilerError) Unwrap() error {
	return e.compilerError
}

func asFatalCompilerError(err error) *fatalCompilerError {
	if ue, ok := errors.AsType[*fatalCompilerError](err); ok {
		return ue
	}

	return nil
}

func numericPathPart(num int) string {
	return fmt.Sprintf("[%d]", num)
}
