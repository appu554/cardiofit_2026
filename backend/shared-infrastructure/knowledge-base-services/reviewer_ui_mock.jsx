import { useState } from "react";

// ── Mock Data ────────────────────────────────────────────────────────
const FACTS = [
  {
    id: "a26c19fc",
    tier: 1,
    section: "1.2",
    sectionTitle: "RAS Inhibitors",
    page: 33,
    channels: ["C", "F"],
    corroboration: 0.7,
    text: 'ACEi should be discontinued if serum creatinine rises >30% from baseline within the first 2 months of initiation.',
    numerics: [">30%", "2 months"],
    conditions: ["if", "within"],
    negation: null,
    alert: null,
    sourceSnippet: "ACEi should be discontinued if serum creatinine rises >30% from baseline within the first 2 months of initiation. Monitor serum potassium and creatinine within 2–4 weeks of initiating or increasing the dose of ACEi or ARB therapy.",
  },
  {
    id: "21a0cea1",
    tier: 1,
    section: "1.4",
    sectionTitle: "Albuminuria Management",
    page: 36,
    channels: ["C"],
    corroboration: 0.6,
    text: 'Urine albumin-to-creatinine ratio ><3 mg should be confirmed on repeat testing.',
    numerics: [""><3 mg""],
    conditions: ["should"],
    negation: null,
    alert: {
      type: "numeric",
      label: "NUMERIC MISMATCH",
      source: ">3 mg/mmol",
      extracted: "><3 mg",
      detail: "Comparator garbled during extraction. Source clearly reads >3 mg/mmol."
    },
    sourceSnippet: "Urine albumin-to-creatinine ratio >3 mg/mmol (>30 mg/g) should be confirmed on repeat testing, ideally using an early morning sample.",
  },
  {
    id: "bf27c28a",
    tier: 1,
    section: "1.3",
    sectionTitle: "ACEi/ARB Use in Pregnancy",
    page: 34,
    channels: ["F"],
    corroboration: 0.4,
    text: 'Women of child-bearing age treated with ACEi or ARBs should be counseled regarding the risks of these medications during pregnancy and the need for contraception.',
    numerics: [],
    conditions: ["should"],
    negation: null,
    alert: {
      type: "llm_only",
      label: "LLM-ONLY EXTRACTION",
      detail: "Found only by Channel F (NuExtract). No deterministic channel corroboration. Manual verification against source required."
    },
    sourceSnippet: "Because of teratogenic effects, women of child-bearing age treated with ACEi or ARBs should be counseled appropriately regarding the risks of these medications during pregnancy and the need for effective contraception.",
  },
  {
    id: "74512f93",
    tier: 1,
    section: "2.2",
    sectionTitle: "Glycemic Targets",
    page: 58,
    channels: ["B", "C", "F"],
    corroboration: 0.9,
    text: 'We suggest an individualized HbA1c target ranging from <6.5% to <8.0% in patients with diabetes and CKD not treated with dialysis.',
    numerics: ["<6.5%", "<8.0%"],
    conditions: ["not treated with"],
    negation: ["not treated with dialysis"],
    alert: {
      type: "branch_loss",
      label: "BRANCH INCOMPLETE",
      source_thresholds: 4,
      extracted_thresholds: 2,
      detail: "Source has 4 HbA1c thresholds (6.5%, 7.0%, 7.5%, 8.0%) with population-specific branches. Only 2 extracted."
    },
    sourceSnippet: "We suggest an individualized HbA1c target ranging from <6.5% to <8.0% in patients with diabetes and CKD not treated with dialysis (2C).\n\nPractice Point 2.2.2: In patients with diabetes and CKD who are treated with insulin or sulfonylurea, a higher HbA1c target (e.g., <8.0%) may be appropriate to reduce hypoglycemia risk.",
  },
  {
    id: "e6cc75fb",
    tier: 1,
    section: "3.1",
    sectionTitle: "MRA Therapy",
    page: 49,
    channels: ["E", "F"],
    corroboration: 0.4,
    text: 'We suggest using a nonsteroidal mineralocorticoid receptor antagonist with proven kidney or cardiovascular benefit for patients with type 2 diabetes, an eGFR ≥25 ml/min per 1.73 m², and serum potassium ≤5.0 mEq/l.',
    numerics: ["≥25 ml/min", "1.73 m²", "≤5.0 mEq/l"],
    conditions: ["with", "and"],
    negation: null,
    alert: null,
    sourceSnippet: "Recommendation 3.1.1: We suggest using a nonsteroidal mineralocorticoid receptor antagonist with proven kidney or cardiovascular benefit for patients with type 2 diabetes, an eGFR ≥25 ml/min per 1.73 m², normal serum potassium concentration, and albuminuria (≥30 mg/g [≥3 mg/mmol]) despite maximum tolerated dose of RAS inhibitor (2A).",
  },
];

