# Makefile for KB Services Unified Monitoring
.PHONY: help deploy start stop restart status health logs clean dashboard prometheus grafana alertmanager

# Default target
help: ## Show this help message
	@echo "KB Services Unified Monitoring"
	@echo "=============================="
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Quick Start:"
	@echo "  make deploy        # Deploy complete monitoring stack"
	@echo "  make dashboard     # Open Grafana dashboard"
	@echo "  make health        # Check all service health"

# Deployment targets
deploy: ## Deploy the complete monitoring stack
	@echo "🚀 Deploying KB Services Unified Monitoring..."
	@chmod +x deploy-monitoring.sh
	@./deploy-monitoring.sh

start: ## Start the monitoring stack
	@echo "▶️  Starting monitoring services..."
	@docker-compose -f docker-compose.monitoring.yml up -d
	@echo "✅ Monitoring services started"

stop: ## Stop the monitoring stack
	@echo "⏹️  Stopping monitoring services..."
	@docker-compose -f docker-compose.monitoring.yml down
	@echo "✅ Monitoring services stopped"

restart: ## Restart the monitoring stack
	@echo "🔄 Restarting monitoring services..."
	@docker-compose -f docker-compose.monitoring.yml restart
	@echo "✅ Monitoring services restarted"

# Status and monitoring targets
status: ## Show status of all monitoring containers
	@echo "📊 Monitoring Stack Status:"
	@echo "=========================="
	@docker-compose -f docker-compose.monitoring.yml ps

health: ## Check health of all monitoring services
	@echo "🏥 Health Check Results:"
	@echo "======================"
	@echo -n "Prometheus:    "; curl -s -o /dev/null -w "%{http_code}" http://localhost:9090/-/healthy || echo "DOWN"
	@echo -n "Grafana:       "; curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/api/health || echo "DOWN"
	@echo -n "Alertmanager:  "; curl -s -o /dev/null -w "%{http_code}" http://localhost:9093/-/healthy || echo "DOWN"
	@echo -n "Loki:          "; curl -s -o /dev/null -w "%{http_code}" http://localhost:3100/ready || echo "DOWN"
	@echo -n "Jaeger:        "; curl -s -o /dev/null -w "%{http_code}" http://localhost:16686/ || echo "DOWN"
	@echo -n "VictoriaMetrics: "; curl -s -o /dev/null -w "%{http_code}" http://localhost:8428/health || echo "DOWN"

logs: ## Show logs from all services (use SERVICE=name for specific service)
ifdef SERVICE
	@echo "📝 Logs for $(SERVICE):"
	@docker-compose -f docker-compose.monitoring.yml logs -f $(SERVICE)
else
	@echo "📝 All monitoring logs:"
	@docker-compose -f docker-compose.monitoring.yml logs -f
endif

# Individual service targets
prometheus: ## Open Prometheus web interface
	@echo "🔍 Opening Prometheus at http://localhost:9090"
	@command -v xdg-open >/dev/null 2>&1 && xdg-open http://localhost:9090 || \
	 command -v open >/dev/null 2>&1 && open http://localhost:9090 || \
	 echo "Please open http://localhost:9090 in your browser"

grafana: ## Open Grafana dashboard
	@echo "📊 Opening Grafana at http://localhost:3000"
	@echo "   Username: admin"
	@echo "   Password: kb-admin-2024"
	@command -v xdg-open >/dev/null 2>&1 && xdg-open http://localhost:3000 || \
	 command -v open >/dev/null 2>&1 && open http://localhost:3000 || \
	 echo "Please open http://localhost:3000 in your browser"

dashboard: ## Open unified KB dashboard directly
	@echo "📈 Opening KB Services Unified Dashboard..."
	@command -v xdg-open >/dev/null 2>&1 && xdg-open "http://localhost:3000/d/kb-unified" || \
	 command -v open >/dev/null 2>&1 && open "http://localhost:3000/d/kb-unified" || \
	 echo "Please open http://localhost:3000/d/kb-unified in your browser"

alertmanager: ## Open Alertmanager web interface  
	@echo "🚨 Opening Alertmanager at http://localhost:9093"
	@command -v xdg-open >/dev/null 2>&1 && xdg-open http://localhost:9093 || \
	 command -v open >/dev/null 2>&1 && open http://localhost:9093 || \
	 echo "Please open http://localhost:9093 in your browser"

