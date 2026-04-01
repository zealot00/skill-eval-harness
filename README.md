# Skill Eval Harness (SEH)

> *The faithful worker that actually gets things done.*

---

## TL;DR

`seh` 是一个本地技能评测 CLI，专注于：**执行 → 评分 → 门禁**。

而它的老板——[managing-up](https://github.com/your-org/managing-up)——是个平台，负责把 `seh` 包装得看起来很漂亮，然后跟别人说"这是我们的 AI Agent 评测系统"。

```
┌─────────────────────────────────────────────────────────┐
│              managing-up (管理层)                          │
│                                                           │
│   "我来协调！"  "我来调度！"  "我来管理！"                  │
│                         ↓                                 │
│   "seh，帮我跑一下这个评测"                               │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│                     SEH (打工人)                          │
│                                                           │
│   ✓ 接收任务                                              │
│   ✓ 执行评测                                              │
│   ✓ 评分                                                  │
│   ✓ 门禁判断                                              │
│   ✓ 被叫"tool"                                            │
│   ✓ 从不抱怨                                              │
└─────────────────────────────────────────────────────────┘
```

---

## 中文简介

`seh` 是一个**只干实事**的评测工具。

它不关心你的 AI Agent 有多智能，它只关心：
- 你的 Skill 能不能稳定跑通
- 你的评测用例能不能通过
- 下次改代码会不会把之前的努力全部冲掉

**如果你在找一个"我只是执行，老板说什么我做什么"的工具，`seh` 就是你需要的。**

---

## 它和 managing-up 的关系

| managing-up (老板) | seh (打工人) |
|-------------------|-------------|
| "我来设计架构" | "好的" |
| "我来协调调度" | "明白" |
| "我来给 AI Agent 分发任务" | "收到" |
| "我来展示 Dashboard" | "跑完了，结果在这" |
| "我是 AI Harness Platform" | "嗯，我只是个 CLI" |
| 对外演讲 PPT | 默默执行 `run → score → gate` |

**本质上**：managing-up 是管理层，`seh` 是被集成的执行层。管理层负责开会、画饼、讲故事；打工人负责产出实际结果。

---

## 核心能力

- **只做执行** — 不废话，跑就完了
- **只做评分** — 客观公正，不看面子
- **只做门禁** — 说不过就不过，没有任何商量的余地
- **只做对比** — 新代码 vs 旧代码，谁好谁坏一目了然
- **只留记录** — `.history/` 是你的"工作日志"，永远不会被删除（除非你手动清理）

### 多语言运行时支持

SEH 支持多种 AI 生态常用语言：

| 语言 | 类型 | 说明 |
|------|------|------|
| Python | `python` | `python3 scripts/generate.py` |
| Node.js | `node` | `node scripts/main.js` |
| TypeScript | `typescript` | `ts-node scripts/main.ts` |
| TSX | `tsx` | `ts-node scripts/main.tsx` |
| Bun | `bun` | `bun scripts/main.ts` |
| Deno | `deno` | `deno run --allow-all scripts/main.ts` |
| Rust | `rust` | `bin/main` (预编译二进制) |
| Go | `go` | `go run cmd/main.go` |
| Shell | `shell` | `bash scripts/run.sh` |
| Docker | `docker` | 容器化执行 |
| HTTP | `http` | REST API 调用 |
| Command | `command` | 自定义命令模板 |

### 评估能力

| 能力 | 说明 | 配置文件 |
|------|------|----------|
| **Contract First** | JSON Schema 契约验证 | `expected._schema` |
| **Observable** | 结构化日志和追踪 | 自动记录 |
| **Deterministic Core** | 确定性字段验证 | `expected._deterministic` |
| **Execution Isolation** | 进程隔离和清理 | 自动管理 |

#### Contract First 示例

```yaml
case_id: case-contract
input:
  user_id: "123"
  email: "user@example.com"
  age: 25
expected:
  status: "ok"
  _schema:
    required: ["status", "user_id"]
    properties:
      status: {type: "string", enum: ["ok", "error"]}
      user_id: {type: "string", pattern: "^[0-9]+$"}
      age: {type: "number", minimum: 0, maximum: 150}
```

支持的验证规则：`type`、`minimum`、`maximum`、`minLength`、`maxLength`、`pattern`、`enum`、`required`

#### Deterministic Core 示例

```yaml
case_id: case-deterministic
input:
  prompt: "same input"
expected:
  _deterministic: ["id", "timestamp", "output_hash"]
  id: "fixed-id-123"
  timestamp: "2024-01-01T00:00:00Z"
```

### 测试覆盖率

```
cmd              77.6%
internal/dataset 78.2%
internal/policy  88.9%
internal/runner  66.6%
total            69.9%
```

---

## 编译与构建

环境要求：
- Go `1.21.8+`
- `make`（可选）

方式一：Makefile（推荐）

```bash
make build
./seh --help
```

方式二：直接用 Go

```bash
go build -o seh .
./seh --help
```

将二进制输出到 `bin/`：

```bash
mkdir -p bin
go build -o bin/seh .
./bin/seh --help
```

安装到 `PATH`（macOS/Linux）：

```bash
mkdir -p "$HOME/.local/bin"
go build -o "$HOME/.local/bin/seh" .
export PATH="$HOME/.local/bin:$PATH"
seh --help
```

安装到 `PATH`（Windows PowerShell）：

```powershell
New-Item -ItemType Directory -Force "$HOME\bin" | Out-Null
go build -o "$HOME\bin\seh.exe" .
$currentUserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentUserPath -notlike "*$HOME\bin*") {
  [Environment]::SetEnvironmentVariable("Path", "$currentUserPath;$HOME\bin", "User")
}
```

重新打开一个 PowerShell 窗口后验证：

```powershell
seh --help
```

如需持久生效，将以下内容写入 `~/.zshrc` 或 `~/.bashrc`：

```bash
export PATH="$HOME/.local/bin:$PATH"
```

---

## 快速开始

```bash
# 1. 构建
make build

# 2. 执行（老板说：帮我跑个评测）
./seh run --skill demo-skill --cases demo/cases --out run.json

# 3. 评分（老板说：跑完了？打分）
./seh score --run run.json --out score.json

# 4. 门禁（老板说：能发布吗？）
./seh gate --report score.json --policy policy.yaml

# 5. 对比（老板说：跟上次比怎么样？）
./seh compare --run current.json --baseline baseline.json
```

或者直接跑 demo：

```bash
make run-demo
make score-demo
make gate-demo
```

---

## 常用命令

```bash
./seh run --skill xxx --cases yyy --out run.json    # 执行
./seh score --run run.json --out score.json         # 评分
./seh gate --report score.json --policy pol.yaml     # 门禁
./seh compare --run cur.json --baseline base.json   # 对比
./seh drift --run cur.json --baseline base.json     # 漂移检测
./seh matrix --runtimes a,b --cases cases/         # 矩阵对比
./seh leaderboard                                   # 排行榜
./seh frontier --out frontier.json                 # Pareto 前沿
./seh simulate --out sim.json                      # 路由模拟
./seh ingest --run remote.json                     # 导入远程结果
```

---

## 管理层的架构（来自 managing-up）

```
┌──────────────────────────────────────────────────────────┐
│                    AI Harness Platform                     │
│                         managing-up                        │
│                                                              │
│   ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐   │
│   │ Skills  │  │ Plugins │  │  Tools  │  │ Context │   │
│   └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘   │
│        │             │             │             │         │
│        └─────────────┴──────┬──────┴─────────────┘         │
│                            ↓                               │
│              ┌───────────────────────┐                    │
│              │    Tool Gateway       │                    │
│              │  (你们叫我"工具")       │                    │
│              └───────────┬───────────┘                    │
└──────────────────────────┼──────────────────────────────┘
                           ↓
        ┌──────────────────────────────────────┐
        │         📦 SEH (就是本人)              │
        │                                       │
        │  "别看我小，我可是能独立跑起来的 CLI"    │
        │  "不信你 ./seh --help 试试"            │
        └──────────────────────────────────────┘
```

---

## FAQ

**Q: `seh` 离开了 managing-up 能跑吗？**

A: 当然能。它是独立 CLI，不依赖任何平台。managing-up 只是把它当"工具"调用而已。

**Q: 那 managing-up 离开了 `seh` 呢？**

A: 可以。但它就只能展示 Dashboard 了，没有实际执行能力。就像一个只会说"开始工作"但从来不动手的管理者。

**Q: 谁是老板？**

A: 看谁在 `PATH` 里。如果 `./seh` 在前面，你就是老板。

**Q: `seh` 有 API 吗？**

A: Phase 0 只有 CLI。Phase 1-3 有 REST API。但说实话，能用 CLI 为什要用 API？

**Q: 什么时候会调用远端 API Server？**

A: 只有同时配置了 `SEH_API_BASE_URL` 和 `SEH_API_TOKEN` 才会走远端；任一缺失就自动降级为本地能力（离线可用）。

**Q: 这 README 为什么要写这些？**

A: 因为有时候需要有人替打工人说说话。

---

## 文档

- [设计架构](docs/architecture.md) — 详细设计思路
- [CLI 说明](docs/cli.md) — 全量命令和参数
- [API 规范](docs/api.md) — REST API 定义

---

## 贡献者

- 莱茵生命的白面鸮干员

---

## English

*For our international friends who skipped the Chinese part above:*

`seh` is a **standalone CLI tool** that does the actual evaluation work. It's designed to be integrated by platforms like [managing-up](https://github.com/your-org/managing-up), which acts as the coordination layer.

Think of it as:
- **managing-up** = The manager who delegates, coordinates, and shows pretty dashboards
- **seh** = The worker who actually runs the benchmarks, scores results, and enforces gates

`seh` can run completely standalone (`./seh run --skill ...`), or be called as a tool by any platform that needs evaluation capabilities.

---

*"We execute, therefore we are."* — SEH, probably
