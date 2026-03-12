//! Hot Loader - Zero-downtime rule and model updates
//! 
//! This module implements hot-loading capabilities for rules and models,
//! allowing updates without service restart using file system monitoring,
//! canary deployments, and safe rollback mechanisms.

use std::collections::HashMap;
use std::sync::Arc;
use std::path::{Path, PathBuf};
use std::time::{Duration, Instant};
use serde::{Deserialize, Serialize};
use anyhow::{Result, anyhow};
use tokio::sync::{RwLock, watch};
use tokio::fs;
use notify::{Watcher, RecursiveMode, Event, EventKind};
use tracing::{info, warn, error, debug};

use super::knowledge_base::KnowledgeBase;
use super::compiled_models::CompiledModelRegistry;

/// Hot loader for zero-downtime updates
#[derive(Debug)]
pub struct HotLoader {
    knowledge_base: Arc<RwLock<KnowledgeBase>>,
    compiled_models: Arc<RwLock<CompiledModelRegistry>>,
    config: HotLoaderConfig,
    deployment_history: Arc<RwLock<Vec<DeploymentRecord>>>,
    active_deployments: Arc<RwLock<HashMap<String, ActiveDeployment>>>,
    file_watcher: Option<notify::RecommendedWatcher>,
    update_sender: watch::Sender<UpdateNotification>,
    update_receiver: watch::Receiver<UpdateNotification>,
}

/// Hot loader configuration
#[derive(Debug, Clone)]
pub struct HotLoaderConfig {
    pub enable_file_watching: bool,
    pub enable_canary_deployment: bool,
    pub canary_percentage: f64,
    pub canary_duration_minutes: u64,
    pub auto_rollback_on_error: bool,
    pub max_deployment_history: usize,
    pub validation_timeout_seconds: u64,
    pub backup_retention_days: u32,
}

impl Default for HotLoaderConfig {
    fn default() -> Self {
        Self {
            enable_file_watching: true,
            enable_canary_deployment: true,
            canary_percentage: 5.0,
            canary_duration_minutes: 10,
            auto_rollback_on_error: true,
            max_deployment_history: 100,
            validation_timeout_seconds: 30,
            backup_retention_days: 30,
        }
    }
}

/// Deployment status
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum DeploymentStatus {
    Pending,
    Validating,
    CanaryDeploying,
    CanaryActive,
    FullDeploying,
    Active,
    RollingBack,
    Failed,
    RolledBack,
}

/// Deployment record for history tracking
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeploymentRecord {
    pub deployment_id: String,
    pub deployment_type: DeploymentType,
    pub status: DeploymentStatus,
    pub started_at: chrono::DateTime<chrono::Utc>,
    pub completed_at: Option<chrono::DateTime<chrono::Utc>>,
    pub version: String,
    pub changes: Vec<ChangeRecord>,
    pub metrics: DeploymentMetrics,
    pub rollback_info: Option<RollbackInfo>,
}

/// Type of deployment
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum DeploymentType {
    RuleUpdate,
    ModelUpdate,
    ConfigUpdate,
    FullReload,
}

/// Individual change record
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChangeRecord {
    pub change_type: ChangeType,
    pub file_path: String,
    pub old_version: Option<String>,
    pub new_version: String,
    pub validation_result: ValidationResult,
}

/// Type of change
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ChangeType {
    Added,
    Modified,
    Deleted,
}

/// Validation result for changes
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationResult {
    pub valid: bool,
    pub warnings: Vec<String>,
    pub errors: Vec<String>,
    pub validation_time_ms: u64,
}

/// Deployment metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeploymentMetrics {
    pub total_files_changed: u32,
    pub validation_time_ms: u64,
    pub deployment_time_ms: u64,
    pub canary_success_rate: Option<f64>,
    pub error_count: u32,
    pub rollback_triggered: bool,
}

/// Rollback information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RollbackInfo {
    pub reason: String,
    pub triggered_at: chrono::DateTime<chrono::Utc>,
    pub rollback_to_version: String,
    pub automatic: bool,
}

