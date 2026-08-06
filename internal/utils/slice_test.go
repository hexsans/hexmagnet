package utils_test

import (
	"strconv"
	"testing"

	"github.com/hexsans/hexmagnet/internal/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSlice(t *testing.T) {
	t.Parallel()

	RegisterFailHandler(Fail)
	RunSpecs(t, "Slice Suite")
}

var _ = Describe("Slice Utils", func() {
	Describe("Map", func() {
		It("returns empty slice for an empty input", func() {
			mapFunc := func(v int) string { return strconv.Itoa(v * 2) }
			result := utils.Map([]int{}, mapFunc)
			Expect(result).To(BeEmpty())
		})

		It("returns a new slice with elements mapped", func() {
			mapFunc := func(v int) string { return strconv.Itoa(v * 2) }
			result := utils.Map([]int{1, 2, 3, 4}, mapFunc)
			Expect(result).To(ConsistOf("2", "4", "6", "8"))
		})
	})
})
