-- BCE v2.0: Gamification (E2), Population Learning (E3), Timing Optimization (E4)

-- E2: Gamification
CREATE TABLE IF NOT EXISTS patient_streaks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    behavior VARCHAR(50) NOT NULL,
    current_streak INT DEFAULT 0,
    longest_streak INT DEFAULT 0,
    last_active_day DATE,
    paused BOOLEAN DEFAULT FALSE,
    paused_at TIMESTAMPTZ,
    pause_reason VARCHAR(30),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(patient_id, behavior)
);

CREATE TABLE IF NOT EXISTS patient_milestones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    milestone_type VARCHAR(50) NOT NULL,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    achieved_at TIMESTAMPTZ NOT NULL,
    celebrated BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_milestones_patient ON patient_milestones(patient_id);

CREATE TABLE IF NOT EXISTS weekly_challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    challenge_name VARCHAR(200) NOT NULL,
    target_days INT NOT NULL DEFAULT 5,
    actual_days INT DEFAULT 0,
    completed BOOLEAN DEFAULT FALSE,
    week_start DATE NOT NULL,
    week_end DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_challenges_patient ON weekly_challenges(patient_id);

-- E3: Population Learning
CREATE TABLE IF NOT EXISTS population_priors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phenotype VARCHAR(30) NOT NULL,
    technique VARCHAR(10) NOT NULL,
    alpha DECIMAL(8,4) NOT NULL,
    beta DECIMAL(8,4) NOT NULL,
    sample_size INT DEFAULT 0,
    version INT DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(phenotype, technique)
);

CREATE TABLE IF NOT EXISTS prior_calibration_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_at TIMESTAMPTZ NOT NULL,
    total_patients INT DEFAULT 0,
    eligible_patients INT DEFAULT 0,
    accuracy_improvement DECIMAL(5,4),
    adopted BOOLEAN DEFAULT FALSE,
    details TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- E4: Timing Optimization
CREATE TABLE IF NOT EXISTS patient_timing_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    slot VARCHAR(10) NOT NULL,
    alpha DECIMAL(8,4) NOT NULL DEFAULT 1.0,
    beta DECIMAL(8,4) NOT NULL DEFAULT 1.0,
    deliveries INT DEFAULT 0,
    responses INT DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(patient_id, slot)
);
