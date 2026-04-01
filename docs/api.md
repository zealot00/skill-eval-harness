# Skill Eval Harness - API Specification v1.0

## 协议约定

- **Base URL:** `https://api.seh.example.com/v1` (实现方填写)
- **Content-Type:** `application/json`
- **认证:** `Authorization: Bearer <api_key>` (无过期时间)

**Phase 判定规则:**
```
if API_BASE_URL 或 API_TOKEN 任一未配置:
    启用 Phase 0 (Local CLI)
else:
    启用 Remote capabilities (Phase 1+)
```

---

## 通用规范

### 错误响应

所有 API 错误统一使用以下格式：

```json
{
  "error": {
    "code": 400,
    "message": "Human readable error message",
    "details": []
  }
}
```

| Code | HTTP Status | Meaning |
|------|-------------|---------|
| 400 | 400 | Bad Request - 参数错误 |
| 401 | 401 | Unauthorized - 无效/缺失 API Key |
| 403 | 403 | Forbidden - 无权限 |
| 404 | 404 | Not Found - 资源不存在 |
| 409 | 409 | Conflict - 资源冲突 |
| 422 | 422 | Unprocessable Entity - 业务校验失败 |
| 429 | 429 | Too Many Requests - 超过 QPS 限制 |
| 500 | 500 | Internal Server Error - 服务端错误 |

### 分页

所有列表类 API 支持分页：

```json
// Query: ?limit=20&offset=0&sort=created_at

// Response
{
  "data": [...],
  "pagination": {
    "limit": 20,
    "offset": 0,
    "total": 100,
    "has_more": true
  }
}
```

**默认值：**
- `limit`: 20
- `max_limit`: 100
- `offset`: 0
- `sort`: created_at (DESC)

**支持的 sort 字段：**
- `created_at`
- `updated_at`
- `name`

### Rate Limiting

- 每 API Key **100 QPS**
- 响应头返回：`X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`
- 超过限制返回 `429 Too Many Requests`

### 数据集限制

| 限制项 | 值 |
|--------|-----|
| 单个 dataset 最大 cases 数量 | 10,000 |
| 单个 case input/expected 总大小 | 1MB |
| 单次请求 body 最大 size | 10MB |
| dataset name 最大长度 | 128 字符 |
| tag 最大数量 per case | 20 |

### 异步任务

**处理流程：**
1. 客户端发起 POST 请求，返回 `202 Accepted`
2. 服务端创建异步任务，返回 `job_id`
3. 客户端轮询 `GET /datasets/synthesize/:job_id` 查询状态
4. 任务完成后返回结果

**可选：Webhook 回调**
- 请求体可包含 `webhook_url` 字段
- 任务完成时服务端 POST 回调

---

## Phase 能力边界

| Phase | 定位 | 用户 | CLI 能力 | Server 能力 |
|-------|------|------|----------|-------------|
| **Phase 0** | Local CLI | 个人开发者 | 全部本地执行 | 无 |
| **Phase 1** | Remote Dataset Store (Read-only) | 团队共享 | `pull`, `ls`, `verify` | Dataset 存储 (管理员上传) |
| **Phase 2** | Evaluation Control Plane | 组织 | `run`, `score`, `gate`, `case submit` | Runs 管理, Case Lifecycle, Policy, Lineage |
| **Phase 3** | AI Engineering Platform | 平台 | `pull approved dataset` | Release Gate, Routing Signal, Dataset Evolution |

**核心原则:**
- Phase 越高，CLI 越瘦 (thin client)
- Approval 审批只在 Dashboard，不通过 CLI
- CLI 被动获取平台审批后的结果
- Phase 1 为 Read-only Store，Dataset 由管理员通过 Web UI 上传

---

## Phase 0 - Local CLI

无 API，纯本地文件操作。

```
CLI Commands:
  seh run --skill <skill> --cases <path> --out <run.json>
  seh score --run <run.json> --out <score.json>
  seh gate --report <score.json> --policy <policy.yaml>
  seh report --in <score.json> --out <report.html>
  seh compare --run <current.json> --baseline <baseline.json>
  seh drift --run <current.json> --baseline <baseline.json>
```

---

## Phase 1 - Remote Dataset Store (Read-only)

