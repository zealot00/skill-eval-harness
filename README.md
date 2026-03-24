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

**Q: 这 README 为什么要写这些？**

A: 因为有时候需要有人替打工人说说话。

---

## 文档

- [设计架构](/Users/zealot/Code/skill-eval-harness/docs/architecture.md) — 详细设计思路
- [CLI 说明](/Users/zealot/Code/skill-eval-harness/docs/cli.md) — 全量命令和参数
- [API 规范](/Users/zealot/Code/skill-eval-harness/docs/api.md) — REST API 定义

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
