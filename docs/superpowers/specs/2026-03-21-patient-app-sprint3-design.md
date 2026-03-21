# Vaidshala Patient App — Sprint 3 Design Spec

**Date:** 2026-03-21
**Status:** Approved
**Scope:** 6 screens merging spec S11-S13 with new data entry capabilities S14-S16
**Location:** `vaidshala/clinical-applications/ui/patient/`
**Tech Stack:** Flutter 3.41.5, Dart 3.11, Riverpod, go_router, Hive, Drift, Freezed, fl_chart
**Builds On:** Sprint 1 (S01-S06) + Sprint 2 (S04, S07-S10)
**Mock Data:** Rajesh Kumar (MRI 82, FBG 178, HbA1c 8.9, SBP 156/98, eGFR 58, steps 2100, protein 32g, BMI 29.4, waist 101cm, weight 82.5kg)

---

## Screen Inventory

| ID | Screen | Route / Trigger | Category |
|----|--------|-----------------|----------|
| S11 | Settings | `/settings` (pushed, no bottom nav) | Spec |
| S12 | Score Detail | `/score-detail` (pushed, Hero animation) | Spec |
| S13 | Notification Center | `/notifications` (pushed) | Spec |
| S14 | Add Vitals | Bottom sheet from FAB on My Day tab | New |
| S15 | Medication Adherence | Section added to Progress tab | New |
| S16 | Symptom Logger | Bottom sheet from FAB on My Day tab | New |

**Design decisions:**
- S14 and S16 are modal bottom sheets triggered by a SpeedDial FAB on My Day tab — not full-screen routes
- S15 is not a standalone screen — it enhances the existing Progress tab with a medication adherence section
- S12 reuses existing `healthScoreProvider` data; domain breakdown uses mock data until backend exposes per-domain scores
- S11 follows the spec doc (VaidshalaPatientApp_ScreenGuide.docx Section S11) exactly

---

## S11: Settings Screen

**Route:** `/settings` — pushed on top of shell (no bottom nav). Auth guard required. Accessed from profile icon on MainShell AppBar.

### Layout

ListView with 5 grouped sections:

1. **Account**
   - Name: "Rajesh Kumar" (read-only, from auth state)
   - Phone: "+91 98765 43210" (read-only)
   - ABHA: status tile showing "Linked" with PHR address, or "Not Linked" → tap navigates to `/abha-verify` (S04)

2. **Preferences**
   - Language: dropdown selector (English, Hindi, Tamil, Telugu, Kannada, Malayalam, Bengali, Marathi)
   - Notifications: toggle switch (on/off)

3. **Family**
   - "Share Health Plan" button → generates mock share URL + displays QR code placeholder
   - Shows existing share link if previously generated

4. **Data**
   - "Download My Data" → mock action (shows "Coming soon" snackbar in Sprint 3)
   - "Delete Account" → double-confirmation dialog with warning text

5. **About**
   - App version (from pubspec.yaml)
   - Terms of Service (mock link)
   - Privacy Policy (mock link)

### State Management

- `settingsProvider` — `StateNotifier<SettingsState>` reading/writing Hive `preferences` box
- `SettingsState` fields: `language` (String), `notificationsEnabled` (bool)
- Language change updates `localeProvider` (new `StateProvider<Locale>`) which triggers `MaterialApp` rebuild
- Sprint 3 scope: language change updates the provider but full i18n (ARB files) is deferred

### Widgets

| Widget | File | Props |
|--------|------|-------|
| `SettingsGroup` | `widgets/settings_group.dart` | `title` (String), `children` (List<Widget>) |
| `SettingsTile` | `widgets/settings_tile.dart` | `icon` (IconData), `title` (String), `trailing` (Widget?), `onTap` (VoidCallback?) |
| `LanguageSelector` | `widgets/language_selector.dart` | `current` (String), `onChanged` (callback) |
| `FamilyShareButton` | `widgets/family_share_button.dart` | `onShare` (callback) |

### Mock Data (Rajesh Kumar)

```
name: "Rajesh Kumar"
phone: "+91 98765 43210"
abhaStatus: linked
phrAddress: "rajesh.kumar@abdm"
language: "en"
notificationsEnabled: true
```

---

## S12: Score Detail Screen

**Route:** `/score-detail` — pushed on top of shell. Auth guard. Hero animation on ScoreRing from Home tab.

