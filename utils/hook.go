package utils

type HookState byte

const (
	// HookStateOK 没有开启Hook回调/Hook响应成功(200应答)
	HookStateOK = HookState(0)
	// HookStateOccupy streamId 已经被其他推流占用
	HookStateOccupy = HookState(1)
	// HookStateFailure Hook响应失败(非200应答)
	HookStateFailure = HookState(2)
)
