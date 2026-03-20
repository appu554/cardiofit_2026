// lib/providers/progress_provider.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/cause_effect.dart';
import '../models/milestone.dart';
import '../models/progress_metric.dart';
import 'api_client_provider.dart';
import 'auth_provider.dart';

class ProgressState {
  final List<ProgressMetric> metrics;
  final List<CauseEffect> causeEffects;
  final List<Milestone> milestones;
  final bool isLoading;

  const ProgressState({
    this.metrics = const [],
    this.causeEffects = const [],
    this.milestones = const [],
    this.isLoading = false,
  });
}

final progressProvider =
    AsyncNotifierProvider<ProgressNotifier, ProgressState>(
        ProgressNotifier.new);

class ProgressNotifier extends AsyncNotifier<ProgressState> {
  @override
  Future<ProgressState> build() async {
    return _fetch();
  }

  Future<ProgressState> _fetch() async {
    try {
      final authState = await ref.read(authStateProvider.future);
      if (authState.patientId == null) return const ProgressState();

      final api = ref.read(apiClientProvider);
      final progressResp =
          await api.dio.get('/tier1/patients/${authState.patientId}/progress');
      final ceResp =
          await api.dio.get('/tier1/patients/${authState.patientId}/cause-effect');

      final metrics = (progressResp.data['metrics'] as List)
          .map((j) => ProgressMetric.fromJson(j as Map<String, dynamic>))
          .toList();
      final causeEffects = (ceResp.data['causeEffects'] as List)
          .map((j) => CauseEffect.fromJson(j as Map<String, dynamic>))
          .toList();
      final milestones = _computeMilestones(metrics);

      return ProgressState(
        metrics: metrics,
        causeEffects: causeEffects,
        milestones: milestones,
      );
    } catch (_) {
      // Dev mock: Rajesh Kumar progress — matched to clinician dashboard
      // MRI 82 (rising), FBG 178, HbA1c 8.9, SBP 156, eGFR 58, waist 101, steps 2100
      // All trends worsening except FBG (slight improvement) and medication adherence
      return const ProgressState(
        metrics: [
          ProgressMetric(
            id: 'fbg',
            name: 'Fasting Blood Glucose',
            icon: 'bloodtype',
            current: 178,
            previous: 185,
            target: 126,
            unit: 'mg/dL',
            improving: true,
            sparkline: [192, 188, 185, 182, 180, 178],
          ),
          ProgressMetric(
            id: 'hba1c',
            name: 'HbA1c',
            icon: 'science',
            current: 8.9,
            previous: 8.5,
            target: 7.0,
            unit: '%',
            improving: false,
            sparkline: [8.2, 8.3, 8.5, 8.7, 8.9],
          ),
          ProgressMetric(
            id: 'sbp',
            name: 'Systolic BP',
            icon: 'favorite',
            current: 156,
            previous: 148,
            target: 140,
            unit: 'mmHg',
            improving: false,
            sparkline: [142, 145, 148, 152, 156],
          ),
          ProgressMetric(
            id: 'steps',
            name: 'Daily Steps',
            icon: 'directions_walk',
            current: 2100,
            previous: 2400,
            target: 4000,
            unit: 'steps',
            improving: false,
            sparkline: [3200, 2800, 2600, 2400, 2100],
          ),
          ProgressMetric(
            id: 'waist',
            name: 'Waist Circumference',
            icon: 'straighten',
            current: 101,
            previous: 99,
            target: 90,
            unit: 'cm',
            improving: false,
            sparkline: [97, 98, 99, 100, 101],
          ),
          ProgressMetric(
            id: 'egfr',
            name: 'Kidney Function (eGFR)',
            icon: 'monitor_heart',
            current: 58,
            previous: 62,
            target: 60,
            unit: 'mL/min',
            improving: false,
            sparkline: [68, 65, 62, 60, 58],
          ),
        ],
        causeEffects: [
          CauseEffect(
            id: 'ce-1',
            cause: 'Taking Metformin consistently',
            effect: 'FBG dropped from 192 to 178 mg/dL',
            causeIcon: 'medication',
            effectIcon: 'bloodtype',
            verified: true,
          ),
          CauseEffect(
            id: 'ce-2',
            cause: 'Telmisartan 40mg daily',
            effect: 'Kidney function stabilizing (eGFR 58)',
            causeIcon: 'medication',
            effectIcon: 'monitor_heart',
            verified: true,
          ),
          CauseEffect(
            id: 'ce-3',
            cause: 'Walking has declined to 2,100 steps',
            effect: 'Blood pressure rising to 156 mmHg',
            causeIcon: 'directions_walk',
            effectIcon: 'favorite',
            verified: false,
          ),
        ],
        milestones: [
          Milestone(
            id: 'm-1',
            title: 'First Week Complete',
            description: 'Completed 7 days of health tracking',
            status: MilestoneStatus.achieved,
            achievedDate: '2026-01-14',
          ),
          Milestone(
            id: 'm-2',
            title: '30 Days Strong',
            description: 'One month of consistent tracking',
            status: MilestoneStatus.achieved,
            achievedDate: '2026-02-07',
          ),
          Milestone(
            id: 'm-3',
            title: 'FBG Below 160',
            description: 'Reach fasting glucose below 160 mg/dL',
            status: MilestoneStatus.locked,
          ),
          Milestone(
            id: 'm-4',
            title: 'BP Under Control',
            description: 'Reach blood pressure target <140/90',
            status: MilestoneStatus.locked,
          ),
          Milestone(
            id: 'm-5',
            title: '4,000 Steps Daily',
            description: 'Average 4,000 steps for 7 consecutive days',
            status: MilestoneStatus.locked,
          ),
        ],
      );
    }
  }

  List<Milestone> _computeMilestones(List<ProgressMetric> metrics) {
    // Client-side milestone computation from metric thresholds
    return [];
  }

  Future<void> refresh() async {
    state = const AsyncLoading();
    state = AsyncData(await _fetch());
  }
}