// ── Utility ──────────────────────────────────────────────────────────
function highlightText(text, numerics, conditions, negation) {
  if (!text) return text;
  let segments = [{ text, type: "normal" }];

  const splitOn = (segs, patterns, type) => {
    const result = [];
    for (const seg of segs) {
      if (seg.type !== "normal") { result.push(seg); continue; }
      let remaining = seg.text;
      for (const pat of patterns) {
        const idx = remaining.toLowerCase().indexOf(pat.toLowerCase());
        if (idx !== -1) {
          if (idx > 0) result.push({ text: remaining.slice(0, idx), type: "normal" });
          result.push({ text: remaining.slice(idx, idx + pat.length), type });
          remaining = remaining.slice(idx + pat.length);
        }
      }
      if (remaining) result.push({ text: remaining, type: "normal" });
    }
    return result;
  };

  if (negation && negation.length) segments = splitOn(segments, negation, "negation");
  if (numerics && numerics.length) segments = splitOn(segments, numerics, "numeric");
  if (conditions && conditions.length) segments = splitOn(segments, conditions, "condition");
  return segments;
}

// ── Components ───────────────────────────────────────────────────────
const COLORS = {
  bg: "#F7F8FA",
  surface: "#FFFFFF",
  navy: "#1B3A5C",
  navyLight: "#2A5580",
  text: "#1A1A2E",
  textMuted: "#6B7280",
  border: "#E2E6EC",
  borderFocus: "#3B82F6",
  numericBg: "#DBEAFE",
  numericText: "#1E40AF",
  condBg: "#FEF3C7",
  condText: "#92400E",
  negBg: "#FEE2E2",
  negText: "#991B1B",
  confirm: "#059669",
  confirmHover: "#047857",
  edit: "#2563EB",
  editHover: "#1D4ED8",
  reject: "#DC2626",
  rejectHover: "#B91C1C",
  add: "#7C3AED",
  addHover: "#6D28D9",
  escalate: "#D97706",
  escalateHover: "#B45309",
  alertRedBg: "#FEF2F2",
  alertRedBorder: "#FECACA",
  alertAmberBg: "#FFFBEB",
  alertAmberBorder: "#FDE68A",
  alertBlueBg: "#EFF6FF",
  alertBlueBorder: "#BFDBFE",
  pdfBg: "#F9FAFB",
  highlightYellow: "#FEF08A",
};

function TopBar({ current, total, verdict, blocksRemaining }) {
  return (
    <div style={{
      height: 52, background: COLORS.navy, display: "flex", alignItems: "center",
      justifyContent: "space-between", padding: "0 24px", flexShrink: 0,
    }}>
      <div style={{ display: "flex", alignItems: "center", gap: 20 }}>
        <span style={{ color: "#fff", fontSize: 14, fontWeight: 600, letterSpacing: 0.3 }}>
          KDIGO 2022 — Diabetes in CKD
        </span>
        <span style={{ color: "rgba(255,255,255,0.5)", fontSize: 12 }}>|</span>
        <span style={{ color: "rgba(255,255,255,0.6)", fontSize: 12 }}>
          Job: dfdb5212
        </span>
      </div>
      <div style={{ display: "flex", alignItems: "center", gap: 16 }}>
        <span style={{
          fontSize: 11, fontWeight: 600, letterSpacing: 0.5,
          padding: "4px 10px", borderRadius: 4,
          background: verdict === "PASS" ? "#065F46" : "#7F1D1D",
          color: verdict === "PASS" ? "#A7F3D0" : "#FECACA",
        }}>
          {verdict === "PASS" ? "✓ PASS" : `⊘ BLOCK (${blocksRemaining})`}
        </span>
        <button style={{
          background: "rgba(255,255,255,0.1)", border: "1px solid rgba(255,255,255,0.2)",
          color: "#fff", fontSize: 11, padding: "5px 12px", borderRadius: 4, cursor: "pointer",
          fontWeight: 500,
        }}>Re-Validate</button>
        <button style={{
          background: "rgba(255,255,255,0.05)", border: "1px solid rgba(255,255,255,0.15)",
          color: "rgba(255,255,255,0.7)", fontSize: 11, padding: "5px 12px", borderRadius: 4, cursor: "pointer",
        }}>Deep Re-Validate</button>
      </div>
    </div>
  );
}

