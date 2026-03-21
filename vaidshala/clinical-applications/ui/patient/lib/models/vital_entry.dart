import 'package:freezed_annotation/freezed_annotation.dart';

part 'vital_entry.freezed.dart';
part 'vital_entry.g.dart';

@freezed
class VitalEntry with _$VitalEntry {
  const factory VitalEntry({
    required String id,
    required String type, // bp, glucose, weight
    required String value, // JSON string
    required String unit,
    required DateTime timestamp,
    @Default(false) bool synced,
  }) = _VitalEntry;

  factory VitalEntry.fromJson(Map<String, dynamic> json) =>
      _$VitalEntryFromJson(json);
}