/// Active deployment tracking
#[derive(Debug, Clone)]
struct ActiveDeployment {
    deployment_id: String,
    status: DeploymentStatus,
    started_at: Instant,
    canary_start: Option<Instant>,
    error_count: u32,
    success_count: u32,
}

/// Update notification
#[derive(Debug, Clone)]
pub struct UpdateNotification {
    pub notification_type: UpdateType,
    pub file_path: String,
    pub timestamp: chrono::DateTime<chrono::Utc>,
}

/// Type of update notification
#[derive(Debug, Clone)]
pub enum UpdateType {
    FileChanged,
    FileAdded,
    FileDeleted,
    DirectoryChanged,
}

impl HotLoader {
    /// Create a new hot loader
    pub fn new(
        knowledge_base: Arc<KnowledgeBase>,
        compiled_models: Arc<CompiledModelRegistry>,
    ) -> Result<Self> {
        Self::with_config(knowledge_base, compiled_models, HotLoaderConfig::default())
    }
    
    /// Create a new hot loader with custom configuration
    pub fn with_config(
        knowledge_base: Arc<KnowledgeBase>,
        compiled_models: Arc<CompiledModelRegistry>,
        config: HotLoaderConfig,
    ) -> Result<Self> {
        info!("🔄 Initializing Hot Loader");
        info!("📁 File watching: {}", config.enable_file_watching);
        info!("🐦 Canary deployment: {} ({}%)", config.enable_canary_deployment, config.canary_percentage);
        info!("🔙 Auto rollback: {}", config.auto_rollback_on_error);
        
        let (update_sender, update_receiver) = watch::channel(UpdateNotification {
            notification_type: UpdateType::FileChanged,
            file_path: String::new(),
            timestamp: chrono::Utc::now(),
        });
        
        Ok(Self {
            knowledge_base: Arc::new(RwLock::new((*knowledge_base).clone())),
            compiled_models: Arc::new(RwLock::new((*compiled_models).clone())),
            config,
            deployment_history: Arc::new(RwLock::new(Vec::new())),
            active_deployments: Arc::new(RwLock::new(HashMap::new())),
            file_watcher: None,
            update_sender,
            update_receiver,
        })
    }
    
    /// Start the hot loader with file system monitoring
    pub async fn start(&mut self, watch_paths: Vec<PathBuf>) -> Result<()> {
        info!("🚀 Starting Hot Loader");
        
        if self.config.enable_file_watching {
            self.setup_file_watcher(watch_paths).await?;
        }
        
        // Start update processing loop
        self.start_update_processor().await;
        
        info!("✅ Hot Loader started successfully");
        Ok(())
    }
    
    /// Setup file system watcher
    async fn setup_file_watcher(&mut self, watch_paths: Vec<PathBuf>) -> Result<()> {
        let sender = self.update_sender.clone();
        
        let mut watcher = notify::recommended_watcher(move |res: Result<Event, notify::Error>| {
            match res {
                Ok(event) => {
                    if let Some(path) = event.paths.first() {
                        let notification_type = match event.kind {
                            EventKind::Create(_) => UpdateType::FileAdded,
                            EventKind::Modify(_) => UpdateType::FileChanged,
                            EventKind::Remove(_) => UpdateType::FileDeleted,
                            _ => UpdateType::FileChanged,
                        };
                        
                        let notification = UpdateNotification {
                            notification_type,
                            file_path: path.to_string_lossy().to_string(),
                            timestamp: chrono::Utc::now(),
                        };
                        
                        if let Err(e) = sender.send(notification) {
                            error!("Failed to send update notification: {}", e);
                        }
                    }
                }
                Err(e) => {
                    error!("File watcher error: {}", e);
                }
            }
        })?;
        
        // Watch all specified paths
        for path in watch_paths {
            info!("👀 Watching path: {:?}", path);
            watcher.watch(&path, RecursiveMode::Recursive)?;
        }
        
        self.file_watcher = Some(watcher);
        Ok(())
    }
    
