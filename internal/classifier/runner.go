package classifier

import (
	"context"
	"fmt"

	"cel.dev/cel-go/common/types/ref"
	"github.com/hexsans/hexmagnet/internal/model"
)

type runner struct {
	dependencies
	flagDefinitions
	compiledFlags
	workflows map[string]action
}

func (r runner) Run(ctx context.Context, workflow string, flags Flags, t model.Torrent) (ClassificationResult, error) {
	w, ok := r.workflows[workflow]
	if !ok {
		return ClassificationResult{}, fmt.Errorf("workflow not found: %s", workflow)
	}

	cfs := make(map[string]ref.Val, len(r.flagDefinitions))

	for k, d := range r.flagDefinitions {
		if runtimeRawVal, ok := flags[k]; ok {
			rcf, err := d.celVal(runtimeRawVal)
			if err != nil {
				return ClassificationResult{}, fmt.Errorf(
					"invalid value for runtime flag '%s': %w",
					k,
					err,
				)
			}

			cfs[k] = rcf
		} else {
			cfs[k] = r.compiledFlags[k]
		}
	}

	cl := ClassificationResult{}

	exCtx := executionContext{
		Context:      ctx,
		dependencies: r.dependencies,
		workflows:    r.workflows,
		flags:        cfs,
		torrent:      t,
		torrentPb:    NewTorrentFromModel(t),
		result:       cl,
		resultPb:     NewClassificationFromResult(cl),
	}

	return w.run(exCtx)
}
