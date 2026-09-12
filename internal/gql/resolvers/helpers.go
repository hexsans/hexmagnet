package resolvers

import (
	"github.com/hexsans/hexmagnet/internal/gql/gqlmodel/gen"
	"github.com/hexsans/hexmagnet/internal/processor"
	"github.com/hexsans/hexmagnet/internal/processor/enrich/indexer"
)

func ptr(s string) *string { return &s }

func nilStr(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}

func reindexProgressGQL(status indexer.ReindexStatus) gen.ReindexProgress {
	return gen.ReindexProgress{
		Total:         status.Total,
		Indexed:       status.Indexed,
		Done:          status.Done,
		Running:       status.Running,
		Resumable:     status.Resumable,
		ConfigChanged: status.ConfigChanged,
		Error:         nilStr(status.Error),
	}
}

func reclassifyProgressGQL(status processor.ReclassifyStatus) gen.ReclassifyProgress {
	return gen.ReclassifyProgress{
		Total:         status.Total,
		Processed:     status.Processed,
		Done:          status.Done,
		Running:       status.Running,
		Resumable:     status.Resumable,
		ConfigChanged: status.ConfigChanged,
		Error:         nilStr(status.Error),
	}
}
