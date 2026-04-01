package apiclient

import "context"

// HybridClient uses remote API when available and falls back to local client.
type HybridClient struct {
	remote       Client
	local        Client
	remoteStrict bool
}

// NewHybridClient creates a client with JWT-enabled remote calls and local fallback.
func NewHybridClient(cfg Config) Client {
	local := NewLocalClient(cfg.MockDataDir)
	remote, err := NewRemoteClient(cfg)
	if err != nil {
		return local
	}
	return &HybridClient{remote: remote, local: local, remoteStrict: cfg.RemoteStrict}
}

func (c *HybridClient) ListDatasets(ctx context.Context, query ListDatasetsQuery) (ListDatasetsResponse, error) {
	var zero ListDatasetsResponse
	return runWithFallback(c, zero, func(client Client) (ListDatasetsResponse, error) { return client.ListDatasets(ctx, query) })
}

func (c *HybridClient) GetDataset(ctx context.Context, datasetID string) (DatasetDetailDTO, error) {
	var zero DatasetDetailDTO
	return runWithFallback(c, zero, func(client Client) (DatasetDetailDTO, error) { return client.GetDataset(ctx, datasetID) })
}

func (c *HybridClient) GetDatasetCases(ctx context.Context, datasetID string, query ListDatasetCasesQuery) (GetDatasetCasesResponse, error) {
	var zero GetDatasetCasesResponse
	return runWithFallback(c, zero, func(client Client) (GetDatasetCasesResponse, error) {
		return client.GetDatasetCases(ctx, datasetID, query)
	})
}

func (c *HybridClient) VerifyDataset(ctx context.Context, datasetID string) (DatasetVerifyDTO, error) {
	var zero DatasetVerifyDTO
	return runWithFallback(c, zero, func(client Client) (DatasetVerifyDTO, error) { return client.VerifyDataset(ctx, datasetID) })
}

func (c *HybridClient) CreateRun(ctx context.Context, run RunResultDTO) (CreateRunResponse, error) {
	var zero CreateRunResponse
	return runWithFallback(c, zero, func(client Client) (CreateRunResponse, error) { return client.CreateRun(ctx, run) })
}

func (c *HybridClient) GetRun(ctx context.Context, runID string) (RunResultDTO, error) {
	var zero RunResultDTO
	return runWithFallback(c, zero, func(client Client) (RunResultDTO, error) { return client.GetRun(ctx, runID) })
}

func (c *HybridClient) ListRuns(ctx context.Context, query ListRunsQuery) (ListRunsResponse, error) {
	var zero ListRunsResponse
	return runWithFallback(c, zero, func(client Client) (ListRunsResponse, error) { return client.ListRuns(ctx, query) })
}

func (c *HybridClient) GateRun(ctx context.Context, runID string, req GateRunRequest) (GateRunResponse, error) {
	var zero GateRunResponse
	return runWithFallback(c, zero, func(client Client) (GateRunResponse, error) { return client.GateRun(ctx, runID, req) })
}

func runWithFallback[T any](c *HybridClient, zero T, fn func(client Client) (T, error)) (T, error) {
	resp, err := fn(c.remote)
	if err == nil {
		return resp, nil
	}
	if c.remoteStrict || !shouldFallback(err) {
		return zero, err
	}
	return fn(c.local)
}
