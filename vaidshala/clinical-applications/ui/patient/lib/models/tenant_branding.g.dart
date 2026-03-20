// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'tenant_branding.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$TenantBrandingImpl _$$TenantBrandingImplFromJson(Map<String, dynamic> json) =>
    _$TenantBrandingImpl(
      tenantId: json['tenantId'] as String,
      tenantName: json['tenantName'] as String,
      logoUrl: json['logoUrl'] as String?,
      primaryColorValue:
          (json['primaryColorValue'] as num?)?.toInt() ?? 0xFF00897B,
      secondaryColorValue:
          (json['secondaryColorValue'] as num?)?.toInt() ?? 0xFF1B3A5C,
    );

Map<String, dynamic> _$$TenantBrandingImplToJson(
  _$TenantBrandingImpl instance,
) => <String, dynamic>{
  'tenantId': instance.tenantId,
  'tenantName': instance.tenantName,
  'logoUrl': instance.logoUrl,
  'primaryColorValue': instance.primaryColorValue,
  'secondaryColorValue': instance.secondaryColorValue,
};