**说明:**
- CLI 为 Read-only，只能 pull dataset
- Dataset 由管理员通过 Web UI 上传和管理
- 所有操作需要 Auth (API Key)

### 认证

```
POST /auth/token
Request:  { "api_key": "xxx" }
Response: 200 { "token": "jwt_xxx" }  # Token 不过期
```

### Dataset API (Read-only)

| Method | Path | Description | CLI 对应 |
|--------|------|-------------|----------|
| GET | /datasets | 列出 datasets | `seh ls datasets` |
| GET | /datasets/:dataset_id | 获取 dataset metadata | - |
| GET | /datasets/:dataset_id/cases | 下载 cases | `seh pull dataset:<version>` |
| GET | /datasets/:dataset_id/verify | 校验完整性 | `seh verify dataset:<version>` |

**注意: Phase 1 不支持 POST /datasets (上传) 和 DELETE /datasets/:dataset_id (删除)**

### GET /datasets

```json
// Query: ?tag=regression&owner=team-a&limit=20&offset=0&sort=created_at

// Response 200
{
  "datasets": [
    {
      "dataset_id": "ds_abc123",
      "name": "my-dataset",
      "version": "v1",
      "owner": "team-a",
      "case_count": 50,
      "created_at": "2025-01-15T10:00:00Z"
    }
  ],
  "pagination": {
    "limit": 20,
    "offset": 0,
    "total": 100,
    "has_more": true
  }
}
```

**支持的查询参数：**
| 参数 | 类型 | 说明 |
|------|------|------|
| `tag` | string | 过滤包含特定 tag 的 dataset |
| `owner` | string | 过滤特定 owner |
| `name` | string | 模糊匹配 dataset name |
| `limit` | int | 默认 20，最大 100 |
| `offset` | int | 默认 0 |
| `sort` | string | `created_at` (默认), `updated_at`, `name` |

### GET /datasets/:dataset_id

```json
// Response 200
{
  "dataset_id": "ds_abc123",
  "name": "my-dataset",
  "version": "v1",
  "owner": "team-a",
  "description": "...",
  "manifest": {
    "dataset_name": "my-dataset",
    "version": "v1",
    "owner": "team-a",
    "description": "..."
  },
  "case_count": 50,
  "checksum": "sha256:xxx",
  "created_at": "2025-01-15T10:00:00Z"
}
```

### GET /datasets/:dataset_id/cases

```json
// Query: ?source=golden&status=active&limit=100&offset=0

// Response 200
{
  "manifest": { ... },
  "cases": [ /* EvaluationCase[] */ ],
  "pagination": {
    "limit": 100,
    "offset": 0,
    "total": 500,
    "has_more": true
  }
}
```

**支持的查询参数：**
| 参数 | 类型 | 说明 |
|------|------|------|
| `source` | string | 过滤特定 source (golden, production, ...) |
| `status` | string | 过滤特定 status (approved, pending_review, ...) |
| `tag` | string | 过滤特定 tag |
| `limit` | int | 默认 100，最大 500 |
| `offset` | int | 默认 0 |

### GET /datasets/:dataset_id/verify

```json
// Response 200
{
  "valid": true,
  "checksum": "sha256:xxx",
  "case_count": 50,
  "verified_at": "2025-01-15T10:00:00Z"
}

// Response 200 (校验失败)
{
  "valid": false,
  "expected": "sha256:abc123",
  "actual": "sha256:def456",
  "verified_at": "2025-01-15T10:00:00Z"
}
```

### CLI Commands (Phase 1)

```bash
# 列出 remote datasets
seh ls datasets

# 下载 dataset 到本地
seh pull dataset:v3 --out ./cases/

# 下载特定 dataset (通过 dataset_id)
seh pull dataset:ds_abc123 --out ./cases/

# 校验 dataset 完整性
seh verify dataset:v3
```

---

## Phase 2 - Evaluation Control Plane

### Runs API

| Method | Path | Description | CLI 对应 |
|--------|------|-------------|----------|
| POST | /runs | 提交 evaluation run | `seh run --dataset dataset:v3` |
| GET | /runs/:run_id | 获取 run 详情 | - |
| GET | /runs | 查询历史 runs | - |

#### POST /runs

