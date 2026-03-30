# cronix

一款面向 Linux 的定时任务管理工具。cronix 封装系统 crontab，提供友好的 CLI 与交互式 TUI，让你无需手动编辑 crontab 文件即可管理所有计划任务。它将任务定义持久化到 `~/.config/cronix/tasks.json`，为每个任务生成独立的 wrapper 脚本，并按任务名 + 日期记录结构化执行日志。

特别适合以下场景：
- 深度学习模型的定时训练（支持任务级环境变量、上次未完成时自动跳过）
- 服务器定期备份、数据同步
- 开机后只执行一次的初始化脚本（`--once`）

---

## 目录

- [安装](#安装)
- [快速开始](#快速开始)
- [CLI 子命令详解](#cli-子命令详解)
  - [init](#cronix-init)
  - [add](#cronix-add)
  - [list](#cronix-list)
  - [enable](#cronix-enable)
  - [disable](#cronix-disable)
  - [remove](#cronix-remove)
  - [run](#cronix-run)
  - [logs](#cronix-logs)
  - [tui](#cronix-tui)
- [TUI 界面](#tui-界面)
- [日志说明](#日志说明)
- [配置文件](#配置文件)
- [Wrapper 脚本机制](#wrapper-脚本机制)
- [项目结构](#项目结构)
- [开发与构建](#开发与构建)
- [License](#license)

---

## 安装

### 一键安装（Linux amd64 / arm64）

```bash
curl -fsSL https://raw.githubusercontent.com/QMZCLL/cronix/main/install.sh | bash
```

脚本会自动检测当前系统架构（x86_64 或 aarch64），从 GitHub Releases 下载对应二进制，安装到 `/usr/local/bin/cronix`，并验证安装是否成功。

**前置要求：** `curl`、`bash`（绝大多数 Linux 发行版默认已有）

### 从源码构建

**前置要求：** Go 1.21+

```bash
git clone https://github.com/QMZCLL/cronix.git
cd cronix
make build
# 二进制输出到 dist/cronix
```

交叉编译 Linux 目标：

```bash
make build-linux-amd64   # 输出 dist/cronix-linux-amd64
make build-linux-arm64   # 输出 dist/cronix-linux-arm64
```

---

## 快速开始

```bash
# 1. 初始化 cronix（创建配置目录，注入 crontab 管理块）
cronix init

# 2. 添加一个每天凌晨 2 点执行的任务
cronix add --name nightly-backup \
  --cron "0 2 * * *" \
  --cmd "/usr/local/bin/backup.sh" \
  --desc "夜间备份"

# 3. 查看所有任务
cronix list

# 4. 查看今日日志
cronix logs nightly-backup

# 5. 立即手动触发执行
cronix run nightly-backup

# 6. 打开 TUI 界面
cronix tui
```

---

## CLI 子命令详解

### `cronix init`

初始化 cronix 配置目录（`~/.config/cronix/`），若 `tasks.json` 不存在则创建，并向用户 crontab 注入 cronix 管理块。可以安全地重复执行。

```bash
cronix init
```

**执行后效果：**
- 创建 `~/.config/cronix/tasks.json`（若不存在）
- 在用户 crontab 中插入：
  ```
  # cronix-managed-start
  # cronix-managed-end
  ```

---

### `cronix add`

添加一个新的计划任务，写入配置文件，生成 wrapper 脚本，并注册到系统 crontab。

```bash
cronix add --name <名称> --cron <cron表达式> --cmd <命令> [其他选项]
cronix add --name <名称> --once --cmd <命令> [其他选项]
```

| 参数 | 是否必填 | 说明 |
|------|----------|------|
| `--name` | 是 | 任务唯一名称，用作标识符和日志目录名，只能包含字母、数字、连字符 |
| `--cron` | 与 `--once` 二选一 | 标准 5 字段 cron 表达式，如 `"0 2 * * *"` |
| `--once` | 与 `--cron` 二选一 | 任务只运行一次，成功后自动禁用；不指定 `--cron` 时默认 `@reboot`（开机运行一次） |
| `--cmd` | 是 | 要执行的 shell 命令 |
| `--desc` | 否 | 任务描述，仅作备注 |
| `--env KEY=VALUE` | 否 | 任务级环境变量，可重复使用多次 |

**示例：**

```bash
# 每 5 分钟 ping 一次，设置代理环境变量
cronix add --name ping-check \
  --cron "*/5 * * * *" \
  --cmd "ping -c1 8.8.8.8" \
  --env HTTP_PROXY=http://proxy:3128

# 深度学习训练任务，每天凌晨 3 点执行，传入 GPU 配置
cronix add --name dl-train \
  --cron "0 3 * * *" \
  --cmd "python /workspace/train.py --epochs 100" \
  --env CUDA_VISIBLE_DEVICES=0 \
  --env PYTHONPATH=/workspace \
  --desc "每日模型训练"

# 开机后只执行一次的初始化脚本
cronix add --name setup-env \
  --once \
  --cmd "/opt/scripts/setup.sh" \
  --desc "环境初始化（仅运行一次）"

# 指定时间只运行一次（如某次临时数据迁移）
cronix add --name migrate-data \
  --cron "0 4 15 3 *" \
  --once \
  --cmd "./migrate.sh" \
  --desc "3月15日凌晨4点数据迁移，成功后自动禁用"
```

**关于 `--once` 机制：**
- 任务成功（exit code = 0）后，wrapper 脚本自动调用 `cronix disable <name>`
- 若任务失败（exit code ≠ 0），任务保持启用状态，下次 cron 周期仍会触发，直到成功为止
- `cronix list` 中该任务状态显示为 `once`

---

### `cronix list`

以表格形式显示所有已注册任务。

```bash
cronix list [--json]
```

| 参数 | 说明 |
|------|------|
| `--json` | 以 JSON 格式输出，便于脚本处理 |

**输出示例：**

```
NAME            CRON          STATUS    COMMAND
nightly-backup  0 2 * * *     enabled   /usr/local/bin/backup.sh
ping-check      */5 * * * *   enabled   ping -c1 8.8.8.8
setup-env       @reboot       once      /opt/scripts/setup.sh
dl-train        0 3 * * *     disabled  python /workspace/train.py...
```

状态说明：
- `enabled` — 已启用，cron 周期正常触发
- `disabled` — 已禁用，不在 crontab 中
- `once` — `--once` 任务，成功后自动变为 `disabled`

---

### `cronix enable`

启用一个已禁用的任务，将其重新注册到系统 crontab。

```bash
cronix enable <任务名>
```

```bash
cronix enable nightly-backup
```

---

### `cronix disable`

禁用一个任务，从系统 crontab 移除，但保留任务定义（不删除配置）。

```bash
cronix disable <任务名>
```

```bash
cronix disable dl-train
```

---

### `cronix remove`

永久删除一个任务：从 crontab 移除、删除 wrapper 脚本、从 `tasks.json` 删除记录。执行前会提示确认。

```bash
cronix remove <任务名>
```

```bash
cronix remove old-task
# Remove task "old-task"? [y/N] y
# ✓ Task "old-task" removed
```

---

### `cronix run`

立即手动执行某个任务（不受 cron 调度，直接运行）。输出实时流式打印到终端，同时写入当日日志文件。

```bash
cronix run <任务名>
```

```bash
cronix run nightly-backup
# 实时输出任务 stdout/stderr
# ✓ Exit: 0 | Duration: 1.23s | Log: ~/cronix-logs/nightly-backup/2026-03-31.log
```

**注意：** `cronix run` 绕过 lockfile 机制，即使 cron 正在运行同一任务也会执行。

---

### `cronix logs`

查看某个任务的执行日志。

```bash
cronix logs <任务名> [--date YYYY-MM-DD] [--tail N]
```

| 参数 | 说明 |
|------|------|
| `--date` | 查看指定日期的日志，格式 `YYYY-MM-DD`，默认今日 |
| `--tail N` | 只显示最后 N 行 |

**示例：**

```bash
# 查看今日日志
cronix logs nightly-backup

# 查看指定日期
cronix logs nightly-backup --date 2026-03-15

# 只看最后 20 行
cronix logs nightly-backup --tail 20
```

---

### `cronix tui`

打开交互式 TUI（终端用户界面），在终端中可视化管理所有任务。

```bash
cronix tui
```

详见 [TUI 界面](#tui-界面) 章节。

---

## TUI 界面

`cronix tui` 启动后进入交互式任务管理界面，包含以下功能：

### 任务列表页

显示所有任务，列包括 `NAME | CRON | STATUS | COMMAND`。

```
     NAME            CRON          STATUS    COMMAND
  ›  nightly-backup  0 2 * * *     enabled   /usr/local/bin/backup.sh
     dl-train        0 3 * * *     disabled  python /workspace/train.py...
     setup-env       @reboot       once      /opt/scripts/setup.sh

  [e]nable [d]isable [r]un [l]ogs [x]delete [q]uit
```

**颜色说明：**
- 绿色 — `enabled` 状态
- 灰色 — `disabled` / `once` 状态
- 选中行加粗高亮

**快捷键：**

| 按键 | 功能 |
|------|------|
| `↑` / `k` | 向上移动光标 |
| `↓` / `j` | 向下移动光标 |
| `e` | 启用选中任务 |
| `d` | 禁用选中任务 |
| `r` | 立即运行选中任务，完成后状态栏显示退出码和耗时 |
| `l` | 进入日志查看页 |
| `x` | 删除选中任务（弹出确认框，按 `y` 确认，其他键取消） |
| `q` / `Ctrl+C` | 退出 TUI |

### 日志查看页

按 `l` 进入，显示选中任务的当日执行日志。

```
  Logs: nightly-backup | 2026-03-31

  === Run at 2026-03-31T02:00:01Z ===
  Backup started...
  [OK] /data backed up to /backup/2026-03-31.tar.gz
  === Exit: 0 ===

  [b/esc] back  [p] prev day  [↑/↓] scroll  [pgup/pgdn] page
```

**快捷键：**

| 按键 | 功能 |
|------|------|
| `↑` / `↓` | 逐行滚动 |
| `PgUp` / `PgDn` | 翻页 |
| `p` | 切换到前一天的日志（若存在） |
| `b` / `Esc` | 返回任务列表 |

---

## 日志说明

每个任务的执行日志按日期存储：

```
~/cronix-logs/
  <任务名>/
    YYYY-MM-DD.log
    YYYY-MM-DD.log
    ...
```

**默认路径：** `~/cronix-logs/`

可通过修改 `~/.config/cronix/tasks.json` 中的 `log_dir` 字段自定义日志根目录。

**日志格式：**

```
=== Run at 2026-03-31T02:00:01Z ===
<任务标准输出和标准错误>
=== Exit: 0 | Duration: 1.23s ===
```

每次执行都会在日志文件末尾追加，同一天多次运行的记录依次排列。

---

## 配置文件

cronix 配置文件位于 `~/.config/cronix/tasks.json`，格式如下：

```json
{
  "tasks": [
    {
      "name": "nightly-backup",
      "command": "/usr/local/bin/backup.sh",
      "cron_expr": "0 2 * * *",
      "enabled": true,
      "run_once": false,
      "envs": {
        "BACKUP_DST": "/mnt/nas/backup"
      },
      "created_at": "2026-03-31T00:00:00Z",
      "last_run_at": "2026-03-31T02:00:01Z",
      "description": "夜间备份"
    }
  ],
  "log_dir": ""
}
```

**字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 任务唯一名称 |
| `command` | string | 执行的 shell 命令 |
| `cron_expr` | string | cron 表达式，支持 `@reboot` |
| `enabled` | bool | 是否在 crontab 中生效 |
| `run_once` | bool | 是否为一次性任务 |
| `envs` | object | 任务级环境变量键值对 |
| `created_at` | string | 任务创建时间（ISO 8601） |
| `last_run_at` | string/null | 最近一次运行时间 |
| `description` | string | 任务描述（可选） |
| `log_dir` | string | 自定义日志根目录（空字符串使用默认值 `~/cronix-logs`） |

> 该文件由 cronix 自动维护，通常不需要手动编辑。如有必要手动修改，请确保 JSON 格式合法。

---

## Wrapper 脚本机制

cronix 为每个任务生成一个独立的 bash wrapper 脚本，存储在 `~/.config/cronix/wrappers/<任务名>.sh`。wrapper 脚本负责：

1. **设置环境变量** — 注入任务级 `envs` 配置
2. **重定向日志** — 将 stdout/stderr 写入 `~/cronix-logs/<任务名>/YYYY-MM-DD.log`
3. **Lockfile 防重叠** — 若上次运行尚未完成，跳过本次执行并记录日志，不强杀上次进程
4. **一次性任务自禁用** — `run_once=true` 且 exit code=0 时自动调用 `cronix disable <name>`

**Lockfile 路径：** `/tmp/cronix-<任务名>.lock`

crontab 中注册的是 wrapper 脚本路径，而非直接命令，确保日志和 lockfile 逻辑始终生效。

**cronix 管理的 crontab 块：**

```
# cronix-managed-start
*/5 * * * * /home/user/.config/cronix/wrappers/ping-check.sh
0 2 * * * /home/user/.config/cronix/wrappers/nightly-backup.sh
# cronix-managed-end
```

该块以外的 crontab 内容永远不会被 cronix 修改。

---

## 项目结构

```
cronix/
  cmd/cronix/           # CLI 入口与各子命令实现
    main.go             # 程序入口
    root.go             # 根命令工厂
    init.go             # cronix init
    add.go              # cronix add
    list.go             # cronix list
    enable.go           # cronix enable
    disable.go          # cronix disable
    remove.go           # cronix remove
    run.go              # cronix run
    logs.go             # cronix logs
    tui.go              # cronix tui（桥接到 internal/tui）
  internal/
    config/             # 配置文件读写（tasks.json）
    cron/               # crontab 注入/移除，wrapper 脚本生成
    logger/             # 日志路径生成，文件读写
    task/               # 任务 CRUD，cron 表达式校验
    tui/                # Bubble Tea TUI（任务列表、日志页、操作快捷键）
  .github/workflows/
    release.yml         # tag 推送自动构建并发布二进制
  Makefile              # build / build-linux-amd64 / build-linux-arm64
  install.sh            # 一键安装脚本
  README.md
  go.mod
```

---

## 开发与构建

```bash
# 克隆仓库
git clone https://github.com/QMZCLL/cronix.git
cd cronix

# 安装依赖
go mod download

# 运行测试
go test ./...

# 带覆盖率的测试
go test ./... -coverprofile=cover.out
go tool cover -func=cover.out

# 本地构建（当前平台）
make build              # 输出 dist/cronix

# 交叉编译
make build-linux-amd64  # 输出 dist/cronix-linux-amd64
make build-linux-arm64  # 输出 dist/cronix-linux-arm64
```

**发布新版本：**

推送带 `v` 前缀的 tag 即可触发 GitHub Actions 自动构建并创建 Release：

```bash
git tag v0.2.0
git push origin v0.2.0
```

---

## License

MIT
