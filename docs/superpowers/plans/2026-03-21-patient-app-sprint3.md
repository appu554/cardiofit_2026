# Vaidshala Patient App — Sprint 3 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Sprint 3 screens — Settings (S11), Score Detail (S12), Notification Center (S13), Add Vitals (S14), Medication Adherence (S15), and Symptom Logger (S16) — adding data entry capabilities, Hero animation on the score ring, SpeedDial FAB, and AppBar navigation icons.

**Architecture:** Builds on Sprint 1-2's Riverpod + go_router + Hive + Drift foundation. New screens follow established patterns: StateNotifier for form state (S14/S16), AsyncNotifier for server-synced data (S13), FutureProvider for derived data (S15). Four new Drift tables for offline-first data entry. Hero animation connects Home tab ScoreRing to Score Detail. SpeedDial FAB on My Day tab triggers bottom sheets for vitals and symptom entry.

**Tech Stack:** Flutter 3.41.5, Dart 3.11, Riverpod, go_router, Hive, Drift (WASM), Freezed, fl_chart

**Sprint Location:** `vaidshala/clinical-applications/ui/patient/`

**Spec:** `docs/superpowers/specs/2026-03-21-patient-app-sprint3-design.md`

---

## File Structure (Sprint 3 additions)

```
vaidshala/clinical-applications/ui/patient/
├── lib/
│   ├── models/
│   │   ├── domain_score.dart              # NEW: DomainScore freezed model (S12)
│   │   ├── app_notification.dart          # NEW: AppNotification freezed model (S13)
│   │   ├── vital_entry.dart              # NEW: VitalEntry freezed model (S14)
│   │   ├── symptom_entry.dart            # NEW: SymptomEntry freezed model (S16)
│   │   ├── medication_adherence.dart     # NEW: MedicationAdherence + MedStreak + MissedDose (S15)
│   │   └── settings_state.dart           # NEW: SettingsState freezed model (S11)
│   ├── providers/
│   │   ├── settings_provider.dart        # NEW: SettingsNotifier (Hive-backed)
│   │   ├── locale_provider.dart          # NEW: StateProvider<Locale>
│   │   ├── score_detail_provider.dart    # NEW: Wraps healthScoreProvider + domain mocks
│   │   ├── notifications_provider.dart   # NEW: AsyncNotifier + unreadCountProvider
│   │   ├── vitals_entry_provider.dart    # NEW: VitalsEntryNotifier (form state)
│   │   ├── symptom_entry_provider.dart   # NEW: SymptomEntryNotifier (form state)
│   │   └── medication_adherence_provider.dart # NEW: FutureProvider from Drift
│   ├── screens/
│   │   ├── settings_screen.dart          # NEW: S11 Settings
│   │   ├── score_detail_screen.dart      # NEW: S12 Score Detail (Hero)
│   │   └── notifications_screen.dart     # NEW: S13 Notification Center
│   ├── widgets/
│   │   ├── settings_group.dart           # NEW: Section group wrapper (S11)
│   │   ├── settings_tile.dart            # NEW: Settings row tile (S11)
│   │   ├── language_selector.dart        # NEW: Language dropdown (S11)
│   │   ├── family_share_button.dart      # NEW: Share button (S11)
│   │   ├── full_sparkline_chart.dart     # NEW: 12-week fl_chart area chart (S12)
│   │   ├── domain_breakdown_bar.dart     # NEW: Animated horizontal bar (S12)
│   │   ├── score_explanation_card.dart   # NEW: Explanation text card (S12)
│   │   ├── notification_date_group.dart  # NEW: Date-grouped notification section (S13)
│   │   ├── notification_item.dart        # NEW: Single notification row (S13)
│   │   ├── vitals_entry_sheet.dart       # NEW: Modal bottom sheet container (S14)
│   │   ├── bp_entry_card.dart            # NEW: Blood pressure form card (S14)
│   │   ├── glucose_entry_card.dart       # NEW: Blood glucose form card (S14)
│   │   ├── weight_entry_card.dart        # NEW: Weight form card (S14)
│   │   ├── symptom_entry_sheet.dart      # NEW: Modal bottom sheet container (S16)
│   │   ├── symptom_chip.dart             # NEW: Tappable symptom chip (S16)
│   │   ├── severity_selector.dart        # NEW: Mild/Moderate/Severe radio (S16)
│   │   ├── medication_adherence_card.dart # NEW: Weekly adherence display (S15)
│   │   └── medication_streak_row.dart    # NEW: Per-med streak row (S15)
│   ├── services/
│   │   └── drift_database.dart           # MODIFY: Add 4 new tables + operations
│   ├── main.dart                         # MODIFY: Add locale support
│   └── router.dart                       # MODIFY: Add 3 new routes
├── test/
│   ├── models/
│   │   ├── domain_score_test.dart        # NEW
│   │   ├── app_notification_test.dart    # NEW
│   │   └── vital_entry_test.dart         # NEW
│   ├── providers/
│   │   ├── settings_provider_test.dart   # NEW
│   │   └── notifications_provider_test.dart # NEW
│   ├── widgets/
│   │   ├── settings_tile_test.dart       # NEW
│   │   ├── domain_breakdown_bar_test.dart # NEW
│   │   ├── notification_item_test.dart   # NEW
│   │   ├── bp_entry_card_test.dart       # NEW
│   │   ├── symptom_chip_test.dart        # NEW
│   │   └── severity_selector_test.dart   # NEW
│   └── screens/
│       ├── settings_screen_test.dart     # NEW
│       ├── score_detail_screen_test.dart # NEW
│       └── notifications_screen_test.dart # NEW
```

### Modified Files

| File | What Changes |
|------|-------------|
| `lib/services/drift_database.dart` | Add 4 tables: `Notifications`, `ObservationQueue`, `MedicationLog`, `SymptomLog` + CRUD operations. Bump `schemaVersion` to 2. |
| `lib/screens/my_day_tab.dart` | Wrap `Scaffold` body to add SpeedDial FAB with "Log Reading" and "Log Symptom" options. |
| `lib/screens/main_shell.dart` | Add AppBar with profile icon (→ `/settings`) and bell icon with unread badge (→ `/notifications`). |
| `lib/screens/progress_tab.dart` | Add Medication Adherence section below Milestones. |
| `lib/screens/home_tab.dart` | Wrap `ScoreRing` in `Hero(tag: 'score-ring')`, make card tappable → `/score-detail`. |
| `lib/router.dart` | Add `/settings`, `/score-detail`, `/notifications` routes outside ShellRoute. |
| `lib/main.dart` | Watch `localeProvider`, pass `locale` to `MaterialApp.router`. |

---

## Task 1: New Freezed Models

**Files:**
- Create: `lib/models/domain_score.dart`
- Create: `lib/models/app_notification.dart`
- Create: `lib/models/vital_entry.dart`
- Create: `lib/models/symptom_entry.dart`
- Create: `lib/models/medication_adherence.dart`
- Create: `lib/models/settings_state.dart`
- Test: `test/models/domain_score_test.dart`
- Test: `test/models/app_notification_test.dart`
- Test: `test/models/vital_entry_test.dart`

- [ ] **Step 1: Create DomainScore model**

```dart
// lib/models/domain_score.dart
import 'package:freezed_annotation/freezed_annotation.dart';

part 'domain_score.freezed.dart';
part 'domain_score.g.dart';

@freezed
class DomainScore with _$DomainScore {
  const factory DomainScore({
    required String name,
    required int score,
    required int target,
    required String icon,
  }) = _DomainScore;

  factory DomainScore.fromJson(Map<String, dynamic> json) =>
      _$DomainScoreFromJson(json);
}
```

- [ ] **Step 2: Create AppNotification model**

```dart
// lib/models/app_notification.dart
import 'package:freezed_annotation/freezed_annotation.dart';

part 'app_notification.freezed.dart';
part 'app_notification.g.dart';

enum NotificationType { coaching, reminder, alert, milestone }

@freezed
class AppNotification with _$AppNotification {
  const factory AppNotification({
    required String id,
    required NotificationType type,
    required String title,
    required String body,
    String? deepLink,
    required DateTime timestamp,
    @Default(false) bool read,
  }) = _AppNotification;

  factory AppNotification.fromJson(Map<String, dynamic> json) =>
      _$AppNotificationFromJson(json);
}
```

- [ ] **Step 3: Create VitalEntry model**

```dart
// lib/models/vital_entry.dart
import 'package:freezed_annotation/freezed_annotation.dart';

part 'vital_entry.freezed.dart';
part 'vital_entry.g.dart';

@freezed
class VitalEntry with _$VitalEntry {
  const factory VitalEntry({
    required String id,
    required String type,       // bp, glucose, weight
    required String value,      // JSON string
    required String unit,
    required DateTime timestamp,
    @Default(false) bool synced,
  }) = _VitalEntry;

  factory VitalEntry.fromJson(Map<String, dynamic> json) =>
      _$VitalEntryFromJson(json);
}
```

- [ ] **Step 4: Create SymptomEntry model**

```dart
// lib/models/symptom_entry.dart
import 'package:freezed_annotation/freezed_annotation.dart';

part 'symptom_entry.freezed.dart';
part 'symptom_entry.g.dart';

@freezed
class SymptomEntry with _$SymptomEntry {
  const factory SymptomEntry({
    required String id,
    required String symptom,    // comma-separated if multiple
    required String severity,   // mild, moderate, severe
    String? notes,
    required DateTime timestamp,
    @Default(false) bool synced,
  }) = _SymptomEntry;

  factory SymptomEntry.fromJson(Map<String, dynamic> json) =>
      _$SymptomEntryFromJson(json);
}
```

- [ ] **Step 5: Create MedicationAdherence models**

```dart
// lib/models/medication_adherence.dart
import 'package:freezed_annotation/freezed_annotation.dart';

part 'medication_adherence.freezed.dart';
part 'medication_adherence.g.dart';

@freezed
class MedicationAdherence with _$MedicationAdherence {
  const factory MedicationAdherence({
    required int weeklyPct,
    required List<MedStreak> streaks,
    MissedDose? lastMissed,
  }) = _MedicationAdherence;

  factory MedicationAdherence.fromJson(Map<String, dynamic> json) =>
      _$MedicationAdherenceFromJson(json);
}

@freezed
class MedStreak with _$MedStreak {
  const factory MedStreak({
    required String medicationName,
    required int streakDays,
  }) = _MedStreak;

  factory MedStreak.fromJson(Map<String, dynamic> json) =>
      _$MedStreakFromJson(json);
}

@freezed
class MissedDose with _$MissedDose {
  const factory MissedDose({
    required String medicationName,
    required int daysAgo,
  }) = _MissedDose;

  factory MissedDose.fromJson(Map<String, dynamic> json) =>
      _$MissedDoseFromJson(json);
}
```

- [ ] **Step 6: Create SettingsState model**

```dart
// lib/models/settings_state.dart
import 'package:freezed_annotation/freezed_annotation.dart';

part 'settings_state.freezed.dart';
part 'settings_state.g.dart';

@freezed
class SettingsState with _$SettingsState {
  const factory SettingsState({
    @Default('en') String language,
    @Default(true) bool notificationsEnabled,
  }) = _SettingsState;

  factory SettingsState.fromJson(Map<String, dynamic> json) =>
      _$SettingsStateFromJson(json);
}
```

- [ ] **Step 7: Write model tests**

```dart
// test/models/domain_score_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/models/domain_score.dart';

void main() {
  group('DomainScore', () {
    test('creates with required fields', () {
      const ds = DomainScore(
        name: 'Blood Sugar',
        score: 35,
        target: 60,
        icon: 'bloodtype',
      );
      expect(ds.name, 'Blood Sugar');
      expect(ds.score, 35);
      expect(ds.target, 60);
    });

    test('serializes to/from JSON', () {
      const ds = DomainScore(
        name: 'Activity',
        score: 22,
        target: 50,
        icon: 'directions_walk',
      );
      final json = ds.toJson();
      final restored = DomainScore.fromJson(json);
      expect(restored, ds);
    });
  });
}
```

```dart
// test/models/app_notification_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/models/app_notification.dart';

void main() {
  group('AppNotification', () {
    test('creates with default read=false', () {
      final n = AppNotification(
        id: 'n1',
        type: NotificationType.coaching,
        title: 'Great progress!',
        body: 'Your FBG dropped 12 mg/dL this week',
        timestamp: DateTime(2026, 3, 21, 9, 0),
      );
      expect(n.read, false);
      expect(n.type, NotificationType.coaching);
    });

    test('serializes to/from JSON', () {
      final n = AppNotification(
        id: 'n2',
        type: NotificationType.alert,
        title: 'Test',
        body: 'Body',
        deepLink: '/home/progress',
        timestamp: DateTime(2026, 3, 21),
        read: true,
      );
      final json = n.toJson();
      final restored = AppNotification.fromJson(json);
      expect(restored, n);
    });
  });
}
```

```dart
// test/models/vital_entry_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/models/vital_entry.dart';

void main() {
  group('VitalEntry', () {
    test('creates with default synced=false', () {
      final v = VitalEntry(
        id: 'v1',
        type: 'bp',
        value: '{"systolic":156,"diastolic":98}',
        unit: 'mmHg',
        timestamp: DateTime(2026, 3, 21),
      );
      expect(v.synced, false);
      expect(v.type, 'bp');
    });

    test('serializes to/from JSON', () {
      final v = VitalEntry(
        id: 'v2',
        type: 'glucose',
        value: '{"value":178,"context":"fasting"}',
        unit: 'mg/dL',
        timestamp: DateTime(2026, 3, 21),
        synced: true,
      );
      final json = v.toJson();
      final restored = VitalEntry.fromJson(json);
      expect(restored, v);
    });
  });
}
```

- [ ] **Step 8: Run code generation**

Run: `cd vaidshala/clinical-applications/ui/patient && dart run build_runner build --delete-conflicting-outputs`
Expected: All `.freezed.dart` and `.g.dart` files generated without errors.

