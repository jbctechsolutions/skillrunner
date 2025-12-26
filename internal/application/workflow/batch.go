// Package workflow provides the workflow executor for skill execution.
package workflow

import (
	"context"
	"sync"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
)

// BatchRequest represents a request waiting to be batched.
type BatchRequest struct {
	Request   ports.CompletionRequest
	ResultCh  chan BatchResult
	Ctx       context.Context
	CreatedAt time.Time
}

// BatchResult contains the response or error for a batched request.
type BatchResult struct {
	Response *ports.CompletionResponse
	Error    error
}

// BatchConfig holds configuration for the batch aggregator.
type BatchConfig struct {
	MaxBatchSize int           // Maximum number of requests per batch
	MaxWaitTime  time.Duration // Maximum time to wait for batch to fill
	Enabled      bool          // Whether batching is enabled
}

// DefaultBatchConfig returns the default batch configuration.
func DefaultBatchConfig() BatchConfig {
	return BatchConfig{
		MaxBatchSize: 10,
		MaxWaitTime:  100 * time.Millisecond,
		Enabled:      true,
	}
}

// BatchAggregator collects requests and executes them in batches.
// This is useful for providers that support batch APIs or for reducing
// overhead when multiple requests can be combined.
type BatchAggregator struct {
	provider ports.ProviderPort
	config   BatchConfig
	mu       sync.Mutex
	pending  []*BatchRequest
	timer    *time.Timer
	stopped  bool

	// Statistics
	batchCount   int64
	requestCount int64
}

// NewBatchAggregator creates a new batch aggregator.
func NewBatchAggregator(provider ports.ProviderPort, config BatchConfig) *BatchAggregator {
	return &BatchAggregator{
		provider: provider,
		config:   config,
		pending:  make([]*BatchRequest, 0, config.MaxBatchSize),
	}
}

// Submit adds a request to the batch queue and waits for the result.
func (b *BatchAggregator) Submit(ctx context.Context, req ports.CompletionRequest) (*ports.CompletionResponse, error) {
	if !b.config.Enabled || b.config.MaxBatchSize <= 1 {
		// Batching disabled or not useful, execute directly
		return b.provider.Complete(ctx, req)
	}

	resultCh := make(chan BatchResult, 1)
	batchReq := &BatchRequest{
		Request:   req,
		ResultCh:  resultCh,
		Ctx:       ctx,
		CreatedAt: time.Now(),
	}

	b.addToBatch(batchReq)

	// Wait for result
	select {
	case result := <-resultCh:
		return result.Response, result.Error
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// addToBatch adds a request to the pending batch.
func (b *BatchAggregator) addToBatch(req *BatchRequest) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.stopped {
		req.ResultCh <- BatchResult{Error: context.Canceled}
		return
	}

	b.pending = append(b.pending, req)
	b.requestCount++

	// If batch is full, execute immediately
	if len(b.pending) >= b.config.MaxBatchSize {
		b.executeBatchLocked()
		return
	}

	// If this is the first request, start the timer
	if len(b.pending) == 1 {
		b.timer = time.AfterFunc(b.config.MaxWaitTime, b.timerExpired)
	}
}

// timerExpired is called when the wait timer expires.
func (b *BatchAggregator) timerExpired() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.pending) > 0 {
		b.executeBatchLocked()
	}
}

// executeBatchLocked executes the current batch. Must be called with lock held.
func (b *BatchAggregator) executeBatchLocked() {
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}

	if len(b.pending) == 0 {
		return
	}

	// Take all pending requests
	batch := b.pending
	b.pending = make([]*BatchRequest, 0, b.config.MaxBatchSize)
	b.batchCount++

	// Execute batch in goroutine to release the lock
	go b.processBatch(batch)
}

// processBatch processes a batch of requests.
func (b *BatchAggregator) processBatch(batch []*BatchRequest) {
	// Currently, we execute each request individually since most providers
	// don't support true batch APIs. However, this architecture allows
	// us to easily add batch API support for providers that have it.
	//
	// The benefit of batching here is reduced lock contention and the
	// ability to implement provider-specific optimizations.

	var wg sync.WaitGroup
	for _, req := range batch {
		wg.Add(1)
		go func(r *BatchRequest) {
			defer wg.Done()
			b.processRequest(r)
		}(req)
	}
	wg.Wait()
}

// processRequest processes a single request from the batch.
func (b *BatchAggregator) processRequest(req *BatchRequest) {
	// Check if context is already cancelled
	if err := req.Ctx.Err(); err != nil {
		req.ResultCh <- BatchResult{Error: err}
		return
	}

	// Execute the request
	resp, err := b.provider.Complete(req.Ctx, req.Request)
	req.ResultCh <- BatchResult{
		Response: resp,
		Error:    err,
	}
}

// Stop stops the batch aggregator and cancels any pending requests.
func (b *BatchAggregator) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.stopped = true

	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}

	// Cancel all pending requests
	for _, req := range b.pending {
		req.ResultCh <- BatchResult{Error: context.Canceled}
	}
	b.pending = nil
}

// Stats returns batch aggregator statistics.
func (b *BatchAggregator) Stats() (batchCount, requestCount int64, avgBatchSize float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.batchCount > 0 {
		avgBatchSize = float64(b.requestCount) / float64(b.batchCount)
	}
	return b.batchCount, b.requestCount, avgBatchSize
}

// PendingCount returns the number of pending requests.
func (b *BatchAggregator) PendingCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.pending)
}

// BatchingProvider wraps a provider with batch aggregation support.
type BatchingProvider struct {
	provider   ports.ProviderPort
	aggregator *BatchAggregator
}

// NewBatchingProvider creates a new batching provider wrapper.
func NewBatchingProvider(provider ports.ProviderPort, config BatchConfig) *BatchingProvider {
	return &BatchingProvider{
		provider:   provider,
		aggregator: NewBatchAggregator(provider, config),
	}
}

// Info returns provider metadata.
func (b *BatchingProvider) Info() ports.ProviderInfo {
	return b.provider.Info()
}

// ListModels returns available models.
func (b *BatchingProvider) ListModels(ctx context.Context) ([]string, error) {
	return b.provider.ListModels(ctx)
}

// SupportsModel checks if the provider supports a model.
func (b *BatchingProvider) SupportsModel(ctx context.Context, modelID string) (bool, error) {
	return b.provider.SupportsModel(ctx, modelID)
}

// IsAvailable checks if a model is available.
func (b *BatchingProvider) IsAvailable(ctx context.Context, modelID string) (bool, error) {
	return b.provider.IsAvailable(ctx, modelID)
}

// Complete executes a completion request through the batch aggregator.
func (b *BatchingProvider) Complete(ctx context.Context, req ports.CompletionRequest) (*ports.CompletionResponse, error) {
	return b.aggregator.Submit(ctx, req)
}

// Stream streams a completion response (not batched, passed directly to provider).
func (b *BatchingProvider) Stream(ctx context.Context, req ports.CompletionRequest, cb ports.StreamCallback) (*ports.CompletionResponse, error) {
	// Streaming requests bypass batching
	return b.provider.Stream(ctx, req, cb)
}

// HealthCheck checks provider health.
func (b *BatchingProvider) HealthCheck(ctx context.Context, modelID string) (*ports.HealthStatus, error) {
	return b.provider.HealthCheck(ctx, modelID)
}

// Close stops the batch aggregator.
func (b *BatchingProvider) Close() {
	b.aggregator.Stop()
}

// Ensure BatchingProvider implements ProviderPort
var _ ports.ProviderPort = (*BatchingProvider)(nil)
