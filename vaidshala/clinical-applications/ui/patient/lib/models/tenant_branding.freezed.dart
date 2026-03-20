// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'tenant_branding.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

TenantBranding _$TenantBrandingFromJson(Map<String, dynamic> json) {
  return _TenantBranding.fromJson(json);
}

/// @nodoc
mixin _$TenantBranding {
  String get tenantId => throw _privateConstructorUsedError;
  String get tenantName => throw _privateConstructorUsedError;
  String? get logoUrl => throw _privateConstructorUsedError;
  int get primaryColorValue => throw _privateConstructorUsedError;
  int get secondaryColorValue => throw _privateConstructorUsedError;

  /// Serializes this TenantBranding to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of TenantBranding
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $TenantBrandingCopyWith<TenantBranding> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $TenantBrandingCopyWith<$Res> {
  factory $TenantBrandingCopyWith(
    TenantBranding value,
    $Res Function(TenantBranding) then,
  ) = _$TenantBrandingCopyWithImpl<$Res, TenantBranding>;
  @useResult
  $Res call({
    String tenantId,
    String tenantName,
    String? logoUrl,
    int primaryColorValue,
    int secondaryColorValue,
  });
}

/// @nodoc
class _$TenantBrandingCopyWithImpl<$Res, $Val extends TenantBranding>
    implements $TenantBrandingCopyWith<$Res> {
  _$TenantBrandingCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of TenantBranding
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? tenantId = null,
    Object? tenantName = null,
    Object? logoUrl = freezed,
    Object? primaryColorValue = null,
    Object? secondaryColorValue = null,
  }) {
    return _then(
      _value.copyWith(
            tenantId: null == tenantId
                ? _value.tenantId
                : tenantId // ignore: cast_nullable_to_non_nullable
                      as String,
            tenantName: null == tenantName
                ? _value.tenantName
                : tenantName // ignore: cast_nullable_to_non_nullable
                      as String,
            logoUrl: freezed == logoUrl
                ? _value.logoUrl
                : logoUrl // ignore: cast_nullable_to_non_nullable
                      as String?,
            primaryColorValue: null == primaryColorValue
                ? _value.primaryColorValue
                : primaryColorValue // ignore: cast_nullable_to_non_nullable
                      as int,
            secondaryColorValue: null == secondaryColorValue
                ? _value.secondaryColorValue
                : secondaryColorValue // ignore: cast_nullable_to_non_nullable
                      as int,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$TenantBrandingImplCopyWith<$Res>
    implements $TenantBrandingCopyWith<$Res> {
  factory _$$TenantBrandingImplCopyWith(
    _$TenantBrandingImpl value,
    $Res Function(_$TenantBrandingImpl) then,
  ) = __$$TenantBrandingImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String tenantId,
    String tenantName,
    String? logoUrl,
    int primaryColorValue,
    int secondaryColorValue,
  });
}

/// @nodoc
class __$$TenantBrandingImplCopyWithImpl<$Res>
    extends _$TenantBrandingCopyWithImpl<$Res, _$TenantBrandingImpl>
    implements _$$TenantBrandingImplCopyWith<$Res> {
  __$$TenantBrandingImplCopyWithImpl(
    _$TenantBrandingImpl _value,
    $Res Function(_$TenantBrandingImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of TenantBranding
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? tenantId = null,
    Object? tenantName = null,
    Object? logoUrl = freezed,
    Object? primaryColorValue = null,
    Object? secondaryColorValue = null,
  }) {
    return _then(
      _$TenantBrandingImpl(
        tenantId: null == tenantId
            ? _value.tenantId
            : tenantId // ignore: cast_nullable_to_non_nullable
                  as String,
        tenantName: null == tenantName
            ? _value.tenantName
            : tenantName // ignore: cast_nullable_to_non_nullable
                  as String,
        logoUrl: freezed == logoUrl
            ? _value.logoUrl
            : logoUrl // ignore: cast_nullable_to_non_nullable
                  as String?,
        primaryColorValue: null == primaryColorValue
            ? _value.primaryColorValue
            : primaryColorValue // ignore: cast_nullable_to_non_nullable
                  as int,
        secondaryColorValue: null == secondaryColorValue
            ? _value.secondaryColorValue
            : secondaryColorValue // ignore: cast_nullable_to_non_nullable
                  as int,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$TenantBrandingImpl implements _TenantBranding {
  const _$TenantBrandingImpl({
    required this.tenantId,
    required this.tenantName,
    this.logoUrl,
    this.primaryColorValue = 0xFF00897B,
    this.secondaryColorValue = 0xFF1B3A5C,
  });

  factory _$TenantBrandingImpl.fromJson(Map<String, dynamic> json) =>
      _$$TenantBrandingImplFromJson(json);

  @override
  final String tenantId;
  @override
  final String tenantName;
  @override
  final String? logoUrl;
  @override
  @JsonKey()
  final int primaryColorValue;
  @override
  @JsonKey()
  final int secondaryColorValue;

  @override
  String toString() {
    return 'TenantBranding(tenantId: $tenantId, tenantName: $tenantName, logoUrl: $logoUrl, primaryColorValue: $primaryColorValue, secondaryColorValue: $secondaryColorValue)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$TenantBrandingImpl &&
            (identical(other.tenantId, tenantId) ||
                other.tenantId == tenantId) &&
            (identical(other.tenantName, tenantName) ||
                other.tenantName == tenantName) &&
            (identical(other.logoUrl, logoUrl) || other.logoUrl == logoUrl) &&
            (identical(other.primaryColorValue, primaryColorValue) ||
                other.primaryColorValue == primaryColorValue) &&
            (identical(other.secondaryColorValue, secondaryColorValue) ||
                other.secondaryColorValue == secondaryColorValue));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    tenantId,
    tenantName,
    logoUrl,
    primaryColorValue,
    secondaryColorValue,
  );

  /// Create a copy of TenantBranding
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$TenantBrandingImplCopyWith<_$TenantBrandingImpl> get copyWith =>
      __$$TenantBrandingImplCopyWithImpl<_$TenantBrandingImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$TenantBrandingImplToJson(this);
  }
}

abstract class _TenantBranding implements TenantBranding {
  const factory _TenantBranding({
    required final String tenantId,
    required final String tenantName,
    final String? logoUrl,
    final int primaryColorValue,
    final int secondaryColorValue,
  }) = _$TenantBrandingImpl;

  factory _TenantBranding.fromJson(Map<String, dynamic> json) =
      _$TenantBrandingImpl.fromJson;

  @override
  String get tenantId;
  @override
  String get tenantName;
  @override
  String? get logoUrl;
  @override
  int get primaryColorValue;
  @override
  int get secondaryColorValue;

  /// Create a copy of TenantBranding
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$TenantBrandingImplCopyWith<_$TenantBrandingImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