function AlertBanner({ alert }) {
  if (!alert) return null;
  const configs = {
    numeric: { bg: COLORS.alertRedBg, border: COLORS.alertRedBorder, icon: "⚠", color: "#991B1B" },
    branch_loss: { bg: COLORS.alertAmberBg, border: COLORS.alertAmberBorder, icon: "⚠", color: "#92400E" },
    llm_only: { bg: COLORS.alertBlueBg, border: COLORS.alertBlueBorder, icon: "◎", color: "#1E40AF" },
  };
  const c = configs[alert.type] || configs.llm_only;
  return (
    <div style={{
      background: c.bg, border: `1px solid ${c.border}`, borderRadius: 6,
      padding: "10px 14px", marginBottom: 16,
      borderLeft: `3px solid ${c.color}`,
    }}>
      <div style={{ fontSize: 11, fontWeight: 700, color: c.color, letterSpacing: 0.5, marginBottom: 4 }}>
        {c.icon} {alert.label}
      </div>
      {alert.type === "numeric" && (
        <div style={{ fontSize: 12, color: COLORS.text, lineHeight: 1.5 }}>
          <span style={{ color: COLORS.textMuted }}>Source: </span>
          <span style={{ fontWeight: 600, fontFamily: "monospace" }}>{alert.source}</span>
          <span style={{ color: COLORS.textMuted, margin: "0 8px" }}>→</span>
          <span style={{ color: COLORS.textMuted }}>Extracted: </span>
          <span style={{ fontWeight: 600, fontFamily: "monospace", color: "#DC2626", textDecoration: "line-through" }}>{alert.extracted}</span>
          <div style={{ marginTop: 4, fontSize: 11, color: COLORS.textMuted }}>{alert.detail}</div>
        </div>
      )}
      {alert.type === "branch_loss" && (
        <div style={{ fontSize: 12, color: COLORS.text, lineHeight: 1.5 }}>
          <span style={{ color: COLORS.textMuted }}>Source thresholds: </span>
          <span style={{ fontWeight: 600 }}>{alert.source_thresholds}</span>
          <span style={{ color: COLORS.textMuted, margin: "0 8px" }}>→</span>
          <span style={{ color: COLORS.textMuted }}>Extracted: </span>
          <span style={{ fontWeight: 600, color: "#DC2626" }}>{alert.extracted_thresholds}</span>
          <div style={{ marginTop: 4, fontSize: 11, color: COLORS.textMuted }}>{alert.detail}</div>
        </div>
      )}
      {alert.type === "llm_only" && (
        <div style={{ fontSize: 12, color: COLORS.text }}>{alert.detail}</div>
      )}
    </div>
  );
}

