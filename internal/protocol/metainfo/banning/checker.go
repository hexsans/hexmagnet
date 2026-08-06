package banning

import (
	"errors"

	"github.com/hexsans/hexmagnet/internal/protocol/metainfo"
	"github.com/hexsans/hexmagnet/internal/utils"
	"go.uber.org/fx"
)

type Params struct {
	fx.In
	Checkers []Checker `group:"metainfo_banning_checkers"`
}

type Result struct {
	fx.Out
	Checker Checker `name:"metainfo_banning_checker"`
}

func New(p Params) Result {
	checkers := p.Checkers
	checkers = append(
		checkers,
		nameLengthChecker{min: 8},
		sizeChecker{min: 1024},
		utf8Checker{},
		contentChecker{},
	)

	return Result{
		Checker: combinedChecker{
			checkers: checkers,
		},
	}
}

type Checker interface {
	Check(metainfo.Info) error
}

type combinedChecker struct {
	checkers []Checker
}

func (c combinedChecker) Check(info metainfo.Info) error {
	return errors.Join(utils.Map(c.checkers, func(checker Checker) error {
		return checker.Check(info)
	})...)
}
