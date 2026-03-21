// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'medication_adherence.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

MedicationAdherence _$MedicationAdherenceFromJson(Map<String, dynamic> json) {
  return _MedicationAdherence.fromJson(json);
}

/// @nodoc
mixin _$MedicationAdherence {
  int get weeklyPct => throw _privateConstructorUsedError;
  List<MedStreak> get streaks => throw _privateConstructorUsedError;
  MissedDose? get lastMissed => throw _privateConstructorUsedError;

  /// Serializes this MedicationAdherence to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of MedicationAdherence
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $MedicationAdherenceCopyWith<MedicationAdherence> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $MedicationAdherenceCopyWith<$Res> {
  factory $MedicationAdherenceCopyWith(
    MedicationAdherence value,
    $Res Function(MedicationAdherence) then,
  ) = _$MedicationAdherenceCopyWithImpl<$Res, MedicationAdherence>;
  @useResult
  $Res call({int weeklyPct, List<MedStreak> streaks, MissedDose? lastMissed});

  $MissedDoseCopyWith<$Res>? get lastMissed;
}

/// @nodoc
class _$MedicationAdherenceCopyWithImpl<$Res, $Val extends MedicationAdherence>
    implements $MedicationAdherenceCopyWith<$Res> {
  _$MedicationAdherenceCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of MedicationAdherence
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? weeklyPct = null,
    Object? streaks = null,
    Object? lastMissed = freezed,
  }) {
    return _then(
      _value.copyWith(
            weeklyPct: null == weeklyPct
                ? _value.weeklyPct
                : weeklyPct // ignore: cast_nullable_to_non_nullable
                      as int,
            streaks: null == streaks
                ? _value.streaks
                : streaks // ignore: cast_nullable_to_non_nullable
                      as List<MedStreak>,
            lastMissed: freezed == lastMissed
                ? _value.lastMissed
                : lastMissed // ignore: cast_nullable_to_non_nullable
                      as MissedDose?,
          )
          as $Val,
    );
  }

  /// Create a copy of MedicationAdherence
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $MissedDoseCopyWith<$Res>? get lastMissed {
    if (_value.lastMissed == null) {
      return null;
    }

    return $MissedDoseCopyWith<$Res>(_value.lastMissed!, (value) {
      return _then(_value.copyWith(lastMissed: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$MedicationAdherenceImplCopyWith<$Res>
    implements $MedicationAdherenceCopyWith<$Res> {
  factory _$$MedicationAdherenceImplCopyWith(
    _$MedicationAdherenceImpl value,
    $Res Function(_$MedicationAdherenceImpl) then,
  ) = __$$MedicationAdherenceImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({int weeklyPct, List<MedStreak> streaks, MissedDose? lastMissed});

  @override
  $MissedDoseCopyWith<$Res>? get lastMissed;
}

/// @nodoc
class __$$MedicationAdherenceImplCopyWithImpl<$Res>
    extends _$MedicationAdherenceCopyWithImpl<$Res, _$MedicationAdherenceImpl>
    implements _$$MedicationAdherenceImplCopyWith<$Res> {
  __$$MedicationAdherenceImplCopyWithImpl(
    _$MedicationAdherenceImpl _value,
    $Res Function(_$MedicationAdherenceImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of MedicationAdherence
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? weeklyPct = null,
    Object? streaks = null,
    Object? lastMissed = freezed,
  }) {
    return _then(
      _$MedicationAdherenceImpl(
        weeklyPct: null == weeklyPct
            ? _value.weeklyPct
            : weeklyPct // ignore: cast_nullable_to_non_nullable
                  as int,
        streaks: null == streaks
            ? _value._streaks
            : streaks // ignore: cast_nullable_to_non_nullable
                  as List<MedStreak>,
        lastMissed: freezed == lastMissed
            ? _value.lastMissed
            : lastMissed // ignore: cast_nullable_to_non_nullable
                  as MissedDose?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$MedicationAdherenceImpl implements _MedicationAdherence {
  const _$MedicationAdherenceImpl({
    required this.weeklyPct,
    required final List<MedStreak> streaks,
    this.lastMissed,
  }) : _streaks = streaks;

  factory _$MedicationAdherenceImpl.fromJson(Map<String, dynamic> json) =>
      _$$MedicationAdherenceImplFromJson(json);

  @override
  final int weeklyPct;
  final List<MedStreak> _streaks;
  @override
  List<MedStreak> get streaks {
    if (_streaks is EqualUnmodifiableListView) return _streaks;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_streaks);
  }

  @override
  final MissedDose? lastMissed;

  @override
  String toString() {
    return 'MedicationAdherence(weeklyPct: $weeklyPct, streaks: $streaks, lastMissed: $lastMissed)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$MedicationAdherenceImpl &&
            (identical(other.weeklyPct, weeklyPct) ||
                other.weeklyPct == weeklyPct) &&
            const DeepCollectionEquality().equals(other._streaks, _streaks) &&
            (identical(other.lastMissed, lastMissed) ||
                other.lastMissed == lastMissed));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    weeklyPct,
    const DeepCollectionEquality().hash(_streaks),
    lastMissed,
  );

  /// Create a copy of MedicationAdherence
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$MedicationAdherenceImplCopyWith<_$MedicationAdherenceImpl> get copyWith =>
      __$$MedicationAdherenceImplCopyWithImpl<_$MedicationAdherenceImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$MedicationAdherenceImplToJson(this);
  }
}

abstract class _MedicationAdherence implements MedicationAdherence {
  const factory _MedicationAdherence({
    required final int weeklyPct,
    required final List<MedStreak> streaks,
    final MissedDose? lastMissed,
  }) = _$MedicationAdherenceImpl;

  factory _MedicationAdherence.fromJson(Map<String, dynamic> json) =
      _$MedicationAdherenceImpl.fromJson;

  @override
  int get weeklyPct;
  @override
  List<MedStreak> get streaks;
  @override
  MissedDose? get lastMissed;

  /// Create a copy of MedicationAdherence
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$MedicationAdherenceImplCopyWith<_$MedicationAdherenceImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

MedStreak _$MedStreakFromJson(Map<String, dynamic> json) {
  return _MedStreak.fromJson(json);
}

/// @nodoc
mixin _$MedStreak {
  String get medicationName => throw _privateConstructorUsedError;
  int get streakDays => throw _privateConstructorUsedError;

  /// Serializes this MedStreak to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of MedStreak
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $MedStreakCopyWith<MedStreak> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $MedStreakCopyWith<$Res> {
  factory $MedStreakCopyWith(MedStreak value, $Res Function(MedStreak) then) =
      _$MedStreakCopyWithImpl<$Res, MedStreak>;
  @useResult
  $Res call({String medicationName, int streakDays});
}

/// @nodoc
class _$MedStreakCopyWithImpl<$Res, $Val extends MedStreak>
    implements $MedStreakCopyWith<$Res> {
  _$MedStreakCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of MedStreak
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? medicationName = null, Object? streakDays = null}) {
    return _then(
      _value.copyWith(
            medicationName: null == medicationName
                ? _value.medicationName
                : medicationName // ignore: cast_nullable_to_non_nullable
                      as String,
            streakDays: null == streakDays
                ? _value.streakDays
                : streakDays // ignore: cast_nullable_to_non_nullable
                      as int,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$MedStreakImplCopyWith<$Res>
    implements $MedStreakCopyWith<$Res> {
  factory _$$MedStreakImplCopyWith(
    _$MedStreakImpl value,
    $Res Function(_$MedStreakImpl) then,
  ) = __$$MedStreakImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String medicationName, int streakDays});
}

/// @nodoc
class __$$MedStreakImplCopyWithImpl<$Res>
    extends _$MedStreakCopyWithImpl<$Res, _$MedStreakImpl>
    implements _$$MedStreakImplCopyWith<$Res> {
  __$$MedStreakImplCopyWithImpl(
    _$MedStreakImpl _value,
    $Res Function(_$MedStreakImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of MedStreak
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? medicationName = null, Object? streakDays = null}) {
    return _then(
      _$MedStreakImpl(
        medicationName: null == medicationName
            ? _value.medicationName
            : medicationName // ignore: cast_nullable_to_non_nullable
                  as String,
        streakDays: null == streakDays
            ? _value.streakDays
            : streakDays // ignore: cast_nullable_to_non_nullable
                  as int,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$MedStreakImpl implements _MedStreak {
  const _$MedStreakImpl({
    required this.medicationName,
    required this.streakDays,
  });

  factory _$MedStreakImpl.fromJson(Map<String, dynamic> json) =>
      _$$MedStreakImplFromJson(json);

  @override
  final String medicationName;
  @override
  final int streakDays;

  @override
  String toString() {
    return 'MedStreak(medicationName: $medicationName, streakDays: $streakDays)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$MedStreakImpl &&
            (identical(other.medicationName, medicationName) ||
                other.medicationName == medicationName) &&
            (identical(other.streakDays, streakDays) ||
                other.streakDays == streakDays));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, medicationName, streakDays);

  /// Create a copy of MedStreak
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$MedStreakImplCopyWith<_$MedStreakImpl> get copyWith =>
      __$$MedStreakImplCopyWithImpl<_$MedStreakImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$MedStreakImplToJson(this);
  }
}

abstract class _MedStreak implements MedStreak {
  const factory _MedStreak({
    required final String medicationName,
    required final int streakDays,
  }) = _$MedStreakImpl;

  factory _MedStreak.fromJson(Map<String, dynamic> json) =
      _$MedStreakImpl.fromJson;

  @override
  String get medicationName;
  @override
  int get streakDays;

  /// Create a copy of MedStreak
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$MedStreakImplCopyWith<_$MedStreakImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

MissedDose _$MissedDoseFromJson(Map<String, dynamic> json) {
  return _MissedDose.fromJson(json);
}

/// @nodoc
mixin _$MissedDose {
  String get medicationName => throw _privateConstructorUsedError;
  int get daysAgo => throw _privateConstructorUsedError;

  /// Serializes this MissedDose to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of MissedDose
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $MissedDoseCopyWith<MissedDose> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $MissedDoseCopyWith<$Res> {
  factory $MissedDoseCopyWith(
    MissedDose value,
    $Res Function(MissedDose) then,
  ) = _$MissedDoseCopyWithImpl<$Res, MissedDose>;
  @useResult
  $Res call({String medicationName, int daysAgo});
}

/// @nodoc
class _$MissedDoseCopyWithImpl<$Res, $Val extends MissedDose>
    implements $MissedDoseCopyWith<$Res> {
  _$MissedDoseCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of MissedDose
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? medicationName = null, Object? daysAgo = null}) {
    return _then(
      _value.copyWith(
            medicationName: null == medicationName
                ? _value.medicationName
                : medicationName // ignore: cast_nullable_to_non_nullable
                      as String,
            daysAgo: null == daysAgo
                ? _value.daysAgo
                : daysAgo // ignore: cast_nullable_to_non_nullable
                      as int,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$MissedDoseImplCopyWith<$Res>
    implements $MissedDoseCopyWith<$Res> {
  factory _$$MissedDoseImplCopyWith(
    _$MissedDoseImpl value,
    $Res Function(_$MissedDoseImpl) then,
  ) = __$$MissedDoseImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String medicationName, int daysAgo});
}

/// @nodoc
class __$$MissedDoseImplCopyWithImpl<$Res>
    extends _$MissedDoseCopyWithImpl<$Res, _$MissedDoseImpl>
    implements _$$MissedDoseImplCopyWith<$Res> {
  __$$MissedDoseImplCopyWithImpl(
    _$MissedDoseImpl _value,
    $Res Function(_$MissedDoseImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of MissedDose
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? medicationName = null, Object? daysAgo = null}) {
    return _then(
      _$MissedDoseImpl(
        medicationName: null == medicationName
            ? _value.medicationName
            : medicationName // ignore: cast_nullable_to_non_nullable
                  as String,
        daysAgo: null == daysAgo
            ? _value.daysAgo
            : daysAgo // ignore: cast_nullable_to_non_nullable
                  as int,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$MissedDoseImpl implements _MissedDose {
  const _$MissedDoseImpl({required this.medicationName, required this.daysAgo});

  factory _$MissedDoseImpl.fromJson(Map<String, dynamic> json) =>
      _$$MissedDoseImplFromJson(json);

  @override
  final String medicationName;
  @override
  final int daysAgo;

  @override
  String toString() {
    return 'MissedDose(medicationName: $medicationName, daysAgo: $daysAgo)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$MissedDoseImpl &&
            (identical(other.medicationName, medicationName) ||
                other.medicationName == medicationName) &&
            (identical(other.daysAgo, daysAgo) || other.daysAgo == daysAgo));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, medicationName, daysAgo);

  /// Create a copy of MissedDose
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$MissedDoseImplCopyWith<_$MissedDoseImpl> get copyWith =>
      __$$MissedDoseImplCopyWithImpl<_$MissedDoseImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$MissedDoseImplToJson(this);
  }
}

abstract class _MissedDose implements MissedDose {
  const factory _MissedDose({
    required final String medicationName,
    required final int daysAgo,
  }) = _$MissedDoseImpl;

  factory _MissedDose.fromJson(Map<String, dynamic> json) =
      _$MissedDoseImpl.fromJson;

  @override
  String get medicationName;
  @override
  int get daysAgo;

  /// Create a copy of MissedDose
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$MissedDoseImplCopyWith<_$MissedDoseImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
