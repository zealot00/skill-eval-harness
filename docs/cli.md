# CLI Guide / 命令说明

## 中文

### 构建与测试

```bash
make build
make test
make coverage
```

也可以直接：

```bash
go build -o seh .
```

### 基础工作流

#### 1. 执行

```bash
./seh run \
  --skill demo-skill \
  --cases demo/cases \
  --out run.json
```

常用参数：

- `--workers N` 并发 worker 数
- `--case-timeout 250ms` 单 case 超时
- `--max-retries 1` 重试次数
- `--tag regression` 标签过滤
- `--sample 0.2` 抽样比例
- `--seed 42` 确定性抽样与确定性输出
- `--strict` 任一 case 失败则退出码非零

#### 2. 评分

```bash
./seh score --run run.json --out score.json
```

使用配置化权重：

```bash
./seh score --run run.json --out score.json --config score.yaml
```

#### 3. 门禁

```bash
./seh gate --report score.json --policy demo/policy.yaml
```

#### 4. HTML 报告

```bash
./seh report --in score.json --out report.html
```

### 历史与治理

#### 查看历史 run

```bash
./seh history list
```

#### 提升 baseline

```bash
./seh baseline promote --run RUN_ID
```

#### 导入远程 run

```bash
./seh ingest --run remote.json
```

### 对比与分析

#### 回归比较

```bash
./seh compare --run current.json --baseline baseline.json
```

若检测到回归则退出 1：

```bash
./seh compare \
  --run current.json \
  --baseline baseline.json \
  --fail-on-regression
```

回归判断规则：

- `success_rate_delta < 0`
- 或 `latency_delta > 0`
- 或 `token_delta > 0`

#### 漂移分析

```bash
./seh drift \
  --run current.json \
  --baseline baseline.json \
  --threshold 0.95 \
  --out drift.json
```

#### 多运行时矩阵

```bash
./seh matrix \
  --runtimes demo-skill,other-runtime \
  --cases demo/cases \
  --out matrix.json
```

可选参数：

- `--workers`
- `--case-timeout`
- `--max-retries`
- `--tag`
- `--sample`
- `--seed`

#### 排行榜

```bash
./seh leaderboard
```

#### 路由模拟

```bash
./seh simulate --out simulation.json
```

包含策略：

- `round_robin`
- `best_score`
- `cost_aware`

#### Pareto 前沿

```bash
./seh frontier --out frontier.json
```

### 示例流程

#### 本地 demo

```bash
make run-demo
make score-demo
make gate-demo
```

#### 手工流程

```bash
./seh run --skill demo-skill --cases demo/cases --out run.json
./seh score --run run.json --out score.json
./seh gate --report score.json --policy demo/policy.yaml
./seh report --in score.json --out report.html
```

#### 回归分析流程

```bash
./seh compare --run current.json --baseline baseline.json --fail-on-regression
./seh drift --run current.json --baseline baseline.json --out drift.json
```

#### 多运行时对比流程

```bash
./seh matrix --runtimes demo-skill,other-runtime --cases demo/cases --out matrix.json
./seh frontier --out frontier.json
./seh leaderboard
```

### 输出说明

#### `run.json`

包含：

- run 元数据
- dataset 信息
- metrics
- 逐 case 结果
- failure cluster id

#### `score.json`

在 `run.json` 基础上补齐 score 相关字段。

#### `report.html`

本地可打开的 HTML 仪表板。

#### `matrix.json`

同一数据集在多个 runtime 上的对比矩阵。

#### `drift.json`

两次 run 输出相似度和 drift 标记。

#### `simulation.json`

不同路由策略下的模拟结果。

#### `frontier.json`

cost vs score 的 Pareto frontier 点集。

### 常见建议

- 日常开发建议使用 `--seed`，便于复现
- 大数据集试跑建议使用 `--sample`
- CI 环境建议使用 `--strict`
- 发布前建议组合使用 `score + gate + compare + drift`

---

## English

### Build and Test

```bash
make build
make test
make coverage
```

Or directly:

```bash
go build -o seh .
```

### Core Workflow

#### 1. Run

```bash
./seh run \
  --skill demo-skill \
  --cases demo/cases \
  --out run.json
```

Common flags:

- `--workers N`
- `--case-timeout 250ms`
- `--max-retries 1`
- `--tag regression`
- `--sample 0.2`
- `--seed 42`
- `--strict`

#### 2. Score

```bash
./seh score --run run.json --out score.json
```

With config-driven weights:

```bash
./seh score --run run.json --out score.json --config score.yaml
```

#### 3. Gate

```bash
./seh gate --report score.json --policy demo/policy.yaml
```

#### 4. HTML Report

```bash
./seh report --in score.json --out report.html
```

### History and Governance

List runs:

```bash
./seh history list
```

Promote baseline:

```bash
./seh baseline promote --run RUN_ID
```

Ingest remote run:

```bash
./seh ingest --run remote.json
```

### Comparison and Analytics

Regression compare:

```bash
./seh compare --run current.json --baseline baseline.json
./seh compare --run current.json --baseline baseline.json --fail-on-regression
```

Regression is currently defined as:

- lower success rate
- higher latency
- higher token usage

Drift:

```bash
./seh drift --run current.json --baseline baseline.json --threshold 0.95 --out drift.json
```

Multi-runtime matrix:

```bash
./seh matrix --runtimes demo-skill,other-runtime --cases demo/cases --out matrix.json
```

Leaderboard:

```bash
./seh leaderboard
```

Simulation:

```bash
./seh simulate --out simulation.json
```

Policies:

- `round_robin`
- `best_score`
- `cost_aware`

Frontier:

```bash
./seh frontier --out frontier.json
```

### Example Flows

Demo:

```bash
make run-demo
make score-demo
make gate-demo
```

Manual flow:

```bash
./seh run --skill demo-skill --cases demo/cases --out run.json
./seh score --run run.json --out score.json
./seh gate --report score.json --policy demo/policy.yaml
./seh report --in score.json --out report.html
```

Regression flow:

```bash
./seh compare --run current.json --baseline baseline.json --fail-on-regression
./seh drift --run current.json --baseline baseline.json --out drift.json
```

Multi-runtime flow:

```bash
./seh matrix --runtimes demo-skill,other-runtime --cases demo/cases --out matrix.json
./seh frontier --out frontier.json
./seh leaderboard
```

### Output Files

- `run.json`: structured execution result
- `score.json`: scored report
- `report.html`: local dashboard
- `matrix.json`: cross-runtime comparison matrix
- `drift.json`: similarity drift report
- `simulation.json`: routing simulation report
- `frontier.json`: Pareto frontier points