jaeger: ## Open Jaeger tracing interface
	@echo "🕸️  Opening Jaeger at http://localhost:16686"
	@command -v xdg-open >/dev/null 2>&1 && xdg-open http://localhost:16686 || \
	 command -v open >/dev/null 2>&1 && open http://localhost:16686 || \
	 echo "Please open http://localhost:16686 in your browser"

# Configuration and maintenance targets
validate-config: ## Validate monitoring configurations
	@echo "🔧 Validating monitoring configurations..."
	@echo "Checking Prometheus config..."
	@docker run --rm -v $(PWD)/prometheus-unified.yml:/etc/prometheus/prometheus.yml prom/prometheus:v2.45.0 promtool check config /etc/prometheus/prometheus.yml
	@echo "Checking Alertmanager config..."
	@docker run --rm -v $(PWD)/alertmanager.yml:/etc/alertmanager/alertmanager.yml prom/alertmanager:v0.26.0 amtool check-config /etc/alertmanager/alertmanager.yml
	@echo "✅ All configurations are valid"

reload-config: ## Reload Prometheus and Alertmanager configurations
	@echo "🔄 Reloading configurations..."
	@echo "Reloading Prometheus..."
	@curl -X POST http://localhost:9090/-/reload || echo "❌ Failed to reload Prometheus"
	@echo "Reloading Alertmanager..."
	@curl -X POST http://localhost:9093/-/reload || echo "❌ Failed to reload Alertmanager"
	@echo "✅ Configuration reload completed"

update-images: ## Update all Docker images to latest versions
	@echo "📥 Updating monitoring Docker images..."
	@docker-compose -f docker-compose.monitoring.yml pull
	@echo "✅ Images updated"

backup: ## Backup monitoring data
	@echo "💾 Creating backup of monitoring data..."
	@mkdir -p backups/$(shell date +%Y%m%d_%H%M%S)
	@docker run --rm -v kb-monitoring_prometheus-data:/data -v $(PWD)/backups/$(shell date +%Y%m%d_%H%M%S):/backup alpine tar czf /backup/prometheus-data.tar.gz -C /data .
	@docker run --rm -v kb-monitoring_grafana-data:/data -v $(PWD)/backups/$(shell date +%Y%m%d_%H%M%S):/backup alpine tar czf /backup/grafana-data.tar.gz -C /data .
	@echo "✅ Backup completed in backups/$(shell date +%Y%m%d_%H%M%S)/"

# Cleanup targets
clean: ## Stop and remove all containers and volumes
	@echo "🧹 Cleaning up monitoring stack..."
	@docker-compose -f docker-compose.monitoring.yml down -v --remove-orphans
	@docker system prune -f
	@echo "✅ Cleanup completed"

clean-volumes: ## Remove only the monitoring volumes (keeps containers)
	@echo "🗑️  Removing monitoring volumes..."
	@docker volume rm -f kb-monitoring_prometheus-data kb-monitoring_grafana-data kb-monitoring_alertmanager-data kb-monitoring_loki-data kb-monitoring_victoria-data 2>/dev/null || true
	@echo "✅ Volumes removed"

reset: ## Complete reset (clean + redeploy)
	@echo "🔄 Performing complete reset..."
	@$(MAKE) clean
	@$(MAKE) deploy
	@echo "✅ Reset completed"

# Development and testing targets
dev-mode: ## Start monitoring in development mode with debug logging
	@echo "🔧 Starting in development mode..."
	@PROMETHEUS_LOG_LEVEL=debug GRAFANA_LOG_LEVEL=debug docker-compose -f docker-compose.monitoring.yml up -d
	@echo "✅ Development mode started"

test-alerts: ## Test alert configurations by sending test alerts
	@echo "🧪 Testing alert configurations..."
	@curl -XPOST http://localhost:9093/api/v1/alerts \
	  -H 'Content-Type: application/json' \
	  -d '[{"labels":{"alertname":"TestAlert","service":"kb-test","severity":"warning"},"annotations":{"summary":"Test alert from Makefile","description":"This is a test alert to verify alerting is working"},"startsAt":"'$(shell date -u +%Y-%m-%dT%H:%M:%S.000Z)'","endsAt":"'$(shell date -u -d '+1 minute' +%Y-%m-%dT%H:%M:%S.000Z)'"}]'
	@echo "✅ Test alert sent"