### Layout

SingleChildScrollView, top to bottom:

1. **Large ScoreRing** (180px) — Hero-wrapped, shared element transition from Home tab's 120px ring. Score: 18 (patient score = 100 - MRI 82). Color: red zone (<40).

2. **12-week sparkline** — fl_chart AreaChart, full width, 120px height. Gradient fill below line. Dot marker on last point. Dashed reference line at score 60 ("target zone"). X-axis: week labels. Y-axis: 0-100.

3. **Domain breakdown** — 4 horizontal animated bars:
   - Blood Sugar: 35/100 (target 60) — red zone
   - Activity: 22/100 (target 50) — red zone
   - Body Health: 58/100 (target 70) — yellow zone
   - Heart Health: 72/100 (target 80) — green zone

4. **Explanation card** — Patient-friendly text explaining what the score means and what drives it.

### State Management

- Reuses `healthScoreProvider` (no new API call)
- New `DomainScore` Freezed model: `name`, `score` (int), `target` (int), `icon` (String)
- Domain breakdown is mock data in Sprint 3 — returned from `scoreDetailProvider` which wraps `healthScoreProvider` and adds domain mocks
- `scoreHistoryProvider` — derives 12-week history from `healthScoreProvider.scoreHistory`

### Widgets

| Widget | File | Props |
|--------|------|-------|
| `FullSparklineChart` | `widgets/full_sparkline_chart.dart` | `data` (List<double>), `targetLine` (double?), `height` (double) |
| `DomainBreakdownBar` | `widgets/domain_breakdown_bar.dart` | `label` (String), `score` (int), `target` (int), `icon` (IconData), `color` (Color) |
| `ScoreExplanationCard` | `widgets/score_explanation_card.dart` | `score` (int), `trend` (String) |

### Mock Data (Rajesh Kumar)

```
patientScore: 18  (100 - MRI 82)
scoreHistory: [25, 28, 24, 22, 20, 19, 18, 18, 19, 17, 18, 18]  (12 weeks)
domains:
  - Blood Sugar: 35 (target 60)
  - Activity: 22 (target 50)
  - Body Health: 58 (target 70)
  - Heart Health: 72 (target 80)
explanation: "Your metabolic health score reflects how well your blood sugar, activity levels, body composition, and heart health markers are tracking against clinical targets. Focus on the areas with the biggest gaps to see the most improvement."
```

---

## S13: Notification Center

**Route:** `/notifications` — pushed on top of shell. Auth guard. Accessed from bell icon on MainShell AppBar.

### Layout

ListView grouped by date ("Today", "Yesterday", "This Week"):
- Each notification row: type icon (left) + title + body + relative timestamp (center) + unread blue dot (right)
- Tap: navigates to relevant screen via deep link stored in notification
- Swipe left: Dismissible to delete
- Empty state: illustration + "No notifications yet"

### State Management

- `notificationsProvider` — `AsyncNotifier<List<AppNotification>>` reading from Drift `notifications` table
- `unreadCountProvider` — `Provider<int>` derived from `notificationsProvider`, drives badge on bell icon
- `AppNotification` Freezed model: `id`, `type` (enum: coaching/reminder/alert/milestone), `title`, `body`, `deepLink`, `timestamp` (DateTime), `read` (bool)
- Sprint 3: notifications are seeded as mock data in Drift on first load. Real FCM integration deferred.

### Drift Table

```sql
notifications (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,        -- coaching, reminder, alert, milestone
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  deep_link TEXT,
  timestamp INTEGER NOT NULL,
  read INTEGER NOT NULL DEFAULT 0
)
```

### Widgets

| Widget | File | Props |
|--------|------|-------|
| `NotificationDateGroup` | `widgets/notification_date_group.dart` | `label` (String), `items` (List<AppNotification>) |
| `NotificationItem` | `widgets/notification_item.dart` | `notification` (AppNotification), `onTap`, `onDismiss` |

### Mock Data (Rajesh Kumar)

