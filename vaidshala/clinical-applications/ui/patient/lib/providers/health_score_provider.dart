import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/health_score.dart';
import 'api_client_provider.dart';
import 'auth_provider.dart';

final healthScoreProvider =
    FutureProvider<HealthScore?>((ref) async {
  try {
    final authState = await ref.watch(authStateProvider.future);
    if (authState.patientId == null) return null;
    final api = ref.read(apiClientProvider);
    final resp =
        await api.dio.get('/tier1/patients/${authState.patientId}/health-score');
    return HealthScore.fromJson(resp.data as Map<String, dynamic>);
  } catch (_) {
    // Dev mock: Rajesh Kumar MRI score
    return const HealthScore(
      score: 82,
      label: 'Needs Attention',
      delta: -3,
      sparkline: [78, 80, 82, 81, 79, 82],
    );
  }
});
