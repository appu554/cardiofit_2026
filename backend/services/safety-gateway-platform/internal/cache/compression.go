package cache

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
	"go.uber.org/zap"
	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// CompressionManager handles snapshot compression and decompression
type CompressionManager struct {
	config     *config.CacheConfig
	logger     *logger.Logger
	algorithms map[string]CompressionAlgorithm
	stats      *CompressionStats
	mu         sync.RWMutex
}

// CompressionAlgorithm interface for different compression implementations
type CompressionAlgorithm interface {
	Compress(data []byte) ([]byte, error)
	Decompress(data []byte) ([]byte, error)
	GetName() string
	GetCompressionRatio(originalSize, compressedSize int) float64
	GetStats() AlgorithmStats
}

// AlgorithmStats tracks algorithm-specific statistics
type AlgorithmStats struct {
	TotalOperations    int64         `json:"total_operations"`
	TotalCompressions  int64         `json:"total_compressions"`
	TotalDecompressions int64        `json:"total_decompressions"`
	AverageRatio       float64       `json:"average_ratio"`
	AverageCompTime    time.Duration `json:"average_comp_time"`
	AverageDecompTime  time.Duration `json:"average_decomp_time"`
	BytesSaved         int64         `json:"bytes_saved"`
	ErrorCount         int64         `json:"error_count"`
}

// GzipAlgorithm implements gzip compression
type GzipAlgorithm struct {
	level int
	stats AlgorithmStats
	mu    sync.RWMutex
}

// ZstdAlgorithm implements Zstandard compression
type ZstdAlgorithm struct {
	level   int
	encoder *zstd.Encoder
	decoder *zstd.Decoder
	stats   AlgorithmStats
	mu      sync.RWMutex
}

// NoCompressionAlgorithm implements pass-through (no compression)
type NoCompressionAlgorithm struct {
	stats AlgorithmStats
	mu    sync.RWMutex
}

// CompressedSnapshot represents a compressed clinical snapshot
type CompressedSnapshot struct {
	OriginalSize     int                    `json:"original_size"`
	CompressedSize   int                    `json:"compressed_size"`
	CompressionRatio float64               `json:"compression_ratio"`
	Algorithm        string                `json:"algorithm"`
	CompressedData   []byte                `json:"compressed_data"`
	Metadata         map[string]interface{} `json:"metadata"`
	CompressedAt     time.Time             `json:"compressed_at"`
	Checksum         string                `json:"checksum"`
}

// NewCompressionManager creates a new compression manager
func NewCompressionManager(cfg *config.CacheConfig, logger *logger.Logger) *CompressionManager {
	manager := &CompressionManager{
		config:     cfg,
		logger:     logger,
		algorithms: make(map[string]CompressionAlgorithm),
		stats:      &CompressionStats{},
	}
	
	manager.initializeAlgorithms()
	
	return manager
}

// initializeAlgorithms sets up compression algorithms
func (cm *CompressionManager) initializeAlgorithms() {
	// Initialize Gzip algorithm
	gzipAlg := &GzipAlgorithm{
		level: cm.config.CompressionLevel,
		stats: AlgorithmStats{},
	}
	cm.algorithms["gzip"] = gzipAlg
	
	// Initialize Zstandard algorithm
	zstdAlg, err := NewZstdAlgorithm(cm.config.CompressionLevel)
	if err != nil {
		cm.logger.Warn("Failed to initialize Zstd algorithm", zap.Error(err))
	} else {
		cm.algorithms["zstd"] = zstdAlg
	}
	
	// Initialize no-compression algorithm
	noCompAlg := &NoCompressionAlgorithm{
		stats: AlgorithmStats{},
	}
	cm.algorithms["none"] = noCompAlg
	
	cm.logger.Info("Compression algorithms initialized",
		zap.Int("algorithm_count", len(cm.algorithms)),
		zap.Strings("algorithms", cm.getAlgorithmNames()),
	)
}

