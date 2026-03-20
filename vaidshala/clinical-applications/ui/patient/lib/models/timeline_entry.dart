import 'package:freezed_annotation/freezed_annotation.dart';

part 'timeline_entry.freezed.dart';
part 'timeline_entry.g.dart';

@freezed
class TimelineEntry with _$TimelineEntry {
  const factory TimelineEntry({
    required String id,
    required String time,
    required String text,
    required String icon,
    required bool done,
  }) = _TimelineEntry;

  factory TimelineEntry.fromJson(Map<String, dynamic> json) =>
      _$TimelineEntryFromJson(json);
}
