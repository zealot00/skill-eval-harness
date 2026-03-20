package runner

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
)

var reportTemplate = template.Must(template.New("report").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>SEH Report</title>
  <style>
    :root {
      --bg: #f6f3ea;
      --panel: #fffdf7;
      --ink: #1c1a17;
      --muted: #6e675d;
      --border: #d7cfbe;
      --good: #226c3a;
      --bad: #9c2f2f;
      --accent: #b96d1f;
    }
    body {
      margin: 0;
      font-family: Georgia, "Iowan Old Style", serif;
      color: var(--ink);
      background:
        radial-gradient(circle at top left, #fff7df 0, transparent 28%),
        linear-gradient(180deg, #f0eadc 0%, var(--bg) 100%);
    }
    .wrap {
      max-width: 1100px;
      margin: 0 auto;
      padding: 32px 20px 56px;
    }
    .hero {
      margin-bottom: 24px;
      padding: 24px;
      border: 1px solid var(--border);
      background: linear-gradient(135deg, #fffef9, var(--panel));
      box-shadow: 0 14px 40px rgba(45, 36, 18, 0.08);
    }
    h1, h2 { margin: 0 0 12px; }
    .meta, .metrics {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
      gap: 12px;
      margin-top: 18px;
    }
    .card {
      padding: 14px;
      border: 1px solid var(--border);
      background: rgba(255,255,255,0.72);
    }
    .label {
      display: block;
      font-size: 12px;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      color: var(--muted);
      margin-bottom: 6px;
    }
    .value {
      font-size: 22px;
      font-weight: 700;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      background: var(--panel);
      border: 1px solid var(--border);
    }
    th, td {
      padding: 12px;
      border-bottom: 1px solid var(--border);
      text-align: left;
      vertical-align: top;
    }
    th {
      font-size: 12px;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      color: var(--muted);
    }
    .ok { color: var(--good); font-weight: 700; }
    .fail { color: var(--bad); font-weight: 700; }
    .muted { color: var(--muted); }
    .tools, .steps {
      margin: 0;
      padding-left: 18px;
    }
    .section { margin-top: 24px; }
  </style>
</head>
<body>
  <div class="wrap">
    <section class="hero">
      <h1>Skill Evaluation Report</h1>
      <div class="{{if .Success}}ok{{else}}fail{{end}}">
        {{if .Success}}PASS{{else}}FAIL{{end}}
      </div>
      <div class="meta">
        <div class="card"><span class="label">Model</span><span class="value">{{.ModelName}}</span></div>
        <div class="card"><span class="label">Git Commit</span><span class="value">{{.GitCommit}}</span></div>
        <div class="card"><span class="label">Timestamp</span><span class="value">{{.Timestamp}}</span></div>
        <div class="card"><span class="label">Seed</span><span class="value">{{.Seed}}</span></div>
      </div>
      {{if .Metrics}}
      <div class="metrics">
        <div class="card"><span class="label">Score</span><span class="value">{{printf "%.6f" .Metrics.Score}}</span></div>
        <div class="card"><span class="label">Success Rate</span><span class="value">{{printf "%.6f" .Metrics.SuccessRate}}</span></div>
        <div class="card"><span class="label">Avg Tokens</span><span class="value">{{printf "%.6f" .Metrics.AvgTokens}}</span></div>
        <div class="card"><span class="label">P95 Latency</span><span class="value">{{.Metrics.P95Latency}} ms</span></div>
        <div class="card"><span class="label">Cost</span><span class="value">${{printf "%.6f" .Metrics.CostUSD}}</span></div>
        <div class="card"><span class="label">Stability Variance</span><span class="value">{{printf "%.6f" .Metrics.StabilityVariance}}</span></div>
      </div>
      {{end}}
    </section>

    <section class="section">
      <h2>Case Results</h2>
      <table>
        <thead>
          <tr>
            <th>#</th>
            <th>Status</th>
            <th>Classification</th>
            <th>Latency</th>
            <th>Tokens</th>
            <th>Error</th>
            <th>Cluster</th>
            <th>Steps</th>
            <th>Tool Calls</th>
          </tr>
        </thead>
        <tbody>
          {{range $index, $result := .Results}}
          <tr>
            <td>{{$index}}</td>
            <td class="{{if $result.Success}}ok{{else}}fail{{end}}">{{if $result.Success}}ok{{else}}fail{{end}}</td>
            <td>{{if $result.Classification}}{{$result.Classification}}{{else}}<span class="muted">none</span>{{end}}</td>
            <td>{{$result.LatencyMS}} ms</td>
            <td>{{$result.TokenUsage}}</td>
            <td>{{if $result.Error}}{{$result.Error}}{{else}}<span class="muted">none</span>{{end}}</td>
            <td>{{if $result.FailureClusterID}}{{$result.FailureClusterID}}{{else}}<span class="muted">none</span>{{end}}</td>
            <td>
              <ul class="steps">
                {{range $result.Trajectory.Steps}}<li>{{.}}</li>{{else}}<li class="muted">none</li>{{end}}
              </ul>
            </td>
            <td>
              <ul class="tools">
                {{range $result.Trajectory.ToolCalls}}<li>{{.}}</li>{{else}}<li class="muted">none</li>{{end}}
              </ul>
            </td>
          </tr>
          {{end}}
        </tbody>
      </table>
    </section>
  </div>
</body>
</html>
`))

// WriteHTMLReport renders a simple local HTML dashboard for a run result.
func WriteHTMLReport(path string, run RunResult) error {
	var buf bytes.Buffer
	if err := reportTemplate.Execute(&buf, run); err != nil {
		return fmt.Errorf("render html report: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write html report %q: %w", path, err)
	}

	return nil
}
