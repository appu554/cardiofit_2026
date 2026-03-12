"""
Repository package for the Patient Service.

This package provides repositories for accessing data in the database.
"""

from .patient_repository import PatientRepository, get_patient_repository

__all__ = ["PatientRepository", "get_patient_repository"]
