# CRED-Level Motion Design — Vaidshala Patient App

> **Goal:** Transform the Vaidshala Patient App from a clean Material 3 MVP into a premium, CRED-inspired experience where every interaction feels alive — spring physics, staggered entrances, glassmorphic sheets, count-up numbers — while keeping the clinical light theme.

> **Scope:** Full app retrofit. All 9 screens (Home, My Day, Progress, Settings, Score Detail, Notifications, Add Vitals, Medication Adherence, Symptom Logger) plus shell (AppBar, bottom nav, FAB).

> **Approach:** Custom animation system built from Flutter primitives. Single new dependency: `google_fonts` for Poppins. Zero animation packages — full control over spring curves and timing.

> **Target:** Chrome (Flutter web)

---

## 1. Animation Toolkit (`lib/widgets/animations/`)

### 1.1 Motion Constants (`lib/theme/motion.dart`)

Central file for all animation parameters. Every animation in the app references these constants — never hardcoded durations or curves.

| Constant | Value | Purpose |
|----------|-------|---------|
| `kCreditSpring` | `SpringDescription(mass: 1, stiffness: 300, damping: 22)` | Primary spring for tap feedback |
| `kDecelerate` | `Cubic(0.25, 0.46, 0.45, 0.94)` | CRED-style soft decelerate for entrances |
| `kStaggerDelay` | `80ms` | Delay between staggered items |
| `kEntranceDuration` | `400ms` | Slide-up + fade entrance |
| `kSpringScaleMin` | `0.96` | Card press-down scale |
| `kCountUpDuration` | `800ms` | Number count-up |
| `kProgressFillDuration` | `600ms` | Progress bar fill |
| `kPageTransitionDuration` | `300ms` | Route transitions |
| `kGlassBlurSigma` | `20.0` | Glassmorphic blur radius |
| `kGlassOpacity` | `0.15` | Glassmorphic fill opacity |
| `kSheetBorderRadius` | `24.0` | Bottom sheet top corners |
| `kPulseDuration` | `2000ms` | Pulsing badge cycle |
| `kPulseScale` | `1.05` | Pulsing badge max scale |
| `kSlideOffset` | `30.0` | Pixels to slide up on entrance |

### 1.2 Animation Widgets

#### `FadeSlideTransition`
- **Purpose:** Building block — combines `FadeTransition` + `SlideTransition`
- **Props:** `animation` (Animation<double>), `slideOffset` (default 30px up), `child`
- **Usage:** Internal to `StaggeredItem`, but also standalone for one-off fade-slides

#### `StaggeredItem`
- **Purpose:** Wraps any child widget. On first build, slides up 30px + fades in over 400ms, delayed by `index × 80ms`
- **Props:** `index` (int), `child` (Widget), `duration` (default 400ms), `delay` (default 80ms per index)
- **Behavior:** Uses `AnimationController` + `CurvedAnimation` with `kDecelerate`. The delay is computed as `index * delay` via a `Future.delayed` that triggers `controller.forward()`. Disposes controller properly.
- **State:** `AutomaticKeepAliveClientMixin` to prevent re-animation on tab switch

#### `StaggeredColumn`
- **Purpose:** Drop-in replacement for `Column` that auto-wraps each child in `StaggeredItem` with sequential indices
- **Props:** Same as `Column` (mainAxisAlignment, crossAxisAlignment, children) + `staggerDelay` (default 80ms)
- **Behavior:** Maps `children.asMap()` to wrap each child: `StaggeredItem(index: i, child: child)`

#### `SpringTapCard`
- **Purpose:** Card that responds to tap with spring-physics scale bounce
- **Props:** `child`, `onTap`, `borderRadius` (default 12), `elevation` (default 1)
- **Behavior:**
  - `onTapDown`: Animate scale to `kSpringScaleMin` (0.96) using `SpringSimulation`
  - `onTapUp` / `onTapCancel`: Animate scale back to 1.0 using spring (overshoot bounce)
  - Wraps child in `Transform.scale` driven by spring animation
  - Applies `Material` with elevation and borderRadius for card appearance