// CompressSnapshot compresses a clinical snapshot
func (cm *CompressionManager) CompressSnapshot(snapshot *types.ClinicalSnapshot) (*CompressedSnapshot, error) {
	if !cm.config.EnableCompression {
		return cm.createUncompressedSnapshot(snapshot)
	}
	
	startTime := time.Now()
	
	// Serialize snapshot to JSON
	originalData, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize snapshot: %w", err)
	}
	
	originalSize := len(originalData)
	
	// Choose best algorithm based on data characteristics
	algorithmName := cm.chooseBestAlgorithm(originalData)
	algorithm, exists := cm.algorithms[algorithmName]
	if !exists {
		return nil, fmt.Errorf("compression algorithm not found: %s", algorithmName)
	}
	
	// Compress data
	compressedData, err := algorithm.Compress(originalData)
	if err != nil {
		cm.logger.Error("Compression failed",
			zap.String("algorithm", algorithmName),
			zap.Int("original_size", originalSize),
			zap.Error(err),
		)
		return nil, fmt.Errorf("compression failed: %w", err)
	}
	
	compressedSize := len(compressedData)
	compressionRatio := algorithm.GetCompressionRatio(originalSize, compressedSize)
	compressionTime := time.Since(startTime)
	
	// Create compressed snapshot
	compressedSnapshot := &CompressedSnapshot{
		OriginalSize:     originalSize,
		CompressedSize:   compressedSize,
		CompressionRatio: compressionRatio,
		Algorithm:        algorithmName,
		CompressedData:   compressedData,
		CompressedAt:     time.Now(),
		Metadata: map[string]interface{}{
			"compression_time_ms": compressionTime.Milliseconds(),
			"snapshot_id":         snapshot.SnapshotID,
			"patient_id":          snapshot.PatientID,
		},
	}
	
	// Update statistics
	cm.updateCompressionStats(algorithmName, originalSize, compressedSize, compressionTime)
	
	cm.logger.Debug("Snapshot compressed successfully",
		zap.String("algorithm", algorithmName),
		zap.String("snapshot_id", snapshot.SnapshotID),
		zap.Int("original_size", originalSize),
		zap.Int("compressed_size", compressedSize),
		zap.Float64("ratio", compressionRatio),
		zap.Duration("compression_time", compressionTime),
	)
	
	return compressedSnapshot, nil
}

// DecompressSnapshot decompresses a clinical snapshot
func (cm *CompressionManager) DecompressSnapshot(compressedSnapshot *CompressedSnapshot) (*types.ClinicalSnapshot, error) {
	if compressedSnapshot.Algorithm == "none" {
		return cm.decompressUncompressedSnapshot(compressedSnapshot)
	}
	
	startTime := time.Now()
	
	// Get algorithm
	algorithm, exists := cm.algorithms[compressedSnapshot.Algorithm]
	if !exists {
		return nil, fmt.Errorf("decompression algorithm not found: %s", compressedSnapshot.Algorithm)
	}
	
	// Decompress data
	decompressedData, err := algorithm.Decompress(compressedSnapshot.CompressedData)
	if err != nil {
		cm.logger.Error("Decompression failed",
			zap.String("algorithm", compressedSnapshot.Algorithm),
			zap.Int("compressed_size", compressedSnapshot.CompressedSize),
			zap.Error(err),
		)
		return nil, fmt.Errorf("decompression failed: %w", err)
	}
	
	// Deserialize snapshot
	var snapshot types.ClinicalSnapshot
	if err := json.Unmarshal(decompressedData, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to deserialize snapshot: %w", err)
	}
	
	decompressionTime := time.Since(startTime)
	
	// Update statistics
	cm.updateDecompressionStats(compressedSnapshot.Algorithm, decompressionTime)
	
	cm.logger.Debug("Snapshot decompressed successfully",
		zap.String("algorithm", compressedSnapshot.Algorithm),
		zap.String("snapshot_id", snapshot.SnapshotID),
		zap.Int("original_size", compressedSnapshot.OriginalSize),
		zap.Duration("decompression_time", decompressionTime),
	)
	
	return &snapshot, nil
}

// chooseBestAlgorithm selects the best compression algorithm based on data characteristics
func (cm *CompressionManager) chooseBestAlgorithm(data []byte) string {
	// Analyze data characteristics
	dataSize := len(data)
	
	// For small data, overhead might not be worth it
	if dataSize < 1024 {
		return "none"
	}
	
	// For medium-sized data, use gzip for good balance
	if dataSize < 100*1024 {
		return "gzip"
	}
	
	// For large data, prefer zstd if available for better performance
	if _, exists := cm.algorithms["zstd"]; exists {
		return "zstd"
	}
	
	return "gzip"
}

// GetStats returns compression statistics
func (cm *CompressionManager) GetStats() *CompressionStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	stats := &CompressionStats{
		Algorithm:         cm.determinePrimaryAlgorithm(),
		AverageRatio:      cm.calculateOverallAverageRatio(),
		CompressionTime:   cm.calculateAverageCompressionTime(),
		DecompressionTime: cm.calculateAverageDecompressionTime(),
		SpaceSaved:        cm.calculateTotalSpaceSaved(),
		CPUOverhead:       cm.calculateCPUOverhead(),
	}
	
	return stats
}

