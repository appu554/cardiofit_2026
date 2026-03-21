import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/family_view_data.dart';
import 'api_client_provider.dart';

final familyViewProvider =
    FutureProvider.family<FamilyViewData?, String>((ref, token) async {
  try {
    final api = ref.read(apiClientProvider);
    final response = await api.dio.get('/family/$token/today');
    return FamilyViewData.fromJson(response.data);
  } catch (_) {
    // Dev mock: Rajesh Kumar's family view
    return const FamilyViewData(
      patientName: 'Rajesh',
      mealActions: [
        FamilyAction(
          text: 'Prepare low-GI breakfast (oats, dal, vegetables)',
          icon: 'restaurant',
          time: '07:30',
        ),
        FamilyAction(
          text: 'Ensure 50g protein across meals today',
          icon: 'egg',
          time: 'All day',
        ),
        FamilyAction(
          text: 'Limit rice portion to 1 small bowl at lunch',
          icon: 'rice_bowl',
          time: '12:30',
        ),
        FamilyAction(
          text: 'Prepare dinner by 19:30 (early dinner helps glucose)',
          icon: 'dinner_dining',
          time: '19:30',
        ),
      ],
      activityActions: [
        FamilyAction(
          text: 'Encourage a 15-min walk after dinner together',
          icon: 'directions_walk',
          time: '20:00',
        ),
        FamilyAction(
          text: 'Remind to drink 8 glasses of water',
          icon: 'water_drop',
          time: 'All day',
        ),
      ],
      supportMessage:
          "Your support makes a real difference! Rajesh's care team appreciates your help with meals and encouragement.",
    );
  }
});
