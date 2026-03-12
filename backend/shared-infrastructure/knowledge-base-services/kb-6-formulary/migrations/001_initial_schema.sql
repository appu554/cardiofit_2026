-- KB-6 Formulary Management Database Schema
-- This schema supports formulary management, stock tracking, cost optimization, and demand prediction

-- Formulary entries by payer and plan
CREATE TABLE formulary_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payer_id VARCHAR(50) NOT NULL,
    payer_name VARCHAR(200),
    plan_id VARCHAR(50) NOT NULL,
    plan_name VARCHAR(200),
    plan_year INTEGER NOT NULL,
    
    -- Drug identification
    drug_rxnorm VARCHAR(20) NOT NULL,
    drug_name TEXT NOT NULL,
    drug_type VARCHAR(50), -- brand, generic, biosimilar
    
    -- Coverage details
    tier VARCHAR(20) NOT NULL,
    -- Tiers: tier1_generic, tier2_preferred_brand, tier3_non_preferred, tier4_specialty, not_covered
    status VARCHAR(20) DEFAULT 'active',
    
    -- Cost sharing
    copay_amount DECIMAL(10,2),
    coinsurance_percent INTEGER,
    deductible_applies BOOLEAN DEFAULT FALSE,
    
    -- Restrictions
    prior_authorization BOOLEAN DEFAULT FALSE,
    step_therapy BOOLEAN DEFAULT FALSE,
    quantity_limit JSONB,
    /* Structure:
    {
      "max_quantity": 30,
      "per_days": 30,
      "max_fills_per_year": 12
    }
    */
    
    -- Age and gender restrictions
    age_limits JSONB,
    gender_restriction VARCHAR(10),
    
    -- Clinical requirements
    required_diagnosis_codes TEXT[],
    required_lab_values JSONB,
    
    -- Alternatives
    preferred_alternatives JSONB DEFAULT '[]',
    generic_available BOOLEAN DEFAULT FALSE,
    generic_rxnorm VARCHAR(20),
    
    -- Metadata
    effective_date DATE NOT NULL,
    termination_date DATE,
    last_updated TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT unique_formulary_entry 
        UNIQUE (payer_id, plan_id, drug_rxnorm, plan_year),
    CONSTRAINT valid_tier CHECK (tier IN (
        'tier1_generic', 
        'tier2_preferred_brand', 
        'tier3_non_preferred', 
        'tier4_specialty', 
        'not_covered'
    )),
    CONSTRAINT valid_status CHECK (status IN ('active', 'inactive', 'pending'))
);

-- Stock inventory tracking
CREATE TABLE drug_inventory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    location_id VARCHAR(100) NOT NULL,
    location_name VARCHAR(200),
    drug_rxnorm VARCHAR(20) NOT NULL,
    drug_ndc VARCHAR(20),
    
    -- Stock levels
    quantity_on_hand INTEGER NOT NULL DEFAULT 0,
    quantity_allocated INTEGER NOT NULL DEFAULT 0,
    quantity_available INTEGER GENERATED ALWAYS AS 
        (quantity_on_hand - quantity_allocated) STORED,
    
    -- Reorder parameters
    reorder_point INTEGER,
    reorder_quantity INTEGER,
    max_stock_level INTEGER,
    
    -- Lot tracking
    lot_number VARCHAR(50),
    expiration_date DATE,
    manufacturer VARCHAR(100),
    
    -- Cost information
    unit_cost DECIMAL(10,4),
    acquisition_cost DECIMAL(10,2),
    
    -- Timestamps
    last_counted TIMESTAMPTZ,
    last_ordered TIMESTAMPTZ,
    last_received TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_location_drug_lot 
        UNIQUE (location_id, drug_rxnorm, lot_number)
);