```
notifications:
  - id: "n1", type: coaching, title: "Great progress!", body: "Your FBG dropped 12 mg/dL this week", deepLink: "/home/progress", timestamp: today 09:00, read: true
  - id: "n2", type: alert, title: "FBG trending down", body: "Your fasting glucose is moving toward target", deepLink: "/home/progress", timestamp: today 08:00, read: false
  - id: "n3", type: reminder, title: "Time for evening walk", body: "A 15-min post-dinner walk can lower glucose by 15-20%", deepLink: "/home/my-day", timestamp: today 19:00, read: false
  - id: "n4", type: coaching, title: "Weekly progress summary", body: "You completed 85% of actions this week", deepLink: "/home/progress", timestamp: yesterday 20:00, read: true
  - id: "n5", type: milestone, title: "New health tip available", body: "Learn about protein's role in metabolic health", deepLink: "/home/learn", timestamp: 3 days ago, read: true
```

---

## S14: Add Vitals (Quick Entry Bottom Sheet)

**Trigger:** SpeedDial FAB on My Day tab → "Log Reading" option → opens modal bottom sheet.

### Layout

Modal bottom sheet with 3 independent quick-entry cards stacked vertically:

1. **Blood Pressure Card**
   - Two TextFormFields side by side: Systolic (hint: "156") / Diastolic (hint: "98")
   - Input: number keyboard, range validation (systolic 60-250, diastolic 40-150)
   - Validation: systolic must be > diastolic
   - Unit label: "mmHg"
   - "Save" button per card

2. **Blood Glucose Card**
   - One TextFormField (hint: "178") + dropdown: Fasting / Post-meal / Random
   - Input: number keyboard, range 20-600 mg/dL
   - Context dropdown required before save
   - "Save" button per card

3. **Weight Card**
   - One TextFormField (hint: "82.5")
   - Input: decimal keyboard, range 20-300 kg
   - "Save" button per card

Each card saves independently to Drift `observation_queue` → shows success snackbar → resets form fields.

### State Management

- `vitalsEntryProvider` — `StateNotifier<VitalsEntryState>` managing form values + validation errors for all 3 cards
- `VitalsEntryState` fields: `systolic`, `diastolic`, `glucoseValue`, `glucoseContext`, `weight`, `errors` (Map<String, String?>)
- On save: writes to Drift `observation_queue`, invalidates relevant cached providers (e.g., `progressProvider` for latest readings)

### Drift Table

```sql
observation_queue (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,          -- bp, glucose, weight
  value TEXT NOT NULL,         -- JSON: {"systolic":156,"diastolic":98} or {"value":178,"context":"fasting"} or {"value":82.5}
  unit TEXT NOT NULL,          -- mmHg, mg/dL, kg
  timestamp INTEGER NOT NULL,
  synced INTEGER NOT NULL DEFAULT 0
)
```

### Freezed Model

```dart
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
}
```

### Widgets

| Widget | File | Props |
|--------|------|-------|
| `VitalsEntrySheet` | `widgets/vitals_entry_sheet.dart` | Container bottom sheet |
| `BpEntryCard` | `widgets/bp_entry_card.dart` | `onSave` (callback) |
| `GlucoseEntryCard` | `widgets/glucose_entry_card.dart` | `onSave` (callback) |
| `WeightEntryCard` | `widgets/weight_entry_card.dart` | `onSave` (callback) |

### Validation Rules

| Field | Rule | Error Message |
|-------|------|---------------|
| Systolic | 60-250, required | "Enter a value between 60 and 250" |
| Diastolic | 40-150, required | "Enter a value between 40 and 150" |
| Systolic > Diastolic | Cross-field | "Systolic must be higher than diastolic" |
| Glucose | 20-600, required | "Enter a value between 20 and 600" |
| Glucose context | Required | "Select when this was measured" |
| Weight | 20-300, required | "Enter a value between 20 and 300" |

---

## S15: Medication Adherence (Progress Tab Enhancement)

**Not a standalone screen.** Adds a "Medication Adherence" section to the existing `progress_tab.dart`, below the Milestones section.

### Layout

New section in Progress tab:

1. **Section header:** "Medication Adherence"
2. **Weekly adherence bar:** circular percentage indicator + "This week: 85% — 6 of 7 days all meds taken"
3. **Per-medication streak list:**
   - "Metformin 1000mg BD — 12-day streak" (green check icon)
   - "Glimepiride 2mg OD — 8-day streak" (green check icon)
   - "Telmisartan 40mg OD — 14-day streak" (green check icon)
4. **Missed dose indicator:** "Last missed: Metformin PM, 2 days ago" (amber warning)

