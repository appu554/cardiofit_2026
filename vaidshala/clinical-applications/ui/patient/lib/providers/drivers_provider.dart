import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/driver.dart';
import 'api_client_provider.dart';
import 'auth_provider.dart';

final healthDriversProvider =
    FutureProvider<List<HealthDriver>>((ref) async {
  try {
    final authState = await ref.watch(authStateProvider.future);
    if (authState.patientId == null) return [];
    final api = ref.read(apiClientProvider);
    final resp =
        await api.dio.get('/tier1/patients/${authState.patientId}/drivers');
    return (resp.data['drivers'] as List)
        .map((j) => HealthDriver.fromJson(j as Map<String, dynamic>))
        .toList();
  } catch (_) {
    // Dev mock: Rajesh Kumar health drivers
    return const [
      HealthDriver(
        id: 'd1',
        name: 'Blood Sugar',
        icon: 'bloodtype',
        current: 178,
        target: 126,
        unit: 'mg/dL',
        improving: true,
      ),
      HealthDriver(
        id: 'd2',
        name: 'Blood Pressure',
        icon: 'favorite',
        current: 156,
        target: 140,
        unit: 'mmHg',
        improving: false,
      ),
      HealthDriver(
        id: 'd3',
        name: 'Kidney Function',
        icon: 'monitor_heart',
        current: 58,
        target: 60,
        unit: 'eGFR',
        improving: false,
      ),
      HealthDriver(
        id: 'd4',
        name: 'Activity',
        icon: 'directions_walk',
        current: 2100,
        target: 4000,
        unit: 'steps',
        improving: false,
      ),
    ];
  }
});
