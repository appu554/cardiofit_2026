// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'clinical_translation.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

ClinicalTranslation _$ClinicalTranslationFromJson(Map<String, dynamic> json) {
  return _ClinicalTranslation.fromJson(json);
}

/// @nodoc
mixin _$ClinicalTranslation {
  String get clinicalTerm => throw _privateConstructorUsedError;
  String get patientTerm => throw _privateConstructorUsedError;
  String get explanation => throw _privateConstructorUsedError;

  /// Serializes this ClinicalTranslation to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ClinicalTranslation
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ClinicalTranslationCopyWith<ClinicalTranslation> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ClinicalTranslationCopyWith<$Res> {
  factory $ClinicalTranslationCopyWith(
    ClinicalTranslation value,
    $Res Function(ClinicalTranslation) then,
  ) = _$ClinicalTranslationCopyWithImpl<$Res, ClinicalTranslation>;
  @useResult
  $Res call({String clinicalTerm, String patientTerm, String explanation});
}

/// @nodoc
class _$ClinicalTranslationCopyWithImpl<$Res, $Val extends ClinicalTranslation>
    implements $ClinicalTranslationCopyWith<$Res> {
  _$ClinicalTranslationCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ClinicalTranslation
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? clinicalTerm = null,
    Object? patientTerm = null,
    Object? explanation = null,
  }) {
    return _then(
      _value.copyWith(
            clinicalTerm: null == clinicalTerm
                ? _value.clinicalTerm
                : clinicalTerm // ignore: cast_nullable_to_non_nullable
                      as String,
            patientTerm: null == patientTerm
                ? _value.patientTerm
                : patientTerm // ignore: cast_nullable_to_non_nullable
                      as String,
            explanation: null == explanation
                ? _value.explanation
                : explanation // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$ClinicalTranslationImplCopyWith<$Res>
    implements $ClinicalTranslationCopyWith<$Res> {
  factory _$$ClinicalTranslationImplCopyWith(
    _$ClinicalTranslationImpl value,
    $Res Function(_$ClinicalTranslationImpl) then,
  ) = __$$ClinicalTranslationImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String clinicalTerm, String patientTerm, String explanation});
}

/// @nodoc
class __$$ClinicalTranslationImplCopyWithImpl<$Res>
    extends _$ClinicalTranslationCopyWithImpl<$Res, _$ClinicalTranslationImpl>
    implements _$$ClinicalTranslationImplCopyWith<$Res> {
  __$$ClinicalTranslationImplCopyWithImpl(
    _$ClinicalTranslationImpl _value,
    $Res Function(_$ClinicalTranslationImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ClinicalTranslation
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? clinicalTerm = null,
    Object? patientTerm = null,
    Object? explanation = null,
  }) {
    return _then(
      _$ClinicalTranslationImpl(
        clinicalTerm: null == clinicalTerm
            ? _value.clinicalTerm
            : clinicalTerm // ignore: cast_nullable_to_non_nullable
                  as String,
        patientTerm: null == patientTerm
            ? _value.patientTerm
            : patientTerm // ignore: cast_nullable_to_non_nullable
                  as String,
        explanation: null == explanation
            ? _value.explanation
            : explanation // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ClinicalTranslationImpl implements _ClinicalTranslation {
  const _$ClinicalTranslationImpl({
    required this.clinicalTerm,
    required this.patientTerm,
    required this.explanation,
  });

  factory _$ClinicalTranslationImpl.fromJson(Map<String, dynamic> json) =>
      _$$ClinicalTranslationImplFromJson(json);

  @override
  final String clinicalTerm;
  @override
  final String patientTerm;
  @override
  final String explanation;

  @override
  String toString() {
    return 'ClinicalTranslation(clinicalTerm: $clinicalTerm, patientTerm: $patientTerm, explanation: $explanation)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ClinicalTranslationImpl &&
            (identical(other.clinicalTerm, clinicalTerm) ||
                other.clinicalTerm == clinicalTerm) &&
            (identical(other.patientTerm, patientTerm) ||
                other.patientTerm == patientTerm) &&
            (identical(other.explanation, explanation) ||
                other.explanation == explanation));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode =>
      Object.hash(runtimeType, clinicalTerm, patientTerm, explanation);

  /// Create a copy of ClinicalTranslation
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ClinicalTranslationImplCopyWith<_$ClinicalTranslationImpl> get copyWith =>
      __$$ClinicalTranslationImplCopyWithImpl<_$ClinicalTranslationImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ClinicalTranslationImplToJson(this);
  }
}

abstract class _ClinicalTranslation implements ClinicalTranslation {
  const factory _ClinicalTranslation({
    required final String clinicalTerm,
    required final String patientTerm,
    required final String explanation,
  }) = _$ClinicalTranslationImpl;

  factory _ClinicalTranslation.fromJson(Map<String, dynamic> json) =
      _$ClinicalTranslationImpl.fromJson;

  @override
  String get clinicalTerm;
  @override
  String get patientTerm;
  @override
  String get explanation;

  /// Create a copy of ClinicalTranslation
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ClinicalTranslationImplCopyWith<_$ClinicalTranslationImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
