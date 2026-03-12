//! # Context Normalization
//!
//! Normalizes JIT Safety context including dose unit conversion,
//! CrCl calculation, interval coercion, and metric selection.

use crate::jit_safety::{domain::*, error::JitSafetyError};
use tracing::{debug, warn};

/// Normalize JIT Safety context
pub fn normalize_context(ctx: &mut JitSafetyContext) -> Result<(), JitSafetyError> {
    debug!("Starting context normalization for request {}", ctx.request_id);

    // Step 1: Validate and normalize dose units
    normalize_dose_units(&mut ctx.proposal)?;

    // Step 2: Normalize interval to hours
    normalize_interval(&mut ctx.proposal)?;

    // Step 3: Validate route
    validate_route(&ctx.proposal)?;

    // Step 4: Compute CrCl if missing and needed
    compute_missing_crcl(&mut ctx.patient)?;

    // Step 5: Validate patient context
    validate_patient_context(&ctx.patient)?;

    debug!("Context normalization completed successfully");
    Ok(())
}

/// Normalize dose units to mg
fn normalize_dose_units(dose: &mut ProposedDose) -> Result<(), JitSafetyError> {
    // Validate dose is positive
    if dose.dose_mg <= 0.0 {
        return Err(JitSafetyError::input_validation(
            format!("Invalid dose: {} mg must be positive", dose.dose_mg)
        ));
    }

    // For now, assume all doses are already in mg
    // In a real implementation, you might convert from other units:
    // - mcg to mg (divide by 1000)
    // - g to mg (multiply by 1000)
    // - units (for insulin) - special handling

    debug!("Dose normalized: {} mg", dose.dose_mg);
    Ok(())
}

/// Normalize interval to hours
fn normalize_interval(dose: &mut ProposedDose) -> Result<(), JitSafetyError> {
    // Validate interval is positive
    if dose.interval_h == 0 {
        return Err(JitSafetyError::input_validation(
            "Invalid interval: must be greater than 0 hours"
        ));
    }

    // Common interval validations
    match dose.interval_h {
        1..=168 => {}, // 1 hour to 1 week is reasonable
        _ => {
            warn!("Unusual dosing interval: {} hours", dose.interval_h);
        }
    }

    debug!("Interval normalized: {} hours", dose.interval_h);
    Ok(())
}

/// Validate route of administration
fn validate_route(dose: &ProposedDose) -> Result<(), JitSafetyError> {
    let valid_routes = ["po", "iv", "im", "sc", "sl", "pr", "topical", "inhaled"];
    
    if !valid_routes.contains(&dose.route.as_str()) {
        return Err(JitSafetyError::input_validation(
            format!("Invalid route: '{}'. Valid routes: {:?}", dose.route, valid_routes)
        ));
    }

    debug!("Route validated: {}", dose.route);
    Ok(())
}

/// Compute CrCl using Cockcroft-Gault equation if missing
fn compute_missing_crcl(patient: &mut PatientCtx) -> Result<(), JitSafetyError> {
    // If CrCl is already available, no need to compute
    if patient.renal.crcl.is_some() {
        debug!("CrCl already available: {:.1} mL/min", patient.renal.crcl.unwrap());
        return Ok(());
    }

    // If eGFR is available and height/weight are available, we can estimate CrCl
    if let (Some(egfr), Some(height_cm)) = (patient.renal.egfr, patient.height_cm) {
        // Simplified conversion from eGFR to CrCl
        // Real implementation would use proper formulas and consider BSA
        let estimated_crcl = estimate_crcl_from_egfr(egfr, patient.age_years, patient.weight_kg, height_cm, &patient.sex)?;
        
        patient.renal.crcl = Some(estimated_crcl);
        debug!("Estimated CrCl from eGFR: {:.1} mL/min", estimated_crcl);
    } else {
        debug!("Insufficient data to compute CrCl (eGFR: {:?}, height: {:?})", 
               patient.renal.egfr, patient.height_cm);
    }

    Ok(())
}

