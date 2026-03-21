import 'package:freezed_annotation/freezed_annotation.dart';

part 'symptom_entry.freezed.dart';
part 'symptom_entry.g.dart';

@freezed
class SymptomEntry with _$SymptomEntry {
  const factory SymptomEntry({
    required String id,
    required String symptom, // comma-separated if multiple
    required String severity, // mild, moderate, severe
    String? notes,
    required DateTime timestamp,
    @Default(false) bool synced,
  }) = _SymptomEntry;

  factory SymptomEntry.fromJson(Map<String, dynamic> json) =>
      _$SymptomEntryFromJson(json);
}
