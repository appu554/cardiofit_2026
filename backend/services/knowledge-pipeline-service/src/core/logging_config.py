"""
Comprehensive Logging Configuration for Knowledge Pipeline
Captures all errors, debug info, and pipeline execution details
"""

import logging
import structlog
import sys
from pathlib import Path
from datetime import datetime
from typing import Dict, Any
import json
import traceback


class PipelineLogger:
    """Enhanced logger for knowledge pipeline with file and console output"""
    
    def __init__(self, log_dir: str = "logs"):
        self.log_dir = Path(log_dir)
        self.log_dir.mkdir(exist_ok=True)
        
        # Create timestamped log files
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        self.main_log_file = self.log_dir / f"pipeline_{timestamp}.log"
        self.error_log_file = self.log_dir / f"pipeline_errors_{timestamp}.log"
        self.debug_log_file = self.log_dir / f"pipeline_debug_{timestamp}.log"
        
        self.setup_logging()
    
    def setup_logging(self):
        """Setup comprehensive logging configuration"""
        
        # Clear any existing handlers
        logging.getLogger().handlers.clear()
        
        # Create formatters
        detailed_formatter = logging.Formatter(
            '%(asctime)s | %(levelname)-8s | %(name)-30s | %(funcName)-20s | %(lineno)-4d | %(message)s',
            datefmt='%Y-%m-%d %H:%M:%S'
        )
        
        simple_formatter = logging.Formatter(
            '%(asctime)s | %(levelname)-8s | %(message)s',
            datefmt='%H:%M:%S'
        )
        
        # Console handler (INFO and above)
        console_handler = logging.StreamHandler(sys.stdout)
        console_handler.setLevel(logging.INFO)
        console_handler.setFormatter(simple_formatter)
        
        # Main log file handler (DEBUG and above)
        main_file_handler = logging.FileHandler(self.main_log_file, encoding='utf-8')
        main_file_handler.setLevel(logging.DEBUG)
        main_file_handler.setFormatter(detailed_formatter)
        
        # Error log file handler (ERROR and above)
        error_file_handler = logging.FileHandler(self.error_log_file, encoding='utf-8')
        error_file_handler.setLevel(logging.ERROR)
        error_file_handler.setFormatter(detailed_formatter)
        
        # Debug log file handler (DEBUG only)
        debug_file_handler = logging.FileHandler(self.debug_log_file, encoding='utf-8')
        debug_file_handler.setLevel(logging.DEBUG)
        debug_file_handler.setFormatter(detailed_formatter)
        
        # Configure root logger
        root_logger = logging.getLogger()
        root_logger.setLevel(logging.DEBUG)
        root_logger.addHandler(console_handler)
        root_logger.addHandler(main_file_handler)
        root_logger.addHandler(error_file_handler)
        root_logger.addHandler(debug_file_handler)
        
        # Configure structlog
        structlog.configure(
            processors=[
                structlog.stdlib.filter_by_level,
                structlog.stdlib.add_logger_name,
                structlog.stdlib.add_log_level,
                structlog.stdlib.PositionalArgumentsFormatter(),
                structlog.processors.TimeStamper(fmt="iso"),
                structlog.processors.StackInfoRenderer(),
                structlog.processors.format_exc_info,
                structlog.processors.UnicodeDecoder(),
                structlog.processors.JSONRenderer()
            ],
            context_class=dict,
            logger_factory=structlog.stdlib.LoggerFactory(),
            wrapper_class=structlog.stdlib.BoundLogger,
            cache_logger_on_first_use=True,
        )
        
        # Log initialization
        logger = structlog.get_logger(__name__)
        logger.info("Logging system initialized",
                   main_log=str(self.main_log_file),
                   error_log=str(self.error_log_file),
                   debug_log=str(self.debug_log_file))
    
    def log_exception(self, logger, message: str, exception: Exception, **kwargs):
        """Log exception with full traceback and context"""
        error_details = {
            "message": message,
            "exception_type": type(exception).__name__,
            "exception_message": str(exception),
            "traceback": traceback.format_exc(),
            **kwargs
        }
        
        logger.error(message, **error_details)
        
        # Also write to a separate exception file
        exception_file = self.log_dir / f"exceptions_{datetime.now().strftime('%Y%m%d')}.log"
        with open(exception_file, 'a', encoding='utf-8') as f:
            f.write(f"\n{'='*80}\n")
            f.write(f"TIMESTAMP: {datetime.now().isoformat()}\n")
            f.write(f"MESSAGE: {message}\n")
            f.write(f"EXCEPTION: {type(exception).__name__}: {str(exception)}\n")
            f.write(f"CONTEXT: {json.dumps(kwargs, indent=2, default=str)}\n")
            f.write(f"TRACEBACK:\n{traceback.format_exc()}\n")
            f.write(f"{'='*80}\n")
    
    def log_pipeline_start(self, sources: list, config: Dict[str, Any]):
        """Log pipeline execution start"""
        logger = structlog.get_logger("pipeline.start")
        logger.info("🚀 Knowledge Pipeline Starting",
                   sources=sources,
                   database_type=config.get('DATABASE_TYPE'),
                   timestamp=datetime.now().isoformat())
        
        # Create execution summary file
        summary_file = self.log_dir / f"execution_summary_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json"
        summary = {
            "start_time": datetime.now().isoformat(),
            "sources": sources,
            "config": {k: v for k, v in config.items() if 'PASSWORD' not in k},  # Exclude passwords
            "status": "STARTED"
        }
        
        with open(summary_file, 'w', encoding='utf-8') as f:
            json.dump(summary, f, indent=2, default=str)
        
        return summary_file
    
    def log_pipeline_end(self, summary_file: Path, status: str, stats: Dict[str, Any]):
        """Log pipeline execution end"""
        logger = structlog.get_logger("pipeline.end")
        logger.info("🏁 Knowledge Pipeline Completed",
                   status=status,
                   stats=stats,
                   timestamp=datetime.now().isoformat())
        
        # Update execution summary
        if summary_file.exists():
            with open(summary_file, 'r', encoding='utf-8') as f:
                summary = json.load(f)
            
            summary.update({
                "end_time": datetime.now().isoformat(),
                "status": status,
                "stats": stats
            })
            
            with open(summary_file, 'w', encoding='utf-8') as f:
                json.dump(summary, f, indent=2, default=str)
    
    def get_log_files(self) -> Dict[str, Path]:
        """Get all log file paths"""
        return {
            "main_log": self.main_log_file,
            "error_log": self.error_log_file,
            "debug_log": self.debug_log_file,
            "log_directory": self.log_dir
        }