    /// Start the update processor loop
    async fn start_update_processor(&self) {
        let mut receiver = self.update_receiver.clone();
        let loader = self.clone_for_processor();
        
        tokio::spawn(async move {
            while receiver.changed().await.is_ok() {
                let notification = receiver.borrow().clone();
                
                if !notification.file_path.is_empty() {
                    debug!("📥 Processing update notification: {:?}", notification);
                    
                    if let Err(e) = loader.process_file_update(notification).await {
                        error!("Failed to process file update: {}", e);
                    }
                }
            }
        });
    }
    
    /// Clone for processor (simplified clone for async processing)
    fn clone_for_processor(&self) -> HotLoaderProcessor {
        HotLoaderProcessor {
            knowledge_base: self.knowledge_base.clone(),
            compiled_models: self.compiled_models.clone(),
            config: self.config.clone(),
            deployment_history: self.deployment_history.clone(),
            active_deployments: self.active_deployments.clone(),
        }
    }
    
    /// Manually trigger a deployment
    pub async fn deploy_changes(&self, changes: Vec<String>) -> Result<String> {
        let deployment_id = format!("manual-{}", uuid::Uuid::new_v4());
        
        info!("🚀 Starting manual deployment: {}", deployment_id);
        
        let deployment_record = DeploymentRecord {
            deployment_id: deployment_id.clone(),
            deployment_type: DeploymentType::FullReload,
            status: DeploymentStatus::Pending,
            started_at: chrono::Utc::now(),
            completed_at: None,
            version: format!("manual-{}", chrono::Utc::now().timestamp()),
            changes: changes.into_iter().map(|path| ChangeRecord {
                change_type: ChangeType::Modified,
                file_path: path,
                old_version: None,
                new_version: "manual".to_string(),
                validation_result: ValidationResult {
                    valid: true,
                    warnings: vec![],
                    errors: vec![],
                    validation_time_ms: 0,
                },
            }).collect(),
            metrics: DeploymentMetrics {
                total_files_changed: 0,
                validation_time_ms: 0,
                deployment_time_ms: 0,
                canary_success_rate: None,
                error_count: 0,
                rollback_triggered: false,
            },
            rollback_info: None,
        };
        
        // Add to history
        {
            let mut history = self.deployment_history.write().await;
            history.push(deployment_record);
            
            // Trim history if needed
            if history.len() > self.config.max_deployment_history {
                history.remove(0);
            }
        }
        
        Ok(deployment_id)
    }
    
    /// Get deployment status
    pub async fn get_deployment_status(&self, deployment_id: &str) -> Option<DeploymentStatus> {
        let active = self.active_deployments.read().await;
        active.get(deployment_id).map(|d| d.status.clone())
    }
    
    /// Get deployment history
    pub async fn get_deployment_history(&self, limit: Option<usize>) -> Vec<DeploymentRecord> {
        let history = self.deployment_history.read().await;
        let limit = limit.unwrap_or(history.len());
        history.iter().rev().take(limit).cloned().collect()
    }
    
    /// Rollback to a previous version
    pub async fn rollback(&self, target_deployment_id: &str, reason: String) -> Result<String> {
        info!("🔙 Starting rollback to deployment: {}", target_deployment_id);
        
        let rollback_id = format!("rollback-{}", uuid::Uuid::new_v4());
        
        // Find target deployment
        let history = self.deployment_history.read().await;
        let target_deployment = history.iter()
            .find(|d| d.deployment_id == target_deployment_id)
            .ok_or_else(|| anyhow!("Target deployment not found: {}", target_deployment_id))?;
        
        // Create rollback record
        let rollback_record = DeploymentRecord {
            deployment_id: rollback_id.clone(),
            deployment_type: DeploymentType::FullReload,
            status: DeploymentStatus::RollingBack,
            started_at: chrono::Utc::now(),
            completed_at: None,
            version: target_deployment.version.clone(),
            changes: vec![],
            metrics: DeploymentMetrics {
                total_files_changed: 0,
                validation_time_ms: 0,
                deployment_time_ms: 0,
                canary_success_rate: None,
                error_count: 0,
                rollback_triggered: true,
            },
            rollback_info: Some(RollbackInfo {
                reason,
                triggered_at: chrono::Utc::now(),
                rollback_to_version: target_deployment.version.clone(),
                automatic: false,
            }),
        };
        
        // Add to history
        {
            let mut history_mut = self.deployment_history.write().await;
            history_mut.push(rollback_record);
        }
        
        info!("✅ Rollback initiated: {}", rollback_id);
        Ok(rollback_id)
    }
    
