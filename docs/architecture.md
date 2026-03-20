# Architecture / 设计说明

## 中文

图文件：

- 系统流程图：[system-flow.excalidraw](/Users/zealot/Code/skill-eval-harness/docs/system-flow.excalidraw)
- 系统架构图：[system-architecture.excalidraw](/Users/zealot/Code/skill-eval-harness/docs/system-architecture.excalidraw)
- 系统流程图 SVG：[system-flow.svg](/Users/zealot/Code/skill-eval-harness/docs/system-flow.svg)
- 系统架构图 SVG：[system-architecture.svg](/Users/zealot/Code/skill-eval-harness/docs/system-architecture.svg)
- 系统流程图 PNG：[system-flow.png](/Users/zealot/Code/skill-eval-harness/docs/system-flow.png)
- 系统架构图 PNG：[system-architecture.png](/Users/zealot/Code/skill-eval-harness/docs/system-architecture.png)

### 项目目标

`skill-eval-harness` 的目标不是只跑一次样例，而是建立一个可持续演进的评测闭环：

- 数据集定义评测输入
- 运行时执行生成结构化结果
- 结果进入历史库
- 历史数据驱动比较、门禁、分析与可视化

### 设计原则

1. 数据优先  
   所有评测围绕数据集展开，而不是围绕手写脚本展开。

2. 结构化优先  
   所有关键对象都用结构体和 JSON/YAML 明确表达，便于测试、比较、沉淀。

3. 历史优先  
   历史运行结果不是日志，而是可被继续分析的资产。

4. 可扩展优先  
   运行时、评分指标、对比方式、治理策略都应可扩展。

### 分层架构

#### `internal/dataset`

负责：

- `EvaluationCase`
- `Manifest`
- `LoadCases`
- `LoadManifest`
- `ComputeDatasetHash`
- `FilterByTag`
- `FilterBySample`

这一层解决“输入是什么”的问题。

#### `internal/runner`

负责：

- `SkillRuntime` 接口
- worker pool 并发执行
- timeout / retry / sampling / seed
- `RunResult` / `CaseRunResult` / `SkillResult`
- metrics 与 score
- runtime registry
- history 持久化
- drift / compare / matrix / simulate / leaderboard / frontier
- HTML report

这一层解决“如何执行、如何记录、如何分析”的问题。

#### `internal/policy`

负责：

- policy YAML 解析
- score / success_rate / p95_latency / avg_tokens 阈值校验

这一层解决“是否允许通过”的问题。

### 数据流

典型流程如下：

1. 从 `cases/` 读取数据集与 manifest
2. 计算 `dataset_hash`
3. 从 registry 解析 runtime
4. 执行 case，生成逐 case 结果
5. 聚合为 `RunResult`
6. 计算 metrics 与 score
7. 写出 JSON 报告
8. 持久化到 `.history/`
9. 基于历史做 compare / simulate / leaderboard / frontier / stability

### 数据集设计

数据集目录：

```text
cases/
  manifest.yaml
  *.yaml
```

`manifest.yaml` 字段：

- `dataset_name`
- `version`
- `owner`
- `description`

case 字段：

- `case_id`
- `skill`
- `input`
- `expected`
- `tags`

说明：

- `case_id` 和 `skill` 为必填
- `expected.output_hash` 可用于输出哈希校验
- `tags` 用于筛选不同测试切片

### 结果模型设计

#### `RunResult`

包含：

- 执行状态：`success`、`error`
- 聚合统计：`latency_ms`、`token_usage`
- 元数据：`git_commit`、`model_name`、`timestamp`
- 数据集信息：`dataset_version`、`dataset_hash`
- 历史追踪：`run_id`
- 逐 case 结果：`results`
- 评分信息：`metrics`

#### `CaseRunResult`

包含：

- `success`
- `latency_ms`
- `token_usage`
- `output`
- `error`
- `classification`
- `failure_cluster_id`
- `trajectory`

#### `RunMetrics`

包含：

- `success_rate`
- `avg_tokens`
- `p95_latency`
- `cost_factor`
- `classification_factor`
- `cost_usd`
- `stability_variance`
- `score`

### 调度设计

执行层支持：

- 单线程顺序执行
- `--workers N` worker pool
- `--case-timeout`
- `--max-retries`
- `--sample`
- `--tag`
- `--seed`

设计意图：

- 默认简单
- 需要吞吐时开启并发
- 需要可复现时使用 seed
- 需要降低成本时使用 sample

### 评分设计

系统把原始执行结果转为更高层指标：

- 成功率
- token 平均值
- p95 延迟
- 成本
- 失败分类因子
- 稳定性方差

然后通过权重配置合成综合分数。

这使得系统不仅能回答“能不能跑”，还能回答：

- 值不值得用
- 相比基线有没有变差
- 成本和质量是否更平衡

### 历史设计

`.history/` 中保存所有 run 结果。

历史系统提供：

- `PersistRun`
- `ListRuns`
- `LoadRun`
- `PromoteBaseline`
- `IngestRun`
- `ComputeStabilityVariance`

因此 `.history/` 是所有高级分析的基础。

