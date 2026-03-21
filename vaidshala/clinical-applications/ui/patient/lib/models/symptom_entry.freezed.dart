// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'symptom_entry.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

SymptomEntry _$SymptomEntryFromJson(Map<String, dynamic> json) {
  return _SymptomEntry.fromJson(json);
}

/// @nodoc
mixin _$SymptomEntry {
  String get id => throw _privateConstructorUsedError;
  String get symptom =>
      throw _privateConstructorUsedError; // comma-separated if multiple
  String get severity =>
      throw _privateConstructorUsedError; // mild, moderate, severe
  String? get notes => throw _privateConstructorUsedError;
  DateTime get timestamp => throw _privateConstructorUsedError;
  bool get synced => throw _privateConstructorUsedError;

  /// Serializes this SymptomEntry to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of SymptomEntry
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SymptomEntryCopyWith<SymptomEntry> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SymptomEntryCopyWith<$Res> {
  factory $SymptomEntryCopyWith(
    SymptomEntry value,
    $Res Function(SymptomEntry) then,
  ) = _$SymptomEntryCopyWithImpl<$Res, SymptomEntry>;
  @useResult
  $Res call({
    String id,
    String symptom,
    String severity,
    String? notes,
    DateTime timestamp,
    bool synced,
  });
}

/// @nodoc
class _$SymptomEntryCopyWithImpl<$Res, $Val extends SymptomEntry>
    implements $SymptomEntryCopyWith<$Res> {
  _$SymptomEntryCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SymptomEntry
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? symptom = null,
    Object? severity = null,
    Object? notes = freezed,
    Object? timestamp = null,
    Object? synced = null,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as String,
            symptom: null == symptom
                ? _value.symptom
                : symptom // ignore: cast_nullable_to_non_nullable
                      as String,
            severity: null == severity
                ? _value.severity
                : severity // ignore: cast_nullable_to_non_nullable
                      as String,
            notes: freezed == notes
                ? _value.notes
                : notes // ignore: cast_nullable_to_non_nullable
                      as String?,
            timestamp: null == timestamp
                ? _value.timestamp
                : timestamp // ignore: cast_nullable_to_non_nullable
                      as DateTime,
            synced: null == synced
                ? _value.synced
                : synced // ignore: cast_nullable_to_non_nullable
                      as bool,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$SymptomEntryImplCopyWith<$Res>
    implements $SymptomEntryCopyWith<$Res> {
  factory _$$SymptomEntryImplCopyWith(
    _$SymptomEntryImpl value,
    $Res Function(_$SymptomEntryImpl) then,
  ) = __$$SymptomEntryImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String id,
    String symptom,
    String severity,
    String? notes,
    DateTime timestamp,
    bool synced,
  });
}

/// @nodoc
class __$$SymptomEntryImplCopyWithImpl<$Res>
    extends _$SymptomEntryCopyWithImpl<$Res, _$SymptomEntryImpl>
    implements _$$SymptomEntryImplCopyWith<$Res> {
  __$$SymptomEntryImplCopyWithImpl(
    _$SymptomEntryImpl _value,
    $Res Function(_$SymptomEntryImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of SymptomEntry
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? symptom = null,
    Object? severity = null,
    Object? notes = freezed,
    Object? timestamp = null,
    Object? synced = null,
  }) {
    return _then(
      _$SymptomEntryImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as String,
        symptom: null == symptom
            ? _value.symptom
            : symptom // ignore: cast_nullable_to_non_nullable
                  as String,
        severity: null == severity
            ? _value.severity
            : severity // ignore: cast_nullable_to_non_nullable
                  as String,
        notes: freezed == notes
            ? _value.notes
            : notes // ignore: cast_nullable_to_non_nullable
                  as String?,
        timestamp: null == timestamp
            ? _value.timestamp
            : timestamp // ignore: cast_nullable_to_non_nullable
                  as DateTime,
        synced: null == synced
            ? _value.synced
            : synced // ignore: cast_nullable_to_non_nullable
                  as bool,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$SymptomEntryImpl implements _SymptomEntry {
  const _$SymptomEntryImpl({
    required this.id,
    required this.symptom,
    required this.severity,
    this.notes,
    required this.timestamp,
    this.synced = false,
  });

  factory _$SymptomEntryImpl.fromJson(Map<String, dynamic> json) =>
      _$$SymptomEntryImplFromJson(json);

  @override
  final String id;
  @override
  final String symptom;
  // comma-separated if multiple
  @override
  final String severity;
  // mild, moderate, severe
  @override
  final String? notes;
  @override
  final DateTime timestamp;
  @override
  @JsonKey()
  final bool synced;

  @override
  String toString() {
    return 'SymptomEntry(id: $id, symptom: $symptom, severity: $severity, notes: $notes, timestamp: $timestamp, synced: $synced)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SymptomEntryImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.symptom, symptom) || other.symptom == symptom) &&
            (identical(other.severity, severity) ||
                other.severity == severity) &&
            (identical(other.notes, notes) || other.notes == notes) &&
            (identical(other.timestamp, timestamp) ||
                other.timestamp == timestamp) &&
            (identical(other.synced, synced) || other.synced == synced));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode =>
      Object.hash(runtimeType, id, symptom, severity, notes, timestamp, synced);

  /// Create a copy of SymptomEntry
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SymptomEntryImplCopyWith<_$SymptomEntryImpl> get copyWith =>
      __$$SymptomEntryImplCopyWithImpl<_$SymptomEntryImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$SymptomEntryImplToJson(this);
  }
}

abstract class _SymptomEntry implements SymptomEntry {
  const factory _SymptomEntry({
    required final String id,
    required final String symptom,
    required final String severity,
    final String? notes,
    required final DateTime timestamp,
    final bool synced,
  }) = _$SymptomEntryImpl;

  factory _SymptomEntry.fromJson(Map<String, dynamic> json) =
      _$SymptomEntryImpl.fromJson;

  @override
  String get id;
  @override
  String get symptom; // comma-separated if multiple
  @override
  String get severity; // mild, moderate, severe
  @override
  String? get notes;
  @override
  DateTime get timestamp;
  @override
  bool get synced;

  /// Create a copy of SymptomEntry
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SymptomEntryImplCopyWith<_$SymptomEntryImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
