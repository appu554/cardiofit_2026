import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class ApiClient {
  final Dio dio;

  ApiClient({String? baseUrl})
      : dio = Dio(BaseOptions(
          baseUrl: baseUrl ?? 'http://localhost:8000',
          connectTimeout: const Duration(seconds: 10),
          receiveTimeout: const Duration(seconds: 10),
        ));
}

final apiClientProvider = Provider<ApiClient>((ref) {
  return ApiClient();
});
