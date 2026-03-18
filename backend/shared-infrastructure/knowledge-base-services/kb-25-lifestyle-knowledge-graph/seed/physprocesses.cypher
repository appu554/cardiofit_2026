// PhysProcess nodes — intermediate physiological processes in causal chains
CREATE (:PhysProcess {code: 'GLUT4_TRANSLOCATION', name: 'GLUT4 Translocation', description: 'Insulin-stimulated glucose uptake in muscle'});
CREATE (:PhysProcess {code: 'HEPATIC_GLYCOGENESIS', name: 'Hepatic Glycogenesis', description: 'Liver glycogen synthesis from glucose'});
CREATE (:PhysProcess {code: 'PERIPHERAL_GLUCOSE_UPTAKE', name: 'Peripheral Glucose Uptake', description: 'Skeletal muscle glucose disposal'});
CREATE (:PhysProcess {code: 'GLUCONEOGENESIS', name: 'Gluconeogenesis', description: 'Hepatic glucose production from non-carb precursors'});
CREATE (:PhysProcess {code: 'LIPOGENESIS', name: 'De Novo Lipogenesis', description: 'Fatty acid synthesis from excess carbohydrate'});
CREATE (:PhysProcess {code: 'LIPOLYSIS', name: 'Lipolysis', description: 'Fat breakdown for energy'});
CREATE (:PhysProcess {code: 'BETA_OXIDATION', name: 'Beta Oxidation', description: 'Fatty acid oxidation in mitochondria'});
CREATE (:PhysProcess {code: 'MITOCHONDRIAL_BIOGENESIS', name: 'Mitochondrial Biogenesis', description: 'New mitochondria formation'});
CREATE (:PhysProcess {code: 'AMPK_ACTIVATION', name: 'AMPK Activation', description: 'Energy sensor activation'});
CREATE (:PhysProcess {code: 'ENDOTHELIAL_FUNCTION', name: 'Endothelial Function', description: 'Vascular relaxation via NO'});
CREATE (:PhysProcess {code: 'RAAS_MODULATION', name: 'RAAS Modulation', description: 'Renin-angiotensin-aldosterone system'});
CREATE (:PhysProcess {code: 'SODIUM_RETENTION', name: 'Sodium Retention', description: 'Renal sodium reabsorption'});
CREATE (:PhysProcess {code: 'INSULIN_SECRETION', name: 'Insulin Secretion', description: 'Beta cell insulin release'});
CREATE (:PhysProcess {code: 'GLP1_SECRETION', name: 'GLP-1 Secretion', description: 'Incretin hormone release'});
CREATE (:PhysProcess {code: 'GASTRIC_EMPTYING', name: 'Gastric Emptying', description: 'Rate of stomach content transit'});
CREATE (:PhysProcess {code: 'MUSCLE_PROTEIN_SYNTHESIS', name: 'Muscle Protein Synthesis', description: 'Skeletal muscle repair and growth'});
CREATE (:PhysProcess {code: 'INFLAMMATORY_RESPONSE', name: 'Inflammatory Response', description: 'Systemic inflammation markers'});
CREATE (:PhysProcess {code: 'OXIDATIVE_STRESS', name: 'Oxidative Stress', description: 'ROS production and antioxidant balance'});
CREATE (:PhysProcess {code: 'SYMPATHETIC_TONE', name: 'Sympathetic Tone', description: 'Autonomic nervous system activation'});
CREATE (:PhysProcess {code: 'ADIPONECTIN_SECRETION', name: 'Adiponectin Secretion', description: 'Anti-inflammatory adipokine'});

// E-02: Triglyceride causal chain intermediate processes
CREATE (:PhysProcess {code: 'VLDL_PRODUCTION', name: 'Hepatic VLDL Production', description: 'Liver synthesis and secretion of very-low-density lipoproteins'});
CREATE (:PhysProcess {code: 'BILE_ACID_BINDING', name: 'Bile Acid Binding', description: 'Intestinal bile acid sequestration increasing hepatic cholesterol uptake'});