- **Feel:** The spring overshoot (damping 0.6) creates a subtle bounce-back — the signature CRED "alive" feeling

#### `GlassmorphicContainer`
- **Purpose:** Frosted glass effect for bottom sheets and overlays
- **Props:** `child`, `borderRadius` (default 24 top), `blurSigma` (default 20), `opacity` (default 0.15), `borderColor` (default white30)
- **Behavior:** `ClipRRect` → `BackdropFilter(filter: ImageFilter.blur)` → semi-transparent `Container` with subtle white border
- **Note:** On Flutter web, `BackdropFilter` performance is acceptable for sheets (not full-screen). Falls back gracefully to solid color if blur not supported.

#### `CountUpText`
- **Purpose:** Animates a number from 0 → target value on first build
- **Props:** `value` (double), `duration` (default 800ms), `style` (TextStyle), `suffix` (String, e.g., "%"), `decimalPlaces` (default 0)
- **Behavior:** `AnimationController` driving `Tween<double>(begin: 0, end: value)` with `kDecelerate` curve. Builds `Text('${animatedValue.toStringAsFixed(decimalPlaces)}$suffix')`
- **Smart:** Only re-animates when `value` changes (compare in `didUpdateWidget`)

#### `AnimatedProgressBar`
- **Purpose:** Progress bar that fills left→right on first build
- **Props:** `value` (0.0–1.0), `duration` (default 600ms), `color`, `backgroundColor`, `height` (default 6), `borderRadius` (default 3)
- **Behavior:** `AnimationController` + `kDecelerate` drives width from 0% → value%. Uses `FractionallySizedBox` inside a `Stack` with `ClipRRect` for rounded corners.
- **Gradient option:** Optional `gradient` prop for premium feel (e.g., green→teal)

#### `PulsingWidget`
- **Purpose:** Infinite subtle scale pulse for attention-grabbing elements
- **Props:** `child`, `duration` (default 2000ms), `minScale` (default 1.0), `maxScale` (default 1.05)
- **Behavior:** `AnimationController` with `repeat(reverse: true)`. `Transform.scale` driven by `Tween(begin: minScale, end: maxScale)` with `Curves.easeInOut`.
- **Usage:** Notification badge, alert indicators

#### `ShakeWidget`
- **Purpose:** Horizontal shake for validation errors
- **Props:** `child`, `shakeCount` (default 3), `shakeOffset` (default 6px)
- **Behavior:** Exposes a `shake()` method via `GlobalKey<ShakeWidgetState>`. When called, runs a quick sinusoidal horizontal translation (3 oscillations over 400ms). Uses `sin(controller.value * pi * shakeCount)`.

### 1.3 Page Transitions

Custom `GoRouter` page builder using `CustomTransitionPage`:
- **Default routes** (`/settings`, `/notifications`): Fade-through transition (300ms, `kDecelerate`)
- **Score Detail** (`/score-detail`): Keep existing Hero animation, add vertical shared-axis feel (slight slide-up + fade)
- **Bottom sheets** (`VitalsEntrySheet`, `SymptomLoggerSheet`): Standard `showModalBottomSheet` but with `GlassmorphicContainer` as the sheet background and barrier color `Colors.black54`

---

## 2. Typography System

### 2.1 Font: Poppins via `google_fonts`

Add `google_fonts: ^6.1.0` to `pubspec.yaml`. Poppins is geometric, modern, and the closest widely-available font to CRED's custom typeface.

### 2.2 Type Scale

Update `buildAppTheme()` in `lib/theme.dart` to use Poppins for all text styles:

| Role | Weight | Size | Letter Spacing | Usage |
|------|--------|------|----------------|-------|
| `displayLarge` | 700 (Bold) | 32px | -0.5 | Score number on detail screen |
| `headlineLarge` | 700 (Bold) | 28px | -0.3 | Screen titles ("Namaste, Rajesh") |
| `headlineMedium` | 600 (SemiBold) | 24px | -0.2 | Section headers |
| `titleLarge` | 600 (SemiBold) | 20px | 0 | Card titles |
| `titleMedium` | 600 (SemiBold) | 16px | 0.1 | Sub-headers, settings group titles |
| `bodyLarge` | 400 (Regular) | 16px | 0.2 | Primary body text |
| `bodyMedium` | 400 (Regular) | 14px | 0.2 | Secondary body text |
| `bodySmall` | 400 (Regular) | 12px | 0.3 | Timestamps, metadata |
| `labelLarge` | 500 (Medium) | 14px | 0.5 | Button text |
| `labelSmall` | 500 (Medium) | 11px | 0.5 | Badges, chips, captions |

**Key design choice:** Negative letter-spacing on headlines creates the "tight, premium" feel. Positive spacing on body/labels preserves readability.

### 2.3 Theme Integration

- `GoogleFonts.poppinsTextTheme()` as base, then override individual styles
- Keep existing `AppColors` palette unchanged
- Card elevation stays at 1 (the motion design provides depth, not shadows)
- Border radius stays at 12dp for cards, 24dp for sheets

---

## 3. Screen-by-Screen Application

### 3.1 Shell (`main_shell.dart`)

| Element | Current | Upgrade |
|---------|---------|---------|
| AppBar | Static, flat | Elevation transitions 0→2 on scroll via `ScrollController` listener. "Vaidshala" title uses `headlineMedium` Poppins. |
| Bottom Nav | Standard `NavigationBar` | Active tab icon wrapped in `TweenAnimationBuilder` scale 1.0→1.2 with spring. Inactive icons at 1.0. Label uses `labelSmall` Poppins. |
| Notification badge | Static dot | Wrapped in `PulsingWidget`. Badge count uses `CountUpText`. |
| FAB (Speed Dial) | ScaleTransition, 250ms | Spring curves from `kCreditSpring`. Sub-buttons wrapped in `StaggeredItem(index: 0)`, `StaggeredItem(index: 1)`. Glassmorphic scrim on open. |

### 3.2 Home Tab (`home_tab.dart`)

| Element | Animation |
|---------|-----------|
| Greeting section | `StaggeredItem(index: 0)` |
| Score card | `StaggeredItem(index: 1)` + `SpringTapCard` wrapper. Score text becomes `CountUpText`. Keep existing animated ring. `GestureDetector` for push to `/score-detail`. |
| Health driver bars | `StaggeredItem(index: 2)`. Replace `LinearProgressIndicator` with `AnimatedProgressBar`. |
| Coaching card | `StaggeredItem(index: 3)`. Left border animates width 0→3px via `TweenAnimationBuilder`. |
| Actions section | `StaggeredItem(index: 4)`. Checkbox toggle gets `SpringTapCard` bounce. Expand/collapse chevron keeps `AnimatedRotation`. |
| Offline banner | Keep current `AnimatedContainer` slide-down. |

### 3.3 My Day Tab (`my_day_tab.dart`)

| Element | Animation |
|---------|-----------|
| Date header | `StaggeredItem(index: 0)` |
| Timeline items | `StaggeredColumn`. Completed checkmarks scale in 0→1 with spring. |
| Medication cards | Each wrapped in `SpringTapCard`. |
| Speed Dial FAB | Shared with shell — spring curves + glassmorphic scrim. |

### 3.4 Progress Tab (`progress_tab.dart`)

