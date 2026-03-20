# 评分算法模型

## 核心公式

```
FinalScore = Σ (metric_value × weight)
```

**默认权重配置：**

| 指标 | 权重 | 说明 |
|------|------|------|
| success_rate | 0.5 | 成功率 |
| cost_factor | 0.3 | 成本（token 消耗） |
| classification_factor | 0.2 | 失败严重度 |

## 各指标计算

| 指标 | 计算方式 | 取值范围 |
|------|----------|----------|
| **success_rate** | 成功数 / 总数 | 0~1 |
| **cost_factor** | `1 / (1 + avg_tokens)` | 越低越好 |
| **latency** | `1 / (1 + p95_latency_ms)` | 越低越好 |
| **classification_factor** | 失败分类加权平均 | 0~1 |

## Classification 权重

| 分类 | 权重值 |
|------|--------|
| 无失败（空） | 1.0 |
| semantic_failure | 0.9 |
| validation_error | 0.8 |
| runtime_error | 0.7 |
| timeout | 0.6 |

## 设计思路

- **成功率为王**（50% 权重）- 核心指标
- **成本感知**（30%）- token 消耗越少分数越高
- **失败严重度惩罚**（20%）- 语义失败比超时更轻微

权重可通过 YAML 配置自定义替换。

## 代码位置

- `internal/runner/metric_evaluator.go` - 评分计算入口
- `internal/runner/runtime.go:326` - `classificationWeight()` 函数