```json
// Request
{
  "dataset_id": "ds_abc123",
  "skill": "my-skill",
  "runtime": "v1.2.3",
  "results": [ /* CaseRunResult[] */ ],
  "metrics": { /* RunMetrics */ }
}

// Response 201
{
  "run_id": "run_xyz789",
  "score": 0.85,
  "success_rate": 0.92,
  "created_at": "2025-01-15T10:00:00Z"
}

// Error Response (422 - 校验失败)
{
  "error": {
    "code": 422,
    "message": "Invalid run data",
    "details": [
      { "field": "dataset_id", "message": "Dataset not found" }
    ]
  }
}
```

**校验规则：**
- `dataset_id` 必填，必须是已存在的 dataset
- `skill` 必填
- `runtime` 必填，格式语义化版本
- `results` 必填，非空数组
- `metrics` 可选

#### GET /runs

```json
// Query: ?skill=xxx&from=2025-01-01&to=2025-01-31&limit=20&offset=0

// Response 200
{
  "runs": [ /* RunResult[] */ ],
  "pagination": {
    "limit": 20,
    "offset": 0,
    "total": 100,
    "has_more": true
  }
}
```

**支持的查询参数：**
| 参数 | 类型 | 说明 |
|------|------|------|
| `skill` | string | 过滤特定 skill |
| `dataset_id` | string | 过滤特定 dataset |
| `from` | ISO8601 | 起始时间 |
| `to` | ISO8601 | 结束时间 |
| `min_score` | float | 最低分数过滤 |
| `limit` | int | 默认 20，最大 100 |
| `offset` | int | 默认 0 |

### Case Lifecycle API

| Method | Path | Description | CLI 对应 |
|--------|------|-------------|----------|
| POST | /cases | 提交 case | `seh case submit <case.yaml>` |
| GET | /cases/:case_id | 查看 case | - |
| GET | /cases | 查询 cases | `seh case review <case_id>` |
| POST | /cases/:case_id/review | Review case (Dashboard) | **Dashboard only** |
| POST | /cases/:case_id/deprecate | 标记 deprecated | - |

#### POST /cases

```json
// Request: { /* EvaluationCase */ }

// Response 201
{
  "case_id": "case_xxx",
  "status": "pending_review",
  "created_at": "2025-01-15T10:00:00Z"
}
```

**校验规则：**
- `case_id` 必填，唯一
- `skill` 必填
- `source` 必填，可选值：`golden`, `production`, `adversarial`, `synthetic`, `cross_team`, `human`
- `input` 必填，JSON 对象
- `expected` 必填，JSON 对象

**Status 流转规则：**
```
draft → pending_review (submit 后)
pending_review → approved (review 通过)
pending_review → deprecated (review 拒绝或手动 deprecated)
approved → deprecated (手动 deprecated)
deprecated → archived (手动 archived)
```

#### POST /cases/:case_id/review

**Note: 此接口仅 Dashboard 调用，CLI 不开放。**

```json
// Request
{
  "approved": true,
  "reviewer": "user_123",
  "reason": "LGTM"
}

// Response 200
{
  "case_id": "case_xxx",
  "status": "approved",
  "reviewed_by": "user_123",
  "reviewed_at": "2025-01-15T11:00:00Z"
}

// Error Response (422 - 状态不允许 review)
{
  "error": {
    "code": 422,
    "message": "Case status 'deprecated' does not allow review",
    "details": []
  }
}
```

### Policy API

| Method | Path | Description | CLI 对应 |
|--------|------|-------------|----------|
| POST | /policies | 创建 policy | `seh policy create <policy.yaml>` |
| GET | /policies/:policy_id | 获取 policy | - |
| GET | /policies | 列出 policies | - |
| POST | /runs/:run_id/gate | 执行 gate check | `seh gate --report score.json --policy policy.yaml` |

#### POST /policies

```json
// Request: { /* GovernancePolicy */ }

// Response 201
{
  "policy_id": "pol_xxx",
  "created_at": "2025-01-15T10:00:00Z"
}
```

#### POST /runs/:run_id/gate

