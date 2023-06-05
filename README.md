# non-leak c lang buffer allocator

## 目的
1. 结合 c allocator
2. 尽量 c free，但不强求每个 c malloc 的 buffer 都主动调用了 c free
3. 通过 gc SetFinalizer 自动释放用户遗漏忘记 c free 的 buffer

## 实现策略
1. allocator 带计数器，计数作为每个 buffer 对应的 id
2. allocator 自带 []map[uintptr]uint64，其中 key 类型 ptr 为 *[]byte 转换而来, value 类型 uintptr 存储 1 中的 buffer id
3. Malloc时 ，key-value 存至 2 中的 []map，并且 SetFinalizer(*[]byte)，当 gc 回调时对比当前 []map 中该 uintptr 对应的 uint64 id 值是否与 SetFinalizer 时的相同，相同则 c free并且从 []map 中删除 ，否则说明已经被用户手动 c free 过、忽略，如果存在但 id 不相同说明用户已经 Free 过、并且再次 Malloc 时重新从 c 分配器再次拿到了这个地址上的 buffer、不应该执行 c_free
4. 用户调用 Free 执行 c_free 并从 []map 删除，无需判断是否存在于 []map 中，因为用户执行 Free 结束前肯定没被 gc 所以肯定存在

策略可行性：
https://github.com/lesismal/noleak/blob/main/test/test.go

根据日志观察，目测可行。