generate-load: ## Generate test load for monitoring (requires curl and wrk)
	@echo "📈 Generating test load on monitoring endpoints..."
	@echo "Testing Prometheus queries..."
	@for i in {1..10}; do curl -s "http://localhost:9090/api/v1/query?query=up" > /dev/null & done
	@echo "Testing Grafana dashboards..."
	@for i in {1..5}; do curl -s -u admin:kb-admin-2024 "http://localhost:3000/api/dashboards/home" > /dev/null & done
	@wait
	@echo "✅ Load generation completed"

# Documentation targets
docs: ## Show monitoring documentation
	@echo "📚 KB Services Monitoring Documentation"
	@echo "======================================"
	@echo ""
	@echo "🔗 Service URLs:"
	@echo "  • Grafana Dashboard:    http://localhost:3000"
	@echo "  • Prometheus:           http://localhost:9090"  
	@echo "  • Alertmanager:         http://localhost:9093"
	@echo "  • Jaeger Tracing:       http://localhost:16686"
	@echo "  • Loki Logs:            http://localhost:3100"
	@echo "  • VictoriaMetrics:      http://localhost:8428"
	@echo ""
	@echo "👤 Default Credentials:"
	@echo "  • Grafana: admin / kb-admin-2024"
	@echo ""
	@echo "📊 Key Dashboards:"
	@echo "  • Unified KB Dashboard: http://localhost:3000/d/kb-unified"
	@echo "  • Service Health:       http://localhost:3000/d/kb-health"
	@echo "  • Database Performance: http://localhost:3000/d/kb-databases"
	@echo ""
	@echo "🔧 Configuration Files:"
	@echo "  • Prometheus:     prometheus-unified.yml"
	@echo "  • Alert Rules:    alert_rules.yml"
	@echo "  • Recording Rules: recording_rules.yml" 
	@echo "  • Alertmanager:   alertmanager.yml"
	@echo "  • Dashboard:      unified-kb-dashboard.json"

version: ## Show version information
	@echo "KB Services Unified Monitoring v1.0.0"
	@echo "Components:"
	@docker-compose -f docker-compose.monitoring.yml config --services | while read service; do \
		image=$$(docker-compose -f docker-compose.monitoring.yml config | grep -A1 "^  $$service:" | grep image | awk '{print $$2}'); \
		echo "  $$service: $$image"; \
	done

# Monitoring-specific targets for KB services
kb-health: ## Check health of all KB services from monitoring perspective
	@echo "🏥 KB Services Health from Monitoring:"
	@echo "===================================="
	@curl -s 'http://localhost:9090/api/v1/query?query=up{job=~"kb-.*"}' | jq -r '.data.result[] | "\(.metric.job): \(if .value[1] == "1" then "✅ UP" else "❌ DOWN" end)"' 2>/dev/null || echo "❌ Cannot connect to Prometheus"

kb-metrics: ## Show current KB services metrics
	@echo "📊 Current KB Services Metrics:"
	@echo "=============================="
	@curl -s 'http://localhost:9090/api/v1/query?query=kb:request_rate_5m' | jq -r '.data.result[] | "\(.metric.job): \(.value[1]) req/sec"' 2>/dev/null || echo "❌ Cannot fetch metrics"

kb-alerts: ## Show active alerts for KB services
	@echo "🚨 Active KB Services Alerts:"
	@echo "==========================="
	@curl -s 'http://localhost:9093/api/v1/alerts' | jq -r '.data[] | select(.labels.job | contains("kb-")) | "\(.labels.alertname) (\(.labels.job)): \(.annotations.summary)"' 2>/dev/null || echo "❌ Cannot connect to Alertmanager"

# Windows compatibility targets
deploy-windows: ## Deploy monitoring stack (Windows compatible)
	@echo "🚀 Deploying KB Services Unified Monitoring (Windows)..."
	@powershell -Command "docker-compose -f docker-compose.monitoring.yml up -d"
	@echo "✅ Monitoring services started"

health-windows: ## Health check (Windows compatible)
	@echo "🏥 Health Check (Windows):"
	@powershell -Command "$$services = @('prometheus:9090', 'grafana:3000', 'alertmanager:9093'); foreach($$service in $$services) { $$name, $$port = $$service.Split(':'); try { Invoke-WebRequest -Uri \"http://localhost:$$port\" -UseBasicParsing -TimeoutSec 5 | Out-Null; Write-Host \"✅ $$name is healthy\" } catch { Write-Host \"❌ $$name is not responding\" } }"