function FactCard({ fact }) {
  const segments = highlightText(fact.text, fact.numerics, fact.conditions, fact.negation);
  const channelLabel = fact.channels.join(" + ");
  const scoreColor = fact.corroboration >= 0.7 ? "#059669" : fact.corroboration >= 0.5 ? "#D97706" : "#DC2626";
  const scoreBg = fact.corroboration >= 0.7 ? "#ECFDF5" : fact.corroboration >= 0.5 ? "#FFFBEB" : "#FEF2F2";

  return (
    <div style={{ flex: 1, display: "flex", flexDirection: "column" }}>
      {/* Meta row */}
      <div style={{
        display: "flex", alignItems: "center", gap: 8, marginBottom: 12, flexWrap: "wrap",
      }}>
        <span style={{
          background: "#7F1D1D", color: "#FECACA", fontSize: 10, fontWeight: 700,
          padding: "2px 8px", borderRadius: 3, letterSpacing: 0.6,
        }}>TIER 1</span>
        <span style={{
          background: "#F3F4F6", color: COLORS.textMuted, fontSize: 11,
          padding: "2px 8px", borderRadius: 3,
        }}>§ {fact.section} — {fact.sectionTitle}</span>
        <span style={{
          background: "#F3F4F6", color: COLORS.textMuted, fontSize: 11,
          padding: "2px 8px", borderRadius: 3,
        }}>p. {fact.page}</span>
        <span style={{
          background: "#EEF2FF", color: "#4338CA", fontSize: 11,
          padding: "2px 8px", borderRadius: 3, fontWeight: 500,
        }}>Ch {channelLabel}</span>
        <span style={{
          background: scoreBg, color: scoreColor, fontSize: 11,
          padding: "2px 8px", borderRadius: 3, fontWeight: 600,
        }}>{fact.corroboration.toFixed(1)}</span>
      </div>

      {/* Alert */}
      <AlertBanner alert={fact.alert} />

      {/* Fact text */}
      <div style={{
        background: COLORS.surface, border: `1px solid ${COLORS.border}`, borderRadius: 8,
        padding: 20, lineHeight: 1.75, fontSize: 15, color: COLORS.text,
        fontFamily: "'Charter', 'Georgia', serif",
        marginBottom: 16, flex: 1,
      }}>
        {segments.map((seg, i) => {
          if (seg.type === "numeric") return (
            <span key={i} style={{
              background: COLORS.numericBg, color: COLORS.numericText,
              padding: "1px 4px", borderRadius: 3, fontWeight: 600,
              fontFamily: "monospace", fontSize: 14,
            }}>{seg.text}</span>
          );
          if (seg.type === "condition") return (
            <span key={i} style={{
              background: COLORS.condBg, color: COLORS.condText,
              padding: "1px 4px", borderRadius: 3, fontWeight: 500,
              fontStyle: "italic", fontSize: 14,
            }}>{seg.text}</span>
          );
          if (seg.type === "negation") return (
            <span key={i} style={{
              background: COLORS.negBg, color: COLORS.negText,
              padding: "1px 5px", borderRadius: 3, fontWeight: 700,
              textDecoration: "underline", textDecorationStyle: "wavy",
              textDecorationColor: "#EF4444", fontSize: 14,
            }}>{seg.text}</span>
          );
          return <span key={i}>{seg.text}</span>;
        })}
      </div>

      {/* Token legend */}
      <div style={{
        display: "flex", gap: 16, fontSize: 10, color: COLORS.textMuted, marginBottom: 20,
        letterSpacing: 0.3,
      }}>
        <span><span style={{ background: COLORS.numericBg, padding: "1px 5px", borderRadius: 2, color: COLORS.numericText, fontWeight: 600 }}>123</span> Numeric</span>
        <span><span style={{ background: COLORS.condBg, padding: "1px 5px", borderRadius: 2, color: COLORS.condText, fontStyle: "italic" }}>if</span> Condition</span>
        <span><span style={{ background: COLORS.negBg, padding: "1px 5px", borderRadius: 2, color: COLORS.negText, fontWeight: 600, textDecoration: "underline wavy #EF4444" }}>not</span> Negation</span>
      </div>

      {/* Notes */}
      <textarea
        placeholder="Reviewer notes (optional)..."
        style={{
          width: "100%", height: 48, border: `1px solid ${COLORS.border}`, borderRadius: 6,
          padding: "8px 12px", fontSize: 12, fontFamily: "inherit", color: COLORS.text,
          resize: "none", background: "#FAFBFC", marginBottom: 16, boxSizing: "border-box",
          outline: "none",
        }}
      />
    </div>
  );
}

