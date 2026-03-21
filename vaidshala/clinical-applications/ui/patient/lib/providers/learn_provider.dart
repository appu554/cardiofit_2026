import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/clinical_translation.dart';
import 'insights_provider.dart';

class LearnState {
  final List<String> alerts;
  final List<String> tips;
  final List<ClinicalTranslation> translations;

  const LearnState({
    this.alerts = const [],
    this.tips = const [],
    this.translations = const [],
  });
}

final learnProvider = Provider<LearnState>((ref) {
  final insightsAsync = ref.watch(insightsProvider);

  return insightsAsync.when(
    data: (insight) => LearnState(
      alerts: insight.alerts,
      tips: insight.tips,
      translations: _defaultTranslations,
    ),
    loading: () => const LearnState(translations: _defaultTranslations),
    error: (_, __) => const LearnState(translations: _defaultTranslations),
  );
});

const _defaultTranslations = [
  ClinicalTranslation(
    clinicalTerm: 'HbA1c',
    patientTerm: 'Average blood sugar over 3 months',
    explanation:
        'A blood test that shows your average blood sugar level over the past 2-3 months. Target is usually below 7%.',
  ),
  ClinicalTranslation(
    clinicalTerm: 'eGFR',
    patientTerm: 'How well your kidneys filter blood',
    explanation:
        'Estimated Glomerular Filtration Rate measures kidney function. Higher is better — above 60 is normal.',
  ),
  ClinicalTranslation(
    clinicalTerm: 'FBG (Fasting Blood Glucose)',
    patientTerm: 'Blood sugar level before eating',
    explanation:
        'Measured after 8+ hours of fasting. Normal is below 100 mg/dL, target for diabetes management is below 126 mg/dL.',
  ),
  ClinicalTranslation(
    clinicalTerm: 'Systolic Blood Pressure',
    patientTerm: 'Pressure when your heart beats',
    explanation:
        'The top number in your blood pressure reading. Target is usually below 140 mmHg.',
  ),
  ClinicalTranslation(
    clinicalTerm: 'Metformin',
    patientTerm: 'Medicine that lowers blood sugar',
    explanation:
        'A first-line diabetes medication that reduces sugar production by the liver. Take with food to reduce stomach upset.',
  ),
  ClinicalTranslation(
    clinicalTerm: 'ARB (Telmisartan)',
    patientTerm: 'Blood pressure medicine that protects kidneys',
    explanation:
        'Angiotensin Receptor Blocker — lowers blood pressure and provides extra protection for your kidneys.',
  ),
  ClinicalTranslation(
    clinicalTerm: 'BMI',
    patientTerm: 'Body weight relative to height',
    explanation:
        "Body Mass Index — a screening tool. Rajesh's BMI is 29.4, which is in the overweight range (target: below 25).",
  ),
  ClinicalTranslation(
    clinicalTerm: 'Creatinine',
    patientTerm: 'Waste product filtered by kidneys',
    explanation:
        'A substance your muscles produce that kidneys filter out. High levels may indicate kidney issues.',
  ),
];
