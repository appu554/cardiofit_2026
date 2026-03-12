"""
Logging configuration for Clinical Assertion Engine (CAE)
"""

import logging
import logging.handlers
import os
from datetime import datetime

def setup_logging(log_level=logging.INFO, log_file=None):
    """
    Setup logging configuration for CAE service
    
    Args:
        log_level: Logging level (default: INFO)
        log_file: Optional log file path
    """
    
    # Create logs directory if it doesn't exist
    log_dir = "logs"
    if not os.path.exists(log_dir):
        os.makedirs(log_dir)
    
    # Default log file with timestamp
    if log_file is None:
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        log_file = os.path.join(log_dir, f"cae_service_{timestamp}.log")
    
    # Create formatter
    formatter = logging.Formatter(
        '%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        datefmt='%Y-%m-%d %H:%M:%S'
    )
    
    # Configure root logger
    root_logger = logging.getLogger()
    root_logger.setLevel(log_level)
    
    # Remove existing handlers
    for handler in root_logger.handlers[:]:
        root_logger.removeHandler(handler)
    
    # Console handler
    console_handler = logging.StreamHandler()
    console_handler.setLevel(log_level)
    console_handler.setFormatter(formatter)
    root_logger.addHandler(console_handler)
    
    # File handler with rotation
    file_handler = logging.handlers.RotatingFileHandler(
        log_file, 
        maxBytes=10*1024*1024,  # 10MB
        backupCount=5
    )
    file_handler.setLevel(log_level)
    file_handler.setFormatter(formatter)
    root_logger.addHandler(file_handler)
    
    # Error file handler (only errors and above)
    error_file = os.path.join(log_dir, "cae_errors.log")
    error_handler = logging.handlers.RotatingFileHandler(
        error_file,
        maxBytes=5*1024*1024,  # 5MB
        backupCount=3
    )
    error_handler.setLevel(logging.ERROR)
    error_handler.setFormatter(formatter)
    root_logger.addHandler(error_handler)
    
    # Reduce verbosity of specific loggers
    logging.getLogger('orchestration.parallel_executor').setLevel(logging.WARNING)
    logging.getLogger('reasoners.medication_interaction').setLevel(logging.WARNING)
    logging.getLogger('reasoners.dosing_calculator').setLevel(logging.WARNING)
    logging.getLogger('reasoners.contraindication').setLevel(logging.WARNING)
    logging.getLogger('reasoners.duplicate_therapy').setLevel(logging.WARNING)
    
    # Keep important loggers at INFO level
    logging.getLogger('grpc_server').setLevel(logging.INFO)
    logging.getLogger('orchestration.orchestration_engine').setLevel(logging.INFO)
    logging.getLogger('orchestration.decision_aggregator').setLevel(logging.INFO)
    
    logging.info(f"Logging configured - Level: {logging.getLevelName(log_level)}")
    logging.info(f"Log file: {log_file}")
    logging.info(f"Error log file: {error_file}")
    
    return log_file

def get_logger(name):
    """Get a logger with the specified name"""
    return logging.getLogger(name)

def log_grpc_error(error_msg, patient_id=None, request_id=None):
    """Log gRPC specific errors with context"""
    logger = get_logger('grpc_server')
    context = []
    if patient_id:
        context.append(f"patient_id={patient_id}")
    if request_id:
        context.append(f"request_id={request_id}")
    
    context_str = f" [{', '.join(context)}]" if context else ""
    logger.error(f"gRPC Error{context_str}: {error_msg}")

def log_enum_conversion_error(enum_value, enum_type, patient_id=None):
    """Log enum conversion errors specifically"""
    logger = get_logger('grpc_server')
    context = f" [patient_id={patient_id}]" if patient_id else ""
    logger.error(f"Enum Conversion Error{context}: unknown {enum_type} label '{enum_value}'")