### State Management

- `medicationAdherenceProvider` — `FutureProvider<MedicationAdherence>` reading from Drift `medication_log` table
- `MedicationAdherence` Freezed model: `weeklyPct` (int), `streaks` (List<MedStreak>), `lastMissed` (MissedDose?)
- `MedStreak`: `medicationName` (String), `streakDays` (int)
- `MissedDose`: `medicationName` (String), `daysAgo` (int)

### Enhancement to existing code

- `ActionsNotifier.toggleAction()` enhanced: when toggling a medication-type action, also writes to `medication_log` table
- Detection: action icons matching "medication" are medication-type actions

### Drift Table

```sql
medication_log (
  id TEXT PRIMARY KEY,
  action_id TEXT NOT NULL,
  medication_name TEXT NOT NULL,
  completed INTEGER NOT NULL,    -- 1 = taken, 0 = missed/skipped
  timestamp INTEGER NOT NULL
)
```

### Widgets

| Widget | File | Props |
|--------|------|-------|
| `MedicationAdherenceCard` | `widgets/medication_adherence_card.dart` | `adherence` (MedicationAdherence) |
| `MedicationStreakRow` | `widgets/medication_streak_row.dart` | `name` (String), `streakDays` (int) |

### Mock Data (Rajesh Kumar — 14-day history)

```
weeklyPct: 85
streaks:
  - Metformin 1000mg BD: 12 days
  - Glimepiride 2mg OD: 8 days
  - Telmisartan 40mg OD: 14 days
lastMissed: Metformin PM, 2 days ago
```

---

## S16: Symptom Logger (Bottom Sheet)

**Trigger:** SpeedDial FAB on My Day tab → "Log Symptom" option → opens modal bottom sheet.

### Layout

Modal bottom sheet:

1. **Symptom grid** — 7 tappable chips (multi-select allowed):
   - Dizziness, Nausea, Fatigue, Chest Pain, Swelling, Breathlessness, Low Sugar Feeling
   - Selected chips get teal fill + check icon

2. **Severity selector** — 3 radio buttons:
   - Mild (green) / Moderate (yellow) / Severe (red)
   - Required — must select one before save

3. **Free text** — "Tell us more (optional)" TextFormField, max 200 chars, multi-line

4. **Time selector** — Defaults to "Now". Tap to pick earlier time today (TimePicker constrained to today only)

5. **Save button** — Full width, disabled until symptom + severity selected

### State Management

- `symptomEntryProvider` — `StateNotifier<SymptomEntryState>`
- `SymptomEntryState`: `selectedSymptoms` (Set<String>), `severity` (String?), `notes` (String), `timestamp` (DateTime)
- On save: writes to Drift `symptom_log`, shows success snackbar, resets form

### Drift Table

```sql
symptom_log (
  id TEXT PRIMARY KEY,
  symptom TEXT NOT NULL,         -- comma-separated if multiple: "dizziness,fatigue"
  severity TEXT NOT NULL,        -- mild, moderate, severe
  notes TEXT,                    -- optional free text, max 200 chars
  timestamp INTEGER NOT NULL,
  synced INTEGER NOT NULL DEFAULT 0
)
```

### Freezed Model

```dart
@freezed
class SymptomEntry with _$SymptomEntry {
  const factory SymptomEntry({
    required String id,
    required String symptom,
    required String severity,
    String? notes,
    required DateTime timestamp,
    @Default(false) bool synced,
  }) = _SymptomEntry;
}
```

### Widgets

| Widget | File | Props |
|--------|------|-------|
| `SymptomEntrySheet` | `widgets/symptom_entry_sheet.dart` | Container bottom sheet |
| `SymptomChip` | `widgets/symptom_chip.dart` | `label` (String), `selected` (bool), `onTap` |
| `SeveritySelector` | `widgets/severity_selector.dart` | `value` (String?), `onChanged` (callback) |

---

## Router & Navigation Changes

### New Routes

```dart
GoRoute(path: '/settings', builder: ... => SettingsScreen()),
GoRoute(path: '/score-detail', builder: ... => ScoreDetailScreen()),
GoRoute(path: '/notifications', builder: ... => NotificationsScreen()),
```

All 3 are pushed routes (outside ShellRoute) with auth guard. No new routes for S14/S16 (bottom sheets).