class ErrorCapture:
    """Context manager to capture and log all errors"""
    
    def __init__(self, logger, operation_name: str, **context):
        self.logger = logger
        self.operation_name = operation_name
        self.context = context
        self.pipeline_logger = None
    
    def __enter__(self):
        self.logger.info(f"🔄 Starting: {self.operation_name}", **self.context)
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        if exc_type is not None:
            # Error occurred
            if self.pipeline_logger:
                self.pipeline_logger.log_exception(
                    self.logger,
                    f"❌ Failed: {self.operation_name}",
                    exc_val,
                    operation=self.operation_name,
                    **self.context
                )
            else:
                self.logger.error(f"❌ Failed: {self.operation_name}",
                                operation=self.operation_name,
                                error=str(exc_val),
                                error_type=exc_type.__name__,
                                **self.context)
            return False  # Re-raise the exception
        else:
            # Success
            self.logger.info(f"✅ Completed: {self.operation_name}", **self.context)
            return True


def setup_pipeline_logging(log_dir: str = "logs") -> PipelineLogger:
    """Setup comprehensive pipeline logging"""
    return PipelineLogger(log_dir)


def get_logger(name: str):
    """Get a configured logger instance"""
    return structlog.get_logger(name)


# Global exception handler
def handle_exception(exc_type, exc_value, exc_traceback):
    """Global exception handler"""
    if issubclass(exc_type, KeyboardInterrupt):
        sys.__excepthook__(exc_type, exc_value, exc_traceback)
        return
    
    logger = structlog.get_logger("global.exception")
    logger.error("Uncaught exception",
                exc_type=exc_type.__name__,
                exc_value=str(exc_value),
                traceback=traceback.format_exception(exc_type, exc_value, exc_traceback))


# Install global exception handler
sys.excepthook = handle_exception