-- Demand prediction data
CREATE TABLE demand_history (
    id BIGSERIAL PRIMARY KEY,
    location_id VARCHAR(100) NOT NULL,
    drug_rxnorm VARCHAR(20) NOT NULL,
    date DATE NOT NULL,
    quantity_dispensed INTEGER NOT NULL,
    quantity_ordered INTEGER,
    stockout_occurred BOOLEAN DEFAULT FALSE,
    
    -- Factors affecting demand
    day_of_week INTEGER,
    month INTEGER,
    is_holiday BOOLEAN DEFAULT FALSE,
    weather_impact VARCHAR(20),
    seasonal_factor DECIMAL(3,2),
    
    -- Patient demographics impact
    patient_age_avg INTEGER,
    patient_count INTEGER,
    
    UNIQUE(location_id, drug_rxnorm, date)
);

-- Pricing information from multiple sources
CREATE TABLE drug_pricing (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drug_rxnorm VARCHAR(20) NOT NULL,
    drug_ndc VARCHAR(20),
    price_type VARCHAR(50) NOT NULL, -- AWP, WAC, NADAC, MAC, Contract
    price DECIMAL(10,4) NOT NULL,
    unit VARCHAR(20),
    package_size INTEGER,
    effective_date DATE NOT NULL,
    termination_date DATE,
    source VARCHAR(100),
    contract_id VARCHAR(50),
    
    UNIQUE(drug_rxnorm, price_type, effective_date, source)
);

-- Insurance payers and plans
CREATE TABLE insurance_payers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payer_id VARCHAR(50) UNIQUE NOT NULL,
    payer_name VARCHAR(200) NOT NULL,
    payer_type VARCHAR(50), -- commercial, medicare, medicaid, federal
    market_share DECIMAL(5,4),
    contract_tier VARCHAR(20),
    contact_info JSONB,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE insurance_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payer_id VARCHAR(50) REFERENCES insurance_payers(payer_id),
    plan_id VARCHAR(50) NOT NULL,
    plan_name VARCHAR(200) NOT NULL,
    plan_type VARCHAR(50), -- HMO, PPO, EPO, POS
    plan_year INTEGER NOT NULL,
    members_covered INTEGER,
    formulary_type VARCHAR(50), -- open, closed, tiered
    active BOOLEAN DEFAULT TRUE,
    
    UNIQUE(payer_id, plan_id, plan_year)
);

-- Drug alternatives and therapeutic equivalents
CREATE TABLE drug_alternatives (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    primary_drug_rxnorm VARCHAR(20) NOT NULL,
    alternative_drug_rxnorm VARCHAR(20) NOT NULL,
    alternative_type VARCHAR(50) NOT NULL, -- generic, therapeutic, biosimilar
    therapeutic_class VARCHAR(100),
    equivalence_rating VARCHAR(10), -- AB, AN, AO, AP, etc.
    cost_difference_percent DECIMAL(5,2),
    efficacy_rating DECIMAL(3,2),
    safety_profile VARCHAR(50),
    switch_complexity VARCHAR(20), -- simple, moderate, complex
    clinical_notes TEXT,
    evidence_level VARCHAR(10),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_drug_alternative 
        UNIQUE (primary_drug_rxnorm, alternative_drug_rxnorm)
);

-- Formulary coverage analysis
CREATE TABLE coverage_analysis (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analysis_date DATE NOT NULL,
    payer_id VARCHAR(50) NOT NULL,
    plan_id VARCHAR(50) NOT NULL,
    total_drugs INTEGER,
    covered_drugs INTEGER,
    coverage_percentage DECIMAL(5,2),
    tier_distribution JSONB,
    prior_auth_percentage DECIMAL(5,2),
    step_therapy_percentage DECIMAL(5,2),
    avg_patient_cost DECIMAL(10,2),
    analysis_notes TEXT,
    
    UNIQUE(analysis_date, payer_id, plan_id)
);

