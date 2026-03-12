package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// startTime records when the server started.
var startTime = time.Now()

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Version   string            `json:"version"`
	Uptime    string            `json:"uptime"`
	Timestamp string            `json:"timestamp"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// ReadyResponse represents the readiness check response.
type ReadyResponse struct {
	Ready     bool              `json:"ready"`
	Service   string            `json:"service"`
	Checks    map[string]bool   `json:"checks"`
	Timestamp string            `json:"timestamp"`
}

// LiveResponse represents the liveness check response.
type LiveResponse struct {
	Alive     bool   `json:"alive"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}

// MetricsResponse represents service metrics.
type MetricsResponse struct {
	Service      string                 `json:"service"`
	Uptime       string                 `json:"uptime"`
	Calculations map[string]int64       `json:"calculations"`
	Memory       MemoryStats            `json:"memory"`
	Goroutines   int                    `json:"goroutines"`
	Timestamp    string                 `json:"timestamp"`
}

// MemoryStats represents memory statistics.
type MemoryStats struct {
	Alloc      uint64 `json:"alloc"`      // bytes allocated and in use
	TotalAlloc uint64 `json:"totalAlloc"` // bytes allocated (even if freed)
	Sys        uint64 `json:"sys"`        // bytes obtained from system
	NumGC      uint32 `json:"numGC"`      // number of GC cycles
}

// healthHandler handles GET /health
//
// @Summary Health Check
// @Description Returns the health status of the service
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (s *Server) healthHandler(c *gin.Context) {
	uptime := time.Since(startTime).Round(time.Second)

	response := HealthResponse{
		Status:    "healthy",
		Service:   "kb-8-calculator-service",
		Version:   "1.0.0",
		Uptime:    uptime.String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks: map[string]string{
			"egfr_calculator": "available",
			"crcl_calculator": "available",
			"bmi_calculator":  "available",
		},
	}

	c.JSON(http.StatusOK, response)
}

// readyHandler handles GET /ready
//
// @Summary Readiness Check
// @Description Returns whether the service is ready to accept traffic
// @Tags health
// @Produce json
// @Success 200 {object} ReadyResponse
// @Failure 503 {object} ReadyResponse
// @Router /ready [get]
func (s *Server) readyHandler(c *gin.Context) {
	// Check all dependencies
	checks := map[string]bool{
		"calculator_service": s.service != nil,
		"config_loaded":      s.cfg != nil,
	}

	// Determine overall readiness
	ready := true
	for _, ok := range checks {
		if !ok {
			ready = false
			break
		}
	}

	response := ReadyResponse{
		Ready:     ready,
		Service:   "kb-8-calculator-service",
		Checks:    checks,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if ready {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusServiceUnavailable, response)
	}
}

// liveHandler handles GET /live
//
// @Summary Liveness Check
// @Description Returns whether the service is alive (for Kubernetes liveness probes)
// @Tags health
// @Produce json
// @Success 200 {object} LiveResponse
// @Router /live [get]
func (s *Server) liveHandler(c *gin.Context) {
	response := LiveResponse{
		Alive:     true,
		Service:   "kb-8-calculator-service",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

// playgroundHandler serves a simple HTML playground for testing
func (s *Server) playgroundHandler(c *gin.Context) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>KB-8 Calculator Service Playground</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; }
        .section { margin: 20px 0; padding: 20px; background: #f9f9f9; border-radius: 4px; }
        label { display: block; margin: 10px 0 5px; font-weight: 600; }
        input, select { width: 100%; padding: 8px; margin-bottom: 10px; border: 1px solid #ddd; border-radius: 4px; box-sizing: border-box; }
        button { background: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; margin-right: 10px; }
        button:hover { background: #0056b3; }
        pre { background: #2d2d2d; color: #f8f8f2; padding: 15px; border-radius: 4px; overflow-x: auto; }
        .result { margin-top: 20px; }
        .info { background: #e7f3ff; padding: 15px; border-radius: 4px; margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🧮 KB-8 Calculator Service</h1>
        <div class="info">
            <strong>Available Calculators:</strong> eGFR (CKD-EPI 2021), CrCl (Cockcroft-Gault), BMI (Western + Asian)
        </div>

        <div class="section">
            <h2>eGFR Calculator</h2>
            <label>Serum Creatinine (mg/dL)</label>
            <input type="number" id="egfr-cr" step="0.1" value="1.2">
            <label>Age (years)</label>
            <input type="number" id="egfr-age" value="65">
            <label>Sex</label>
            <select id="egfr-sex">
                <option value="male">Male</option>
                <option value="female">Female</option>
            </select>
            <button onclick="calculateEGFR()">Calculate eGFR</button>
        </div>

        <div class="section">
            <h2>CrCl Calculator (Cockcroft-Gault)</h2>
            <label>Serum Creatinine (mg/dL)</label>
            <input type="number" id="crcl-cr" step="0.1" value="1.2">
            <label>Age (years)</label>
            <input type="number" id="crcl-age" value="65">
            <label>Sex</label>
            <select id="crcl-sex">
                <option value="male">Male</option>
                <option value="female">Female</option>
            </select>
            <label>Weight (kg)</label>
            <input type="number" id="crcl-weight" step="0.1" value="70">
            <button onclick="calculateCrCl()">Calculate CrCl</button>
        </div>

        <div class="section">
            <h2>BMI Calculator</h2>
            <label>Weight (kg)</label>
            <input type="number" id="bmi-weight" step="0.1" value="70">
            <label>Height (cm)</label>
            <input type="number" id="bmi-height" step="0.1" value="170">
            <label>Region</label>
            <select id="bmi-region">
                <option value="GLOBAL">Global (WHO Standard)</option>
                <option value="INDIA">India (Asian Cutoffs)</option>
            </select>
            <button onclick="calculateBMI()">Calculate BMI</button>
        </div>

        <div class="result">
            <h3>Result</h3>
            <pre id="result">Click a calculate button to see results...</pre>
        </div>
    </div>

    <script>
        async function fetchCalc(endpoint, data) {
            try {
                const response = await fetch('/api/v1/calculate/' + endpoint, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(data)
                });
                const result = await response.json();
                document.getElementById('result').textContent = JSON.stringify(result, null, 2);
            } catch (error) {
                document.getElementById('result').textContent = 'Error: ' + error.message;
            }
        }

        function calculateEGFR() {
            fetchCalc('egfr', {
                serumCreatinine: parseFloat(document.getElementById('egfr-cr').value),
                ageYears: parseInt(document.getElementById('egfr-age').value),
                sex: document.getElementById('egfr-sex').value
            });
        }

        function calculateCrCl() {
            fetchCalc('crcl', {
                serumCreatinine: parseFloat(document.getElementById('crcl-cr').value),
                ageYears: parseInt(document.getElementById('crcl-age').value),
                sex: document.getElementById('crcl-sex').value,
                weightKg: parseFloat(document.getElementById('crcl-weight').value)
            });
        }

        function calculateBMI() {
            fetchCalc('bmi', {
                weightKg: parseFloat(document.getElementById('bmi-weight').value),
                heightCm: parseFloat(document.getElementById('bmi-height').value),
                region: document.getElementById('bmi-region').value
            });
        }
    </script>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