function ActionBar({ onAction, alertType }) {
  const mustResolve = alertType === "numeric" || alertType === "branch_loss";
  const btnStyle = (bg, hover, disabled) => ({
    padding: "9px 20px", borderRadius: 6, border: "none", cursor: disabled ? "not-allowed" : "pointer",
    fontSize: 12, fontWeight: 600, color: "#fff", background: disabled ? "#D1D5DB" : bg,
    letterSpacing: 0.3, transition: "background 0.15s", opacity: disabled ? 0.6 : 1,
    display: "flex", alignItems: "center", gap: 6,
  });

  return (
    <div style={{
      display: "flex", justifyContent: "space-between", alignItems: "center",
      paddingTop: 12, borderTop: `1px solid ${COLORS.border}`,
    }}>
      <div style={{ display: "flex", gap: 8 }}>
        <button
          onClick={() => onAction("confirm")}
          disabled={mustResolve}
          title={mustResolve ? "Resolve alert before confirming" : "Confirm (C)"}
          style={btnStyle(COLORS.confirm, COLORS.confirmHover, mustResolve)}
        >
          <span style={{ fontSize: 14 }}>✓</span> Confirm
        </button>
        <button onClick={() => onAction("edit")} style={btnStyle(COLORS.edit, COLORS.editHover, false)}>
          <span style={{ fontSize: 13 }}>✎</span> Edit
        </button>
        <button onClick={() => onAction("reject")} style={btnStyle(COLORS.reject, COLORS.rejectHover, false)}>
          <span style={{ fontSize: 13 }}>✕</span> Reject
        </button>
      </div>
      <div style={{ display: "flex", gap: 8 }}>
        <button onClick={() => onAction("add")} style={btnStyle(COLORS.add, COLORS.addHover, false)}>
          + Add Fact
        </button>
        <button onClick={() => onAction("escalate")} style={btnStyle(COLORS.escalate, COLORS.escalateHover, false)}>
          ⇡ Escalate
        </button>
      </div>
    </div>
  );
}

function PDFPanel({ fact }) {
  const lines = fact.sourceSnippet.split('\n');
  return (
    <div style={{
      flex: 1, background: COLORS.pdfBg, borderLeft: `1px solid ${COLORS.border}`,
      display: "flex", flexDirection: "column", overflow: "hidden",
    }}>
      {/* PDF header */}
      <div style={{
        padding: "10px 20px", borderBottom: `1px solid ${COLORS.border}`,
        display: "flex", justifyContent: "space-between", alignItems: "center",
        background: "#fff", flexShrink: 0,
      }}>
        <span style={{ fontSize: 12, fontWeight: 600, color: COLORS.navy }}>
          Source PDF — Page {fact.page}
        </span>
        <span style={{ fontSize: 10, color: COLORS.textMuted }}>
          Auto-scrolled to match
        </span>
      </div>

      {/* PDF content simulation */}
      <div style={{ flex: 1, padding: 24, overflow: "auto" }}>
        {/* Simulated page */}
        <div style={{
          background: "#fff", border: "1px solid #E5E7EB",
          borderRadius: 2, padding: "40px 36px",
          boxShadow: "0 1px 4px rgba(0,0,0,0.06)",
          maxWidth: 520, margin: "0 auto",
          fontFamily: "'Times New Roman', serif",
          fontSize: 13, lineHeight: 1.8, color: "#222",
        }}>
          {/* Context before */}
          <div style={{ color: "#999", marginBottom: 16 }}>
            <div style={{ fontSize: 10, letterSpacing: 1, marginBottom: 8, fontFamily: "sans-serif" }}>
              CHAPTER {fact.section.split('.')[0]} — {fact.sectionTitle.toUpperCase()}
            </div>
          </div>

          {/* Highlighted source text */}
          {lines.map((line, i) => (
            <p key={i} style={{
              margin: "0 0 12px 0",
              background: i === 0 ? COLORS.highlightYellow : "transparent",
              padding: i === 0 ? "2px 4px" : 0,
              borderRadius: i === 0 ? 2 : 0,
            }}>
              {line}
            </p>
          ))}

          {/* Context after */}
          <div style={{ color: "#bbb", marginTop: 20, fontSize: 12 }}>
            [Additional guideline text continues...]
          </div>
        </div>

        {/* Page number */}
        <div style={{ textAlign: "center", marginTop: 16, fontSize: 11, color: COLORS.textMuted }}>
          — {fact.page} —
        </div>
      </div>
    </div>
  );
}