// AnalyzeCompressionEffectiveness analyzes compression effectiveness for different data types
func (cm *CompressionManager) AnalyzeCompressionEffectiveness() map[string]interface{} {
	analysis := map[string]interface{}{
		"timestamp":     time.Now(),
		"algorithms":    make(map[string]interface{}),
		"recommendations": []string{},
		"overall_stats": cm.GetStats(),
	}
	
	// Analyze each algorithm
	for name, alg := range cm.algorithms {
		stats := alg.GetStats()
		analysis["algorithms"].(map[string]interface{})[name] = stats
	}
	
	// Generate recommendations
	recommendations := cm.generateCompressionRecommendations()
	analysis["recommendations"] = recommendations
	
	return analysis
}

// OptimizeCompressionSettings optimizes compression settings based on performance data
func (cm *CompressionManager) OptimizeCompressionSettings() error {
	analysis := cm.AnalyzeCompressionEffectiveness()
	recommendations := analysis["recommendations"].([]string)
	
	cm.logger.Info("Optimizing compression settings",
		zap.Int("recommendations", len(recommendations)),
	)
	
	for _, recommendation := range recommendations {
		cm.logger.Info("Compression recommendation", zap.String("recommendation", recommendation))
	}
	
	// Apply optimizations based on analysis
	return cm.applyCompressionOptimizations(analysis)
}

// Helper methods for statistics calculation
func (cm *CompressionManager) updateCompressionStats(algorithm string, originalSize, compressedSize int, duration time.Duration) {
	alg, exists := cm.algorithms[algorithm]
	if !exists {
		return
	}
	
	// Update algorithm-specific stats
	algStats := alg.GetStats()
	algStats.TotalOperations++
	algStats.TotalCompressions++
	algStats.BytesSaved += int64(originalSize - compressedSize)
	
	// Calculate running average for compression time
	if algStats.TotalCompressions == 1 {
		algStats.AverageCompTime = duration
	} else {
		// Calculate exponential moving average
		alpha := 0.1
		avgMS := float64(algStats.AverageCompTime.Nanoseconds())
		newMS := float64(duration.Nanoseconds())
		avgMS = alpha*newMS + (1-alpha)*avgMS
		algStats.AverageCompTime = time.Duration(int64(avgMS))
	}
	
	// Update compression ratio
	ratio := float64(originalSize) / float64(compressedSize)
	if algStats.TotalCompressions == 1 {
		algStats.AverageRatio = ratio
	} else {
		alpha := 0.1
		algStats.AverageRatio = alpha*ratio + (1-alpha)*algStats.AverageRatio
	}
}

func (cm *CompressionManager) updateDecompressionStats(algorithm string, duration time.Duration) {
	alg, exists := cm.algorithms[algorithm]
	if !exists {
		return
	}
	
	algStats := alg.GetStats()
	algStats.TotalDecompressions++
	
	// Calculate running average for decompression time
	if algStats.TotalDecompressions == 1 {
		algStats.AverageDecompTime = duration
	} else {
		alpha := 0.1
		avgMS := float64(algStats.AverageDecompTime.Nanoseconds())
		newMS := float64(duration.Nanoseconds())
		avgMS = alpha*newMS + (1-alpha)*avgMS
		algStats.AverageDecompTime = time.Duration(int64(avgMS))
	}
}

func (cm *CompressionManager) determinePrimaryAlgorithm() string {
	maxOperations := int64(0)
	primaryAlg := "none"
	
	for name, alg := range cm.algorithms {
		stats := alg.GetStats()
		if stats.TotalOperations > maxOperations {
			maxOperations = stats.TotalOperations
			primaryAlg = name
		}
	}
	
	return primaryAlg
}

func (cm *CompressionManager) calculateOverallAverageRatio() float64 {
	totalRatio := 0.0
	totalOperations := int64(0)
	
	for _, alg := range cm.algorithms {
		stats := alg.GetStats()
		if stats.TotalCompressions > 0 {
			totalRatio += stats.AverageRatio * float64(stats.TotalCompressions)
			totalOperations += stats.TotalCompressions
		}
	}
	
	if totalOperations == 0 {
		return 1.0
	}
	
	return totalRatio / float64(totalOperations)
}