### MainShell AppBar Changes

- **Left:** Profile icon → navigates to `/settings`
- **Right:** Bell icon with badge (unread count from `unreadCountProvider`) → navigates to `/notifications`

### My Day Tab FAB

- `FloatingActionButton` replaced with SpeedDial (or expandable FAB):
  - Option 1: "Log Reading" → opens `VitalsEntrySheet`
  - Option 2: "Log Symptom" → opens `SymptomEntrySheet`

---

## New Drift Tables Summary

| Table | Purpose | Key Columns |
|-------|---------|-------------|
| `observation_queue` | Vitals entries (BP, glucose, weight) | id, type, value (JSON), unit, timestamp, synced |
| `medication_log` | Medication adherence tracking | id, actionId, medicationName, completed, timestamp |
| `symptom_log` | Symptom reports | id, symptom, severity, notes, timestamp, synced |
| `notifications` | Notification history | id, type, title, body, deepLink, timestamp, read |

All 4 tables added to existing `drift_database.dart` alongside Sprint 2's `checkin_queue` and `lab_history`.

---

## New Freezed Models

| Model | File | Fields |
|-------|------|--------|
| `DomainScore` | `models/domain_score.dart` | name, score, target, icon |
| `AppNotification` | `models/app_notification.dart` | id, type, title, body, deepLink, timestamp, read |
| `VitalEntry` | `models/vital_entry.dart` | id, type, value, unit, timestamp, synced |
| `SymptomEntry` | `models/symptom_entry.dart` | id, symptom, severity, notes, timestamp, synced |
| `MedicationAdherence` | `models/medication_adherence.dart` | weeklyPct, streaks, lastMissed |
| `MedStreak` | `models/medication_adherence.dart` | medicationName, streakDays |
| `MissedDose` | `models/medication_adherence.dart` | medicationName, daysAgo |
| `SettingsState` | `models/settings_state.dart` | language, notificationsEnabled |

---

## New Providers

| Provider | Type | Source | Consumed By |
|----------|------|--------|-------------|
| `settingsProvider` | `StateNotifier<SettingsState>` | Hive preferences | S11 |
| `localeProvider` | `StateProvider<Locale>` | From settingsProvider | MaterialApp |
| `scoreDetailProvider` | `Provider<ScoreDetailState>` | healthScoreProvider + mock domains | S12 |
| `notificationsProvider` | `AsyncNotifier<List<AppNotification>>` | Drift notifications table | S13 |
| `unreadCountProvider` | `Provider<int>` | Derived from notificationsProvider | MainShell badge |
| `vitalsEntryProvider` | `StateNotifier<VitalsEntryState>` | Form state | S14 |
| `medicationAdherenceProvider` | `FutureProvider<MedicationAdherence>` | Drift medication_log | S15 |
| `symptomEntryProvider` | `StateNotifier<SymptomEntryState>` | Form state | S16 |

---

## Files Created / Modified Summary

### New Files (~30)

**Models (8):** domain_score.dart, app_notification.dart, vital_entry.dart, symptom_entry.dart, medication_adherence.dart, settings_state.dart (+ codegen files)

**Providers (6):** settings_provider.dart, locale_provider.dart, score_detail_provider.dart, notifications_provider.dart, vitals_entry_provider.dart, symptom_entry_provider.dart, medication_adherence_provider.dart

**Screens (3):** settings_screen.dart, score_detail_screen.dart, notifications_screen.dart

**Widgets (14):** settings_group.dart, settings_tile.dart, language_selector.dart, family_share_button.dart, full_sparkline_chart.dart, domain_breakdown_bar.dart, score_explanation_card.dart, notification_date_group.dart, notification_item.dart, vitals_entry_sheet.dart, bp_entry_card.dart, glucose_entry_card.dart, weight_entry_card.dart, symptom_entry_sheet.dart, symptom_chip.dart, severity_selector.dart, medication_adherence_card.dart, medication_streak_row.dart

### Modified Files (~4)

- `lib/services/drift_database.dart` — add 4 new tables
- `lib/screens/my_day_tab.dart` — add SpeedDial FAB
- `lib/screens/main_shell.dart` — add AppBar icons (profile, bell with badge)
- `lib/screens/progress_tab.dart` — add medication adherence section
- `lib/router.dart` — add 3 new routes
