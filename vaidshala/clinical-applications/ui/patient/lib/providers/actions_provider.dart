import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/action.dart';
import 'api_client_provider.dart';
import 'auth_provider.dart';

class ActionsState {
  final List<PatientAction> actions;
  final bool isLoading;

  const ActionsState({
    this.actions = const [],
    this.isLoading = false,
  });

  int get completionPct {
    if (actions.isEmpty) return 0;
    final done = actions.where((a) => a.done).length;
    return (done / actions.length * 100).round();
  }

  ActionsState copyWith({
    List<PatientAction>? actions,
    bool? isLoading,
  }) =>
      ActionsState(
        actions: actions ?? this.actions,
        isLoading: isLoading ?? this.isLoading,
      );
}

final actionsProvider =
    NotifierProvider<ActionsNotifier, ActionsState>(ActionsNotifier.new);

class ActionsNotifier extends Notifier<ActionsState> {
  @override
  ActionsState build() {
    _fetch();
    return const ActionsState(isLoading: true);
  }

  Future<void> _fetch() async {
    try {
      final authState = await ref.read(authStateProvider.future);
      if (authState.patientId == null) {
        state = const ActionsState();
        return;
      }
      final api = ref.read(apiClientProvider);
      final resp =
          await api.dio.get('/tier1/patients/${authState.patientId}/actions');
      final list = (resp.data['actions'] as List)
          .map((j) => PatientAction.fromJson(j as Map<String, dynamic>))
          .toList();
      state = ActionsState(actions: list);
    } catch (_) {
      // Dev mock: Rajesh Kumar's daily actions
      state = const ActionsState(actions: [
        PatientAction(
          id: 'a1',
          text: 'Take Metformin 500mg',
          icon: 'medication',
          time: '8:00 AM',
          done: true,
          why: 'Helps lower fasting blood sugar by reducing liver glucose production',
        ),
        PatientAction(
          id: 'a2',
          text: 'Take Telmisartan 40mg',
          icon: 'medication',
          time: '8:00 AM',
          done: true,
          why: 'Controls blood pressure and protects kidney function',
        ),
        PatientAction(
          id: 'a3',
          text: 'Check fasting blood glucose',
          icon: 'bloodtype',
          time: '7:00 AM',
          done: false,
          why: 'Tracking FBG helps your doctor adjust medications',
        ),
        PatientAction(
          id: 'a4',
          text: '15 min post-dinner walk',
          icon: 'directions_walk',
          time: '8:30 PM',
          done: false,
          why: 'Walking after meals can lower post-meal glucose by 15-20%',
        ),
        PatientAction(
          id: 'a5',
          text: 'Log blood pressure',
          icon: 'favorite',
          time: '9:00 PM',
          done: false,
        ),
      ]);
    }
  }

  void toggleAction(String id) {
    final updated = state.actions.map((a) {
      if (a.id == id) return a.copyWith(done: !a.done);
      return a;
    }).toList();
    state = state.copyWith(actions: updated);
  }

  Future<void> refresh() async {
    state = state.copyWith(isLoading: true);
    await _fetch();
  }
}
