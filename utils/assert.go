package utils

import (
	"fmt"
	"runtime"
)

func Assert(b bool) {
	if !b {
		// 只获取最近的一个调用栈帧
		ptrs := make([]uintptr, 1)
		// 跳过Assert和Callers函数
		callers := runtime.Callers(2, ptrs)
		frames := runtime.CallersFrames(ptrs[:callers])
		frame, _ := frames.Next()
		panic(fmt.Sprintf("Assertion failed, file %s, line: %d", frame.File, frame.Line))
	}
}
