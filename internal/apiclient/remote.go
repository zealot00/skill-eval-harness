package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// RemoteClient calls the remote API server and injects JWT Bearer token.
type RemoteClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewRemoteClient creates a remote API client.
func NewRemoteClient(cfg Config) (*RemoteClient, error) {
	baseURL := cfg.normalizedBaseURL()
	if baseURL == "" || strings.TrimSpace(cfg.APIToken) == "" {
		return nil, ErrRemoteDisabled
	}
	return &RemoteClient{
		baseURL:    baseURL,
		token:      strings.TrimSpace(cfg.APIToken),
		httpClient: cfg.clientOrDefault(),
	}, nil
}

func (c *RemoteClient) ListDatasets(ctx context.Context, query ListDatasetsQuery) (ListDatasetsResponse, error) {
	values := url.Values{}
	addIfNotEmpty(values, "tag", query.Tag)
	addIfNotEmpty(values, "owner", query.Owner)
	addIfNotEmpty(values, "name", query.Name)
	addIfPositive(values, "limit", query.Limit)
	addIfNonNegative(values, "offset", query.Offset)
	addIfNotEmpty(values, "sort", query.Sort)

	var out ListDatasetsResponse
	err := c.doJSON(ctx, http.MethodGet, "/datasets", values, nil, &out)
	return out, err
}

func (c *RemoteClient) GetDataset(ctx context.Context, datasetID string) (DatasetDetailDTO, error) {
	var out DatasetDetailDTO
	err := c.doJSON(ctx, http.MethodGet, "/datasets/"+url.PathEscape(datasetID), nil, nil, &out)
	return out, err
}

func (c *RemoteClient) GetDatasetCases(ctx context.Context, datasetID string, query ListDatasetCasesQuery) (GetDatasetCasesResponse, error) {
	values := url.Values{}
	addIfNotEmpty(values, "source", query.Source)
	addIfNotEmpty(values, "status", query.Status)
	addIfNotEmpty(values, "tag", query.Tag)
	addIfPositive(values, "limit", query.Limit)
	addIfNonNegative(values, "offset", query.Offset)

	var out GetDatasetCasesResponse
	err := c.doJSON(ctx, http.MethodGet, "/datasets/"+url.PathEscape(datasetID)+"/cases", values, nil, &out)
	return out, err
}

func (c *RemoteClient) VerifyDataset(ctx context.Context, datasetID string) (DatasetVerifyDTO, error) {
	var out DatasetVerifyDTO
	err := c.doJSON(ctx, http.MethodGet, "/datasets/"+url.PathEscape(datasetID)+"/verify", nil, nil, &out)
	return out, err
}

func (c *RemoteClient) CreateRun(ctx context.Context, run RunResultDTO) (CreateRunResponse, error) {
	var out CreateRunResponse
	err := c.doJSON(ctx, http.MethodPost, "/runs", nil, run, &out)
	return out, err
}

func (c *RemoteClient) GetRun(ctx context.Context, runID string) (RunResultDTO, error) {
	var out RunResultDTO
	err := c.doJSON(ctx, http.MethodGet, "/runs/"+url.PathEscape(runID), nil, nil, &out)
	return out, err
}

func (c *RemoteClient) ListRuns(ctx context.Context, query ListRunsQuery) (ListRunsResponse, error) {
	values := url.Values{}
	addIfNotEmpty(values, "skill", query.Skill)
	addIfNotEmpty(values, "dataset_id", query.DatasetID)
	addIfNotEmpty(values, "from", query.From)
	addIfNotEmpty(values, "to", query.To)
	if query.MinScore != nil {
		values.Set("min_score", strconv.FormatFloat(*query.MinScore, 'f', -1, 64))
	}
	addIfPositive(values, "limit", query.Limit)
	addIfNonNegative(values, "offset", query.Offset)

	var out ListRunsResponse
	err := c.doJSON(ctx, http.MethodGet, "/runs", values, nil, &out)
	return out, err
}

func (c *RemoteClient) GateRun(ctx context.Context, runID string, req GateRunRequest) (GateRunResponse, error) {
	var out GateRunResponse
	err := c.doJSON(ctx, http.MethodPost, "/runs/"+url.PathEscape(runID)+"/gate", nil, req, &out)
	return out, err
}

func (c *RemoteClient) doJSON(ctx context.Context, method, path string, query url.Values, in, out any) error {
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var body io.Reader
	if in != nil {
		payload, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeAPIError(resp)
	}

	if out == nil {
		io.Copy(io.Discard, resp.Body)
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func decodeAPIError(resp *http.Response) error {
	data, _ := io.ReadAll(resp.Body)
	var errPayload ErrorResponse
	if err := json.Unmarshal(data, &errPayload); err == nil && errPayload.Error.Code != 0 {
		return &APIError{StatusCode: resp.StatusCode, Payload: errPayload.Error}
	}

	payload := APIErrorPayload{Code: resp.StatusCode, Message: strings.TrimSpace(string(data))}
	if payload.Message == "" {
		payload.Message = resp.Status
	}
	return &APIError{StatusCode: resp.StatusCode, Payload: payload}
}

func shouldFallback(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrRemoteDisabled) {
		return true
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode >= 500 || apiErr.StatusCode == 429
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return true
	}
	return false
}

func addIfNotEmpty(values url.Values, key, value string) {
	if strings.TrimSpace(value) != "" {
		values.Set(key, value)
	}
}

func addIfPositive(values url.Values, key string, value int) {
	if value > 0 {
		values.Set(key, strconv.Itoa(value))
	}
}

func addIfNonNegative(values url.Values, key string, value int) {
	if value >= 0 {
		values.Set(key, strconv.Itoa(value))
	}
}
