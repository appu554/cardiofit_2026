import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/insight.dart';
import 'api_client_provider.dart';
import 'auth_provider.dart';

final insightsProvider = FutureProvider<Insight>((ref) async {
  try {
    final authState = await ref.watch(authStateProvider.future);
    if (authState.patientId == null) return const Insight();
    final api = ref.read(apiClientProvider);
    final resp =
        await api.dio.get('/tier1/patients/${authState.patientId}/insights');
    return Insight.fromJson(resp.data as Map<String, dynamic>);
  } catch (_) {
    // Dev mock: Rajesh Kumar coaching insight
    return const Insight(
      coachingMessage:
          'Your fasting glucose dropped from 185 to 178 mg/dL this week. '
          'Consistent Metformin use and your evening walks are making a difference!',
      coachingType: InsightType.encouragement,
      tips: [
        'A 15-minute walk after dinner can lower post-meal glucose by 15-20%',
        'Staying hydrated helps your kidneys filter waste more efficiently',
        'Taking medications at the same time each day improves their effectiveness',
      ],
      alerts: [
        'Your blood pressure has been trending upward — remember to log readings',
      ],
    );
  }
});