function RejectModal({ onClose, onConfirm }) {
  const [reason, setReason] = useState("");
  const reasons = [
    "Not present in source",
    "Numeric mismatch",
    "Out of scope",
    "Duplicate",
    "Hallucinated content",
  ];
  return (
    <div style={{
      position: "fixed", inset: 0, background: "rgba(0,0,0,0.4)", display: "flex",
      alignItems: "center", justifyContent: "center", zIndex: 100,
    }}>
      <div style={{
        background: "#fff", borderRadius: 12, padding: 28, width: 380,
        boxShadow: "0 20px 60px rgba(0,0,0,0.15)",
      }}>
        <div style={{ fontSize: 15, fontWeight: 600, color: COLORS.text, marginBottom: 16 }}>
          Reject — Select Reason
        </div>
        <div style={{ display: "flex", flexDirection: "column", gap: 6, marginBottom: 20 }}>
          {reasons.map(r => (
            <label key={r} style={{
              display: "flex", alignItems: "center", gap: 10, padding: "8px 12px",
              borderRadius: 6, cursor: "pointer",
              background: reason === r ? "#FEF2F2" : "#F9FAFB",
              border: `1px solid ${reason === r ? "#FECACA" : "#E5E7EB"}`,
              fontSize: 13, color: COLORS.text,
            }}>
              <input
                type="radio" name="reason" value={r}
                checked={reason === r}
                onChange={() => setReason(r)}
                style={{ accentColor: COLORS.reject }}
              />
              {r}
            </label>
          ))}
        </div>
        <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
          <button onClick={onClose} style={{
            padding: "8px 16px", borderRadius: 6, border: `1px solid ${COLORS.border}`,
            background: "#fff", fontSize: 12, cursor: "pointer", color: COLORS.textMuted,
          }}>Cancel</button>
          <button
            onClick={() => { if (reason) onConfirm(reason); }}
            disabled={!reason}
            style={{
              padding: "8px 16px", borderRadius: 6, border: "none",
              background: reason ? COLORS.reject : "#D1D5DB",
              color: "#fff", fontSize: 12, fontWeight: 600, cursor: reason ? "pointer" : "not-allowed",
            }}
          >Confirm Reject</button>
        </div>
      </div>
    </div>
  );
}

function SummaryView({ decisions }) {
  const counts = { confirm: 0, edit: 0, reject: 0, add: 0, escalate: 0, pending: 0 };
  FACTS.forEach((f, i) => {
    const d = decisions[f.id];
    if (d) counts[d]++;
    else counts.pending++;
  });
  const total = FACTS.length;
  const done = total - counts.pending;

  return (
    <div style={{
      flex: 1, display: "flex", alignItems: "center", justifyContent: "center",
      background: COLORS.bg,
    }}>
      <div style={{
        background: "#fff", borderRadius: 12, padding: 40, width: 440,
        boxShadow: "0 4px 20px rgba(0,0,0,0.06)", border: `1px solid ${COLORS.border}`,
        textAlign: "center",
      }}>
        <div style={{ fontSize: 20, fontWeight: 700, color: COLORS.navy, marginBottom: 8 }}>
          Review Summary
        </div>
        <div style={{ fontSize: 13, color: COLORS.textMuted, marginBottom: 28 }}>
          Tier 1 Facts: {done}/{total} reviewed
        </div>

        <div style={{ display: "flex", flexDirection: "column", gap: 8, marginBottom: 28, textAlign: "left" }}>
          {[
            { label: "Confirmed", count: counts.confirm, color: COLORS.confirm },
            { label: "Edited", count: counts.edit, color: COLORS.edit },
            { label: "Rejected", count: counts.reject, color: COLORS.reject },
            { label: "Added", count: counts.add, color: COLORS.add },
            { label: "Escalated", count: counts.escalate, color: COLORS.escalate },
            { label: "Pending", count: counts.pending, color: COLORS.textMuted },
          ].map(row => (
            <div key={row.label} style={{
              display: "flex", justifyContent: "space-between", alignItems: "center",
              padding: "8px 14px", borderRadius: 6, background: "#F9FAFB",
            }}>
              <span style={{ fontSize: 13, color: COLORS.text }}>{row.label}</span>
              <span style={{
                fontSize: 14, fontWeight: 700, color: row.color,
                minWidth: 28, textAlign: "right",
              }}>{row.count}</span>
            </div>
          ))}
        </div>

        <div style={{
          padding: "12px 16px", borderRadius: 8, marginBottom: 20,
          background: counts.pending === 0 ? "#ECFDF5" : "#FEF2F2",
          border: `1px solid ${counts.pending === 0 ? "#A7F3D0" : "#FECACA"}`,
        }}>
          <span style={{
            fontSize: 12, fontWeight: 600,
            color: counts.pending === 0 ? "#065F46" : "#991B1B",
          }}>
            {counts.pending === 0
              ? "✓ All Tier 1 facts reviewed — ready for Deep Re-Validate"
              : `${counts.pending} facts remaining`}
          </span>
        </div>

        <button style={{
          width: "100%", padding: "11px 0", borderRadius: 8, border: "none",
          background: counts.pending === 0 ? COLORS.navy : "#D1D5DB",
          color: "#fff", fontSize: 13, fontWeight: 600,
          cursor: counts.pending === 0 ? "pointer" : "not-allowed",
        }}>
          Run Final Deep Validation →
        </button>
      </div>
    </div>
  );
}

