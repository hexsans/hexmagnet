package classifier

import (
	"context"
	"fmt"

	"github.com/hexsans/hexmagnet/internal/model"
)

type runner struct {
	dependencies
	flagDefinitions
	defaultFlags Flags
	exprEnv      ExprEnv
	workflows    map[string]action
}

func (r runner) Run(ctx context.Context, workflow string, flags Flags, t model.Torrent) (ClassificationResult, error) {
	w, ok := r.workflows[workflow]
	if !ok {
		return ClassificationResult{}, fmt.Errorf("workflow not found: %s", workflow)
	}

	f := make(map[string]any, len(r.flagDefinitions))

	for k, d := range r.flagDefinitions {
		if runtimeRawVal, ok := flags[k]; ok {
			rcf, err := d.validate(runtimeRawVal)
			if err != nil {
				return ClassificationResult{}, fmt.Errorf(
					"invalid value for runtime flag '%s': %w",
					k,
					err,
				)
			}

			f[k] = rcf
		} else {
			f[k] = r.defaultFlags[k]
		}
	}

	cl := ClassificationResult{}

	exCtx := executionContext{
		Context:      ctx,
		dependencies: r.dependencies,
		workflows:    r.workflows,
		flags:        f,
		torrent:      t,
		torrentExpr:  NewTorrentFromModel(t),
		result:       cl,
		resultExpr:   NewClassificationFromResult(cl),
		exprEnv:      r.exprEnv,
	}

	return w.run(exCtx)
}
