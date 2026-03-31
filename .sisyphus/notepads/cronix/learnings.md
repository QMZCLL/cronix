
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

## Task 16: TUI add form 与 once 语义对齐

- `internal/tui` 新增 add page 时，最稳妥的做法是继续沿用 list/logs 的 page 状态机，在 `Model` 上挂一个轻量 `addState`，不要把创建逻辑塞回 cobra
- 用 5 个独立字段拼 cron 字符串后，继续交给 `task.Add` 做统一校验，这样 TUI 不需要复制 cron 范围规则
- `RunOnce` 的真实语义应和 wrapper 保持一致: 仍按用户配置的 cron 触发，且只有 wrapper 记录到成功退出码后才自动 disable
- env 输入做成逗号分隔字符串时，抽一个 `task.ParseEnvAssignments` 供 CLI 和 TUI 共用，可以避免两套 KEY=VALUE 解析逻辑再次漂移

## Task 17: cronix TUI 视觉整理

- `internal/tui/styles.go` 适合作为 TUI 视觉 token 的最小落点，把页头、帮助块、状态块、空状态、表格单元格、表单标签集中起来，比在 `list.go`/`logs.go`/`add.go` 各自散落颜色更稳
- lipgloss 的 `Width()` 会把 padding/border 也算进 frame size，列表这种固定列宽布局里给单元格直接加 padding 很容易把长命令提前换行，测试会先炸出来
- 终端里“当前项更明显”最好同时用符号和字重/下划线，不要只靠颜色；这次列表选中态和表单焦点都保留了显式 `›` 标记，tmux/capture-pane 里也清楚
- 日志页适合保留原始内容区域，不要为了美观给 viewport 再套重边框；把层级主要放在页头、任务/日期元信息、帮助区和空状态上，更不容易影响宽高计算

## Task 18: NEXT 相对时间与 cron 守护进程提示

- `task.NextRun` 最稳的是只在一个函数里做时间分段格式化，并把“当前时间”抽成可替换变量，这样 CLI/TUI 共用输出且测试不依赖真实时钟
- 相对时间若要避免 flaky，分钟和小时都向上取整到分钟边界，比展示秒数更稳定；规则控制在 `in Xm` / `in Xh Ym` / `tomorrow HH:MM` / `MM-DD HH:MM` 就够用了
- Linux 下检测 cron 守护进程可以直接扫 `/proc/*/{comm,cmdline}` 查 `cron`/`crond`，不需要 `systemctl`、`service` 或任何会修改系统状态的命令
- 守护进程未运行更适合作为 warning 而不是错误：`init`、`list`、TUI 列表都能提示，但 `add`/`run` 不应被阻断