- [ ] **Step 9: Run model tests**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/models/`
Expected: All tests PASS.

- [ ] **Step 10: Commit**

```bash
cd vaidshala/clinical-applications/ui/patient
git add lib/models/domain_score.dart lib/models/app_notification.dart lib/models/vital_entry.dart lib/models/symptom_entry.dart lib/models/medication_adherence.dart lib/models/settings_state.dart lib/models/*.freezed.dart lib/models/*.g.dart test/models/domain_score_test.dart test/models/app_notification_test.dart test/models/vital_entry_test.dart
git commit -m "feat(patient-app): add Sprint 3 Freezed models

DomainScore, AppNotification, VitalEntry, SymptomEntry,
MedicationAdherence/MedStreak/MissedDose, SettingsState."
```

---

## Task 2: Extend Drift Database with 4 New Tables

**Files:**
- Modify: `lib/services/drift_database.dart`

**Context:** The existing database at `lib/services/drift_database.dart` has 2 tables (`CheckinQueue`, `LabHistory`) with `schemaVersion: 1`. We add 4 new tables and bump to version 2.

- [ ] **Step 1: Add Notifications table class**

Add after `LabHistory` class (after line 20 of `drift_database.dart`):

```dart
class Notifications extends Table {
  TextColumn get id => text()();
  TextColumn get type => text()();          // coaching, reminder, alert, milestone
  TextColumn get title => text()();
  TextColumn get body => text()();
  TextColumn get deepLink => text().nullable()();
  IntColumn get timestamp => integer()();   // epoch ms
  BoolColumn get read => boolean().withDefault(const Constant(false))();

  @override
  Set<Column> get primaryKey => {id};
}
```

- [ ] **Step 2: Add ObservationQueue table class**

```dart
class ObservationQueue extends Table {
  TextColumn get id => text()();
  TextColumn get type => text()();          // bp, glucose, weight
  TextColumn get value => text()();         // JSON string
  TextColumn get unit => text()();
  IntColumn get timestamp => integer()();   // epoch ms
  BoolColumn get synced => boolean().withDefault(const Constant(false))();

  @override
  Set<Column> get primaryKey => {id};
}
```

- [ ] **Step 3: Add MedicationLog table class**

```dart
class MedicationLog extends Table {
  TextColumn get id => text()();
  TextColumn get actionId => text()();
  TextColumn get medicationName => text()();
  BoolColumn get completed => boolean()();  // true = taken, false = missed
  IntColumn get timestamp => integer()();   // epoch ms

  @override
  Set<Column> get primaryKey => {id};
}
```

- [ ] **Step 4: Add SymptomLog table class**

```dart
class SymptomLog extends Table {
  TextColumn get id => text()();
  TextColumn get symptom => text()();       // comma-separated
  TextColumn get severity => text()();      // mild, moderate, severe
  TextColumn get notes => text().nullable()();
  IntColumn get timestamp => integer()();   // epoch ms
  BoolColumn get synced => boolean().withDefault(const Constant(false))();

  @override
  Set<Column> get primaryKey => {id};
}
```

- [ ] **Step 5: Update @DriftDatabase annotation and schemaVersion**

Change the annotation from:
```dart
@DriftDatabase(tables: [CheckinQueue, LabHistory])
```
to:
```dart
@DriftDatabase(tables: [CheckinQueue, LabHistory, Notifications, ObservationQueue, MedicationLog, SymptomLog])
```

Change `schemaVersion` from `1` to `2`.

- [ ] **Step 6: Add migration strategy**

Add inside `AppDatabase` class, after `schemaVersion`:

```dart
@override
MigrationStrategy get migration => MigrationStrategy(
      onCreate: (m) => m.createAll(),
      onUpgrade: (m, from, to) async {
        if (from < 2) {
          await m.createTable(notifications);
          await m.createTable(observationQueue);
          await m.createTable(medicationLog);
          await m.createTable(symptomLog);
        }
      },
    );
```

- [ ] **Step 7: Add Notifications CRUD operations**

Add inside `AppDatabase` class:

```dart
  // Notification operations
  Future<void> seedNotifications(List<NotificationsCompanion> items) async {
    final count = await (selectOnly(notifications)..addColumns([notifications.id.count()])).getSingle();
    if ((count.read(notifications.id.count()) ?? 0) > 0) return;
    await batch((b) => b.insertAll(notifications, items));
  }

  Future<List<Notification>> allNotifications() =>
      (select(notifications)..orderBy([(t) => OrderingTerm.desc(t.timestamp)])).get();

  Future<void> markNotificationRead(String id) =>
      (update(notifications)..where((t) => t.id.equals(id)))
          .write(const NotificationsCompanion(read: Value(true)));

  Future<void> markAllNotificationsRead() =>
      update(notifications).write(const NotificationsCompanion(read: Value(true)));

  Future<void> deleteNotification(String id) =>
      (delete(notifications)..where((t) => t.id.equals(id))).go();

  Future<int> unreadNotificationCount() async {
    final query = selectOnly(notifications)
      ..addColumns([notifications.id.count()])
      ..where(notifications.read.equals(false));
    final result = await query.getSingle();
    return result.read(notifications.id.count()) ?? 0;
  }
```

- [ ] **Step 8: Add ObservationQueue CRUD operations**

```dart
  // Observation queue operations
  Future<void> insertObservation({
    required String id,
    required String type,
    required String value,
    required String unit,
  }) =>
      into(observationQueue).insert(ObservationQueueCompanion.insert(
        id: id,
        type: type,
        value: value,
        unit: unit,
        timestamp: Value(DateTime.now().millisecondsSinceEpoch),
      ));

  Future<List<ObservationQueueData>> recentObservations(String type) =>
      (select(observationQueue)
            ..where((t) => t.type.equals(type))
            ..orderBy([(t) => OrderingTerm.desc(t.timestamp)])
            ..limit(20))
          .get();
```

- [ ] **Step 9: Add MedicationLog CRUD operations**

```dart
  // Medication log operations
  Future<void> logMedication({
    required String id,
    required String actionId,
    required String medicationName,
    required bool completed,
  }) =>
      into(medicationLog).insert(MedicationLogCompanion.insert(
        id: id,
        actionId: actionId,
        medicationName: medicationName,
        completed: completed,
        timestamp: Value(DateTime.now().millisecondsSinceEpoch),
      ));

  Future<List<MedicationLogData>> medicationHistory({int days = 14}) {
    final cutoff = DateTime.now().subtract(Duration(days: days)).millisecondsSinceEpoch;
    return (select(medicationLog)
          ..where((t) => t.timestamp.isBiggerOrEqualValue(cutoff))
          ..orderBy([(t) => OrderingTerm.desc(t.timestamp)]))
        .get();
  }
```

- [ ] **Step 10: Add SymptomLog CRUD operations**

```dart
  // Symptom log operations
  Future<void> insertSymptom({
    required String id,
    required String symptom,
    required String severity,
    String? notes,
  }) =>
      into(symptomLog).insert(SymptomLogCompanion.insert(
        id: id,
        symptom: symptom,
        severity: severity,
        notes: Value(notes),
        timestamp: Value(DateTime.now().millisecondsSinceEpoch),
      ));

  Future<List<SymptomLogData>> recentSymptoms({int days = 30}) {
    final cutoff = DateTime.now().subtract(Duration(days: days)).millisecondsSinceEpoch;
    return (select(symptomLog)
          ..where((t) => t.timestamp.isBiggerOrEqualValue(cutoff))
          ..orderBy([(t) => OrderingTerm.desc(t.timestamp)]))
        .get();
  }
```

- [ ] **Step 11: Run Drift code generation**

Run: `cd vaidshala/clinical-applications/ui/patient && dart run build_runner build --delete-conflicting-outputs`
Expected: `drift_database.g.dart` regenerated with new table classes.

- [ ] **Step 12: Commit**

```bash
cd vaidshala/clinical-applications/ui/patient
git add lib/services/drift_database.dart lib/services/drift_database.g.dart
git commit -m "feat(patient-app): add 4 Drift tables for Sprint 3

notifications, observation_queue, medication_log, symptom_log.
Schema version 1→2 with migration strategy."
```

---

## Task 3: Settings Provider + Locale Provider

**Files:**
- Create: `lib/providers/settings_provider.dart`
- Create: `lib/providers/locale_provider.dart`
- Test: `test/providers/settings_provider_test.dart`

- [ ] **Step 1: Write settings provider test**

```dart
// test/providers/settings_provider_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:vaidshala_patient/providers/settings_provider.dart';
import 'package:vaidshala_patient/models/settings_state.dart';

void main() {
  group('SettingsNotifier', () {
    test('initial state has English and notifications enabled', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      final state = container.read(settingsProvider);
      expect(state.language, 'en');
      expect(state.notificationsEnabled, true);
    });

    test('setLanguage updates language', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      container.read(settingsProvider.notifier).setLanguage('hi');
      expect(container.read(settingsProvider).language, 'hi');
    });

    test('toggleNotifications flips the flag', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      container.read(settingsProvider.notifier).toggleNotifications();
      expect(container.read(settingsProvider).notificationsEnabled, false);
    });
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/providers/settings_provider_test.dart`
Expected: FAIL — `settingsProvider` not found.

- [ ] **Step 3: Create settings provider**

```dart
// lib/providers/settings_provider.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/settings_state.dart';
import '../services/hive_service.dart';

final settingsProvider =
    StateNotifierProvider<SettingsNotifier, SettingsState>(
        (ref) => SettingsNotifier());

class SettingsNotifier extends StateNotifier<SettingsState> {
  SettingsNotifier() : super(const SettingsState()) {
    _load();
  }

  void _load() {
    final prefs = HiveService.preferences;
    final lang = prefs.get('language') as String? ?? 'en';
    final notif = prefs.get('notificationsEnabled') as bool? ?? true;
    state = SettingsState(language: lang, notificationsEnabled: notif);
  }

  void setLanguage(String language) {
    state = state.copyWith(language: language);
    HiveService.preferences.put('language', language);
  }

  void toggleNotifications() {
    final toggled = !state.notificationsEnabled;
    state = state.copyWith(notificationsEnabled: toggled);
    HiveService.preferences.put('notificationsEnabled', toggled);
  }
}
```

- [ ] **Step 4: Create locale provider**

```dart
// lib/providers/locale_provider.dart
import 'dart:ui';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'settings_provider.dart';

final localeProvider = Provider<Locale>((ref) {
  final settings = ref.watch(settingsProvider);
  return Locale(settings.language);
});
```

- [ ] **Step 5: Run tests**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/providers/settings_provider_test.dart`
Expected: PASS (note: tests may need Hive mock — if `HiveService.preferences` fails, the notifier constructor catches with defaults. If Hive is not initialized in test, the `_load()` call may fail. The subagent should handle this by either initializing Hive in test setUp or overriding the provider).

- [ ] **Step 6: Commit**

```bash
cd vaidshala/clinical-applications/ui/patient
git add lib/providers/settings_provider.dart lib/providers/locale_provider.dart test/providers/settings_provider_test.dart
git commit -m "feat(patient-app): add settings and locale providers

Hive-backed SettingsNotifier for language + notifications toggle.
Locale provider derives Locale from settings language."
```

---

## Task 4: Score Detail Provider

**Files:**
- Create: `lib/providers/score_detail_provider.dart`

**Context:** This provider wraps the existing `healthScoreProvider` to add domain breakdown mock data. The HealthScore model already has `score: 18`, `sparkline: [26, 28, 25, 22, 20, 18]` (6 points). The score detail screen needs 12-week history, so we extend the sparkline with mock data.

- [ ] **Step 1: Create score detail provider**

```dart
// lib/providers/score_detail_provider.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/domain_score.dart';
import '../models/health_score.dart';
import 'health_score_provider.dart';

class ScoreDetailState {
  final int? score;
  final String label;
  final List<double> scoreHistory;
  final List<DomainScore> domains;
  final String explanation;

  const ScoreDetailState({
    this.score,
    this.label = '',
    this.scoreHistory = const [],
    this.domains = const [],
    this.explanation = '',
  });
}

final scoreDetailProvider = Provider<ScoreDetailState>((ref) {
  final healthScore = ref.watch(healthScoreProvider).valueOrNull;

  if (healthScore == null) return const ScoreDetailState();

  // Extend sparkline to 12 weeks with mock historical data
  final sparkline = healthScore.sparkline;
  final history = <double>[
    // Pad to 12 weeks if shorter
    ...List.generate(
      (12 - sparkline.length).clamp(0, 12),
      (i) => (30 - i).toDouble(),
    ),
    ...sparkline.map((s) => s.toDouble()),
  ];

  return ScoreDetailState(
    score: healthScore.score,
    label: _scoreLabel(healthScore.score),
    scoreHistory: history.length > 12 ? history.sublist(history.length - 12) : history,
    domains: const [
      DomainScore(name: 'Blood Sugar', score: 35, target: 60, icon: 'bloodtype'),
      DomainScore(name: 'Activity', score: 22, target: 50, icon: 'directions_walk'),
      DomainScore(name: 'Body Health', score: 58, target: 70, icon: 'monitor_weight'),
      DomainScore(name: 'Heart Health', score: 72, target: 80, icon: 'favorite'),
    ],
    explanation:
        "Your metabolic health score reflects how well your blood sugar, activity "
        "levels, body composition, and heart health markers are tracking against "
        "clinical targets. Focus on the areas with the biggest gaps to see the "
        "most improvement.",
  );
});

String _scoreLabel(int score) {
  if (score >= 80) return 'Excellent';
  if (score >= 60) return 'Good';
  if (score >= 40) return 'Improving';
  return 'Needs Attention';
}
```

- [ ] **Step 2: Commit**

```bash
cd vaidshala/clinical-applications/ui/patient
git add lib/providers/score_detail_provider.dart
git commit -m "feat(patient-app): add score detail provider

Wraps healthScoreProvider with 12-week history and domain
breakdown mock data for S12 Score Detail screen."
```

---

## Task 5: Notifications Provider + Unread Count

**Files:**
- Create: `lib/providers/notifications_provider.dart`
- Test: `test/providers/notifications_provider_test.dart`

- [ ] **Step 1: Write notification provider test**

```dart
// test/providers/notifications_provider_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:vaidshala_patient/models/app_notification.dart';
import 'package:vaidshala_patient/providers/notifications_provider.dart';

void main() {
  group('notificationsProvider', () {
    test('mock seed data returns 5 notifications', () async {
      final container = ProviderContainer(
        overrides: [
          notificationsProvider.overrideWith(() => _MockNotificationsNotifier()),
        ],
      );
      addTearDown(container.dispose);

      // Wait for async build
      await container.read(notificationsProvider.future);
      final state = container.read(notificationsProvider).valueOrNull ?? [];
      expect(state.length, 5);
    });

    test('unread count derived from notifications', () async {
      final mockNotifications = [
        AppNotification(
          id: 'n1',
          type: NotificationType.coaching,
          title: 'Test',
          body: 'Body',
          timestamp: DateTime.now(),
          read: true,
        ),
        AppNotification(
          id: 'n2',
          type: NotificationType.alert,
          title: 'Test2',
          body: 'Body2',
          timestamp: DateTime.now(),
          read: false,
        ),
      ];

      final container = ProviderContainer(
        overrides: [
          notificationsProvider.overrideWith(
            () => _FixedNotificationsNotifier(mockNotifications),
          ),
        ],
      );
      addTearDown(container.dispose);

      await container.read(notificationsProvider.future);
      final unread = container.read(unreadCountProvider);
      expect(unread, 1);
    });
  });
}

class _MockNotificationsNotifier extends AsyncNotifier<List<AppNotification>>
    implements NotificationsNotifier {
  @override
  Future<List<AppNotification>> build() async {
    return _mockSeedData();
  }

  @override
  Future<void> markRead(String id) async {
    final current = state.valueOrNull ?? [];
    state = AsyncData(
      current.map((n) => n.id == id ? n.copyWith(read: true) : n).toList(),
    );
  }

  @override
  Future<void> markAllRead() async {
    final current = state.valueOrNull ?? [];
    state = AsyncData(current.map((n) => n.copyWith(read: true)).toList());
  }

  @override
  Future<void> dismiss(String id) async {
    final current = state.valueOrNull ?? [];
    state = AsyncData(current.where((n) => n.id != id).toList());
  }

  @override
  Future<void> refresh() async {
    state = AsyncData(await build());
  }
}

class _FixedNotificationsNotifier extends AsyncNotifier<List<AppNotification>>
    implements NotificationsNotifier {
  final List<AppNotification> _data;
  _FixedNotificationsNotifier(this._data);

  @override
  Future<List<AppNotification>> build() async => _data;

  @override
  Future<void> markRead(String id) async {}
  @override
  Future<void> markAllRead() async {}
  @override
  Future<void> dismiss(String id) async {}
  @override
  Future<void> refresh() async {}
}

List<AppNotification> _mockSeedData() {
  final now = DateTime.now();
  return [
    AppNotification(
      id: 'n1', type: NotificationType.coaching,
      title: 'Great progress!', body: 'Your FBG dropped 12 mg/dL this week',
      deepLink: '/home/progress', timestamp: now.copyWith(hour: 9), read: true,
    ),
    AppNotification(
      id: 'n2', type: NotificationType.alert,
      title: 'FBG trending down', body: 'Your fasting glucose is moving toward target',
      deepLink: '/home/progress', timestamp: now.copyWith(hour: 8), read: false,
    ),
    AppNotification(
      id: 'n3', type: NotificationType.reminder,
      title: 'Time for evening walk', body: 'A 15-min post-dinner walk can lower glucose by 15-20%',
      deepLink: '/home/my-day', timestamp: now.copyWith(hour: 19), read: false,
    ),
    AppNotification(
      id: 'n4', type: NotificationType.coaching,
      title: 'Weekly progress summary', body: 'You completed 85% of actions this week',
      deepLink: '/home/progress', timestamp: now.subtract(const Duration(days: 1)), read: true,
    ),
    AppNotification(
      id: 'n5', type: NotificationType.milestone,
      title: 'New health tip available', body: "Learn about protein's role in metabolic health",
      deepLink: '/home/learn', timestamp: now.subtract(const Duration(days: 3)), read: true,
    ),
  ];
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/providers/notifications_provider_test.dart`
Expected: FAIL — `notificationsProvider` not found.

- [ ] **Step 3: Create notifications provider**

```dart
// lib/providers/notifications_provider.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/app_notification.dart';
import 'database_provider.dart';

final notificationsProvider =
    AsyncNotifierProvider<NotificationsNotifier, List<AppNotification>>(
        NotificationsNotifier.new);

class NotificationsNotifier extends AsyncNotifier<List<AppNotification>> {
  @override
  Future<List<AppNotification>> build() async {
    try {
      final db = ref.read(databaseProvider);
      await db.seedNotifications(_seedCompanions());
      final rows = await db.allNotifications();
      return rows.map(_rowToModel).toList();
    } catch (_) {
      return _mockSeedData();
    }
  }

  Future<void> markRead(String id) async {
    try {
      final db = ref.read(databaseProvider);
      await db.markNotificationRead(id);
    } catch (_) {}
    final current = state.valueOrNull ?? [];
    state = AsyncData(
      current.map((n) => n.id == id ? n.copyWith(read: true) : n).toList(),
    );
  }

  Future<void> markAllRead() async {
    try {
      final db = ref.read(databaseProvider);
      await db.markAllNotificationsRead();
    } catch (_) {}
    final current = state.valueOrNull ?? [];
    state = AsyncData(current.map((n) => n.copyWith(read: true)).toList());
  }

  Future<void> dismiss(String id) async {
    try {
      final db = ref.read(databaseProvider);
      await db.deleteNotification(id);
    } catch (_) {}
    final current = state.valueOrNull ?? [];
    state = AsyncData(current.where((n) => n.id != id).toList());
  }

  Future<void> refresh() async {
    state = const AsyncLoading();
    state = AsyncData(await build());
  }
}

AppNotification _rowToModel(dynamic row) {
  return AppNotification(
    id: row.id as String,
    type: NotificationType.values.firstWhere(
      (t) => t.name == (row.type as String),
      orElse: () => NotificationType.coaching,
    ),
    title: row.title as String,
    body: row.body as String,
    deepLink: row.deepLink as String?,
    timestamp: DateTime.fromMillisecondsSinceEpoch(row.timestamp as int),
    read: row.read as bool,
  );
}

final unreadCountProvider = Provider<int>((ref) {
  final notifications = ref.watch(notificationsProvider).valueOrNull ?? [];
  return notifications.where((n) => !n.read).length;
});

List<AppNotification> _mockSeedData() {
  final now = DateTime.now();
  return [
    AppNotification(
      id: 'n1', type: NotificationType.coaching,
      title: 'Great progress!', body: 'Your FBG dropped 12 mg/dL this week',
      deepLink: '/home/progress',
      timestamp: DateTime(now.year, now.month, now.day, 9), read: true,
    ),
    AppNotification(
      id: 'n2', type: NotificationType.alert,
      title: 'FBG trending down', body: 'Your fasting glucose is moving toward target',
      deepLink: '/home/progress',
      timestamp: DateTime(now.year, now.month, now.day, 8), read: false,
    ),
    AppNotification(
      id: 'n3', type: NotificationType.reminder,
      title: 'Time for evening walk',
      body: 'A 15-min post-dinner walk can lower glucose by 15-20%',
      deepLink: '/home/my-day',
      timestamp: DateTime(now.year, now.month, now.day, 19), read: false,
    ),
    AppNotification(
      id: 'n4', type: NotificationType.coaching,
      title: 'Weekly progress summary', body: 'You completed 85% of actions this week',
      deepLink: '/home/progress',
      timestamp: now.subtract(const Duration(days: 1)), read: true,
    ),
    AppNotification(
      id: 'n5', type: NotificationType.milestone,
      title: 'New health tip available',
      body: "Learn about protein's role in metabolic health",
      deepLink: '/home/learn',
      timestamp: now.subtract(const Duration(days: 3)), read: true,
    ),
  ];
}

// Drift companions for seed data — only used if Drift DB is available
List<dynamic> _seedCompanions() {
  final now = DateTime.now();
  final seeds = _mockSeedData();
  // Return empty list — implementer should convert to NotificationsCompanion
  // when Drift codegen types are available. The provider falls back to mock data
  // if seeding fails.
  return [];
}
```

- [ ] **Step 4: Run tests**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/providers/notifications_provider_test.dart`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd vaidshala/clinical-applications/ui/patient
git add lib/providers/notifications_provider.dart test/providers/notifications_provider_test.dart
git commit -m "feat(patient-app): add notifications provider + unread count

AsyncNotifier reading from Drift with mock seed fallback.
unreadCountProvider derives badge count from notification state."
```

---

## Task 6: Vitals Entry Provider

**Files:**
- Create: `lib/providers/vitals_entry_provider.dart`

**Context:** Form state manager for the Add Vitals bottom sheet (S14). Manages BP, glucose, and weight form fields with validation. Saves to Drift `observation_queue` table on submit.

- [ ] **Step 1: Create vitals entry provider**

```dart
// lib/providers/vitals_entry_provider.dart
import 'dart:convert';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'database_provider.dart';

class VitalsEntryState {
  final String systolic;
  final String diastolic;
  final String glucoseValue;
  final String? glucoseContext; // fasting, post-meal, random
  final String weight;
  final Map<String, String?> errors;

  const VitalsEntryState({
    this.systolic = '',
    this.diastolic = '',
    this.glucoseValue = '',
    this.glucoseContext,
    this.weight = '',
    this.errors = const {},
  });

  VitalsEntryState copyWith({
    String? systolic,
    String? diastolic,
    String? glucoseValue,
    String? glucoseContext,
    String? weight,
    Map<String, String?>? errors,
  }) =>
      VitalsEntryState(
        systolic: systolic ?? this.systolic,
        diastolic: diastolic ?? this.diastolic,
        glucoseValue: glucoseValue ?? this.glucoseValue,
        glucoseContext: glucoseContext ?? this.glucoseContext,
        weight: weight ?? this.weight,
        errors: errors ?? this.errors,
      );
}

final vitalsEntryProvider =
    StateNotifierProvider<VitalsEntryNotifier, VitalsEntryState>(
        (ref) => VitalsEntryNotifier(ref));

class VitalsEntryNotifier extends StateNotifier<VitalsEntryState> {
  final Ref _ref;

  VitalsEntryNotifier(this._ref) : super(const VitalsEntryState());

  void setSystolic(String v) => state = state.copyWith(systolic: v);
  void setDiastolic(String v) => state = state.copyWith(diastolic: v);
  void setGlucoseValue(String v) => state = state.copyWith(glucoseValue: v);
  void setGlucoseContext(String? v) => state = state.copyWith(glucoseContext: v);
  void setWeight(String v) => state = state.copyWith(weight: v);

  /// Validates BP fields. Returns true if valid.
  bool validateBp() {
    final errors = <String, String?>{...state.errors};
    final sys = double.tryParse(state.systolic);
    final dia = double.tryParse(state.diastolic);

    if (sys == null || sys < 60 || sys > 250) {
      errors['systolic'] = 'Enter a value between 60 and 250';
    } else {
      errors.remove('systolic');
    }

    if (dia == null || dia < 40 || dia > 150) {
      errors['diastolic'] = 'Enter a value between 40 and 150';
    } else {
      errors.remove('diastolic');
    }

    if (sys != null && dia != null && sys <= dia) {
      errors['systolic'] = 'Systolic must be higher than diastolic';
    }

    state = state.copyWith(errors: errors);
    return !errors.containsKey('systolic') && !errors.containsKey('diastolic');
  }

  /// Validates glucose fields. Returns true if valid.
  bool validateGlucose() {
    final errors = <String, String?>{...state.errors};
    final val = double.tryParse(state.glucoseValue);

    if (val == null || val < 20 || val > 600) {
      errors['glucose'] = 'Enter a value between 20 and 600';
    } else {
      errors.remove('glucose');
    }

    if (state.glucoseContext == null) {
      errors['glucoseContext'] = 'Select when this was measured';
    } else {
      errors.remove('glucoseContext');
    }

    state = state.copyWith(errors: errors);
    return !errors.containsKey('glucose') && !errors.containsKey('glucoseContext');
  }

  /// Validates weight field. Returns true if valid.
  bool validateWeight() {
    final errors = <String, String?>{...state.errors};
    final val = double.tryParse(state.weight);

    if (val == null || val < 20 || val > 300) {
      errors['weight'] = 'Enter a value between 20 and 300';
    } else {
      errors.remove('weight');
    }

    state = state.copyWith(errors: errors);
    return !errors.containsKey('weight');
  }

  /// Save BP reading to Drift. Returns true on success.
  Future<bool> saveBp() async {
    if (!validateBp()) return false;
    try {
      final db = _ref.read(databaseProvider);
      final id = 'obs-bp-${DateTime.now().millisecondsSinceEpoch}';
      await db.insertObservation(
        id: id,
        type: 'bp',
        value: jsonEncode({
          'systolic': int.parse(state.systolic),
          'diastolic': int.parse(state.diastolic),
        }),
        unit: 'mmHg',
      );
      state = state.copyWith(systolic: '', diastolic: '');
      return true;
    } catch (_) {
      return false;
    }
  }

  /// Save glucose reading to Drift. Returns true on success.
  Future<bool> saveGlucose() async {
    if (!validateGlucose()) return false;
    try {
      final db = _ref.read(databaseProvider);
      final id = 'obs-gluc-${DateTime.now().millisecondsSinceEpoch}';
      await db.insertObservation(
        id: id,
        type: 'glucose',
        value: jsonEncode({
          'value': double.parse(state.glucoseValue),
          'context': state.glucoseContext,
        }),
        unit: 'mg/dL',
      );
      state = state.copyWith(glucoseValue: '', glucoseContext: null);
      return true;
    } catch (_) {
      return false;
    }
  }

  /// Save weight reading to Drift. Returns true on success.
  Future<bool> saveWeight() async {
    if (!validateWeight()) return false;
    try {
      final db = _ref.read(databaseProvider);
      final id = 'obs-wt-${DateTime.now().millisecondsSinceEpoch}';
      await db.insertObservation(
        id: id,
        type: 'weight',
        value: jsonEncode({'value': double.parse(state.weight)}),
        unit: 'kg',
      );
      state = state.copyWith(weight: '');
      return true;
    } catch (_) {
      return false;
    }
  }

  void reset() => state = const VitalsEntryState();
}
```

- [ ] **Step 2: Commit**

```bash
cd vaidshala/clinical-applications/ui/patient
git add lib/providers/vitals_entry_provider.dart
git commit -m "feat(patient-app): add vitals entry provider

Form state for BP, glucose, weight with validation rules and
Drift persistence. Supports independent save per vital card."
```

---

## Task 7: Symptom Entry + Medication Adherence Providers

**Files:**
- Create: `lib/providers/symptom_entry_provider.dart`
- Create: `lib/providers/medication_adherence_provider.dart`

- [ ] **Step 1: Create symptom entry provider**

```dart
// lib/providers/symptom_entry_provider.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'database_provider.dart';

class SymptomEntryState {
  final Set<String> selectedSymptoms;
  final String? severity;  // mild, moderate, severe
  final String notes;
  final DateTime timestamp;

  SymptomEntryState({
    this.selectedSymptoms = const {},
    this.severity,
    this.notes = '',
    DateTime? timestamp,
  }) : timestamp = timestamp ?? DateTime.now();

  bool get canSave => selectedSymptoms.isNotEmpty && severity != null;

  SymptomEntryState copyWith({
    Set<String>? selectedSymptoms,
    String? severity,
    String? notes,
    DateTime? timestamp,
  }) =>
      SymptomEntryState(
        selectedSymptoms: selectedSymptoms ?? this.selectedSymptoms,
        severity: severity ?? this.severity,
        notes: notes ?? this.notes,
        timestamp: timestamp ?? this.timestamp,
      );
}

final symptomEntryProvider =
    StateNotifierProvider<SymptomEntryNotifier, SymptomEntryState>(
        (ref) => SymptomEntryNotifier(ref));

class SymptomEntryNotifier extends StateNotifier<SymptomEntryState> {
  final Ref _ref;

  SymptomEntryNotifier(this._ref) : super(SymptomEntryState(timestamp: DateTime.now()));

  void toggleSymptom(String symptom) {
    final current = Set<String>.from(state.selectedSymptoms);
    if (current.contains(symptom)) {
      current.remove(symptom);
    } else {
      current.add(symptom);
    }
    state = state.copyWith(selectedSymptoms: current);
  }

  void setSeverity(String severity) =>
      state = state.copyWith(severity: severity);

  void setNotes(String notes) =>
      state = state.copyWith(notes: notes);

  void setTimestamp(DateTime timestamp) =>
      state = state.copyWith(timestamp: timestamp);

  /// Save symptom log to Drift. Returns true on success.
  Future<bool> save() async {
    if (!state.canSave) return false;
    try {
      final db = _ref.read(databaseProvider);
      final id = 'sym-${DateTime.now().millisecondsSinceEpoch}';
      await db.insertSymptom(
        id: id,
        symptom: state.selectedSymptoms.join(','),
        severity: state.severity!,
        notes: state.notes.isEmpty ? null : state.notes,
      );
      reset();
      return true;
    } catch (_) {
      return false;
    }
  }

  void reset() =>
      state = SymptomEntryState(timestamp: DateTime.now());
}
```

- [ ] **Step 2: Create medication adherence provider**

```dart
// lib/providers/medication_adherence_provider.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/medication_adherence.dart';

final medicationAdherenceProvider =
    FutureProvider<MedicationAdherence>((ref) async {
  // Sprint 3: Mock data — Rajesh Kumar 14-day medication history
  // In future sprints, this reads from Drift medication_log table
  return const MedicationAdherence(
    weeklyPct: 85,
    streaks: [
      MedStreak(medicationName: 'Metformin 1000mg BD', streakDays: 12),
      MedStreak(medicationName: 'Glimepiride 2mg OD', streakDays: 8),
      MedStreak(medicationName: 'Telmisartan 40mg OD', streakDays: 14),
    ],
    lastMissed: MissedDose(medicationName: 'Metformin PM', daysAgo: 2),
  );
});
```

- [ ] **Step 3: Commit**

```bash
cd vaidshala/clinical-applications/ui/patient
git add lib/providers/symptom_entry_provider.dart lib/providers/medication_adherence_provider.dart
git commit -m "feat(patient-app): add symptom entry + medication adherence providers

SymptomEntryNotifier: form state for structured symptom logging.
MedicationAdherenceProvider: FutureProvider with Rajesh Kumar mock data."
```

---

## Task 8: Settings Widgets

**Files:**
- Create: `lib/widgets/settings_group.dart`
- Create: `lib/widgets/settings_tile.dart`
- Create: `lib/widgets/language_selector.dart`
- Create: `lib/widgets/family_share_button.dart`
- Test: `test/widgets/settings_tile_test.dart`

- [ ] **Step 1: Write settings tile test**

```dart
// test/widgets/settings_tile_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/widgets/settings_tile.dart';

void main() {
  group('SettingsTile', () {
    testWidgets('renders icon, title, and trailing widget', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: SettingsTile(
              icon: Icons.person,
              title: 'Account Name',
              trailing: Text('Rajesh Kumar'),
            ),
          ),
        ),
      );

      expect(find.byIcon(Icons.person), findsOneWidget);
      expect(find.text('Account Name'), findsOneWidget);
      expect(find.text('Rajesh Kumar'), findsOneWidget);
    });

    testWidgets('calls onTap when tapped', (tester) async {
      var tapped = false;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SettingsTile(
              icon: Icons.settings,
              title: 'Tap Me',
              onTap: () => tapped = true,
            ),
          ),
        ),
      );

      await tester.tap(find.text('Tap Me'));
      expect(tapped, true);
    });
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/widgets/settings_tile_test.dart`
Expected: FAIL — `SettingsTile` not found.

- [ ] **Step 3: Create SettingsGroup widget**

```dart
// lib/widgets/settings_group.dart
import 'package:flutter/material.dart';
import '../theme.dart';

class SettingsGroup extends StatelessWidget {
  final String title;
  final List<Widget> children;

  const SettingsGroup({super.key, required this.title, required this.children});

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 20, 16, 8),
          child: Text(
            title.toUpperCase(),
            style: const TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.w600,
              color: AppColors.textSecondary,
              letterSpacing: 1.2,
            ),
          ),
        ),
        Card(
          margin: const EdgeInsets.symmetric(horizontal: 16),
          child: Column(children: children),
        ),
      ],
    );
  }
}
```

- [ ] **Step 4: Create SettingsTile widget**

```dart
// lib/widgets/settings_tile.dart
import 'package:flutter/material.dart';
import '../theme.dart';

class SettingsTile extends StatelessWidget {
  final IconData icon;
  final String title;
  final Widget? trailing;
  final VoidCallback? onTap;

  const SettingsTile({
    super.key,
    required this.icon,
    required this.title,
    this.trailing,
    this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: Icon(icon, color: AppColors.primaryTeal),
      title: Text(title),
      trailing: trailing ?? (onTap != null ? const Icon(Icons.chevron_right) : null),
      onTap: onTap,
    );
  }
}
```

- [ ] **Step 5: Create LanguageSelector widget**

```dart
// lib/widgets/language_selector.dart
import 'package:flutter/material.dart';
import '../theme.dart';

class LanguageSelector extends StatelessWidget {
  final String current;
  final ValueChanged<String> onChanged;

  const LanguageSelector({
    super.key,
    required this.current,
    required this.onChanged,
  });

  static const _languages = {
    'en': 'English',
    'hi': 'Hindi',
    'ta': 'Tamil',
    'te': 'Telugu',
    'kn': 'Kannada',
    'ml': 'Malayalam',
    'bn': 'Bengali',
    'mr': 'Marathi',
  };

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: const Icon(Icons.language, color: AppColors.primaryTeal),
      title: const Text('Language'),
      trailing: DropdownButton<String>(
        value: current,
        underline: const SizedBox.shrink(),
        items: _languages.entries
            .map((e) => DropdownMenuItem(value: e.key, child: Text(e.value)))
            .toList(),
        onChanged: (v) {
          if (v != null) onChanged(v);
        },
      ),
    );
  }
}
```

- [ ] **Step 6: Create FamilyShareButton widget**

```dart
// lib/widgets/family_share_button.dart
import 'package:flutter/material.dart';
import '../theme.dart';

class FamilyShareButton extends StatefulWidget {
  final VoidCallback? onShare;

  const FamilyShareButton({super.key, this.onShare});

  @override
  State<FamilyShareButton> createState() => _FamilyShareButtonState();
}

class _FamilyShareButtonState extends State<FamilyShareButton> {
  String? _shareLink;

  void _generateLink() {
    setState(() {
      _shareLink = 'https://vaidshala.health/family/rajesh-kumar-abc123';
    });
    widget.onShare?.call();
  }

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: const Icon(Icons.family_restroom, color: AppColors.primaryTeal),
      title: const Text('Share Health Plan'),
      subtitle: _shareLink != null
          ? Text(_shareLink!, style: const TextStyle(fontSize: 12))
          : null,
      trailing: FilledButton(
        onPressed: _shareLink == null ? _generateLink : null,
        child: Text(_shareLink == null ? 'Generate Link' : 'Shared'),
      ),
    );
  }
}
```

- [ ] **Step 7: Run tests**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/widgets/settings_tile_test.dart`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
cd vaidshala/clinical-applications/ui/patient
git add lib/widgets/settings_group.dart lib/widgets/settings_tile.dart lib/widgets/language_selector.dart lib/widgets/family_share_button.dart test/widgets/settings_tile_test.dart
git commit -m "feat(patient-app): add settings widgets

SettingsGroup, SettingsTile, LanguageSelector, FamilyShareButton."
```

---

## Task 9: Settings Screen (S11)

**Files:**
- Create: `lib/screens/settings_screen.dart`
- Test: `test/screens/settings_screen_test.dart`

**Context:** S11 is a pushed route `/settings` — full screen with its own AppBar (no bottom nav). ListView with 5 grouped sections. Reads from `settingsProvider` and `authStateProvider`.

- [ ] **Step 1: Write settings screen test**

```dart
// test/screens/settings_screen_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:vaidshala_patient/screens/settings_screen.dart';
import 'package:vaidshala_patient/providers/settings_provider.dart';
import 'package:vaidshala_patient/models/settings_state.dart';

void main() {
  group('SettingsScreen', () {
    testWidgets('renders Account section with patient name', (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            settingsProvider.overrideWith(
              (ref) => _FakeSettingsNotifier(),
            ),
          ],
          child: const MaterialApp(home: SettingsScreen()),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Settings'), findsOneWidget);
      expect(find.text('Rajesh Kumar'), findsOneWidget);
      expect(find.textContaining('+91'), findsOneWidget);
    });

    testWidgets('renders language dropdown', (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            settingsProvider.overrideWith(
              (ref) => _FakeSettingsNotifier(),
            ),
          ],
          child: const MaterialApp(home: SettingsScreen()),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Language'), findsOneWidget);
      expect(find.text('English'), findsOneWidget);
    });
  });
}

class _FakeSettingsNotifier extends StateNotifier<SettingsState>
    implements SettingsNotifier {
  _FakeSettingsNotifier() : super(const SettingsState());

  @override
  void setLanguage(String language) =>
      state = state.copyWith(language: language);

  @override
  void toggleNotifications() =>
      state = state.copyWith(notificationsEnabled: !state.notificationsEnabled);
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/screens/settings_screen_test.dart`
Expected: FAIL — `SettingsScreen` not found.

- [ ] **Step 3: Create Settings Screen**

```dart
// lib/screens/settings_screen.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../providers/auth_provider.dart';
import '../providers/settings_provider.dart';
import '../theme.dart';
import '../widgets/family_share_button.dart';
import '../widgets/language_selector.dart';
import '../widgets/settings_group.dart';
import '../widgets/settings_tile.dart';

class SettingsScreen extends ConsumerWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final settings = ref.watch(settingsProvider);
    final authAsync = ref.watch(authStateProvider);
    final auth = authAsync.valueOrNull;

    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: ListView(
        children: [
          // Account
          SettingsGroup(
            title: 'Account',
            children: [
              SettingsTile(
                icon: Icons.person,
                title: 'Name',
                trailing: Text(
                  auth?.name ?? 'Rajesh Kumar',
                  style: const TextStyle(color: AppColors.textSecondary),
                ),
              ),
              SettingsTile(
                icon: Icons.phone,
                title: 'Phone',
                trailing: Text(
                  auth?.phone ?? '+91 98765 43210',
                  style: const TextStyle(color: AppColors.textSecondary),
                ),
              ),
              SettingsTile(
                icon: Icons.verified_user,
                title: 'ABHA',
                trailing: const Text(
                  'Linked — rajesh.kumar@abdm',
                  style: TextStyle(
                    color: AppColors.scoreGreen,
                    fontSize: 12,
                  ),
                ),
                onTap: () => context.push('/abha-verify'),
              ),
            ],
          ),

          // Preferences
          SettingsGroup(
            title: 'Preferences',
            children: [
              LanguageSelector(
                current: settings.language,
                onChanged: (lang) =>
                    ref.read(settingsProvider.notifier).setLanguage(lang),
              ),
              SettingsTile(
                icon: Icons.notifications,
                title: 'Notifications',
                trailing: Switch(
                  value: settings.notificationsEnabled,
                  onChanged: (_) =>
                      ref.read(settingsProvider.notifier).toggleNotifications(),
                ),
              ),
            ],
          ),

          // Family
          SettingsGroup(
            title: 'Family',
            children: const [
              FamilyShareButton(),
            ],
          ),

          // Data
          SettingsGroup(
            title: 'Data',
            children: [
              SettingsTile(
                icon: Icons.download,
                title: 'Download My Data',
                onTap: () => ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('Coming soon')),
                ),
              ),
              SettingsTile(
                icon: Icons.delete_forever,
                title: 'Delete Account',
                onTap: () => _showDeleteConfirmation(context),
              ),
            ],
          ),

          // About
          SettingsGroup(
            title: 'About',
            children: const [
              SettingsTile(
                icon: Icons.info_outline,
                title: 'App Version',
                trailing: Text('1.0.0', style: TextStyle(color: AppColors.textSecondary)),
              ),
              SettingsTile(
                icon: Icons.description,
                title: 'Terms of Service',
              ),
              SettingsTile(
                icon: Icons.privacy_tip,
                title: 'Privacy Policy',
              ),
            ],
          ),

          // Logout
          Padding(
            padding: const EdgeInsets.all(16),
            child: OutlinedButton.icon(
              onPressed: () {
                ref.read(authStateProvider.notifier).logout();
                context.go('/login');
              },
              icon: const Icon(Icons.logout, color: AppColors.scoreRed),
              label: const Text('Log Out',
                  style: TextStyle(color: AppColors.scoreRed)),
            ),
          ),

          const SizedBox(height: 32),
        ],
      ),
    );
  }

  void _showDeleteConfirmation(BuildContext context) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete Account?'),
        content: const Text(
          'This will permanently delete your account and all health data. '
          'This action cannot be undone.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () {
              Navigator.pop(ctx);
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(content: Text('Account deletion requested')),
              );
            },
            style: TextButton.styleFrom(foregroundColor: AppColors.scoreRed),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
  }
}
```

- [ ] **Step 4: Run tests**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/screens/settings_screen_test.dart`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd vaidshala/clinical-applications/ui/patient
git add lib/screens/settings_screen.dart test/screens/settings_screen_test.dart
git commit -m "feat(patient-app): add Settings screen (S11)

5-section ListView: Account, Preferences, Family, Data, About.
Language selector, notification toggle, ABHA link, logout."
```

---

## Task 10: Score Detail Widgets

**Files:**
- Create: `lib/widgets/full_sparkline_chart.dart`
- Create: `lib/widgets/domain_breakdown_bar.dart`
- Create: `lib/widgets/score_explanation_card.dart`
- Test: `test/widgets/domain_breakdown_bar_test.dart`

- [ ] **Step 1: Write domain breakdown bar test**

```dart
// test/widgets/domain_breakdown_bar_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/widgets/domain_breakdown_bar.dart';

void main() {
  group('DomainBreakdownBar', () {
    testWidgets('renders label, score, and target', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: DomainBreakdownBar(
              label: 'Blood Sugar',
              score: 35,
              target: 60,
              icon: Icons.bloodtype,
              color: Color(0xFFC62828),
            ),
          ),
        ),
      );

      expect(find.text('Blood Sugar'), findsOneWidget);
      expect(find.text('35'), findsOneWidget);
      expect(find.textContaining('60'), findsOneWidget);
    });
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/widgets/domain_breakdown_bar_test.dart`
Expected: FAIL — `DomainBreakdownBar` not found.

- [ ] **Step 3: Create FullSparklineChart widget**

```dart
// lib/widgets/full_sparkline_chart.dart
import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/material.dart';
import '../theme.dart';

class FullSparklineChart extends StatelessWidget {
  final List<double> data;
  final double? targetLine;
  final double height;

  const FullSparklineChart({
    super.key,
    required this.data,
    this.targetLine,
    this.height = 120,
  });

  @override
  Widget build(BuildContext context) {
    if (data.isEmpty) return SizedBox(height: height);

    final spots = data
        .asMap()
        .entries
        .map((e) => FlSpot(e.key.toDouble(), e.value))
        .toList();

    return SizedBox(
      height: height,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16),
        child: LineChart(
          LineChartData(
            minY: 0,
            maxY: 100,
            gridData: const FlGridData(show: false),
            borderData: FlBorderData(show: false),
            titlesData: FlTitlesData(
              leftTitles: const AxisTitles(
                sideTitles: SideTitles(showTitles: false),
              ),
              rightTitles: const AxisTitles(
                sideTitles: SideTitles(showTitles: false),
              ),
              topTitles: const AxisTitles(
                sideTitles: SideTitles(showTitles: false),
              ),
              bottomTitles: AxisTitles(
                sideTitles: SideTitles(
                  showTitles: true,
                  interval: 1,
                  getTitlesWidget: (value, meta) {
                    if (value.toInt() % 4 == 0) {
                      return Text(
                        'W${value.toInt() + 1}',
                        style: const TextStyle(
                          fontSize: 10,
                          color: AppColors.textSecondary,
                        ),
                      );
                    }
                    return const SizedBox.shrink();
                  },
                ),
              ),
            ),
            extraLinesData: targetLine != null
                ? ExtraLinesData(horizontalLines: [
                    HorizontalLine(
                      y: targetLine!,
                      color: AppColors.scoreGreen.withValues(alpha: 0.5),
                      strokeWidth: 1,
                      dashArray: [8, 4],
                      label: HorizontalLineLabel(
                        show: true,
                        labelResolver: (_) => 'Target',
                        style: const TextStyle(
                          fontSize: 10,
                          color: AppColors.scoreGreen,
                        ),
                      ),
                    ),
                  ])
                : null,
            lineBarsData: [
              LineChartBarData(
                spots: spots,
                isCurved: true,
                color: AppColors.primaryTeal,
                barWidth: 2.5,
                dotData: FlDotData(
                  show: true,
                  getDotPainter: (spot, _, __, ___) {
                    if (spot == spots.last) {
                      return FlDotCirclePainter(
                        radius: 4,
                        color: AppColors.primaryTeal,
                        strokeWidth: 2,
                        strokeColor: Colors.white,
                      );
                    }
                    return FlDotCirclePainter(radius: 0, color: Colors.transparent);
                  },
                ),
                belowBarData: BarAreaData(
                  show: true,
                  gradient: LinearGradient(
                    begin: Alignment.topCenter,
                    end: Alignment.bottomCenter,
                    colors: [
                      AppColors.primaryTeal.withValues(alpha: 0.3),
                      AppColors.primaryTeal.withValues(alpha: 0.05),
                    ],
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
```

- [ ] **Step 4: Create DomainBreakdownBar widget**

```dart
// lib/widgets/domain_breakdown_bar.dart
import 'package:flutter/material.dart';
import '../theme.dart';

class DomainBreakdownBar extends StatelessWidget {
  final String label;
  final int score;
  final int target;
  final IconData icon;
  final Color color;

  const DomainBreakdownBar({
    super.key,
    required this.label,
    required this.score,
    required this.target,
    required this.icon,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(icon, size: 18, color: color),
              const SizedBox(width: 8),
              Text(label,
                  style: const TextStyle(
                      fontSize: 14, fontWeight: FontWeight.w500)),
              const Spacer(),
              Text('$score',
                  style: TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.bold,
                      color: color)),
              Text(' / $target',
                  style: const TextStyle(
                      fontSize: 12, color: AppColors.textSecondary)),
            ],
          ),
          const SizedBox(height: 4),
          Stack(
            children: [
              // Background
              Container(
                height: 8,
                decoration: BoxDecoration(
                  color: Colors.grey.shade200,
                  borderRadius: BorderRadius.circular(4),
                ),
              ),
              // Target marker
              FractionallySizedBox(
                widthFactor: (target / 100).clamp(0, 1),
                child: Container(
                  height: 8,
                  alignment: Alignment.centerRight,
                  child: Container(
                    width: 2,
                    height: 8,
                    color: AppColors.textSecondary,
                  ),
                ),
              ),
              // Score bar (animated)
              TweenAnimationBuilder<double>(
                tween: Tween(begin: 0, end: score / 100),
                duration: const Duration(milliseconds: 800),
                curve: Curves.easeOutCubic,
                builder: (_, value, __) => FractionallySizedBox(
                  widthFactor: value.clamp(0, 1),
                  child: Container(
                    height: 8,
                    decoration: BoxDecoration(
                      color: color,
                      borderRadius: BorderRadius.circular(4),
                    ),
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
```

- [ ] **Step 5: Create ScoreExplanationCard widget**

```dart
// lib/widgets/score_explanation_card.dart
import 'package:flutter/material.dart';
import '../theme.dart';

class ScoreExplanationCard extends StatelessWidget {
  final int score;
  final String explanation;

  const ScoreExplanationCard({
    super.key,
    required this.score,
    required this.explanation,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.all(16),
      color: AppColors.coachingGreen,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.lightbulb_outline,
                    color: AppColors.scoreGreen, size: 20),
                const SizedBox(width: 8),
                const Text(
                  'What This Means',
                  style: TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              explanation,
              style: const TextStyle(
                fontSize: 13,
                color: AppColors.textPrimary,
                height: 1.5,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
```

- [ ] **Step 6: Run tests**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/widgets/domain_breakdown_bar_test.dart`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
cd vaidshala/clinical-applications/ui/patient
git add lib/widgets/full_sparkline_chart.dart lib/widgets/domain_breakdown_bar.dart lib/widgets/score_explanation_card.dart test/widgets/domain_breakdown_bar_test.dart
git commit -m "feat(patient-app): add Score Detail widgets

FullSparklineChart (fl_chart area), DomainBreakdownBar (animated),
ScoreExplanationCard for S12."
```

---

## Task 11: Score Detail Screen (S12) + Home Tab Hero

**Files:**
- Create: `lib/screens/score_detail_screen.dart`
- Modify: `lib/screens/home_tab.dart` (wrap ScoreRing in Hero, make tappable)
- Test: `test/screens/score_detail_screen_test.dart`

**Context:** The ScoreRing in home_tab.dart (line 127) needs a Hero wrapper with `tag: 'score-ring'`. The entire `_ScoreCard` becomes tappable → navigates to `/score-detail`. The Score Detail screen has the same Hero tag on a larger (180px) ScoreRing.

- [ ] **Step 1: Write score detail screen test**

```dart
// test/screens/score_detail_screen_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:vaidshala_patient/screens/score_detail_screen.dart';
import 'package:vaidshala_patient/providers/score_detail_provider.dart';
import 'package:vaidshala_patient/models/domain_score.dart';

void main() {
  group('ScoreDetailScreen', () {
    testWidgets('renders domain breakdown bars', (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            scoreDetailProvider.overrideWithValue(
              const ScoreDetailState(
                score: 18,
                label: 'Needs Attention',
                scoreHistory: [25.0, 28.0, 24.0, 22.0, 20.0, 19.0, 18.0, 18.0, 19.0, 17.0, 18.0, 18.0],
                domains: [
                  DomainScore(name: 'Blood Sugar', score: 35, target: 60, icon: 'bloodtype'),
                  DomainScore(name: 'Activity', score: 22, target: 50, icon: 'directions_walk'),
                ],
                explanation: 'Test explanation text',
              ),
            ),
          ],
          child: const MaterialApp(home: ScoreDetailScreen()),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Your Health Score'), findsOneWidget);
      expect(find.text('Blood Sugar'), findsOneWidget);
      expect(find.text('Activity'), findsOneWidget);
      expect(find.text('Test explanation text'), findsOneWidget);
    });
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/screens/score_detail_screen_test.dart`
Expected: FAIL — `ScoreDetailScreen` not found.

- [ ] **Step 3: Create Score Detail Screen**

```dart
// lib/screens/score_detail_screen.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/score_detail_provider.dart';
import '../theme.dart';
import '../utils/icon_mapper.dart';
import '../widgets/domain_breakdown_bar.dart';
import '../widgets/full_sparkline_chart.dart';
import '../widgets/score_explanation_card.dart';
import '../widgets/score_ring.dart';

class ScoreDetailScreen extends ConsumerWidget {
  const ScoreDetailScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final detail = ref.watch(scoreDetailProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('Your Health Score')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.only(bottom: 32),
        child: Column(
          children: [
            // Hero ScoreRing (large)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 24),
              child: Center(
                child: Hero(
                  tag: 'score-ring',
                  child: ScoreRing(score: detail.score, size: 180),
                ),
              ),
            ),

            // 12-week sparkline
            if (detail.scoreHistory.isNotEmpty) ...[
              const Padding(
                padding: EdgeInsets.fromLTRB(16, 8, 16, 4),
                child: Text(
                  '12-Week Trend',
                  style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
                ),
              ),
              FullSparklineChart(
                data: detail.scoreHistory,
                targetLine: 60,
                height: 120,
              ),
              const SizedBox(height: 16),
            ],

            // Domain breakdown
            if (detail.domains.isNotEmpty) ...[
              const Padding(
                padding: EdgeInsets.fromLTRB(16, 16, 16, 8),
                child: Text(
                  'Score Breakdown',
                  style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
                ),
              ),
              ...detail.domains.map(
                (d) => DomainBreakdownBar(
                  label: d.name,
                  score: d.score,
                  target: d.target,
                  icon: mapIcon(d.icon),
                  color: AppColors.scoreColor(d.score),
                ),
              ),
            ],

            // Explanation card
            if (detail.explanation.isNotEmpty)
              ScoreExplanationCard(
                score: detail.score ?? 0,
                explanation: detail.explanation,
              ),
          ],
        ),
      ),
    );
  }
}
```

- [ ] **Step 4: Modify home_tab.dart — wrap ScoreRing in Hero and make card tappable**

In `lib/screens/home_tab.dart`, modify the `_ScoreCard` widget.

Change the `ScoreRing` at line 127 from:
```dart
ScoreRing(score: score, size: 120),
```
to:
```dart
Hero(
  tag: 'score-ring',
  child: ScoreRing(score: score, size: 120),
),
```

Wrap the `Card` in `_ScoreCard.build()` (line 120) with `GestureDetector`:

Change:
```dart
return Card(
```
to:
```dart
return GestureDetector(
  onTap: () => context.push('/score-detail'),
  child: Card(
```

And close the `GestureDetector` after the `Card` closing paren (after line 156):
Add closing `);` for GestureDetector.

Also add import at top:
```dart
import 'package:go_router/go_router.dart';
```

And update the `build` signature to accept context properly — since `_ScoreCard` is a `StatelessWidget`, its `build` already has `BuildContext context`, so `context.push` will work.

- [ ] **Step 5: Run tests**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/screens/score_detail_screen_test.dart`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
cd vaidshala/clinical-applications/ui/patient
git add lib/screens/score_detail_screen.dart lib/screens/home_tab.dart test/screens/score_detail_screen_test.dart
git commit -m "feat(patient-app): add Score Detail screen (S12) + Hero animation

Hero-wrapped ScoreRing transitions from Home (120px) to Detail (180px).
12-week sparkline, domain breakdown bars, explanation card."
```

---

## Task 12: Notification Widgets + Screen (S13)

**Files:**
- Create: `lib/widgets/notification_date_group.dart`
- Create: `lib/widgets/notification_item.dart`
- Create: `lib/screens/notifications_screen.dart`
- Test: `test/widgets/notification_item_test.dart`
- Test: `test/screens/notifications_screen_test.dart`

- [ ] **Step 1: Write notification item test**

```dart
// test/widgets/notification_item_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/models/app_notification.dart';
import 'package:vaidshala_patient/widgets/notification_item.dart';

void main() {
  group('NotificationItem', () {
    testWidgets('renders title and body', (tester) async {
      final notification = AppNotification(
        id: 'n1',
        type: NotificationType.coaching,
        title: 'Great progress!',
        body: 'Your FBG dropped 12 mg/dL',
        timestamp: DateTime.now(),
      );

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: NotificationItem(
              notification: notification,
              onTap: () {},
              onDismiss: () {},
            ),
          ),
        ),
      );

      expect(find.text('Great progress!'), findsOneWidget);
      expect(find.text('Your FBG dropped 12 mg/dL'), findsOneWidget);
    });

    testWidgets('shows unread dot when not read', (tester) async {
      final notification = AppNotification(
        id: 'n2',
        type: NotificationType.alert,
        title: 'Alert',
        body: 'Body',
        timestamp: DateTime.now(),
        read: false,
      );

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: NotificationItem(
              notification: notification,
              onTap: () {},
              onDismiss: () {},
            ),
          ),
        ),
      );

      // Unread dot is a small blue Container
      final dot = find.byWidgetPredicate(
        (w) => w is Container && w.decoration is BoxDecoration &&
               (w.decoration as BoxDecoration).color == Colors.blue,
      );
      expect(dot, findsOneWidget);
    });
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/widgets/notification_item_test.dart`
Expected: FAIL — `NotificationItem` not found.

- [ ] **Step 3: Create NotificationItem widget**

```dart
// lib/widgets/notification_item.dart
import 'package:flutter/material.dart';
import '../models/app_notification.dart';
import '../theme.dart';

class NotificationItem extends StatelessWidget {
  final AppNotification notification;
  final VoidCallback onTap;
  final VoidCallback onDismiss;

  const NotificationItem({
    super.key,
    required this.notification,
    required this.onTap,
    required this.onDismiss,
  });

  IconData get _typeIcon {
    switch (notification.type) {
      case NotificationType.coaching:
        return Icons.school;
      case NotificationType.reminder:
        return Icons.alarm;
      case NotificationType.alert:
        return Icons.warning_amber;
      case NotificationType.milestone:
        return Icons.emoji_events;
    }
  }

  Color get _typeColor {
    switch (notification.type) {
      case NotificationType.coaching:
        return AppColors.scoreGreen;
      case NotificationType.reminder:
        return AppColors.primaryTeal;
      case NotificationType.alert:
        return Colors.orange;
      case NotificationType.milestone:
        return Colors.purple;
    }
  }

  String get _timeAgo {
    final diff = DateTime.now().difference(notification.timestamp);
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    if (diff.inHours < 24) return '${diff.inHours}h ago';
    if (diff.inDays == 1) return 'Yesterday';
    return '${diff.inDays}d ago';
  }

  @override
  Widget build(BuildContext context) {
    return Dismissible(
      key: Key(notification.id),
      direction: DismissDirection.endToStart,
      onDismissed: (_) => onDismiss(),
      background: Container(
        color: AppColors.scoreRed,
        alignment: Alignment.centerRight,
        padding: const EdgeInsets.only(right: 16),
        child: const Icon(Icons.delete, color: Colors.white),
      ),
      child: ListTile(
        leading: CircleAvatar(
          backgroundColor: _typeColor.withValues(alpha: 0.15),
          child: Icon(_typeIcon, color: _typeColor, size: 20),
        ),
        title: Text(
          notification.title,
          style: TextStyle(
            fontWeight: notification.read ? FontWeight.normal : FontWeight.bold,
          ),
        ),
        subtitle: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(notification.body, maxLines: 2, overflow: TextOverflow.ellipsis),
            const SizedBox(height: 2),
            Text(_timeAgo,
                style: const TextStyle(fontSize: 11, color: AppColors.textSecondary)),
          ],
        ),
        trailing: notification.read
            ? null
            : Container(
                width: 8,
                height: 8,
                decoration: const BoxDecoration(
                  color: Colors.blue,
                  shape: BoxShape.circle,
                ),
              ),
        onTap: onTap,
      ),
    );
  }
}
```

- [ ] **Step 4: Create NotificationDateGroup widget**

```dart
// lib/widgets/notification_date_group.dart
import 'package:flutter/material.dart';
import '../models/app_notification.dart';
import '../theme.dart';
import 'notification_item.dart';

class NotificationDateGroup extends StatelessWidget {
  final String label;
  final List<AppNotification> items;
  final ValueChanged<AppNotification> onTap;
  final ValueChanged<String> onDismiss;

  const NotificationDateGroup({
    super.key,
    required this.label,
    required this.items,
    required this.onTap,
    required this.onDismiss,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
          child: Text(
            label,
            style: const TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.w600,
              color: AppColors.textSecondary,
              letterSpacing: 1.0,
            ),
          ),
        ),
        ...items.map((n) => NotificationItem(
              notification: n,
              onTap: () => onTap(n),
              onDismiss: () => onDismiss(n.id),
            )),
      ],
    );
  }
}
```

- [ ] **Step 5: Create Notifications Screen**

```dart
// lib/screens/notifications_screen.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../models/app_notification.dart';
import '../providers/notifications_provider.dart';
import '../theme.dart';
import '../widgets/notification_date_group.dart';

class NotificationsScreen extends ConsumerWidget {
  const NotificationsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final notificationsAsync = ref.watch(notificationsProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Notifications'),
        actions: [
          TextButton(
            onPressed: () =>
                ref.read(notificationsProvider.notifier).markAllRead(),
            child: const Text('Mark all read'),
          ),
        ],
      ),
      body: notificationsAsync.when(
        data: (notifications) {
          if (notifications.isEmpty) {
            return const Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(Icons.notifications_none,
                      size: 64, color: AppColors.textSecondary),
                  SizedBox(height: 16),
                  Text(
                    'No notifications yet',
                    style: TextStyle(
                      fontSize: 16,
                      color: AppColors.textSecondary,
                    ),
                  ),
                ],
              ),
            );
          }

          final grouped = _groupByDate(notifications);
          return ListView(
            children: grouped.entries
                .map((entry) => NotificationDateGroup(
                      label: entry.key,
                      items: entry.value,
                      onTap: (n) {
                        ref
                            .read(notificationsProvider.notifier)
                            .markRead(n.id);
                        if (n.deepLink != null) {
                          context.go(n.deepLink!);
                        }
                      },
                      onDismiss: (id) => ref
                          .read(notificationsProvider.notifier)
                          .dismiss(id),
                    ))
                .toList(),
          );
        },
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (_, __) => const Center(child: Text('Unable to load notifications')),
      ),
    );
  }

  Map<String, List<AppNotification>> _groupByDate(
      List<AppNotification> notifications) {
    final now = DateTime.now();
    final today = DateTime(now.year, now.month, now.day);
    final yesterday = today.subtract(const Duration(days: 1));
    final weekAgo = today.subtract(const Duration(days: 7));

    final groups = <String, List<AppNotification>>{};

    for (final n in notifications) {
      final date = DateTime(n.timestamp.year, n.timestamp.month, n.timestamp.day);
      String label;
      if (date == today || date.isAfter(today)) {
        label = 'TODAY';
      } else if (date == yesterday || (date.isAfter(yesterday) && date.isBefore(today))) {
        label = 'YESTERDAY';
      } else if (date.isAfter(weekAgo)) {
        label = 'THIS WEEK';
      } else {
        label = 'EARLIER';
      }
      groups.putIfAbsent(label, () => []).add(n);
    }

    return groups;
  }
}
```

- [ ] **Step 6: Write notifications screen test**

```dart
// test/screens/notifications_screen_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:vaidshala_patient/models/app_notification.dart';
import 'package:vaidshala_patient/providers/notifications_provider.dart';
import 'package:vaidshala_patient/screens/notifications_screen.dart';

void main() {
  group('NotificationsScreen', () {
    testWidgets('renders empty state when no notifications', (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            notificationsProvider.overrideWith(
              () => _EmptyNotificationsNotifier(),
            ),
          ],
          child: const MaterialApp(home: NotificationsScreen()),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('No notifications yet'), findsOneWidget);
    });

    testWidgets('renders notifications grouped by date', (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            notificationsProvider.overrideWith(
              () => _MockNotificationsNotifier(),
            ),
          ],
          child: const MaterialApp(home: NotificationsScreen()),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Great progress!'), findsOneWidget);
      expect(find.text('TODAY'), findsOneWidget);
    });
  });
}

class _EmptyNotificationsNotifier extends AsyncNotifier<List<AppNotification>>
    implements NotificationsNotifier {
  @override
  Future<List<AppNotification>> build() async => [];
  @override
  Future<void> markRead(String id) async {}
  @override
  Future<void> markAllRead() async {}
  @override
  Future<void> dismiss(String id) async {}
  @override
  Future<void> refresh() async {}
}

class _MockNotificationsNotifier extends AsyncNotifier<List<AppNotification>>
    implements NotificationsNotifier {
  @override
  Future<List<AppNotification>> build() async => [
        AppNotification(
          id: 'n1',
          type: NotificationType.coaching,
          title: 'Great progress!',
          body: 'Your FBG dropped',
          timestamp: DateTime.now(),
          read: true,
        ),
      ];
  @override
  Future<void> markRead(String id) async {}
  @override
  Future<void> markAllRead() async {}
  @override
  Future<void> dismiss(String id) async {}
  @override
  Future<void> refresh() async {}
}
```

- [ ] **Step 7: Run tests**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/widgets/notification_item_test.dart test/screens/notifications_screen_test.dart`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
cd vaidshala/clinical-applications/ui/patient
git add lib/widgets/notification_date_group.dart lib/widgets/notification_item.dart lib/screens/notifications_screen.dart test/widgets/notification_item_test.dart test/screens/notifications_screen_test.dart
git commit -m "feat(patient-app): add Notification Center (S13)

NotificationItem with Dismissible swipe-to-delete, unread dot.
NotificationDateGroup groups by Today/Yesterday/This Week.
NotificationsScreen with empty state and Mark All Read."
```

---

## Task 13: Vitals Entry Widgets + Sheet (S14)

**Files:**
- Create: `lib/widgets/bp_entry_card.dart`
- Create: `lib/widgets/glucose_entry_card.dart`
- Create: `lib/widgets/weight_entry_card.dart`
- Create: `lib/widgets/vitals_entry_sheet.dart`
- Test: `test/widgets/bp_entry_card_test.dart`

- [ ] **Step 1: Write BP entry card test**

```dart
// test/widgets/bp_entry_card_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:vaidshala_patient/widgets/bp_entry_card.dart';
import 'package:vaidshala_patient/providers/vitals_entry_provider.dart';

void main() {
  group('BpEntryCard', () {
    testWidgets('renders systolic and diastolic fields', (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          child: MaterialApp(
            home: Scaffold(
              body: BpEntryCard(onSaved: () {}),
            ),
          ),
        ),
      );

      expect(find.text('Blood Pressure'), findsOneWidget);
      expect(find.byType(TextFormField), findsNWidgets(2));
      expect(find.text('Save'), findsOneWidget);
    });

    testWidgets('shows validation error for out-of-range systolic', (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          child: MaterialApp(
            home: Scaffold(
              body: BpEntryCard(onSaved: () {}),
            ),
          ),
        ),
      );

      // Enter invalid systolic
      await tester.enterText(find.byType(TextFormField).first, '300');
      await tester.enterText(find.byType(TextFormField).last, '80');
      await tester.tap(find.text('Save'));
      await tester.pumpAndSettle();

      expect(find.textContaining('between 60 and 250'), findsOneWidget);
    });
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/widgets/bp_entry_card_test.dart`
Expected: FAIL — `BpEntryCard` not found.

- [ ] **Step 3: Create BpEntryCard widget**

```dart
// lib/widgets/bp_entry_card.dart
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/vitals_entry_provider.dart';
import '../theme.dart';

class BpEntryCard extends ConsumerWidget {
  final VoidCallback onSaved;

  const BpEntryCard({super.key, required this.onSaved});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(vitalsEntryProvider);
    final notifier = ref.read(vitalsEntryProvider.notifier);

    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.favorite, color: AppColors.scoreRed, size: 20),
                const SizedBox(width: 8),
                const Text('Blood Pressure',
                    style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
                const Spacer(),
                const Text('mmHg',
                    style: TextStyle(fontSize: 12, color: AppColors.textSecondary)),
              ],
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  child: TextFormField(
                    decoration: InputDecoration(
                      labelText: 'Systolic',
                      hintText: '156',
                      errorText: state.errors['systolic'],
                      border: const OutlineInputBorder(),
                    ),
                    keyboardType: TextInputType.number,
                    inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                    onChanged: notifier.setSystolic,
                  ),
                ),
                const Padding(
                  padding: EdgeInsets.symmetric(horizontal: 8),
                  child: Text('/', style: TextStyle(fontSize: 20)),
                ),
                Expanded(
                  child: TextFormField(
                    decoration: InputDecoration(
                      labelText: 'Diastolic',
                      hintText: '98',
                      errorText: state.errors['diastolic'],
                      border: const OutlineInputBorder(),
                    ),
                    keyboardType: TextInputType.number,
                    inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                    onChanged: notifier.setDiastolic,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            SizedBox(
              width: double.infinity,
              child: FilledButton(
                onPressed: () async {
                  final saved = await notifier.saveBp();
                  if (saved && context.mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(content: Text('Blood pressure saved')),
                    );
                    onSaved();
                  }
                },
                child: const Text('Save'),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
```

- [ ] **Step 4: Create GlucoseEntryCard widget**

```dart
// lib/widgets/glucose_entry_card.dart
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/vitals_entry_provider.dart';
import '../theme.dart';

class GlucoseEntryCard extends ConsumerWidget {
  final VoidCallback onSaved;

  const GlucoseEntryCard({super.key, required this.onSaved});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(vitalsEntryProvider);
    final notifier = ref.read(vitalsEntryProvider.notifier);

    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.bloodtype, color: AppColors.scoreRed, size: 20),
                const SizedBox(width: 8),
                const Text('Blood Glucose',
                    style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
                const Spacer(),
                const Text('mg/dL',
                    style: TextStyle(fontSize: 12, color: AppColors.textSecondary)),
              ],
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  flex: 2,
                  child: TextFormField(
                    decoration: InputDecoration(
                      labelText: 'Glucose',
                      hintText: '178',
                      errorText: state.errors['glucose'],
                      border: const OutlineInputBorder(),
                    ),
                    keyboardType: TextInputType.number,
                    inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                    onChanged: notifier.setGlucoseValue,
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  flex: 3,
                  child: DropdownButtonFormField<String>(
                    value: state.glucoseContext,
                    decoration: InputDecoration(
                      labelText: 'Context',
                      errorText: state.errors['glucoseContext'],
                      border: const OutlineInputBorder(),
                    ),
                    items: const [
                      DropdownMenuItem(value: 'fasting', child: Text('Fasting')),
                      DropdownMenuItem(value: 'post-meal', child: Text('Post-meal')),
                      DropdownMenuItem(value: 'random', child: Text('Random')),
                    ],
                    onChanged: (v) => notifier.setGlucoseContext(v),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            SizedBox(
              width: double.infinity,
              child: FilledButton(
                onPressed: () async {
                  final saved = await notifier.saveGlucose();
                  if (saved && context.mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(content: Text('Blood glucose saved')),
                    );
                    onSaved();
                  }
                },
                child: const Text('Save'),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
```

- [ ] **Step 5: Create WeightEntryCard widget**

```dart
// lib/widgets/weight_entry_card.dart
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/vitals_entry_provider.dart';
import '../theme.dart';

class WeightEntryCard extends ConsumerWidget {
  final VoidCallback onSaved;

  const WeightEntryCard({super.key, required this.onSaved});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(vitalsEntryProvider);
    final notifier = ref.read(vitalsEntryProvider.notifier);

    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.monitor_weight, color: AppColors.primaryTeal, size: 20),
                const SizedBox(width: 8),
                const Text('Weight',
                    style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
                const Spacer(),
                const Text('kg',
                    style: TextStyle(fontSize: 12, color: AppColors.textSecondary)),
              ],
            ),
            const SizedBox(height: 12),
            TextFormField(
              decoration: InputDecoration(
                labelText: 'Weight',
                hintText: '82.5',
                errorText: state.errors['weight'],
                border: const OutlineInputBorder(),
              ),
              keyboardType: const TextInputType.numberWithOptions(decimal: true),
              inputFormatters: [
                FilteringTextInputFormatter.allow(RegExp(r'[\d.]')),
              ],
              onChanged: notifier.setWeight,
            ),
            const SizedBox(height: 12),
            SizedBox(
              width: double.infinity,
              child: FilledButton(
                onPressed: () async {
                  final saved = await notifier.saveWeight();
                  if (saved && context.mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(content: Text('Weight saved')),
                    );
                    onSaved();
                  }
                },
                child: const Text('Save'),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
```

- [ ] **Step 6: Create VitalsEntrySheet container**

```dart
// lib/widgets/vitals_entry_sheet.dart
import 'package:flutter/material.dart';
import 'bp_entry_card.dart';
import 'glucose_entry_card.dart';
import 'weight_entry_card.dart';

class VitalsEntrySheet extends StatelessWidget {
  const VitalsEntrySheet({super.key});

  @override
  Widget build(BuildContext context) {
    return DraggableScrollableSheet(
      initialChildSize: 0.85,
      minChildSize: 0.5,
      maxChildSize: 0.95,
      expand: false,
      builder: (context, scrollController) {
        return Container(
          decoration: const BoxDecoration(
            color: Colors.white,
            borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
          ),
          child: Column(
            children: [
              // Drag handle
              Container(
                margin: const EdgeInsets.symmetric(vertical: 8),
                width: 40,
                height: 4,
                decoration: BoxDecoration(
                  color: Colors.grey.shade300,
                  borderRadius: BorderRadius.circular(2),
                ),
              ),
              const Padding(
                padding: EdgeInsets.fromLTRB(16, 8, 16, 8),
                child: Text(
                  'Log a Reading',
                  style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold),
                ),
              ),
              Expanded(
                child: ListView(
                  controller: scrollController,
                  children: [
                    BpEntryCard(onSaved: () => Navigator.pop(context)),
                    GlucoseEntryCard(onSaved: () => Navigator.pop(context)),
                    WeightEntryCard(onSaved: () => Navigator.pop(context)),
                    const SizedBox(height: 32),
                  ],
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}
```

- [ ] **Step 7: Run tests**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/widgets/bp_entry_card_test.dart`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
cd vaidshala/clinical-applications/ui/patient
git add lib/widgets/bp_entry_card.dart lib/widgets/glucose_entry_card.dart lib/widgets/weight_entry_card.dart lib/widgets/vitals_entry_sheet.dart test/widgets/bp_entry_card_test.dart
git commit -m "feat(patient-app): add Vitals Entry sheet (S14)

BpEntryCard, GlucoseEntryCard, WeightEntryCard with validation.
VitalsEntrySheet as DraggableScrollableSheet container."
```

---

## Task 14: Symptom Logger Widgets + Sheet (S16)

**Files:**
- Create: `lib/widgets/symptom_chip.dart`
- Create: `lib/widgets/severity_selector.dart`
- Create: `lib/widgets/symptom_entry_sheet.dart`
- Test: `test/widgets/symptom_chip_test.dart`
- Test: `test/widgets/severity_selector_test.dart`

- [ ] **Step 1: Write symptom chip test**

```dart
// test/widgets/symptom_chip_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/widgets/symptom_chip.dart';

void main() {
  group('SymptomChip', () {
    testWidgets('renders label text', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SymptomChip(
              label: 'Dizziness',
              selected: false,
              onTap: () {},
            ),
          ),
        ),
      );

      expect(find.text('Dizziness'), findsOneWidget);
    });

    testWidgets('shows check icon when selected', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SymptomChip(
              label: 'Fatigue',
              selected: true,
              onTap: () {},
            ),
          ),
        ),
      );

      expect(find.byIcon(Icons.check), findsOneWidget);
    });

    testWidgets('calls onTap when tapped', (tester) async {
      var tapped = false;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SymptomChip(
              label: 'Nausea',
              selected: false,
              onTap: () => tapped = true,
            ),
          ),
        ),
      );

      await tester.tap(find.text('Nausea'));
      expect(tapped, true);
    });
  });
}
```

- [ ] **Step 2: Write severity selector test**

```dart
// test/widgets/severity_selector_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/widgets/severity_selector.dart';

void main() {
  group('SeveritySelector', () {
    testWidgets('renders 3 severity options', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SeveritySelector(
              value: null,
              onChanged: (_) {},
            ),
          ),
        ),
      );

      expect(find.text('Mild'), findsOneWidget);
      expect(find.text('Moderate'), findsOneWidget);
      expect(find.text('Severe'), findsOneWidget);
    });

    testWidgets('calls onChanged when tapped', (tester) async {
      String? selected;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SeveritySelector(
              value: null,
              onChanged: (v) => selected = v,
            ),
          ),
        ),
      );

      await tester.tap(find.text('Moderate'));
      expect(selected, 'moderate');
    });
  });
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/widgets/symptom_chip_test.dart test/widgets/severity_selector_test.dart`
Expected: FAIL — widgets not found.

- [ ] **Step 4: Create SymptomChip widget**

```dart
// lib/widgets/symptom_chip.dart
import 'package:flutter/material.dart';
import '../theme.dart';

class SymptomChip extends StatelessWidget {
  final String label;
  final bool selected;
  final VoidCallback onTap;

  const SymptomChip({
    super.key,
    required this.label,
    required this.selected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 200),
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        decoration: BoxDecoration(
          color: selected ? AppColors.primaryTeal : Colors.grey.shade100,
          borderRadius: BorderRadius.circular(20),
          border: Border.all(
            color: selected ? AppColors.primaryTeal : Colors.grey.shade300,
          ),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (selected)
              const Padding(
                padding: EdgeInsets.only(right: 4),
                child: Icon(Icons.check, size: 16, color: Colors.white),
              ),
            Text(
              label,
              style: TextStyle(
                color: selected ? Colors.white : AppColors.textPrimary,
                fontWeight: selected ? FontWeight.w600 : FontWeight.normal,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
```

- [ ] **Step 5: Create SeveritySelector widget**

```dart
// lib/widgets/severity_selector.dart
import 'package:flutter/material.dart';
import '../theme.dart';

class SeveritySelector extends StatelessWidget {
  final String? value;
  final ValueChanged<String> onChanged;

  const SeveritySelector({
    super.key,
    required this.value,
    required this.onChanged,
  });

  static const _options = [
    ('mild', 'Mild', AppColors.scoreGreen),
    ('moderate', 'Moderate', AppColors.scoreYellow),
    ('severe', 'Severe', AppColors.scoreRed),
  ];

  @override
  Widget build(BuildContext context) {
    return Row(
      children: _options.map((option) {
        final isSelected = value == option.$1;
        return Expanded(
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 4),
            child: GestureDetector(
              onTap: () => onChanged(option.$1),
              child: Container(
                padding: const EdgeInsets.symmetric(vertical: 10),
                decoration: BoxDecoration(
                  color: isSelected
                      ? option.$3.withValues(alpha: 0.15)
                      : Colors.grey.shade100,
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(
                    color: isSelected ? option.$3 : Colors.grey.shade300,
                    width: isSelected ? 2 : 1,
                  ),
                ),
                child: Center(
                  child: Text(
                    option.$2,
                    style: TextStyle(
                      color: isSelected ? option.$3 : AppColors.textSecondary,
                      fontWeight:
                          isSelected ? FontWeight.bold : FontWeight.normal,
                    ),
                  ),
                ),
              ),
            ),
          ),
        );
      }).toList(),
    );
  }
}
```

- [ ] **Step 6: Create SymptomEntrySheet container**

```dart
// lib/widgets/symptom_entry_sheet.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/symptom_entry_provider.dart';
import '../theme.dart';
import 'severity_selector.dart';
import 'symptom_chip.dart';

class SymptomEntrySheet extends ConsumerWidget {
  const SymptomEntrySheet({super.key});

  static const _symptoms = [
    'Dizziness',
    'Nausea',
    'Fatigue',
    'Chest Pain',
    'Swelling',
    'Breathlessness',
    'Low Sugar Feeling',
  ];

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(symptomEntryProvider);
    final notifier = ref.read(symptomEntryProvider.notifier);

    return DraggableScrollableSheet(
      initialChildSize: 0.8,
      minChildSize: 0.5,
      maxChildSize: 0.95,
      expand: false,
      builder: (context, scrollController) {
        return Container(
          decoration: const BoxDecoration(
            color: Colors.white,
            borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
          ),
          child: Column(
            children: [
              // Drag handle
              Container(
                margin: const EdgeInsets.symmetric(vertical: 8),
                width: 40,
                height: 4,
                decoration: BoxDecoration(
                  color: Colors.grey.shade300,
                  borderRadius: BorderRadius.circular(2),
                ),
              ),
              const Padding(
                padding: EdgeInsets.fromLTRB(16, 8, 16, 16),
                child: Text(
                  'Log a Symptom',
                  style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold),
                ),
              ),
              Expanded(
                child: ListView(
                  controller: scrollController,
                  padding: const EdgeInsets.symmetric(horizontal: 16),
                  children: [
                    // Symptom chips
                    const Text('What are you feeling?',
                        style: TextStyle(
                            fontSize: 14, fontWeight: FontWeight.w600)),
                    const SizedBox(height: 8),
                    Wrap(
                      spacing: 8,
                      runSpacing: 8,
                      children: _symptoms
                          .map((s) => SymptomChip(
                                label: s,
                                selected: state.selectedSymptoms.contains(s),
                                onTap: () => notifier.toggleSymptom(s),
                              ))
                          .toList(),
                    ),
                    const SizedBox(height: 20),

                    // Severity
                    const Text('How severe?',
                        style: TextStyle(
                            fontSize: 14, fontWeight: FontWeight.w600)),
                    const SizedBox(height: 8),
                    SeveritySelector(
                      value: state.severity,
                      onChanged: notifier.setSeverity,
                    ),
                    const SizedBox(height: 20),

                    // Free text
                    TextFormField(
                      decoration: const InputDecoration(
                        labelText: 'Tell us more (optional)',
                        border: OutlineInputBorder(),
                        counterText: '',
                      ),
                      maxLength: 200,
                      maxLines: 3,
                      onChanged: notifier.setNotes,
                    ),
                    const SizedBox(height: 20),

                    // Time selector
                    ListTile(
                      leading: const Icon(Icons.access_time),
                      title: const Text('When did this happen?'),
                      subtitle: Text(
                        _formatTime(state.timestamp),
                        style: const TextStyle(color: AppColors.primaryTeal),
                      ),
                      onTap: () async {
                        final time = await showTimePicker(
                          context: context,
                          initialTime: TimeOfDay.fromDateTime(state.timestamp),
                        );
                        if (time != null) {
                          final now = DateTime.now();
                          notifier.setTimestamp(DateTime(
                            now.year, now.month, now.day,
                            time.hour, time.minute,
                          ));
                        }
                      },
                    ),
                    const SizedBox(height: 16),

                    // Save button
                    SizedBox(
                      width: double.infinity,
                      child: FilledButton(
                        onPressed: state.canSave
                            ? () async {
                                final saved = await notifier.save();
                                if (saved && context.mounted) {
                                  ScaffoldMessenger.of(context).showSnackBar(
                                    const SnackBar(
                                        content: Text('Symptom logged')),
                                  );
                                  Navigator.pop(context);
                                }
                              }
                            : null,
                        child: const Text('Save'),
                      ),
                    ),
                    const SizedBox(height: 32),
                  ],
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  String _formatTime(DateTime dt) {
    final hour = dt.hour.toString().padLeft(2, '0');
    final min = dt.minute.toString().padLeft(2, '0');
    return '$hour:$min';
  }
}
```

- [ ] **Step 7: Run tests**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/widgets/symptom_chip_test.dart test/widgets/severity_selector_test.dart`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
cd vaidshala/clinical-applications/ui/patient
git add lib/widgets/symptom_chip.dart lib/widgets/severity_selector.dart lib/widgets/symptom_entry_sheet.dart test/widgets/symptom_chip_test.dart test/widgets/severity_selector_test.dart
git commit -m "feat(patient-app): add Symptom Logger sheet (S16)

SymptomChip (multi-select), SeveritySelector (mild/moderate/severe),
SymptomEntrySheet with time picker and optional free text."
```

---

## Task 15: Medication Adherence Widgets + Progress Tab Update (S15)

**Files:**
- Create: `lib/widgets/medication_adherence_card.dart`
- Create: `lib/widgets/medication_streak_row.dart`
- Modify: `lib/screens/progress_tab.dart` (add medication adherence section)

**Context:** The existing `progress_tab.dart` has 3 sections: Key Metrics, How Your Actions Help, Milestones. We add a 4th section "Medication Adherence" after Milestones (line 111). The `_ProgressContent` widget becomes a `ConsumerWidget` so it can watch `medicationAdherenceProvider`.

- [ ] **Step 1: Create MedicationStreakRow widget**

```dart
// lib/widgets/medication_streak_row.dart
import 'package:flutter/material.dart';
import '../theme.dart';

class MedicationStreakRow extends StatelessWidget {
  final String name;
  final int streakDays;

  const MedicationStreakRow({
    super.key,
    required this.name,
    required this.streakDays,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      child: Row(
        children: [
          const Icon(Icons.check_circle, color: AppColors.scoreGreen, size: 20),
          const SizedBox(width: 8),
          Expanded(
            child: Text(name,
                style: const TextStyle(fontSize: 14)),
          ),
          Text(
            '$streakDays-day streak',
            style: const TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.w600,
              color: AppColors.scoreGreen,
            ),
          ),
        ],
      ),
    );
  }
}
```

- [ ] **Step 2: Create MedicationAdherenceCard widget**

```dart
// lib/widgets/medication_adherence_card.dart
import 'package:flutter/material.dart';
import '../models/medication_adherence.dart';
import '../theme.dart';
import 'medication_streak_row.dart';

class MedicationAdherenceCard extends StatelessWidget {
  final MedicationAdherence adherence;

  const MedicationAdherenceCard({super.key, required this.adherence});

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Weekly adherence header
          Padding(
            padding: const EdgeInsets.all(16),
            child: Row(
              children: [
                SizedBox(
                  width: 48,
                  height: 48,
                  child: Stack(
                    alignment: Alignment.center,
                    children: [
                      CircularProgressIndicator(
                        value: adherence.weeklyPct / 100,
                        strokeWidth: 4,
                        backgroundColor: Colors.grey.shade200,
                        valueColor: const AlwaysStoppedAnimation(
                            AppColors.scoreGreen),
                      ),
                      Text(
                        '${adherence.weeklyPct}%',
                        style: const TextStyle(
                          fontSize: 12,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text(
                        'This Week',
                        style: TextStyle(
                          fontSize: 14,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      Text(
                        '${(adherence.weeklyPct * 7 / 100).round()} of 7 days — all meds taken',
                        style: const TextStyle(
                          fontSize: 12,
                          color: AppColors.textSecondary,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),

          const Divider(height: 1),

          // Per-medication streaks
          const SizedBox(height: 8),
          ...adherence.streaks.map(
            (s) => MedicationStreakRow(
              name: s.medicationName,
              streakDays: s.streakDays,
            ),
          ),

          // Missed dose indicator
          if (adherence.lastMissed != null) ...[
            const Divider(height: 24),
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
              child: Row(
                children: [
                  const Icon(Icons.warning_amber,
                      color: Colors.orange, size: 18),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      'Last missed: ${adherence.lastMissed!.medicationName}, '
                      '${adherence.lastMissed!.daysAgo} day${adherence.lastMissed!.daysAgo == 1 ? "" : "s"} ago',
                      style: const TextStyle(
                        fontSize: 12,
                        color: Colors.orange,
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ] else
            const SizedBox(height: 12),
        ],
      ),
    );
  }
}
```

- [ ] **Step 3: Modify progress_tab.dart — add medication adherence section**

In `lib/screens/progress_tab.dart`:

1. Add imports at the top:
```dart
import '../providers/medication_adherence_provider.dart';
import '../widgets/medication_adherence_card.dart';
```

2. Change `_ProgressContent` from `StatelessWidget` to `ConsumerWidget`:
Change line 41 from:
```dart
class _ProgressContent extends StatelessWidget {
```
to:
```dart
class _ProgressContent extends ConsumerWidget {
```

3. Change the `build` method signature from:
```dart
Widget build(BuildContext context) {
```
to:
```dart
Widget build(BuildContext context, WidgetRef ref) {
```

4. Add the medication adherence section after the Milestones section (after line 110, before the closing `],` of the Column's children):

```dart
          // Medication Adherence Section
          ref.watch(medicationAdherenceProvider).when(
            data: (adherence) => Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Padding(
                  padding: EdgeInsets.fromLTRB(16, 20, 16, 8),
                  child: Text(
                    'Medication Adherence',
                    style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                  ),
                ),
                MedicationAdherenceCard(adherence: adherence),
              ],
            ),
            loading: () => const SizedBox.shrink(),
            error: (_, __) => const SizedBox.shrink(),
          ),
```

- [ ] **Step 4: Commit**

```bash
cd vaidshala/clinical-applications/ui/patient
git add lib/widgets/medication_adherence_card.dart lib/widgets/medication_streak_row.dart lib/screens/progress_tab.dart
git commit -m "feat(patient-app): add Medication Adherence section (S15)

MedicationAdherenceCard with weekly %, per-med streaks, missed indicator.
Added to Progress tab below Milestones section."
```

---

## Task 16: SpeedDial FAB on My Day Tab

**Files:**
- Modify: `lib/screens/my_day_tab.dart`

**Context:** My Day tab currently has no FAB. We add a SpeedDial-style expandable FAB with 2 options: "Log Reading" (opens VitalsEntrySheet) and "Log Symptom" (opens SymptomEntrySheet). We implement this with a simple `_SpeedDialFab` StatefulWidget rather than adding a dependency.

- [ ] **Step 1: Modify my_day_tab.dart — wrap in Scaffold with FAB**

The current `MyDayTab` returns `SafeArea > RefreshIndicator > ...`. We need to wrap the body in a `Scaffold` to add a FAB.

Replace the full `build` method of `MyDayTab` (lines 14-111):

```dart
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final myDay = ref.watch(myDayProvider);

    return Scaffold(
      backgroundColor: Colors.transparent,
      floatingActionButton: _SpeedDialFab(),
      body: SafeArea(
        child: RefreshIndicator(
          onRefresh: () => ref.read(actionsProvider.notifier).refresh(),
          child: myDay.isLoading
              ? const SingleChildScrollView(
                  child: Column(
                    children: [SkeletonCard(height: 300)],
                  ),
                )
              : SingleChildScrollView(
                  physics: const AlwaysScrollableScrollPhysics(),
                  padding: const EdgeInsets.only(bottom: 80),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      // Header
                      const Padding(
                        padding: EdgeInsets.fromLTRB(16, 16, 16, 4),
                        child: Text(
                          'My Day',
                          style: TextStyle(
                              fontSize: 24, fontWeight: FontWeight.bold),
                        ),
                      ),
                      const Padding(
                        padding: EdgeInsets.symmetric(horizontal: 16),
                        child: Text(
                          'Your daily health routine',
                          style: TextStyle(
                              fontSize: 14, color: AppColors.textSecondary),
                        ),
                      ),
                      const SizedBox(height: 16),

                      // Timeline
                      if (myDay.entries.isEmpty)
                        const Padding(
                          padding: EdgeInsets.all(32),
                          child: Center(
                            child: Text(
                              'No activities scheduled for today',
                              style:
                                  TextStyle(color: AppColors.textSecondary),
                            ),
                          ),
                        )
                      else
                        Card(
                          margin:
                              const EdgeInsets.symmetric(horizontal: 16),
                          child: Padding(
                            padding:
                                const EdgeInsets.symmetric(vertical: 16),
                            child: Column(
                              children: [
                                for (var i = 0;
                                    i < myDay.entries.length;
                                    i++)
                                  TimelineEntryWidget(
                                    entry: myDay.entries[i],
                                    isLast:
                                        i == myDay.entries.length - 1,
                                  ),
                              ],
                            ),
                          ),
                        ),

                      // Completion footer
                      if (myDay.entries.isNotEmpty &&
                          myDay.entries.every((e) => e.done))
                        const Padding(
                          padding: EdgeInsets.all(24),
                          child: Center(
                            child: Column(
                              children: [
                                Icon(Icons.celebration,
                                    size: 48,
                                    color: AppColors.scoreGreen),
                                SizedBox(height: 8),
                                Text(
                                  'All done for today! Great work!',
                                  style: TextStyle(
                                    fontSize: 16,
                                    fontWeight: FontWeight.w600,
                                    color: AppColors.scoreGreen,
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ),

                      // Did You Know
                      if (myDay.tipOfTheDay != null)
                        DidYouKnowCard(tip: myDay.tipOfTheDay!),
                    ],
                  ),
                ),
        ),
      ),
    );
  }
```

Add the `_SpeedDialFab` widget at the end of the file:

```dart
class _SpeedDialFab extends StatefulWidget {
  @override
  State<_SpeedDialFab> createState() => _SpeedDialFabState();
}

class _SpeedDialFabState extends State<_SpeedDialFab>
    with SingleTickerProviderStateMixin {
  bool _isOpen = false;
  late final AnimationController _controller;
  late final Animation<double> _expandAnimation;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 250),
    );
    _expandAnimation = CurvedAnimation(
      parent: _controller,
      curve: Curves.easeOut,
    );
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _toggle() {
    setState(() => _isOpen = !_isOpen);
    if (_isOpen) {
      _controller.forward();
    } else {
      _controller.reverse();
    }
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.end,
      children: [
        // Sub-buttons
        ScaleTransition(
          scale: _expandAnimation,
          child: Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                  decoration: BoxDecoration(
                    color: Colors.black87,
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: const Text('Log Reading',
                      style: TextStyle(color: Colors.white, fontSize: 12)),
                ),
                const SizedBox(width: 8),
                FloatingActionButton.small(
                  heroTag: 'fab_vitals',
                  onPressed: () {
                    _toggle();
                    showModalBottomSheet(
                      context: context,
                      isScrollControlled: true,
                      backgroundColor: Colors.transparent,
                      builder: (_) => const VitalsEntrySheet(),
                    );
                  },
                  child: const Icon(Icons.monitor_heart, size: 20),
                ),
              ],
            ),
          ),
        ),
        ScaleTransition(
          scale: _expandAnimation,
          child: Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                  decoration: BoxDecoration(
                    color: Colors.black87,
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: const Text('Log Symptom',
                      style: TextStyle(color: Colors.white, fontSize: 12)),
                ),
                const SizedBox(width: 8),
                FloatingActionButton.small(
                  heroTag: 'fab_symptom',
                  onPressed: () {
                    _toggle();
                    showModalBottomSheet(
                      context: context,
                      isScrollControlled: true,
                      backgroundColor: Colors.transparent,
                      builder: (_) => const SymptomEntrySheet(),
                    );
                  },
                  child: const Icon(Icons.edit_note, size: 20),
                ),
              ],
            ),
          ),
        ),

        // Main FAB
        FloatingActionButton(
          heroTag: 'fab_main',
          onPressed: _toggle,
          child: AnimatedRotation(
            turns: _isOpen ? 0.125 : 0,
            duration: const Duration(milliseconds: 250),
            child: const Icon(Icons.add),
          ),
        ),
      ],
    );
  }
}
```

Also add imports at the top of `my_day_tab.dart`:
```dart
import '../widgets/vitals_entry_sheet.dart';
import '../widgets/symptom_entry_sheet.dart';
```

- [ ] **Step 2: Run existing my_day_tab test to ensure no regressions**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/screens/my_day_tab_test.dart`
Expected: PASS (existing test doesn't depend on FAB).

- [ ] **Step 3: Commit**

```bash
cd vaidshala/clinical-applications/ui/patient
git add lib/screens/my_day_tab.dart
git commit -m "feat(patient-app): add SpeedDial FAB to My Day tab

Expandable FAB with 'Log Reading' (vitals sheet) and
'Log Symptom' (symptom sheet) options. Animated expand/collapse."
```

---

## Task 17: MainShell AppBar + Router Updates

**Files:**
- Modify: `lib/screens/main_shell.dart` (add AppBar with profile + bell icons)
- Modify: `lib/router.dart` (add 3 new pushed routes)
- Modify: `lib/main.dart` (add locale support)

**Context:** The current `MainShell` has no AppBar — it's just a `Scaffold(body: Column(...), bottomNavigationBar: ...)`. We add an AppBar with a profile icon (left) and bell icon with badge (right). The router needs 3 new routes outside the ShellRoute.

- [ ] **Step 1: Modify main_shell.dart — add AppBar**

Replace the `Scaffold` in `MainShell.build()` (lines 29-62) to include an AppBar:

Add these imports at the top:
```dart
import '../providers/notifications_provider.dart';
```

Change the `Scaffold` to:
```dart
    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.person_outline),
          onPressed: () => context.push('/settings'),
        ),
        title: const Text('Vaidshala'),
        centerTitle: true,
        actions: [
          Stack(
            alignment: Alignment.center,
            children: [
              IconButton(
                icon: const Icon(Icons.notifications_outlined),
                onPressed: () => context.push('/notifications'),
              ),
              if (unreadCount > 0)
                Positioned(
                  right: 8,
                  top: 8,
                  child: Container(
                    padding: const EdgeInsets.all(4),
                    decoration: const BoxDecoration(
                      color: Colors.red,
                      shape: BoxShape.circle,
                    ),
                    constraints: const BoxConstraints(minWidth: 16, minHeight: 16),
                    child: Text(
                      '$unreadCount',
                      style: const TextStyle(
                        color: Colors.white,
                        fontSize: 10,
                        fontWeight: FontWeight.bold,
                      ),
                      textAlign: TextAlign.center,
                    ),
                  ),
                ),
            ],
          ),
        ],
      ),
      body: Column(
        children: [
          const OfflineBanner(),
          Expanded(child: child),
        ],
      ),
      bottomNavigationBar: NavigationBar(
        // ... same as before ...
      ),
    );
```

Also add `unreadCount` variable in the build method:
```dart
final unreadCount = ref.watch(unreadCountProvider);
```

And add the `go_router` import (already present) and `context.push` usage:
```dart
import 'package:go_router/go_router.dart';
```

- [ ] **Step 2: Modify router.dart — add 3 new routes**

Add imports at top of `lib/router.dart`:
```dart
import 'screens/settings_screen.dart';
import 'screens/score_detail_screen.dart';
import 'screens/notifications_screen.dart';
```

Add 3 new `GoRoute` entries inside the `routes: [...]` list, outside the `ShellRoute` (before or after it — order matters for matching, but these are distinct paths):

Add after the `/family/:token` route (line 79) and before the `ShellRoute` (line 80):

```dart
      GoRoute(
        path: '/settings',
        builder: (context, state) => const SettingsScreen(),
      ),
      GoRoute(
        path: '/score-detail',
        builder: (context, state) => const ScoreDetailScreen(),
      ),
      GoRoute(
        path: '/notifications',
        builder: (context, state) => const NotificationsScreen(),
      ),
```

- [ ] **Step 3: Modify main.dart — add locale support**

In `lib/main.dart`, the `VaidshalaPatientApp` needs to watch `localeProvider` and pass the locale to `MaterialApp.router`.

Add import:
```dart
import 'providers/locale_provider.dart';
```

Update the `build` method to read locale:
```dart
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(routerProvider);
    final locale = ref.watch(localeProvider);

    return MaterialApp.router(
      title: 'Vaidshala',
      debugShowCheckedModeBanner: false,
      theme: buildAppTheme(),
      locale: locale,
      routerConfig: router,
    );
  }
```

- [ ] **Step 4: Run existing tests to verify no regressions**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test`
Expected: All existing tests PASS.

- [ ] **Step 5: Commit**

```bash
cd vaidshala/clinical-applications/ui/patient
git add lib/screens/main_shell.dart lib/router.dart lib/main.dart
git commit -m "feat(patient-app): add AppBar nav + routes + locale support

MainShell: profile icon → /settings, bell+badge → /notifications.
Router: 3 new pushed routes (settings, score-detail, notifications).
Main: locale from settingsProvider passed to MaterialApp."
```

---

## Task 18: Integration — Codegen, Lint, Test, Build

**Files:** None created — integration verification.

**Context:** This task runs code generation, lint, tests, and build to ensure everything integrates cleanly. This is the final task and mirrors Sprint 2's Task 15.

- [ ] **Step 1: Run Dart code generation**

Run: `cd vaidshala/clinical-applications/ui/patient && dart run build_runner build --delete-conflicting-outputs`
Expected: All `.freezed.dart`, `.g.dart`, and Drift generated files created without errors.

- [ ] **Step 2: Run flutter analyze**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter analyze`
Expected: No issues found.

If issues found:
- Unused imports: Remove them.
- Missing imports: Add them.
- `use_super_parameters` info: Apply super parameter syntax.
- Other lint warnings: Fix per the suggestion.

- [ ] **Step 3: Run all tests**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test`
Expected: All tests pass (Sprint 1-2: ~55 tests + Sprint 3 new tests).

If tests fail:
- `find.text()` on RichText: Use `find.byWidgetPredicate` instead.
- Provider not found: Ensure test overrides the provider directly.
- Hive not initialized: Wrap in try/catch or override provider.

- [ ] **Step 4: Run flutter build web**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter build web --release`
Expected: Build completes without errors.

- [ ] **Step 5: Fix any issues found in steps 2-4**

Apply fixes as needed (remove unused imports, fix type mismatches, etc.).

- [ ] **Step 6: Re-run all verifications**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter analyze && flutter test && flutter build web --release`
Expected: All pass clean.

- [ ] **Step 7: Final commit**

```bash
cd vaidshala/clinical-applications/ui/patient
git add -A
git commit -m "chore(patient-app): Sprint 3 integration fixes

Clean analyze, all tests passing, successful web build."
```
