// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'driver.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

HealthDriver _$HealthDriverFromJson(Map<String, dynamic> json) {
  return _HealthDriver.fromJson(json);
}

/// @nodoc
mixin _$HealthDriver {
  String get id => throw _privateConstructorUsedError;
  String get name => throw _privateConstructorUsedError;
  String get icon => throw _privateConstructorUsedError;
  double get current => throw _privateConstructorUsedError;
  double get target => throw _privateConstructorUsedError;
  String get unit => throw _privateConstructorUsedError;
  bool get improving => throw _privateConstructorUsedError;

  /// Serializes this HealthDriver to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of HealthDriver
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $HealthDriverCopyWith<HealthDriver> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $HealthDriverCopyWith<$Res> {
  factory $HealthDriverCopyWith(
    HealthDriver value,
    $Res Function(HealthDriver) then,
  ) = _$HealthDriverCopyWithImpl<$Res, HealthDriver>;
  @useResult
  $Res call({
    String id,
    String name,
    String icon,
    double current,
    double target,
    String unit,
    bool improving,
  });
}

/// @nodoc
class _$HealthDriverCopyWithImpl<$Res, $Val extends HealthDriver>
    implements $HealthDriverCopyWith<$Res> {
  _$HealthDriverCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of HealthDriver
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? name = null,
    Object? icon = null,
    Object? current = null,
    Object? target = null,
    Object? unit = null,
    Object? improving = null,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as String,
            name: null == name
                ? _value.name
                : name // ignore: cast_nullable_to_non_nullable
                      as String,
            icon: null == icon
                ? _value.icon
                : icon // ignore: cast_nullable_to_non_nullable
                      as String,
            current: null == current
                ? _value.current
                : current // ignore: cast_nullable_to_non_nullable
                      as double,
            target: null == target
                ? _value.target
                : target // ignore: cast_nullable_to_non_nullable
                      as double,
            unit: null == unit
                ? _value.unit
                : unit // ignore: cast_nullable_to_non_nullable
                      as String,
            improving: null == improving
                ? _value.improving
                : improving // ignore: cast_nullable_to_non_nullable
                      as bool,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$HealthDriverImplCopyWith<$Res>
    implements $HealthDriverCopyWith<$Res> {
  factory _$$HealthDriverImplCopyWith(
    _$HealthDriverImpl value,
    $Res Function(_$HealthDriverImpl) then,
  ) = __$$HealthDriverImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String id,
    String name,
    String icon,
    double current,
    double target,
    String unit,
    bool improving,
  });
}

/// @nodoc
class __$$HealthDriverImplCopyWithImpl<$Res>
    extends _$HealthDriverCopyWithImpl<$Res, _$HealthDriverImpl>
    implements _$$HealthDriverImplCopyWith<$Res> {
  __$$HealthDriverImplCopyWithImpl(
    _$HealthDriverImpl _value,
    $Res Function(_$HealthDriverImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of HealthDriver
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? name = null,
    Object? icon = null,
    Object? current = null,
    Object? target = null,
    Object? unit = null,
    Object? improving = null,
  }) {
    return _then(
      _$HealthDriverImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as String,
        name: null == name
            ? _value.name
            : name // ignore: cast_nullable_to_non_nullable
                  as String,
        icon: null == icon
            ? _value.icon
            : icon // ignore: cast_nullable_to_non_nullable
                  as String,
        current: null == current
            ? _value.current
            : current // ignore: cast_nullable_to_non_nullable
                  as double,
        target: null == target
            ? _value.target
            : target // ignore: cast_nullable_to_non_nullable
                  as double,
        unit: null == unit
            ? _value.unit
            : unit // ignore: cast_nullable_to_non_nullable
                  as String,
        improving: null == improving
            ? _value.improving
            : improving // ignore: cast_nullable_to_non_nullable
                  as bool,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$HealthDriverImpl implements _HealthDriver {
  const _$HealthDriverImpl({
    required this.id,
    required this.name,
    required this.icon,
    required this.current,
    required this.target,
    required this.unit,
    this.improving = false,
  });

  factory _$HealthDriverImpl.fromJson(Map<String, dynamic> json) =>
      _$$HealthDriverImplFromJson(json);

  @override
  final String id;
  @override
  final String name;
  @override
  final String icon;
  @override
  final double current;
  @override
  final double target;
  @override
  final String unit;
  @override
  @JsonKey()
  final bool improving;

  @override
  String toString() {
    return 'HealthDriver(id: $id, name: $name, icon: $icon, current: $current, target: $target, unit: $unit, improving: $improving)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$HealthDriverImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.icon, icon) || other.icon == icon) &&
            (identical(other.current, current) || other.current == current) &&
            (identical(other.target, target) || other.target == target) &&
            (identical(other.unit, unit) || other.unit == unit) &&
            (identical(other.improving, improving) ||
                other.improving == improving));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    name,
    icon,
    current,
    target,
    unit,
    improving,
  );

  /// Create a copy of HealthDriver
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$HealthDriverImplCopyWith<_$HealthDriverImpl> get copyWith =>
      __$$HealthDriverImplCopyWithImpl<_$HealthDriverImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$HealthDriverImplToJson(this);
  }
}

abstract class _HealthDriver implements HealthDriver {
  const factory _HealthDriver({
    required final String id,
    required final String name,
    required final String icon,
    required final double current,
    required final double target,
    required final String unit,
    final bool improving,
  }) = _$HealthDriverImpl;

  factory _HealthDriver.fromJson(Map<String, dynamic> json) =
      _$HealthDriverImpl.fromJson;

  @override
  String get id;
  @override
  String get name;
  @override
  String get icon;
  @override
  double get current;
  @override
  double get target;
  @override
  String get unit;
  @override
  bool get improving;

  /// Create a copy of HealthDriver
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$HealthDriverImplCopyWith<_$HealthDriverImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