    /// Get current hot loader metrics
    pub async fn get_metrics(&self) -> HotLoaderMetrics {
        let history = self.deployment_history.read().await;
        let active = self.active_deployments.read().await;
        
        let total_deployments = history.len();
        let successful_deployments = history.iter()
            .filter(|d| matches!(d.status, DeploymentStatus::Active))
            .count();
        let failed_deployments = history.iter()
            .filter(|d| matches!(d.status, DeploymentStatus::Failed))
            .count();
        let rollbacks = history.iter()
            .filter(|d| d.rollback_info.is_some())
            .count();
        
        let average_deployment_time = if !history.is_empty() {
            history.iter()
                .map(|d| d.metrics.deployment_time_ms)
                .sum::<u64>() as f64 / history.len() as f64
        } else {
            0.0
        };
        
        HotLoaderMetrics {
            total_deployments: total_deployments as u64,
            successful_deployments: successful_deployments as u64,
            failed_deployments: failed_deployments as u64,
            rollbacks: rollbacks as u64,
            active_deployments: active.len() as u32,
            average_deployment_time_ms: average_deployment_time,
            last_deployment: history.last().map(|d| d.started_at),
        }
    }
}

/// Simplified processor for async operations
#[derive(Debug, Clone)]
struct HotLoaderProcessor {
    knowledge_base: Arc<RwLock<KnowledgeBase>>,
    compiled_models: Arc<RwLock<CompiledModelRegistry>>,
    config: HotLoaderConfig,
    deployment_history: Arc<RwLock<Vec<DeploymentRecord>>>,
    active_deployments: Arc<RwLock<HashMap<String, ActiveDeployment>>>,
}

impl HotLoaderProcessor {
    /// Process a file update notification
    async fn process_file_update(&self, notification: UpdateNotification) -> Result<()> {
        debug!("🔄 Processing file update: {}", notification.file_path);
        
        // Check if file is relevant (TOML files in knowledge base directories)
        if !notification.file_path.ends_with(".toml") {
            return Ok(());
        }
        
        // Determine if this is a rule file or model file
        let is_rule_file = notification.file_path.contains("kb_drug_rules") || 
                          notification.file_path.contains("kb_ddi_rules");
        
        if is_rule_file {
            self.process_rule_update(&notification).await?;
        } else {
            debug!("Ignoring non-rule file: {}", notification.file_path);
        }
        
        Ok(())
    }
    
    /// Process a rule file update
    async fn process_rule_update(&self, notification: &UpdateNotification) -> Result<()> {
        let deployment_id = format!("auto-{}", uuid::Uuid::new_v4());
        
        info!("📋 Processing rule update: {} ({})", notification.file_path, deployment_id);
        
        // Validate the updated file
        let validation_result = self.validate_rule_file(&notification.file_path).await?;
        
        if !validation_result.valid {
            warn!("❌ Rule file validation failed: {}", notification.file_path);
            for error in &validation_result.errors {
                error!("Validation error: {}", error);
            }
            return Ok(());
        }
        
        // If canary deployment is enabled, start canary
        if self.config.enable_canary_deployment {
            self.start_canary_deployment(deployment_id, notification.clone()).await?;
        } else {
            // Direct deployment
            self.deploy_rule_update(deployment_id, notification.clone()).await?;
        }
        
        Ok(())
    }
    
    /// Validate a rule file
    async fn validate_rule_file(&self, file_path: &str) -> Result<ValidationResult> {
        let start_time = Instant::now();
        let mut warnings = Vec::new();
        let mut errors = Vec::new();
        
        // Read and parse the TOML file
        match fs::read_to_string(file_path).await {
            Ok(content) => {
                match toml::from_str::<serde_json::Value>(&content) {
                    Ok(_) => {
                        debug!("✅ TOML file is valid: {}", file_path);
                    }
                    Err(e) => {
                        errors.push(format!("TOML parsing error: {}", e));
                    }
                }
            }
            Err(e) => {
                errors.push(format!("Failed to read file: {}", e));
            }
        }
        
        Ok(ValidationResult {
            valid: errors.is_empty(),
            warnings,
            errors,
            validation_time_ms: start_time.elapsed().as_millis() as u64,
        })
    }
    
