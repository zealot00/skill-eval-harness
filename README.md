# Skill Eval Harness

中英文文档：

- 中文与 English 简介都在本页
- 详细设计见 [docs/architecture.md](/Users/zealot/Code/skill-eval-harness/docs/architecture.md)
- 详细 CLI 说明见 [docs/cli.md](/Users/zealot/Code/skill-eval-harness/docs/cli.md)

## 中文简介

`skill-eval-harness` 是一个本地技能评测与分析平台，CLI 名称为 `seh`。它用于把“数据集执行、结果评分、历史沉淀、回归比较、门禁判断、分析报告”整合到一个统一工作流中。

核心能力：

- 数据集驱动评测：`cases/*.yaml` + `manifest.yaml`
- 运行时注册与执行：支持单运行时与多运行时矩阵
- 调度能力：并发、超时、重试、抽样、标签过滤、确定性 seed
- 评分与治理：score、gate、baseline、regression、strict mode
- 历史分析：leaderboard、simulate、frontier、stability variance
- 输出分析：drift、failure clustering、HTML report
- 历史管理：`.history/` 持久化与远程 run 导入

快速开始：

```bash
make build
./seh run --skill demo-skill --cases demo/cases --out demo/out/run.json
./seh score --run demo/out/run.json --out demo/out/score.json
./seh report --in demo/out/score.json --out demo/out/report.html
```

也可以直接运行 demo：

```bash
make run-demo
make score-demo
make gate-demo
```

常用命令：

```bash
./seh run --skill demo-skill --cases demo/cases --out run.json
./seh score --run run.json --out score.json
./seh gate --report score.json --policy demo/policy.yaml
./seh compare --run current.json --baseline baseline.json --fail-on-regression
./seh drift --run current.json --baseline baseline.json --out drift.json
./seh matrix --runtimes demo-skill,other-runtime --cases demo/cases --out matrix.json
./seh leaderboard
./seh frontier --out frontier.json
./seh simulate --out simulation.json
./seh ingest --run remote.json
```

文档导航：

- 设计思路、架构、数据模型、扩展方式：
  [docs/architecture.md](/Users/zealot/Code/skill-eval-harness/docs/architecture.md)
- 全量命令、参数、示例、典型流程：
  [docs/cli.md](/Users/zealot/Code/skill-eval-harness/docs/cli.md)
- 系统流程图：
  [docs/system-flow.excalidraw](/Users/zealot/Code/skill-eval-harness/docs/system-flow.excalidraw)
- 系统架构图：
  [docs/system-architecture.excalidraw](/Users/zealot/Code/skill-eval-harness/docs/system-architecture.excalidraw)
- 系统流程图 SVG：
  [docs/system-flow.svg](/Users/zealot/Code/skill-eval-harness/docs/system-flow.svg)
- 系统架构图 SVG：
  [docs/system-architecture.svg](/Users/zealot/Code/skill-eval-harness/docs/system-architecture.svg)
- 系统流程图 PNG：
  [docs/system-flow.png](/Users/zealot/Code/skill-eval-harness/docs/system-flow.png)
- 系统架构图 PNG：
  [docs/system-architecture.png](/Users/zealot/Code/skill-eval-harness/docs/system-architecture.png)

流程图预览：

![System Flow](/Users/zealot/Code/skill-eval-harness/docs/system-flow.svg)

架构图预览：

![System Architecture](/Users/zealot/Code/skill-eval-harness/docs/system-architecture.svg)

## English

`skill-eval-harness` is a local evaluation and analysis platform for skill runtimes. The CLI entrypoint is `seh`. It combines dataset execution, scoring, history persistence, regression checks, policy gating, and reporting into one workflow.

Core capabilities:

- Dataset-driven evaluation with `cases/*.yaml` and `manifest.yaml`
- Runtime registry and execution across single or multiple runtimes
- Scheduling controls: concurrency, timeout, retry, sampling, tag filtering, deterministic seed
- Scoring and governance: score, gate, baseline, regression checks, strict mode
- Historical analytics: leaderboard, simulation, frontier, stability variance
- Output analytics: drift detection, failure clustering, HTML reports
- History management: `.history/` persistence and remote run ingestion

Quick start:

```bash
make build
./seh run --skill demo-skill --cases demo/cases --out demo/out/run.json
./seh score --run demo/out/run.json --out demo/out/score.json
./seh report --in demo/out/score.json --out demo/out/report.html
```

Or run the bundled demo pipeline:

```bash
make run-demo
make score-demo
make gate-demo
```

Common commands:

```bash
./seh run --skill demo-skill --cases demo/cases --out run.json
./seh score --run run.json --out score.json
./seh gate --report score.json --policy demo/policy.yaml
./seh compare --run current.json --baseline baseline.json --fail-on-regression
./seh drift --run current.json --baseline baseline.json --out drift.json
./seh matrix --runtimes demo-skill,other-runtime --cases demo/cases --out matrix.json
./seh leaderboard
./seh frontier --out frontier.json
./seh simulate --out simulation.json
./seh ingest --run remote.json
```

Documentation:

- Design, architecture, data model, extension points:
  [docs/architecture.md](/Users/zealot/Code/skill-eval-harness/docs/architecture.md)
- Full command reference, flags, examples, and workflows:
  [docs/cli.md](/Users/zealot/Code/skill-eval-harness/docs/cli.md)
- System flow diagram:
  [docs/system-flow.excalidraw](/Users/zealot/Code/skill-eval-harness/docs/system-flow.excalidraw)
- System architecture diagram:
  [docs/system-architecture.excalidraw](/Users/zealot/Code/skill-eval-harness/docs/system-architecture.excalidraw)
- System flow SVG:
  [docs/system-flow.svg](/Users/zealot/Code/skill-eval-harness/docs/system-flow.svg)
- System architecture SVG:
  [docs/system-architecture.svg](/Users/zealot/Code/skill-eval-harness/docs/system-architecture.svg)
- System flow PNG:
  [docs/system-flow.png](/Users/zealot/Code/skill-eval-harness/docs/system-flow.png)
- System architecture PNG:
  [docs/system-architecture.png](/Users/zealot/Code/skill-eval-harness/docs/system-architecture.png)
