package utils

import "fmt"

type AVMediaType int

const (
	AVMediaTypeUnknown    = AVMediaType(-1) ///< Usually treated as AVMediaTypeData
	AVMediaTypeVideo      = AVMediaType(0)
	AVMediaTypeAudio      = AVMediaType(1)
	AVMediaTypeData       = AVMediaType(2) ///< Opaque data information usually continuous
	AVMediaTypeSubtitle   = AVMediaType(3)
	AVMediaTypeAttachment = AVMediaType(4) ///< Opaque data information usually sparse
	AVMediaTypeN          = AVMediaType(5)
)

func (a AVMediaType) String() string {
	if AVMediaTypeUnknown == a {
		return "unknown"
	}

	if AVMediaTypeVideo == a {
		return "video"
	}

	if AVMediaTypeAudio == a {
		return "audio"
	}

	if AVMediaTypeData == a {
		return "data"
	}

	if AVMediaTypeSubtitle == a {
		return "subtitle"
	}

	if AVMediaTypeAttachment == a {
		return "attachment"
	}

	if AVMediaTypeN == a {
		return "n"
	}

	panic(fmt.Sprintf("bad type:%d", a))
}