-- Stock alerts and notifications
CREATE TABLE stock_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    location_id VARCHAR(100) NOT NULL,
    drug_rxnorm VARCHAR(20) NOT NULL,
    alert_type VARCHAR(50) NOT NULL, -- low_stock, stockout, expired, recall
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low')),
    message TEXT NOT NULL,
    current_quantity INTEGER,
    recommended_action TEXT,
    
    -- Alert management
    triggered_at TIMESTAMPTZ DEFAULT NOW(),
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_by VARCHAR(100),
    acknowledged_at TIMESTAMPTZ,
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMPTZ,
    
    -- Escalation
    escalated BOOLEAN DEFAULT FALSE,
    escalated_at TIMESTAMPTZ,
    escalation_level INTEGER DEFAULT 1
);

-- Create indexes for performance
CREATE INDEX idx_formulary_payer_plan ON formulary_entries(payer_id, plan_id);
CREATE INDEX idx_formulary_drug ON formulary_entries(drug_rxnorm);
CREATE INDEX idx_formulary_tier ON formulary_entries(tier);
CREATE INDEX idx_formulary_effective ON formulary_entries(effective_date, termination_date);
CREATE INDEX idx_formulary_text_search ON formulary_entries USING gin(to_tsvector('english', drug_name));

CREATE INDEX idx_inventory_location ON drug_inventory(location_id);
CREATE INDEX idx_inventory_drug ON drug_inventory(drug_rxnorm);
CREATE INDEX idx_inventory_available ON drug_inventory(quantity_available);
CREATE INDEX idx_inventory_expiration ON drug_inventory(expiration_date);
CREATE INDEX idx_inventory_reorder ON drug_inventory(reorder_point) WHERE quantity_available <= reorder_point;

CREATE INDEX idx_demand_location_drug_date ON demand_history(location_id, drug_rxnorm, date DESC);
CREATE INDEX idx_demand_date ON demand_history(date);
CREATE INDEX idx_demand_stockout ON demand_history(stockout_occurred) WHERE stockout_occurred = true;

CREATE INDEX idx_pricing_drug_type_date ON drug_pricing(drug_rxnorm, price_type, effective_date DESC);
CREATE INDEX idx_pricing_effective ON drug_pricing(effective_date, termination_date);

CREATE INDEX idx_alternatives_primary ON drug_alternatives(primary_drug_rxnorm);
CREATE INDEX idx_alternatives_type ON drug_alternatives(alternative_type);

CREATE INDEX idx_stock_alerts_location ON stock_alerts(location_id, triggered_at DESC);
CREATE INDEX idx_stock_alerts_unresolved ON stock_alerts(resolved) WHERE resolved = false;
CREATE INDEX idx_stock_alerts_severity ON stock_alerts(severity, triggered_at DESC);

-- Functions for formulary coverage checking
CREATE OR REPLACE FUNCTION check_formulary_coverage(
    p_drug_rxnorm VARCHAR,
    p_payer_id VARCHAR,
    p_plan_id VARCHAR,
    p_quantity INTEGER DEFAULT 30
) RETURNS TABLE (
    covered BOOLEAN,
    tier VARCHAR,
    patient_cost DECIMAL,
    requires_prior_auth BOOLEAN,
    requires_step_therapy BOOLEAN,
    alternatives JSONB
) AS $$
DECLARE
    v_formulary RECORD;
    v_drug_price DECIMAL;
    v_patient_cost DECIMAL;