```json
// Request: { "policy_id": "pol_xxx" }

// Response 200
{
  "passed": true,
  "run_id": "run_xyz789",
  "policy_id": "pol_xxx",
  "details": {
    "primary_score": { "required": 0.8, "actual": 0.85, "passed": true },
    "source_diversity": { "required": 3, "actual": 4, "passed": true },
    "adversarial_pass_rate": { "required": 0.7, "actual": 0.65, "passed": false }
  },
  "evaluated_at": "2025-01-15T10:00:00Z"
}
```

### Lineage API

| Method | Path | Description |
|--------|------|-------------|
| GET | /cases/:case_id/lineage | Case 血缘追溯 |
| GET | /datasets/:dataset_id/lineage | Dataset 版本血缘 |

#### GET /cases/:case_id/lineage

```json
// Response 200
{
  "case_id": "case_xxx",
  "ancestors": [
    { "case_id": "case_yyy", "relationship": "synthesized_from", "distance": 1 }
  ],
  "descendants": [
    { "case_id": "case_zzz", "relationship": "generated", "distance": 1 }
  ]
}
```

#### GET /datasets/:dataset_id/lineage

```json
// Response 200
{
  "dataset_id": "ds_abc123",
  "name": "my-dataset",
  "versions": [
    {
      "dataset_id": "ds_old",
      "version": "v1",
      "parent_dataset_id": null,
      "created_at": "2025-01-01T10:00:00Z"
    },
    {
      "dataset_id": "ds_abc123",
      "version": "v2",
      "parent_dataset_id": "ds_old",
      "created_at": "2025-01-15T10:00:00Z"
    }
  ],
  "contributors": ["user_a", "user_b"]
}
```

---

## Phase 3 - AI Engineering Platform

**核心变化:**
- CLI 变为 thin client，仅负责发起请求和获取结果
- 所有 Approval 审批流程由 Dashboard 处理
- CLI 被动获取平台审批后的 approved dataset / release 状态

### Release API

| Method | Path | Description | CLI 对应 |
|--------|------|-------------|----------|
| POST | /skills/:skill/releases | 发起 release | `seh release --skill xxx --run run_id` |
| GET | /releases/:release_id | 查看 release 状态 | `seh release status <release_id>` |

**Approval 接口仅 Dashboard 调用，CLI 不开放:**
- ~~POST /releases/:release_id/approve~~
- ~~POST /releases/:release_id/reject~~
- ~~POST /releases/:release_id/rollback~~

#### POST /skills/:skill/releases

```json
// Request
{
  "run_id": "run_xyz789",
  "policy_id": "pol_xxx",
  "release_type": "major",
  "change_summary": "Bug fixes and performance improvements"
}

// Response 202
{
  "release_id": "rel_aaa",
  "skill": "my-skill",
  "status": "pending_approval",
  "run_id": "run_xyz789",
  "policy_id": "pol_xxx",
  "created_at": "2025-01-15T10:00:00Z"
}
```

**Release Type 规则：**
- `major`: 主版本发布，需要完整 review
- `minor`: 次版本发布，简化 review
- `patch`: 补丁发布，仅需 CI 通过

**Status 流转：**
```
pending_approval → approved (Dashboard 审批通过)
pending_approval → rejected (Dashboard 审批拒绝)
approved → rolled_back (Dashboard 触发 rollback)
```

#### GET /releases/:release_id

```json
// Response 200
{
  "release_id": "rel_aaa",
  "skill": "my-skill",
  "status": "approved",
  "run_id": "run_xyz789",
  "policy_id": "pol_xxx",
  "release_type": "major",
  "change_summary": "Bug fixes and performance improvements",
  "approved_by": "manager_456",
  "approved_at": "2025-01-15T11:00:00Z",
  "rejected_reason": null,
  "created_at": "2025-01-15T10:00:00Z"
}
```

### Routing Signal API

| Method | Path | Description |
|--------|------|-------------|
| GET | /routing/recommend | 获取路由推荐 |
| POST | /routing/feedback | 上报路由效果 |

#### GET /routing/recommend

```json
// Query: ?skill=csv_generation&context={"input_size": "large"}

Response 200
{
  "skill": "csv_generation",
  "recommendations": [
    {
      "model_id": "model_a",
      "score": 0.92,
      "confidence": 0.85,
      "avg_latency_ms": 150,
      "avg_cost": 0.002
    },
    {
      "model_id": "model_b",
      "score": 0.88,
      "confidence": 0.72,
      "avg_latency_ms": 100,
      "avg_cost": 0.003
    }
  ],
  "generated_at": "2025-01-15T10:00:00Z"
}
```

