import 'package:hive_flutter/hive_flutter.dart';

class HiveService {
  static late Box _prefsBox;

  static Future<void> init() async {
    await Hive.initFlutter();
    _prefsBox = await Hive.openBox('preferences');
  }

  static Box get preferences => _prefsBox;
}
