package collections

import (
	"github.com/lkmio/avformat/utils"
	"testing"
)

func TestReferenceCounter(t *testing.T) {
	r := ReferenceCounter[int]{}
	r.Refer()
	utils.Assert(r.UseCount() == 1)
	utils.Assert(r.Release())
}