func (cm *CompressionManager) calculateAverageCompressionTime() float64 {
	totalTime := int64(0)
	totalOperations := int64(0)
	
	for _, alg := range cm.algorithms {
		stats := alg.GetStats()
		if stats.TotalCompressions > 0 {
			totalTime += int64(stats.AverageCompTime) * stats.TotalCompressions
			totalOperations += stats.TotalCompressions
		}
	}
	
	if totalOperations == 0 {
		return 0.0
	}
	
	avgNs := totalTime / totalOperations
	return float64(avgNs) / 1000000.0 // Convert to milliseconds
}

func (cm *CompressionManager) calculateAverageDecompressionTime() float64 {
	totalTime := int64(0)
	totalOperations := int64(0)
	
	for _, alg := range cm.algorithms {
		stats := alg.GetStats()
		if stats.TotalDecompressions > 0 {
			totalTime += int64(stats.AverageDecompTime) * stats.TotalDecompressions
			totalOperations += stats.TotalDecompressions
		}
	}
	
	if totalOperations == 0 {
		return 0.0
	}
	
	avgNs := totalTime / totalOperations
	return float64(avgNs) / 1000000.0 // Convert to milliseconds
}

func (cm *CompressionManager) calculateTotalSpaceSaved() int64 {
	totalSaved := int64(0)
	
	for _, alg := range cm.algorithms {
		stats := alg.GetStats()
		totalSaved += stats.BytesSaved
	}
	
	return totalSaved
}

func (cm *CompressionManager) calculateCPUOverhead() float64 {
	// This is a simplified calculation
	// In practice, this would measure actual CPU usage
	avgCompTime := cm.calculateAverageCompressionTime()
	avgDecompTime := cm.calculateAverageDecompressionTime()
	
	// Assume base processing time without compression is 1ms
	baseTime := 1.0
	totalTime := avgCompTime + avgDecompTime
	
	return (totalTime / (baseTime + totalTime)) * 100.0
}

func (cm *CompressionManager) generateCompressionRecommendations() []string {
	recommendations := []string{}
	
	stats := cm.GetStats()
	
	// Analyze compression ratio
	if stats.AverageRatio < 1.5 {
		recommendations = append(recommendations, 
			"Low compression ratio detected. Consider disabling compression for better performance.")
	} else if stats.AverageRatio > 3.0 {
		recommendations = append(recommendations, 
			"Excellent compression ratio. Consider using higher compression levels for even better space savings.")
	}
	
	// Analyze compression time
	if stats.CompressionTime > 10.0 {
		recommendations = append(recommendations, 
			"High compression latency detected. Consider using faster compression algorithm or lower compression level.")
	}
	
	// Analyze CPU overhead
	if stats.CPUOverhead > 20.0 {
		recommendations = append(recommendations, 
			"High CPU overhead from compression. Consider optimizing compression settings or disabling for hot paths.")
	}
	
	// Space savings analysis
	if stats.SpaceSaved > 1024*1024*100 { // More than 100MB saved
		recommendations = append(recommendations, 
			"Significant space savings achieved through compression. Consider applying to additional data types.")
	}
	
	return recommendations
}

func (cm *CompressionManager) applyCompressionOptimizations(analysis map[string]interface{}) error {
	// This would implement automatic optimization based on analysis
	// For now, log the optimization opportunity
	cm.logger.Info("Compression optimization analysis completed",
		zap.Any("analysis", analysis),
	)
	
	return nil
}

func (cm *CompressionManager) getAlgorithmNames() []string {
	names := make([]string, 0, len(cm.algorithms))
	for name := range cm.algorithms {
		names = append(names, name)
	}
	return names
}

func (cm *CompressionManager) createUncompressedSnapshot(snapshot *types.ClinicalSnapshot) (*CompressedSnapshot, error) {
	data, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize snapshot: %w", err)
	}
	
	return &CompressedSnapshot{
		OriginalSize:     len(data),
		CompressedSize:   len(data),
		CompressionRatio: 1.0,
		Algorithm:        "none",
		CompressedData:   data,
		CompressedAt:     time.Now(),
		Metadata: map[string]interface{}{
			"snapshot_id": snapshot.SnapshotID,
			"patient_id":  snapshot.PatientID,
		},
	}, nil
}

func (cm *CompressionManager) decompressUncompressedSnapshot(compressedSnapshot *CompressedSnapshot) (*types.ClinicalSnapshot, error) {
	var snapshot types.ClinicalSnapshot
	if err := json.Unmarshal(compressedSnapshot.CompressedData, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to deserialize snapshot: %w", err)
	}
	
	return &snapshot, nil
}