#### POST /routing/feedback

```json
// Request
{
  "model_id": "model_a",
  "skill": "csv_generation",
  "success": true,
  "latency_ms": 123,
  "token_usage": 456,
  "feedback_at": "2025-01-15T12:00:00Z"
}

// Response 200
{
  "acknowledged": true
}
```

### Dataset Evolution API

| Method | Path | Description |
|--------|------|-------------|
| GET | /insights/failure-patterns | 失败模式分析 |
| POST | /datasets/synthesize | 请求 synthetic 生成 |
| GET | /datasets/synthesize/:job_id | 查询生成任务 |

#### GET /insights/failure-patterns

```json
// Query: ?skill=csv_generation&from=2025-01-01&to=2025-01-31

// Response 200
{
  "skill": "csv_generation",
  "period": {
    "from": "2025-01-01T00:00:00Z",
    "to": "2025-01-31T23:59:59Z"
  },
  "clusters": [
    {
      "cluster_id": "c1",
      "description": "timeout on large files (>10MB)",
      "case_count": 15,
      "first_seen": "2025-01-05T10:00:00Z",
      "last_seen": "2025-01-28T15:30:00Z"
    },
    {
      "cluster_id": "c2",
      "description": "malformed CSV output - missing headers",
      "case_count": 8,
      "first_seen": "2025-01-10T09:00:00Z",
      "last_seen": "2025-01-25T11:00:00Z"
    }
  ],
  "suggestions": [
    {
      "type": "synthetic_case",
      "priority": "high",
      "description": "建议增加 20 个 boundary cases (file size > 10MB)",
      "params": { "type": "rule", "count": 20, "context": { "attack_type": "boundary" } }
    }
  ]
}
```

#### POST /datasets/synthesize

```json
// Request
{
  "type": "failure_driven",
  "count": 20,
  "context": {
    "skill": "csv_generation",
    "failure_cluster_ids": ["c1"]
  },
  "webhook_url": "https://optional-callback.example.com/synthesize"
}

// Response 202
{
  "job_id": "job_xxx",
  "type": "failure_driven",
  "status": "queued",
  "estimated_completion": "2025-01-15T10:05:00Z",
  "created_at": "2025-01-15T10:00:00Z"
}
```

**支持的 type：**
- `rule`: 确定性规则生成
- `llm`: LLM 生成 (需要 validator)
- `failure_driven`: 基于失败模式生成

#### GET /datasets/synthesize/:job_id

```json
// Response 200 (运行中)
{
  "job_id": "job_xxx",
  "type": "failure_driven",
  "status": "running",
  "progress": {
    "current": 12,
    "total": 20
  },
  "created_at": "2025-01-15T10:00:00Z"
}

// Response 200 (完成)
{
  "job_id": "job_xxx",
  "type": "failure_driven",
  "status": "completed",
  "cases": [ /* EvaluationCase[] */ ],
  "stats": {
    "total": 20,
    "by_source": { "synthetic": 20 },
    "by_status": { "draft": 20 }
  },
  "completed_at": "2025-01-15T10:03:00Z"
}

// Response 200 (失败)
{
  "job_id": "job_xxx",
  "type": "failure_driven",
  "status": "failed",
  "error": {
    "code": 500,
    "message": "LLM generator timeout"
  },
  "failed_at": "2025-01-15T10:02:00Z"
}
```

### CLI Commands (Phase 3)

```bash
# 发起 release request
seh release --skill xxx --run run_id --policy policy.yaml

# 查看 release 状态
seh release status <release_id>

# 获取 Dashboard 审批后的 approved dataset
seh pull dataset:v4  # 拿最新 approved 版本
```

---

## Dashboard API (Internal Only)

**以下接口仅 Dashboard 调用，CLI 不开放。**

### Dataset 管理 (Web UI Only)

| Method | Path | Description |
|--------|------|-------------|
| POST | /datasets | 上传 dataset (管理员) |
| DELETE | /datasets/:dataset_id | 删除 dataset (管理员) |

