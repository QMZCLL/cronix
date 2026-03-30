
## Task 15: TUI 操作集成

- task.Enable/Disable 接收 []Task (非指针)，但内部 FindByName 返回指针修改元素，所以能就地修改 slice 元素
- TUI 操作后需重新 config.Load() 再操作，因为 m.tasks 是内存快照，不反映磁盘状态
- 状态栏消息用 tea.Tick(3s) 返回 clearStatusMsg 自动清除
- 删除确认用 m.confirming bool 字段 + status 字符串显示 prompt，updateConfirm 处理 y/其他
- 测试中用 t.Setenv("CRONIX_CONFIG_DIR", t.TempDir()) 隔离真实 crontab/config
- run 操作用异步 Cmd 返回 runResult msg，避免阻塞 TUI 渲染
- renderListView 读取 m.status/m.statusErr 在 helpText 下方渲染状态行
- list.go 新增 statusOk/statusErr lipgloss.Style 用于着色状态消息