### 分析设计

#### 回归比较

基于两个 run 的：

- `success_rate_delta`
- `latency_delta`
- `token_delta`

并支持 `--fail-on-regression`。

#### 漂移分析

通过本地 embedding-style 文本向量相似度比较输出，低于阈值则标记 drift。

#### 失败聚类

对失败 case 的输出与错误文本做相似度聚类，生成 `failure_cluster_id`。

#### 多运行时矩阵

同一数据集可同时运行多个 runtime，产出对比矩阵。

#### 路由模拟

基于历史结果模拟：

- `round_robin`
- `best_score`
- `cost_aware`

#### Pareto Frontier

从历史结果中找出 cost vs score 上的非支配点。

#### 稳定性方差

基于同 skill、同 dataset 的多次 run 计算 score 方差，用于衡量稳定性。

### 扩展点

#### 运行时扩展

接口：

```go
type SkillRuntime interface {
    Execute(ctx context.Context, input map[string]any) (SkillResult, error)
}
```

注册：

```go
runner.RegisterRuntime("my-runtime", myRuntime)
```

#### 指标扩展

接口：

```go
type MetricEvaluator interface {
    Evaluate(run RunResult) float64
}
```

### 当前边界

当前实现更偏本地评测与离线分析，仍未覆盖：

- 分布式执行
- 真实 embedding API
- 向量数据库
- Web UI
- 多租户服务化

但当前结构已经为这些演进方向预留了接口边界。

---

## English

Diagram files:

- System flow: [system-flow.excalidraw](/Users/zealot/Code/skill-eval-harness/docs/system-flow.excalidraw)
- System architecture: [system-architecture.excalidraw](/Users/zealot/Code/skill-eval-harness/docs/system-architecture.excalidraw)
- System flow SVG: [system-flow.svg](/Users/zealot/Code/skill-eval-harness/docs/system-flow.svg)
- System architecture SVG: [system-architecture.svg](/Users/zealot/Code/skill-eval-harness/docs/system-architecture.svg)
- System flow PNG: [system-flow.png](/Users/zealot/Code/skill-eval-harness/docs/system-flow.png)
- System architecture PNG: [system-architecture.png](/Users/zealot/Code/skill-eval-harness/docs/system-architecture.png)

### Goal

`skill-eval-harness` is designed as a reusable evaluation loop:

- datasets define inputs
- runtimes execute cases
- runs become structured results
- results are persisted as history
- history powers comparison, gating, analytics, and reports

### Design Principles

1. Dataset-first
2. Structured outputs
3. History-first analytics
4. Extensibility-first design

### Layers

#### `internal/dataset`

Owns:

- case schema
- manifest schema
- YAML loading
- dataset hashing
- tag filtering
- sampling

This layer defines the input model.

#### `internal/runner`

Owns:

- runtime interface
- execution scheduling
- concurrency, timeout, retry
- result models
- scoring
- registry
- history
- reports and analytics

This layer defines execution and analysis.

#### `internal/policy`

Owns:

- policy parsing
- threshold-based gating

This layer defines pass/fail governance.

### Data Flow

1. load dataset and manifest
2. compute dataset hash
3. resolve runtime
4. execute cases
5. aggregate `RunResult`
6. compute metrics and score
7. write JSON report
8. persist to `.history/`
9. run analytics on history

### Dataset Model

Layout:

```text
cases/
  manifest.yaml
  *.yaml
```

Manifest fields:

- `dataset_name`
- `version`
- `owner`
- `description`

Case fields:

- `case_id`
- `skill`
- `input`
- `expected`
- `tags`

### Result Model

`RunResult` stores aggregate execution, metadata, dataset metadata, run ID, per-case results, and metrics.

`CaseRunResult` stores status, latency, token usage, output, error, classification, failure cluster ID, and trajectory.

`RunMetrics` stores success rate, avg tokens, p95 latency, cost, classification factor, stability variance, and composite score.

### Scheduling

Supported controls:

- sequential execution
- worker pool via `--workers`
- timeout
- retry
- tag filtering
- random sampling
- deterministic seed

### Scoring

Raw execution outcomes are transformed into higher-level metrics:

- success rate
- average token usage
- p95 latency
- cost
- classification factor
- stability variance

These are then combined into a composite score.

### History

`.history/` stores persisted runs and powers:

- listing and loading runs
- baseline promotion
- remote ingestion
- stability analysis

### Analytics

The system currently supports:

- regression comparison
- drift detection
- failure clustering
- multi-runtime matrix
- routing simulation
- leaderboard
- Pareto frontier
- stability variance

### Extension Points

Runtime extension:

```go
type SkillRuntime interface {
    Execute(ctx context.Context, input map[string]any) (SkillResult, error)
}
```

Metric extension:

```go
type MetricEvaluator interface {
    Evaluate(run RunResult) float64
}
```

### Current Scope

The current implementation focuses on local evaluation and offline analysis. It does not yet include:

- distributed execution
- real embedding APIs
- vector databases
- web-native UI
- multi-tenant service architecture

The current structure is intentionally prepared for those evolutions.