BEGIN
    -- Get formulary entry
    SELECT * INTO v_formulary
    FROM formulary_entries
    WHERE drug_rxnorm = p_drug_rxnorm
      AND payer_id = p_payer_id
      AND plan_id = p_plan_id
      AND plan_year = EXTRACT(YEAR FROM CURRENT_DATE)
      AND status = 'active'
      AND CURRENT_DATE BETWEEN effective_date AND COALESCE(termination_date, '9999-12-31');
    
    IF NOT FOUND THEN
        -- Drug not covered, return alternatives
        RETURN QUERY
        SELECT 
            FALSE AS covered,
            'not_covered'::VARCHAR AS tier,
            NULL::DECIMAL AS patient_cost,
            FALSE AS requires_prior_auth,
            FALSE AS requires_step_therapy,
            (SELECT jsonb_agg(jsonb_build_object(
                'drug_rxnorm', alternative_drug_rxnorm,
                'alternative_type', alternative_type,
                'cost_difference_percent', cost_difference_percent
            ))
            FROM drug_alternatives
            WHERE primary_drug_rxnorm = p_drug_rxnorm
            LIMIT 5) AS alternatives;
        RETURN;
    END IF;
    
    -- Get drug price
    SELECT price INTO v_drug_price
    FROM drug_pricing
    WHERE drug_rxnorm = p_drug_rxnorm
      AND price_type = 'AWP'
      AND CURRENT_DATE BETWEEN effective_date AND COALESCE(termination_date, '9999-12-31')
    ORDER BY effective_date DESC
    LIMIT 1;
    
    -- Calculate patient cost
    IF v_formulary.copay_amount IS NOT NULL THEN
        v_patient_cost := v_formulary.copay_amount;
    ELSIF v_formulary.coinsurance_percent IS NOT NULL THEN
        v_patient_cost := (v_drug_price * p_quantity) * (v_formulary.coinsurance_percent / 100.0);
    ELSE
        v_patient_cost := v_drug_price * p_quantity;
    END IF;
    
    RETURN QUERY
    SELECT 
        TRUE AS covered,
        v_formulary.tier,
        v_patient_cost,
        v_formulary.prior_authorization,
        v_formulary.step_therapy,
        v_formulary.preferred_alternatives;
END;
$$ LANGUAGE plpgsql;

-- Function for demand prediction
CREATE OR REPLACE FUNCTION predict_demand(
    p_location_id VARCHAR,
    p_drug_rxnorm VARCHAR,
    p_days_ahead INTEGER DEFAULT 7
) RETURNS TABLE (
    predicted_demand INTEGER,
    confidence_interval_low INTEGER,
    confidence_interval_high INTEGER,
    reorder_recommended BOOLEAN,
    stockout_risk DECIMAL
) AS $$
DECLARE
    v_avg_daily_demand DECIMAL;
    v_std_dev DECIMAL;
    v_seasonal_factor DECIMAL;
    v_current_stock INTEGER;
    v_predicted_demand INTEGER;
    v_stockout_risk DECIMAL;
BEGIN
    -- Calculate average daily demand and seasonal factor (last 90 days)
    SELECT 
        AVG(quantity_dispensed),
        STDDEV(quantity_dispensed),
        AVG(seasonal_factor)
    INTO v_avg_daily_demand, v_std_dev, v_seasonal_factor
    FROM demand_history
    WHERE location_id = p_location_id
      AND drug_rxnorm = p_drug_rxnorm
      AND date >= CURRENT_DATE - INTERVAL '90 days';
    
    -- Apply seasonal adjustment
    v_avg_daily_demand := v_avg_daily_demand * COALESCE(v_seasonal_factor, 1.0);
    
    -- Get current stock
    SELECT quantity_available INTO v_current_stock
    FROM drug_inventory
    WHERE location_id = p_location_id
      AND drug_rxnorm = p_drug_rxnorm
      AND expiration_date > CURRENT_DATE
    ORDER BY expiration_date
    LIMIT 1;
    
    -- Calculate prediction
    v_predicted_demand := CEIL(v_avg_daily_demand * p_days_ahead);
    
    -- Calculate stockout risk
    IF v_current_stock IS NULL OR v_current_stock <= 0 THEN
        v_stockout_risk := 1.0;
    ELSE
        v_stockout_risk := LEAST(1.0, GREATEST(0.0, 
            (v_predicted_demand - v_current_stock)::DECIMAL / NULLIF(v_predicted_demand, 0)
        ));
    END IF;
    
    RETURN QUERY
    SELECT 
        v_predicted_demand,
        GREATEST(0, v_predicted_demand - (2 * COALESCE(v_std_dev, 0)))::INTEGER,
        (v_predicted_demand + (2 * COALESCE(v_std_dev, 0)))::INTEGER,
        (COALESCE(v_current_stock, 0) < v_predicted_demand * 1.5),
        v_stockout_risk;
