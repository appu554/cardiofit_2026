// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'vital_entry.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

VitalEntry _$VitalEntryFromJson(Map<String, dynamic> json) {
  return _VitalEntry.fromJson(json);
}

/// @nodoc
mixin _$VitalEntry {
  String get id => throw _privateConstructorUsedError;
  String get type => throw _privateConstructorUsedError; // bp, glucose, weight
  String get value => throw _privateConstructorUsedError; // JSON string
  String get unit => throw _privateConstructorUsedError;
  DateTime get timestamp => throw _privateConstructorUsedError;
  bool get synced => throw _privateConstructorUsedError;

  /// Serializes this VitalEntry to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of VitalEntry
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $VitalEntryCopyWith<VitalEntry> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $VitalEntryCopyWith<$Res> {
  factory $VitalEntryCopyWith(
    VitalEntry value,
    $Res Function(VitalEntry) then,
  ) = _$VitalEntryCopyWithImpl<$Res, VitalEntry>;
  @useResult
  $Res call({
    String id,
    String type,
    String value,
    String unit,
    DateTime timestamp,
    bool synced,
  });
}

/// @nodoc
class _$VitalEntryCopyWithImpl<$Res, $Val extends VitalEntry>
    implements $VitalEntryCopyWith<$Res> {
  _$VitalEntryCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of VitalEntry
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? type = null,
    Object? value = null,
    Object? unit = null,
    Object? timestamp = null,
    Object? synced = null,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as String,
            type: null == type
                ? _value.type
                : type // ignore: cast_nullable_to_non_nullable
                      as String,
            value: null == value
                ? _value.value
                : value // ignore: cast_nullable_to_non_nullable
                      as String,
            unit: null == unit
                ? _value.unit
                : unit // ignore: cast_nullable_to_non_nullable
                      as String,
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
abstract class _$$VitalEntryImplCopyWith<$Res>
    implements $VitalEntryCopyWith<$Res> {
  factory _$$VitalEntryImplCopyWith(
    _$VitalEntryImpl value,
    $Res Function(_$VitalEntryImpl) then,
  ) = __$$VitalEntryImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String id,
    String type,
    String value,
    String unit,
    DateTime timestamp,
    bool synced,
  });
}

/// @nodoc
class __$$VitalEntryImplCopyWithImpl<$Res>
    extends _$VitalEntryCopyWithImpl<$Res, _$VitalEntryImpl>
    implements _$$VitalEntryImplCopyWith<$Res> {
  __$$VitalEntryImplCopyWithImpl(
    _$VitalEntryImpl _value,
    $Res Function(_$VitalEntryImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of VitalEntry
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? type = null,
    Object? value = null,
    Object? unit = null,
    Object? timestamp = null,
    Object? synced = null,
  }) {
    return _then(
      _$VitalEntryImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as String,
        type: null == type
            ? _value.type
            : type // ignore: cast_nullable_to_non_nullable
                  as String,
        value: null == value
            ? _value.value
            : value // ignore: cast_nullable_to_non_nullable
                  as String,
        unit: null == unit
            ? _value.unit
            : unit // ignore: cast_nullable_to_non_nullable
                  as String,
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
class _$VitalEntryImpl implements _VitalEntry {
  const _$VitalEntryImpl({
    required this.id,
    required this.type,
    required this.value,
    required this.unit,
    required this.timestamp,
    this.synced = false,
  });

  factory _$VitalEntryImpl.fromJson(Map<String, dynamic> json) =>
      _$$VitalEntryImplFromJson(json);

  @override
  final String id;
  @override
  final String type;
  // bp, glucose, weight
  @override
  final String value;
  // JSON string
  @override
  final String unit;
  @override
  final DateTime timestamp;
  @override
  @JsonKey()
  final bool synced;

  @override
  String toString() {
    return 'VitalEntry(id: $id, type: $type, value: $value, unit: $unit, timestamp: $timestamp, synced: $synced)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$VitalEntryImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.type, type) || other.type == type) &&
            (identical(other.value, value) || other.value == value) &&
            (identical(other.unit, unit) || other.unit == unit) &&
            (identical(other.timestamp, timestamp) ||
                other.timestamp == timestamp) &&
            (identical(other.synced, synced) || other.synced == synced));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode =>
      Object.hash(runtimeType, id, type, value, unit, timestamp, synced);

  /// Create a copy of VitalEntry
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$VitalEntryImplCopyWith<_$VitalEntryImpl> get copyWith =>
      __$$VitalEntryImplCopyWithImpl<_$VitalEntryImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$VitalEntryImplToJson(this);
  }
}

abstract class _VitalEntry implements VitalEntry {
  const factory _VitalEntry({
    required final String id,
    required final String type,
    required final String value,
    required final String unit,
    required final DateTime timestamp,
    final bool synced,
  }) = _$VitalEntryImpl;

  factory _VitalEntry.fromJson(Map<String, dynamic> json) =
      _$VitalEntryImpl.fromJson;

  @override
  String get id;
  @override
  String get type; // bp, glucose, weight
  @override
  String get value; // JSON string
  @override
  String get unit;
  @override
  DateTime get timestamp;
  @override
  bool get synced;

  /// Create a copy of VitalEntry
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$VitalEntryImplCopyWith<_$VitalEntryImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
