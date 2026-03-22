import 'package:freezed_annotation/freezed_annotation.dart';

part 'tenant_branding.freezed.dart';
part 'tenant_branding.g.dart';

@freezed
class TenantBranding with _$TenantBranding {
  const factory TenantBranding({
    required String tenantId,
    required String tenantName,
    String? logoUrl,
    required int primaryColorValue,
    required int secondaryColorValue,
  }) = _TenantBranding;

  factory TenantBranding.fromJson(Map<String, dynamic> json) =>
      _$TenantBrandingFromJson(json);
}