// ── Main App ─────────────────────────────────────────────────────────
export default function ReviewerUI() {
  const [currentIdx, setCurrentIdx] = useState(0);
  const [decisions, setDecisions] = useState({});
  const [showReject, setShowReject] = useState(false);
  const [showSummary, setShowSummary] = useState(false);

  const fact = FACTS[currentIdx];
  const decidedCount = Object.keys(decisions).length;

  const handleAction = (action) => {
    if (action === "reject") { setShowReject(true); return; }
    setDecisions(d => ({ ...d, [fact.id]: action }));
    if (currentIdx < FACTS.length - 1) setCurrentIdx(i => i + 1);
    else setShowSummary(true);
  };

  const handleReject = (reason) => {
    setDecisions(d => ({ ...d, [fact.id]: "reject" }));
    setShowReject(false);
    if (currentIdx < FACTS.length - 1) setCurrentIdx(i => i + 1);
    else setShowSummary(true);
  };

  const goTo = (dir) => {
    if (dir === "prev" && currentIdx > 0) { setCurrentIdx(i => i - 1); setShowSummary(false); }
    if (dir === "next" && currentIdx < FACTS.length - 1) { setCurrentIdx(i => i + 1); setShowSummary(false); }
    if (dir === "next" && currentIdx === FACTS.length - 1) setShowSummary(true);
  };

  return (
    <div style={{
      height: "100vh", display: "flex", flexDirection: "column",
      fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
      background: COLORS.bg, color: COLORS.text,
    }}>
      <TopBar
        current={currentIdx + 1}
        total={FACTS.length}
        verdict={decidedCount === FACTS.length ? "PASS" : "BLOCK"}
        blocksRemaining={FACTS.length - decidedCount}
      />

      {showSummary ? (
        <SummaryView decisions={decisions} />
      ) : (
        <div style={{ flex: 1, display: "flex", overflow: "hidden" }}>
          {/* LEFT — Fact Review */}
          <div style={{
            width: "48%", display: "flex", flexDirection: "column",
            padding: "20px 24px 16px", overflow: "auto",
          }}>
            <FactCard fact={fact} />
            <ActionBar
              onAction={handleAction}
              alertType={fact.alert?.type}
            />

            {/* Navigation */}
            <div style={{
              display: "flex", justifyContent: "space-between", alignItems: "center",
              marginTop: 14, paddingTop: 10,
            }}>
              <button
                onClick={() => goTo("prev")}
                disabled={currentIdx === 0}
                style={{
                  background: "none", border: `1px solid ${COLORS.border}`, borderRadius: 5,
                  padding: "5px 14px", fontSize: 12, cursor: currentIdx === 0 ? "not-allowed" : "pointer",
                  color: currentIdx === 0 ? "#ccc" : COLORS.textMuted,
                }}
              >◂ Prev</button>
              <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
                <span style={{ fontSize: 12, color: COLORS.textMuted }}>
                  Fact {currentIdx + 1} of {FACTS.length}
                </span>
                <span style={{ fontSize: 10, color: COLORS.textMuted }}>
                  ({decidedCount} decided)
                </span>
              </div>
              <button
                onClick={() => goTo("next")}
                style={{
                  background: "none", border: `1px solid ${COLORS.border}`, borderRadius: 5,
                  padding: "5px 14px", fontSize: 12, cursor: "pointer", color: COLORS.textMuted,
                }}
              >{currentIdx === FACTS.length - 1 ? "Summary ▸" : "Next ▸"}</button>
            </div>

            {/* Keyboard hint */}
            <div style={{
              textAlign: "center", marginTop: 8, fontSize: 10, color: "#C0C4CC",
              letterSpacing: 0.3,
            }}>
              C confirm · E edit · R reject · ← → navigate
            </div>
          </div>

          {/* RIGHT — PDF */}
          <PDFPanel fact={fact} />
        </div>
      )}

      {showReject && (
        <RejectModal
          onClose={() => setShowReject(false)}
          onConfirm={handleReject}
        />
      )}
    </div>
  );
}