/// Estimate CrCl from eGFR (simplified)
fn estimate_crcl_from_egfr(
    egfr: f64,
    age_years: u32,
    weight_kg: f64,
    height_cm: f64,
    sex: &str,
) -> Result<f64, JitSafetyError> {
    // This is a simplified estimation
    // In practice, you'd use proper pharmacokinetic formulas
    
    // Basic validation
    if egfr <= 0.0 || weight_kg <= 0.0 || height_cm <= 0.0 {
        return Err(JitSafetyError::normalization(
            "Invalid parameters for CrCl calculation"
        ));
    }

    // Calculate BSA (Body Surface Area) using Mosteller formula
    let bsa = ((height_cm * weight_kg) / 3600.0).sqrt();
    
    // Convert eGFR (normalized to 1.73 m²) to absolute GFR
    let absolute_gfr = egfr * (bsa / 1.73);
    
    // Apply age and sex adjustments (simplified)
    let age_factor = if age_years > 65 { 0.95 } else { 1.0 };
    let sex_factor = if sex.to_lowercase() == "female" { 0.85 } else { 1.0 };
    
    let estimated_crcl = absolute_gfr * age_factor * sex_factor;
    
    debug!("CrCl estimation: eGFR={:.1} → CrCl={:.1} (BSA={:.2}, age_factor={:.2}, sex_factor={:.2})",
           egfr, estimated_crcl, bsa, age_factor, sex_factor);
    
    Ok(estimated_crcl)
}

/// Validate patient context
fn validate_patient_context(patient: &PatientCtx) -> Result<(), JitSafetyError> {
    // Age validation
    if patient.age_years == 0 || patient.age_years > 150 {
        return Err(JitSafetyError::input_validation(
            format!("Invalid age: {} years", patient.age_years)
        ));
    }

    // Weight validation
    if patient.weight_kg <= 0.0 || patient.weight_kg > 500.0 {
        return Err(JitSafetyError::input_validation(
            format!("Invalid weight: {} kg", patient.weight_kg)
        ));
    }

    // Height validation (if provided)
    if let Some(height) = patient.height_cm {
        if height <= 0.0 || height > 300.0 {
            return Err(JitSafetyError::input_validation(
                format!("Invalid height: {} cm", height)
            ));
        }
    }

    // Sex validation
    let valid_sex = ["male", "female", "other", "unknown"];
    if !valid_sex.contains(&patient.sex.to_lowercase().as_str()) {
        return Err(JitSafetyError::input_validation(
            format!("Invalid sex: '{}'. Valid values: {:?}", patient.sex, valid_sex)
        ));
    }

    // Renal function validation
    if let Some(egfr) = patient.renal.egfr {
        if egfr < 0.0 || egfr > 200.0 {
            warn!("Unusual eGFR value: {:.1} mL/min/1.73m²", egfr);
        }
    }

    if let Some(crcl) = patient.renal.crcl {
        if crcl < 0.0 || crcl > 300.0 {
            warn!("Unusual CrCl value: {:.1} mL/min", crcl);
        }
    }

    // QTc validation (if provided)
    if let Some(qtc) = patient.qtc_ms {
        if qtc < 300 || qtc > 700 {
            warn!("Unusual QTc value: {} ms", qtc);
        }
    }

    debug!("Patient context validation completed");
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::HashMap;

    fn create_test_context() -> JitSafetyContext {
        JitSafetyContext {
            patient: PatientCtx {
                age_years: 65,
                sex: "female".to_string(),
                weight_kg: 70.0,
                height_cm: Some(165.0),
                pregnancy: false,
                renal: RenalCtx {
                    egfr: Some(45.0),
                    crcl: None,
                },
                hepatic: HepaticCtx {
                    child_pugh: Some('A'),
                },
                qtc_ms: Some(420),
                allergies: vec![],
                conditions: vec![],
                labs: LabsCtx {
                    alt: Some(25.0),
                    ast: Some(30.0),
                    uacr: Some(100.0),
                },
            },
            concurrent_meds: vec![],
            proposal: ProposedDose {
                drug_id: "lisinopril".to_string(),
                dose_mg: 10.0,
                route: "po".to_string(),
                interval_h: 24,
            },
            kb_versions: HashMap::new(),
            request_id: "test-request-123".to_string(),
        }
    }

    #[test]
    fn test_normalize_context_success() {
        let mut ctx = create_test_context();
        let result = normalize_context(&mut ctx);
        assert!(result.is_ok());
        
        // Should have computed CrCl
        assert!(ctx.patient.renal.crcl.is_some());
    }

    #[test]
    fn test_invalid_dose() {
        let mut ctx = create_test_context();
        ctx.proposal.dose_mg = -5.0;
        
        let result = normalize_context(&mut ctx);
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), JitSafetyError::InputValidation { .. }));
    }

    #[test]
    fn test_invalid_route() {
        let mut ctx = create_test_context();
        ctx.proposal.route = "invalid".to_string();
        
        let result = normalize_context(&mut ctx);
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), JitSafetyError::InputValidation { .. }));
    }

    #[test]
    fn test_crcl_estimation() {
        let result = estimate_crcl_from_egfr(45.0, 65, 70.0, 165.0, "female");
        assert!(result.is_ok());
        
        let crcl = result.unwrap();
        assert!(crcl > 0.0);
        assert!(crcl < 100.0); // Should be reasonable for this patient
    }
}
