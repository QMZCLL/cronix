
## Task 15: TUI 操作集成

- task.Enable/Disable 接收 []Task (非指针)，但内部 FindByName 返回指针修改元素，所以能就地修改 slice 元素
- TUI 操作后需重新 config.Load() 再操作，因为 m.tasks 是内存快照，不反映磁盘状态
- 状态栏消息用 tea.Tick(3s) 返回 clearStatusMsg 自动清除
- 删除确认用 m.confirming bool 字段 + status 字符串显示 prompt，updateConfirm 处理 y/其他
- 测试中用 t.Setenv("CRONIX_CONFIG_DIR", t.TempDir()) 隔离真实 crontab/config
- run 操作用异步 Cmd 返回 runResult msg，避免阻塞 TUI 渲染
- renderListView 读取 m.status/m.statusErr 在 helpText 下方渲染状态行
- list.go 新增 statusOk/statusErr lipgloss.Style 用于着色状态消息

## cronix add --once 实现 (2026-03-31)

### 修改点
- `internal/task/types.go`: Task struct 新增 `RunOnce bool \`json:"run_once"\``
- `internal/task/manager.go`: `ValidateCronExpr` 在字段验证前特判 `@reboot`，直接 return nil
- `internal/cron/wrapper.go`: `GenerateWrapper` 末尾，若 `t.RunOnce` 为 true，在 EXIT_CODE=0 时调用 `$CRONIX_BIN disable {name}`；用 `which cronix` 定位二进制
- `cmd/cronix/add.go`: 新增 `--once` bool flag；若设置则 `RunOnce=true`，若未同时给 `--cron` 则默认 `@reboot`
- `cmd/cronix/list.go`: status 列：Enabled+RunOnce 显示 `once`，只 Enabled 显示 `enabled`
- `README.md`: 更新 `cronix add` 说明，加入 `--once` flag 和示例

### 约定
- RunOnce 只在 EXIT_CODE=0 时 disable，避免失败自删
- disable 而非 remove，保留日志和配置
- `@reboot` 绕过 5 字段 cron 校验
- `go test ./...` 全部通过，`make build` 成功