END;
$$ LANGUAGE plpgsql;

-- Function to calculate cost savings from alternatives
CREATE OR REPLACE FUNCTION calculate_cost_savings(
    p_primary_drug_rxnorm VARCHAR,
    p_quantity INTEGER DEFAULT 30
) RETURNS TABLE (
    alternative_rxnorm VARCHAR,
    alternative_type VARCHAR,
    primary_cost DECIMAL,
    alternative_cost DECIMAL,
    cost_savings DECIMAL,
    savings_percent DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        da.alternative_drug_rxnorm,
        da.alternative_type,
        pp.price * p_quantity AS primary_cost,
        ap.price * p_quantity AS alternative_cost,
        (pp.price - ap.price) * p_quantity AS cost_savings,
        CASE 
            WHEN pp.price > 0 THEN ((pp.price - ap.price) / pp.price) * 100
            ELSE 0
        END AS savings_percent
    FROM drug_alternatives da
    JOIN drug_pricing pp ON pp.drug_rxnorm = da.primary_drug_rxnorm
        AND pp.price_type = 'AWP'
        AND CURRENT_DATE BETWEEN pp.effective_date AND COALESCE(pp.termination_date, '9999-12-31')
    JOIN drug_pricing ap ON ap.drug_rxnorm = da.alternative_drug_rxnorm
        AND ap.price_type = 'AWP'
        AND CURRENT_DATE BETWEEN ap.effective_date AND COALESCE(ap.termination_date, '9999-12-31')
    WHERE da.primary_drug_rxnorm = p_primary_drug_rxnorm
      AND ap.price < pp.price
    ORDER BY cost_savings DESC;
END;
$$ LANGUAGE plpgsql;

-- Triggers for automatic stock alerts
CREATE OR REPLACE FUNCTION trigger_stock_alerts() RETURNS TRIGGER AS $$
BEGIN
    -- Low stock alert
    IF NEW.quantity_available <= COALESCE(NEW.reorder_point, 0) AND NEW.quantity_available > 0 THEN
        INSERT INTO stock_alerts (
            location_id, drug_rxnorm, alert_type, severity, message, 
            current_quantity, recommended_action
        ) VALUES (
            NEW.location_id, NEW.drug_rxnorm, 'low_stock', 'medium',
            format('Low stock alert: %s units remaining', NEW.quantity_available),
            NEW.quantity_available,
            format('Reorder %s units', COALESCE(NEW.reorder_quantity, 100))
        )
        ON CONFLICT DO NOTHING;
    END IF;
    
    -- Stockout alert
    IF NEW.quantity_available <= 0 THEN
        INSERT INTO stock_alerts (
            location_id, drug_rxnorm, alert_type, severity, message,
            current_quantity, recommended_action
        ) VALUES (
            NEW.location_id, NEW.drug_rxnorm, 'stockout', 'critical',
            'STOCKOUT: No inventory available',
            NEW.quantity_available,
            'Urgent reorder required'
        )
        ON CONFLICT DO NOTHING;
    END IF;
    
    -- Expiring stock alert (within 30 days)
    IF NEW.expiration_date <= CURRENT_DATE + INTERVAL '30 days' AND NEW.quantity_available > 0 THEN
        INSERT INTO stock_alerts (
            location_id, drug_rxnorm, alert_type, severity, message,
            current_quantity, recommended_action
        ) VALUES (
            NEW.location_id, NEW.drug_rxnorm, 'expired', 'high',
            format('Stock expiring on %s', NEW.expiration_date),
            NEW.quantity_available,
            'Remove expired inventory or use before expiration'
        )
        ON CONFLICT DO NOTHING;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER stock_alert_trigger
    AFTER INSERT OR UPDATE ON drug_inventory
    FOR EACH ROW
    EXECUTE FUNCTION trigger_stock_alerts();