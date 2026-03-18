// Exercise nodes — representative sample (spec §10)
CREATE (:Exercise {code: 'EX001', name: 'Brisk Walking', category: 'AEROBIC', met_value: 3.5, safety_tier: 'T1_SAFE', min_duration_min: 30, max_duration_min: 60, freq_per_week: 5});
CREATE (:Exercise {code: 'EX002', name: 'Jogging', category: 'AEROBIC', met_value: 7.0, safety_tier: 'T2_CONDITIONAL', min_duration_min: 20, max_duration_min: 45, freq_per_week: 3});
CREATE (:Exercise {code: 'EX003', name: 'Swimming', category: 'AEROBIC', met_value: 6.0, safety_tier: 'T2_CONDITIONAL', min_duration_min: 30, max_duration_min: 60, freq_per_week: 3});
CREATE (:Exercise {code: 'EX004', name: 'Cycling', category: 'AEROBIC', met_value: 5.5, safety_tier: 'T1_SAFE', min_duration_min: 30, max_duration_min: 60, freq_per_week: 5});
CREATE (:Exercise {code: 'EX005', name: 'Yoga (Hatha)', category: 'FLEXIBILITY', met_value: 2.5, safety_tier: 'T1_SAFE', min_duration_min: 30, max_duration_min: 60, freq_per_week: 5});
CREATE (:Exercise {code: 'EX006', name: 'Resistance Bands', category: 'RESISTANCE', met_value: 3.5, safety_tier: 'T1_SAFE', min_duration_min: 20, max_duration_min: 40, freq_per_week: 3});
CREATE (:Exercise {code: 'EX007', name: 'Weight Training (Moderate)', category: 'RESISTANCE', met_value: 5.0, safety_tier: 'T2_CONDITIONAL', min_duration_min: 30, max_duration_min: 45, freq_per_week: 3});
CREATE (:Exercise {code: 'EX008', name: 'Surya Namaskar', category: 'FLEXIBILITY', met_value: 3.3, safety_tier: 'T1_SAFE', min_duration_min: 15, max_duration_min: 30, freq_per_week: 5});
CREATE (:Exercise {code: 'EX009', name: 'Tai Chi', category: 'BALANCE', met_value: 3.0, safety_tier: 'T1_SAFE', min_duration_min: 30, max_duration_min: 60, freq_per_week: 5});
CREATE (:Exercise {code: 'EX010', name: 'Dancing', category: 'AEROBIC', met_value: 4.5, safety_tier: 'T1_SAFE', min_duration_min: 30, max_duration_min: 60, freq_per_week: 3});
CREATE (:Exercise {code: 'EX011', name: 'High Intensity Interval Training', category: 'AEROBIC', met_value: 8.0, safety_tier: 'T3_SUPERVISED', min_duration_min: 20, max_duration_min: 30, freq_per_week: 2});
CREATE (:Exercise {code: 'EX012', name: 'Stair Climbing', category: 'AEROBIC', met_value: 4.0, safety_tier: 'T1_SAFE', min_duration_min: 10, max_duration_min: 20, freq_per_week: 5});
CREATE (:Exercise {code: 'EX013', name: 'Stretching', category: 'FLEXIBILITY', met_value: 2.3, safety_tier: 'T1_SAFE', min_duration_min: 10, max_duration_min: 20, freq_per_week: 7});
CREATE (:Exercise {code: 'EX014', name: 'Chair Exercises', category: 'BALANCE', met_value: 2.0, safety_tier: 'T1_SAFE', min_duration_min: 15, max_duration_min: 30, freq_per_week: 5});
CREATE (:Exercise {code: 'EX015', name: 'Heavy Deadlift/Squat', category: 'RESISTANCE', met_value: 6.0, safety_tier: 'T3_SUPERVISED', min_duration_min: 30, max_duration_min: 60, freq_per_week: 2});

// E-06: Resistance band seated row
CREATE (:Exercise {code: 'EX_RESISTANCE_BAND_ROW', name: 'Resistance band seated row', category: 'RESISTANCE', met_value: 3.5, safety_tier: 'T2_CONDITIONAL', min_duration_min: 10, max_duration_min: 20, freq_per_week: 3, equipment: ['resistance_band'], contraindications: []});

// E-07: Micro-workout nodes for baseline <2000 steps
CREATE (:Exercise {code: 'EX_MICRO_SQUAT', name: 'Micro-workout bodyweight squats', category: 'RESISTANCE', met_value: 4.0, safety_tier: 'T1_SAFE', min_duration_min: 1, max_duration_min: 3, freq_per_week: 7, equipment: [], contraindications: []});
CREATE (:Exercise {code: 'EX_MICRO_STAIR', name: 'Micro-workout stair climbing', category: 'AEROBIC', met_value: 6.0, safety_tier: 'T2_CONDITIONAL', min_duration_min: 1, max_duration_min: 3, freq_per_week: 7, equipment: [], contraindications: []});
CREATE (:Exercise {code: 'EX_MICRO_WALK', name: 'Micro-workout brisk walking', category: 'AEROBIC', met_value: 4.3, safety_tier: 'T1_SAFE', min_duration_min: 2, max_duration_min: 5, freq_per_week: 7, equipment: [], contraindications: []});
