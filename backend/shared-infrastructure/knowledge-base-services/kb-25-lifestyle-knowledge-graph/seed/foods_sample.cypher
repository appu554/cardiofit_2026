// Food nodes — representative sample from IFCT-2017 (spec §9.1)
CREATE (:Food {code: 'F001', name: 'Brown Rice', name_local: 'भूरे चावल', region: 'ALL', diet_type: 'VEG', category: 'CEREAL', food_group: 'GRAINS', gi: 50, gl: 16, serving_size_g: 150, fiber_g: 1.8, sodium_mg: 1, potassium_mg: 77});
CREATE (:Food {code: 'F002', name: 'Ragi (Finger Millet)', name_local: 'रागी', region: 'SOUTH', diet_type: 'VEG', category: 'MILLET', food_group: 'GRAINS', gi: 54, gl: 18, serving_size_g: 100, fiber_g: 3.6, sodium_mg: 11, potassium_mg: 408});
CREATE (:Food {code: 'F003', name: 'Bajra (Pearl Millet)', name_local: 'बाजरा', region: 'WEST', diet_type: 'VEG', category: 'MILLET', food_group: 'GRAINS', gi: 55, gl: 17, serving_size_g: 100, fiber_g: 1.2, sodium_mg: 5, potassium_mg: 307});
CREATE (:Food {code: 'F004', name: 'Moong Dal', name_local: 'मूंग दाल', region: 'ALL', diet_type: 'VEG', category: 'PULSE', food_group: 'LEGUMES', gi: 31, gl: 5, serving_size_g: 50, fiber_g: 4.1, sodium_mg: 7, potassium_mg: 334});
CREATE (:Food {code: 'F005', name: 'Methi (Fenugreek) Leaves', name_local: 'मेथी', region: 'ALL', diet_type: 'VEG', category: 'LEAFY_GREEN', food_group: 'VEGETABLES', gi: 15, gl: 1, serving_size_g: 50, fiber_g: 1.1, sodium_mg: 58, potassium_mg: 31});
CREATE (:Food {code: 'F006', name: 'Bitter Gourd (Karela)', name_local: 'करेला', region: 'ALL', diet_type: 'VEG', category: 'VEGETABLE', food_group: 'VEGETABLES', gi: 24, gl: 2, serving_size_g: 100, fiber_g: 2.8, sodium_mg: 5, potassium_mg: 296});
CREATE (:Food {code: 'F007', name: 'Curd (Dahi)', name_local: 'दही', region: 'ALL', diet_type: 'VEG', category: 'DAIRY', food_group: 'DAIRY', gi: 36, gl: 2, serving_size_g: 100, fiber_g: 0, sodium_mg: 40, potassium_mg: 234});
CREATE (:Food {code: 'F008', name: 'Mackerel (Bangda)', name_local: 'बांगडा', region: 'WEST', diet_type: 'NON_VEG', category: 'FISH', food_group: 'SEAFOOD', gi: 0, gl: 0, serving_size_g: 100, fiber_g: 0, sodium_mg: 71, potassium_mg: 314});
CREATE (:Food {code: 'F009', name: 'Almonds', name_local: 'बादाम', region: 'ALL', diet_type: 'VEG', category: 'NUT', food_group: 'NUTS', gi: 15, gl: 0.6, serving_size_g: 30, fiber_g: 3.5, sodium_mg: 1, potassium_mg: 200});
CREATE (:Food {code: 'F010', name: 'Flaxseed (Alsi)', name_local: 'अलसी', region: 'NORTH', diet_type: 'VEG', category: 'SEED', food_group: 'SEEDS', gi: 0, gl: 0, serving_size_g: 15, fiber_g: 4.1, sodium_mg: 5, potassium_mg: 122});
CREATE (:Food {code: 'F011', name: 'Jamun (Indian Blackberry)', name_local: 'जामुन', region: 'ALL', diet_type: 'VEG', category: 'FRUIT', food_group: 'FRUITS', gi: 25, gl: 3, serving_size_g: 100, fiber_g: 0.6, sodium_mg: 14, potassium_mg: 55});
CREATE (:Food {code: 'F012', name: 'Turmeric', name_local: 'हल्दी', region: 'ALL', diet_type: 'VEG', category: 'SPICE', food_group: 'SPICES', gi: 0, gl: 0, serving_size_g: 5, fiber_g: 1.1, sodium_mg: 2, potassium_mg: 125});
CREATE (:Food {code: 'F013', name: 'Coconut Oil', name_local: 'नारियल तेल', region: 'SOUTH', diet_type: 'VEG', category: 'OIL', food_group: 'FATS', gi: 0, gl: 0, serving_size_g: 15, fiber_g: 0, sodium_mg: 0, potassium_mg: 0});
CREATE (:Food {code: 'F014', name: 'White Rice', name_local: 'सफेद चावल', region: 'ALL', diet_type: 'VEG', category: 'CEREAL', food_group: 'GRAINS', gi: 73, gl: 30, serving_size_g: 150, fiber_g: 0.4, sodium_mg: 1, potassium_mg: 35});
CREATE (:Food {code: 'F015', name: 'Maida (Refined Flour)', name_local: 'मैदा', region: 'ALL', diet_type: 'VEG', category: 'REFINED_CEREAL', food_group: 'GRAINS', gi: 71, gl: 25, serving_size_g: 100, fiber_g: 0.7, sodium_mg: 2, potassium_mg: 107});
CREATE (:Food {code: 'F016', name: 'Idli', name_local: 'इडली', region: 'SOUTH', diet_type: 'VEG', category: 'FERMENTED', food_group: 'GRAINS', gi: 63, gl: 14, serving_size_g: 80, fiber_g: 0.9, sodium_mg: 390, potassium_mg: 60});
CREATE (:Food {code: 'F017', name: 'Chana Dal', name_local: 'चना दाल', region: 'ALL', diet_type: 'VEG', category: 'PULSE', food_group: 'LEGUMES', gi: 28, gl: 4, serving_size_g: 50, fiber_g: 5.2, sodium_mg: 20, potassium_mg: 305});
CREATE (:Food {code: 'F018', name: 'Spinach (Palak)', name_local: 'पालक', region: 'ALL', diet_type: 'VEG', category: 'LEAFY_GREEN', food_group: 'VEGETABLES', gi: 15, gl: 1, serving_size_g: 100, fiber_g: 2.2, sodium_mg: 79, potassium_mg: 558});
CREATE (:Food {code: 'F019', name: 'Guava (Amrood)', name_local: 'अमरूद', region: 'ALL', diet_type: 'VEG', category: 'FRUIT', food_group: 'FRUITS', gi: 12, gl: 2, serving_size_g: 100, fiber_g: 5.4, sodium_mg: 2, potassium_mg: 417});
CREATE (:Food {code: 'F020', name: 'Egg (Whole)', name_local: 'अंडा', region: 'ALL', diet_type: 'EGG', category: 'EGG', food_group: 'EGGS', gi: 0, gl: 0, serving_size_g: 50, fiber_g: 0, sodium_mg: 124, potassium_mg: 126});

// Add leucine_g property to high-protein foods (from IFCT-2017 amino acid tables + FAO/INFOODS)
// Egg whole: 1.09g/100g, Paneer: 0.98g/100g, Chicken breast: 1.73g/100g
// Moong dal: 0.82g/100g, Dahi/curd: 0.45g/100g, Mackerel: 1.42g/100g
// Chana dal: 0.89g/100g, Almonds: 1.01g/100g
MATCH (f:Food) WHERE f.code IN ['egg_whole', 'paneer', 'moong_dal', 'dahi', 'mackerel', 'chana_dal', 'almonds']
SET f.leucine_g = CASE f.code
  WHEN 'egg_whole' THEN 1.09
  WHEN 'paneer' THEN 0.98
  WHEN 'moong_dal' THEN 0.82
  WHEN 'dahi' THEN 0.45
  WHEN 'mackerel' THEN 1.42
  WHEN 'chana_dal' THEN 0.89
  WHEN 'almonds' THEN 1.01
END;