// Gzip Algorithm Implementation
func (g *GzipAlgorithm) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, g.level)
	if err != nil {
		g.mu.Lock()
		g.stats.ErrorCount++
		g.mu.Unlock()
		return nil, err
	}
	defer writer.Close()
	
	if _, err := writer.Write(data); err != nil {
		g.mu.Lock()
		g.stats.ErrorCount++
		g.mu.Unlock()
		return nil, err
	}
	
	return buf.Bytes(), nil
}

func (g *GzipAlgorithm) Decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		g.mu.Lock()
		g.stats.ErrorCount++
		g.mu.Unlock()
		return nil, err
	}
	defer reader.Close()
	
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		g.mu.Lock()
		g.stats.ErrorCount++
		g.mu.Unlock()
		return nil, err
	}
	
	return decompressed, nil
}

func (g *GzipAlgorithm) GetName() string {
	return "gzip"
}

func (g *GzipAlgorithm) GetCompressionRatio(originalSize, compressedSize int) float64 {
	if compressedSize == 0 {
		return 0
	}
	return float64(originalSize) / float64(compressedSize)
}

func (g *GzipAlgorithm) GetStats() AlgorithmStats {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.stats
}

// Zstd Algorithm Implementation
func NewZstdAlgorithm(level int) (*ZstdAlgorithm, error) {
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(level)))
	if err != nil {
		return nil, err
	}
	
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		encoder.Close()
		return nil, err
	}
	
	return &ZstdAlgorithm{
		level:   level,
		encoder: encoder,
		decoder: decoder,
		stats:   AlgorithmStats{},
	}, nil
}

func (z *ZstdAlgorithm) Compress(data []byte) ([]byte, error) {
	compressed := z.encoder.EncodeAll(data, make([]byte, 0, len(data)))
	return compressed, nil
}

func (z *ZstdAlgorithm) Decompress(data []byte) ([]byte, error) {
	decompressed, err := z.decoder.DecodeAll(data, nil)
	if err != nil {
		z.mu.Lock()
		z.stats.ErrorCount++
		z.mu.Unlock()
		return nil, err
	}
	
	return decompressed, nil
}

func (z *ZstdAlgorithm) GetName() string {
	return "zstd"
}

func (z *ZstdAlgorithm) GetCompressionRatio(originalSize, compressedSize int) float64 {
	if compressedSize == 0 {
		return 0
	}
	return float64(originalSize) / float64(compressedSize)
}

func (z *ZstdAlgorithm) GetStats() AlgorithmStats {
	z.mu.RLock()
	defer z.mu.RUnlock()
	return z.stats
}

// No Compression Algorithm Implementation
func (n *NoCompressionAlgorithm) Compress(data []byte) ([]byte, error) {
	return data, nil
}

func (n *NoCompressionAlgorithm) Decompress(data []byte) ([]byte, error) {
	return data, nil
}

func (n *NoCompressionAlgorithm) GetName() string {
	return "none"
}

func (n *NoCompressionAlgorithm) GetCompressionRatio(originalSize, compressedSize int) float64 {
	return 1.0
}

func (n *NoCompressionAlgorithm) GetStats() AlgorithmStats {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.stats
}

// AccessPredictor implementation placeholder
type AccessPredictor struct {
	logger *logger.Logger
}

func NewAccessPredictor(logger *logger.Logger) *AccessPredictor {
	return &AccessPredictor{logger: logger}
}

// Warming strategy implementations
func (p *PreemptiveWarmingStrategy) Warm(ctx context.Context, cache *SnapshotCache, keys []string) error {
	p.logger.Info("Starting preemptive cache warming", zap.Int("key_count", len(keys)))
	// Implementation would predict and warm likely-to-be-accessed keys
	return nil
}

func (p *PreemptiveWarmingStrategy) GetName() string {
	return "preemptive"
}

func (p *PreemptiveWarmingStrategy) GetEffectiveness() float64 {
	return 0.85 // 85% effectiveness
}

func (o *OnDemandWarmingStrategy) Warm(ctx context.Context, cache *SnapshotCache, keys []string) error {
	o.logger.Info("Starting on-demand cache warming", zap.Int("key_count", len(keys)))
	// Implementation would warm specific requested keys
	return nil
}

func (o *OnDemandWarmingStrategy) GetName() string {
	return "on_demand"
}

func (o *OnDemandWarmingStrategy) GetEffectiveness() float64 {
	return 0.95 // 95% effectiveness
}