### Case Approval

| Method | Path | Description |
|--------|------|-------------|
| POST | /cases/:case_id/review | Review + Approval |
| POST | /cases/:case_id/deprecate | 标记 deprecated |

**鉴权要求：** 需要 `reviewer` 角色的 API Key

### Release Approval

| Method | Path | Description |
|--------|------|-------------|
| POST | /releases/:release_id/approve | 审批通过 |
| POST | /releases/:release_id/reject | 审批拒绝 |
| POST | /releases/:release_id/rollback | 触发 rollback |

**鉴权要求：** 需要 `approver` 角色的 API Key

---

## 数据模型

### EvaluationCase

```json
{
  "case_id": "string (必填, 最大128字符)",
  "skill": "string (必填)",
  "source": "string (必填, 可选值见下表)",
  "status": "string (默认 draft)",
  "provenance": {
    "approved_by": "string",
    "contributor_id": "string",
    "attack_category": "string",
    "generator_id": "string",
    "method": "string",
    "seed": "string"
  },
  "input": "object (必填)",
  "expected": "object (必填)",
  "tags": ["string (最多20个)"]
}
```

**Source 可选值：**
| 值 | 说明 |
|----|------|
| `golden` | SME 专家标准 |
| `production` | 真实历史流量 |
| `adversarial` | 红队构造 |
| `synthetic` | 自动生成 |
| `cross_team` | 跨团队贡献 |
| `human` | 人工编写 |

**Status 可选值：**
| 值 | 说明 |
|----|------|
| `draft` | 草稿 |
| `pending_review` | 待审核 |
| `approved` | 已批准 |
| `deprecated` | 已废弃 |
| `archived` | 已归档 |

### GovernancePolicy

```json
{
  "policy_id": "string",
  "name": "string (必填, 最大128字符)",
  "require_provenance": "boolean (默认 false)",
  "require_approved_for_score": "boolean (默认 true)",
  "min_source_diversity": "int (默认 1)",
  "min_golden_weight": "float (默认 0.0)",
  "source_policies": [
    {
      "source": "string",
      "weight": "float (0.0-1.0)",
      "count_in_score": "boolean",
      "min_success_rate": "float (0.0-1.0)",
      "adversarial_threshold": "float (仅 adversarial)"
    }
  ]
}
```

### RunResult

```json
{
  "run_id": "string",
  "dataset_id": "string",
  "skill": "string",
  "runtime": "string",
  "metrics": {
    "success_rate": "float",
    "avg_tokens": "float",
    "p95_latency": "int64",
    "cost_factor": "float",
    "classification_factor": "float",
    "cost_usd": "float",
    "stability_variance": "float",
    "score": "float"
  },
  "results": [ /* CaseRunResult[] */ ],
  "created_at": "ISO8601"
}
```

### CaseRunResult

```json
{
  "case_id": "string",
  "success": "boolean",
  "latency_ms": "int64",
  "token_usage": "int64",
  "output": "object",
  "error": "string",
  "classification": "string",
  "failure_cluster_id": "string",
  "provenance": {
    "source": "string",
    "status": "string"
  },
  "trajectory": {
    "steps": []
  }
}
```

---

## 版本策略

### API 版本控制

- 当前版本：**v1**
- 版本通过 URL 路径管理：`/v1/...`
- 未来升级：**v2**，旧版本保留 6 个月后下线

### 非破坏性变更 (兼容)

- 新增可选字段
- 新增可选 API 参数
- 新增新的 enum 值

### 破坏性变更 (不兼容)

- 删除或重命名字段
- 改变字段类型
- 删除 API 端点
- 改变必填参数

---

## 附录：错误码速查

| 业务错误码 | 说明 |
|-----------|------|
| 1001 | Dataset checksum 不匹配 |
| 1002 | Dataset 已存在 |
| 1003 | Case 已存在 |
| 1004 | Case status 不允许此操作 |
| 2001 | Run 关联的 dataset 不存在 |
| 2002 | Policy 不存在 |
| 2003 | Gate check 失败 |
| 3001 | Release 状态不允许此操作 |
| 4001 | Synthesize job 不存在 |
| 4002 | Synthesize job 已失败 |