| Element | Animation |
|---------|-----------|
| Milestone cards | `StaggeredColumn`. Ring percentages use `CountUpText`. |
| Sparkline chart | Chart clip-path animates left→right over 800ms (use `fl_chart`'s `FlClipData` or animated `maxX`). |
| Lab trend cards | `StaggeredItem` + `SpringTapCard`. |
| Medication Adherence section | Adherence ring uses `CountUpText` for "85%". Streak rows stagger in. `AnimatedProgressBar` for overall bar. |

### 3.5 Settings Screen (`settings_screen.dart`) — S11

| Element | Animation |
|---------|-----------|
| Settings groups | `StaggeredColumn` — each group slides in. |
| Setting tiles | `SpringTapCard` on each tile. |
| Language selector | Dropdown options slide in with `FadeSlideTransition`. |
| Toggle switches | Keep Material `Switch` but add haptic-style visual flash (brief opacity pulse on toggle). |
| Logout button | `SpringTapCard` with red accent. |

### 3.6 Score Detail Screen (`score_detail_screen.dart`) — S12

| Element | Animation |
|---------|-----------|
| Score ring | Hero transition (existing). Ring value becomes `CountUpText` after Hero completes. |
| Full sparkline chart | Line draws itself left→right (800ms, `kDecelerate`). |
| Domain breakdown bars | Each bar width animates 0→value% with `AnimatedProgressBar`, staggered 100ms apart. |
| Explanation card | `StaggeredItem` — last element to appear, subtle fade-slide. |

### 3.7 Notifications Screen (`notifications_screen.dart`) — S13

| Element | Animation |
|---------|-----------|
| Date group headers | `FadeSlideTransition` (fade + slight slide-down). |
| Notification items | `StaggeredColumn` within each group. Items slide in from right (slide-left + fade) — unique direction for notifications. |
| Swipe to dismiss | Keep `Dismissible` but add spring snap-back on cancelled swipe (DismissDirection with `movementDuration`). |
| Unread dot | `PulsingWidget` — single pulse on first appear, then static. |
| Mark All Read | Button press triggers all unread dots to scale down simultaneously (batch animation). |

### 3.8 Add Vitals Sheet (`vitals_entry_sheet.dart`) — S14

| Element | Animation |
|---------|-----------|
| Sheet background | `GlassmorphicContainer` wrapping the sheet content. |
| Tab bar (BP/Glucose/Weight) | Keep Material `TabBar`. Active indicator slides with spring. |
| Input fields | `StaggeredColumn` within each tab — fields appear one by one. |
| Validation errors | `ShakeWidget` on invalid fields (horizontal shake on submit attempt). |
| Save button | `SpringTapCard`. On success: brief green flash + sheet dismisses with slide-down. |

### 3.9 Medication Adherence (`medication_adherence_section.dart`) — S15

| Element | Animation |
|---------|-----------|
| Adherence ring | `CountUpText` for the percentage. Ring fill animates like score ring. |
| Streak rows | `StaggeredColumn`. Each row uses `AnimatedProgressBar` for the streak bar. |
| Missed dose items | Slide in from left (contrasts with right-side streaks) with subtle red-tinted entrance. |

### 3.10 Symptom Logger Sheet (`symptom_logger_sheet.dart`) — S16

| Element | Animation |
|---------|-----------|
| Sheet background | `GlassmorphicContainer`. |
| Symptom chip grid | Chips stagger in left→right, top→bottom (grid pattern). `StaggeredItem` with index computed as `row * cols + col`. |
| Chip selection | `SpringTapCard` scale bounce on tap. Selected chip border animates with `AnimatedContainer`. |
| Severity selector | Options slide between with `AnimatedSwitcher` + horizontal slide. |
| Notes field | `FadeSlideTransition` — appears after severity selected. |
| Save button | Same as Vitals — `SpringTapCard` + success flash. |

---

## 4. File Structure

### New Files (create)

```
lib/theme/motion.dart                          # Motion constants
lib/widgets/animations/fade_slide_transition.dart
lib/widgets/animations/staggered_item.dart
lib/widgets/animations/staggered_column.dart
lib/widgets/animations/spring_tap_card.dart
lib/widgets/animations/glassmorphic_container.dart
lib/widgets/animations/count_up_text.dart
lib/widgets/animations/animated_progress_bar.dart
lib/widgets/animations/pulsing_widget.dart
lib/widgets/animations/shake_widget.dart
```

### Modified Files (update)

```
pubspec.yaml                                   # Add google_fonts dependency
lib/theme.dart                                 # Poppins type scale
lib/router.dart                                # Custom page transitions
lib/screens/main_shell.dart                    # Shell animations
lib/screens/home_tab.dart                      # Staggered + spring cards
lib/screens/my_day_tab.dart                    # Staggered timeline
lib/screens/progress_tab.dart                  # Count-up + animated bars
lib/screens/settings_screen.dart               # Staggered + spring tiles
lib/screens/score_detail_screen.dart           # Domain bars + chart draw
lib/screens/notifications_screen.dart          # Slide-from-right + pulse
lib/widgets/vitals_entry_sheet.dart            # Glassmorphic + shake
lib/widgets/symptom_logger_sheet.dart          # Glassmorphic + chip grid stagger
lib/widgets/medication_adherence_section.dart   # Count-up + stagger
lib/widgets/adherence_ring.dart                # CountUpText integration
lib/widgets/full_sparkline_chart.dart          # Animated line draw
lib/widgets/domain_breakdown_bar.dart          # AnimatedProgressBar swap
lib/widgets/notification_item.dart             # Slide-right entrance
lib/widgets/bp_entry_card.dart                 # ShakeWidget on validation
lib/widgets/glucose_entry_card.dart            # ShakeWidget on validation
lib/widgets/weight_entry_card.dart             # ShakeWidget on validation
lib/widgets/settings_tile.dart                 # SpringTapCard wrap
lib/widgets/settings_group.dart                # StaggeredItem wrap
lib/widgets/speed_dial_fab.dart                # Spring + glassmorphic scrim (if extracted)
```

### Total: 9 new files, ~22 modified files

---

## 5. Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `google_fonts` | ^6.1.0 | Poppins typeface |

No other new packages. All animations built from Flutter primitives (`AnimationController`, `SpringSimulation`, `BackdropFilter`, `CurvedAnimation`).

---

## 6. Testing Strategy

- **Animation widgets:** Unit tests verify controller lifecycle (init, forward, dispose). Verify `StaggeredItem` computes correct delay. Verify `CountUpText` re-animates on value change.
- **SpringTapCard:** Widget test with `tester.press()` verifying scale transform changes.
- **GlassmorphicContainer:** Widget test verifying `BackdropFilter` is in tree.
- **Screen integration:** Existing screen tests continue to pass — animation widgets wrap existing content without changing structure.
- **Visual regression:** Manual Chrome testing for smooth 60fps on all screens.

---

## 7. Performance Considerations

- **Stagger limit:** Max 8 staggered items per screen (640ms total entrance). Beyond 8, items appear instantly.
- **Spring simulation:** `SpringSimulation` is computed per frame — lightweight on web.
- **BackdropFilter on web:** Acceptable for bottom sheets (small area). Not used full-screen. Falls back to solid color on low-end devices.
- **AnimationController disposal:** Every widget with a controller uses `SingleTickerProviderStateMixin` and disposes in `dispose()`.
- **AutomaticKeepAlive:** `StaggeredItem` uses keepalive to prevent re-animation on tab switch.

---

## 8. Success Criteria

1. Every tappable element responds with spring-physics feedback
2. Every screen's content staggers in — no "pop" of fully-loaded content
3. Numbers animate to their values (scores, percentages, counts)
4. Progress bars fill on first appear
5. Bottom sheets have frosted glass backdrop
6. Typography is Poppins throughout with premium letter-spacing
7. Page transitions are fade-through, not default slide
8. Notification badge pulses subtly
9. 60fps on Chrome — no jank during animations
10. All existing tests continue to pass