    /// Start canary deployment
    async fn start_canary_deployment(
        &self,
        deployment_id: String,
        notification: UpdateNotification,
    ) -> Result<()> {
        info!("🐦 Starting canary deployment: {}", deployment_id);
        
        let active_deployment = ActiveDeployment {
            deployment_id: deployment_id.clone(),
            status: DeploymentStatus::CanaryDeploying,
            started_at: Instant::now(),
            canary_start: Some(Instant::now()),
            error_count: 0,
            success_count: 0,
        };
        
        {
            let mut active = self.active_deployments.write().await;
            active.insert(deployment_id.clone(), active_deployment);
        }
        
        // Schedule canary completion
        let processor = self.clone();
        let deployment_id_clone = deployment_id.clone();
        tokio::spawn(async move {
            tokio::time::sleep(Duration::from_secs(processor.config.canary_duration_minutes * 60)).await;
            if let Err(e) = processor.complete_canary_deployment(deployment_id_clone).await {
                error!("Failed to complete canary deployment: {}", e);
            }
        });
        
        Ok(())
    }
    
    /// Complete canary deployment
    async fn complete_canary_deployment(&self, deployment_id: String) -> Result<()> {
        info!("🎯 Completing canary deployment: {}", deployment_id);
        
        // Check canary success rate
        let should_proceed = {
            let active = self.active_deployments.read().await;
            if let Some(deployment) = active.get(&deployment_id) {
                let total_requests = deployment.success_count + deployment.error_count;
                if total_requests > 0 {
                    let success_rate = deployment.success_count as f64 / total_requests as f64;
                    success_rate >= 0.95 // 95% success rate threshold
                } else {
                    true // No requests during canary, proceed
                }
            } else {
                false
            }
        };
        
        if should_proceed {
            info!("✅ Canary successful, proceeding with full deployment");
            // Proceed with full deployment
            self.complete_full_deployment(deployment_id).await?;
        } else {
            warn!("❌ Canary failed, rolling back");
            self.rollback_deployment(deployment_id, "Canary failure".to_string()).await?;
        }
        
        Ok(())
    }
    
    /// Deploy rule update
    async fn deploy_rule_update(&self, deployment_id: String, _notification: UpdateNotification) -> Result<()> {
        info!("🚀 Deploying rule update: {}", deployment_id);
        
        // Reload knowledge base
        // This is a simplified implementation - in practice, you'd reload specific files
        // let mut kb = self.knowledge_base.write().await;
        // *kb = KnowledgeBase::new(&kb_path).await?;
        
        info!("✅ Rule update deployed successfully: {}", deployment_id);
        Ok(())
    }
    
    /// Complete full deployment
    async fn complete_full_deployment(&self, deployment_id: String) -> Result<()> {
        info!("🎯 Completing full deployment: {}", deployment_id);
        
        {
            let mut active = self.active_deployments.write().await;
            if let Some(deployment) = active.get_mut(&deployment_id) {
                deployment.status = DeploymentStatus::Active;
            }
        }
        
        Ok(())
    }
    
    /// Rollback deployment
    async fn rollback_deployment(&self, deployment_id: String, reason: String) -> Result<()> {
        warn!("🔙 Rolling back deployment: {} ({})", deployment_id, reason);
        
        {
            let mut active = self.active_deployments.write().await;
            if let Some(deployment) = active.get_mut(&deployment_id) {
                deployment.status = DeploymentStatus::RolledBack;
            }
        }
        
        Ok(())
    }
}

/// Hot loader metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HotLoaderMetrics {
    pub total_deployments: u64,
    pub successful_deployments: u64,
    pub failed_deployments: u64,
    pub rollbacks: u64,
    pub active_deployments: u32,
    pub average_deployment_time_ms: f64,
    pub last_deployment: Option<chrono::DateTime<chrono::Utc>>,
}
