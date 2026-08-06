package classifier

import (
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/stretchr/testify/require"
)

func TestApplyLLMResult_DateParsing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		date      string
		wantYear  model.Year
		wantMonth time.Month
		wantDay   uint8
		wantNil   bool
	}{
		{
			name:      "full ISO date",
			date:      "2024-03-15",
			wantYear:  2024,
			wantMonth: time.March,
			wantDay:   15,
		},
		{
			name:     "year only",
			date:     "2024",
			wantYear: 2024,
			wantNil:  false,
		},
		{
			name:    "year too low",
			date:    "24",
			wantNil: true,
		},
		{
			name:    "empty string",
			date:    "",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var cl ClassificationResult
			applyLLMResult(&cl, &LLMResult{Date: tt.date})

			if tt.wantNil {
				require.True(t, cl.Date.IsNil(), "expected date to be nil, got %+v", cl.Date)
				return
			}

			require.Equal(t, tt.wantYear, cl.Date.Year)
			require.Equal(t, tt.wantMonth, cl.Date.Month)
			require.Equal(t, tt.wantDay, cl.Date.Day)
		})
	}
}
