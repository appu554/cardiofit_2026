import { useState, useCallback, useEffect, useRef } from "react";

// ─── DATA: Actual pipeline output from KDIGO 2022 pp.58-61 ───────────────────
const PIPELINE_META = {
  job_id: "job_marker_b3d324b3",
  guideline: "KDIGO 2022 – Diabetes Management in CKD",
  chapter: "Chapter 2: Glycemic Monitoring and Targets",
  pages: "58–61",
  version: "V4.2.1",
  channels: { B: 17, C: 79, D: 42, E: 6, F: 10 },
  raw_spans: 154,
  merged_spans: 143,
  section_passages: 7,
  reparented: 1,
};

const PAGES = [
  {
    num: 1, label: "p.58", title: "Glycemic Monitoring",
    sections: ["2.1.1", "2.1.2", "2.1.3", "2.1.4"],
    risk: "clean",
    highlights: [
      { id: "h1", text: "HbA1c", type: "lab_test", channel: "C", conf: 0.85, x: 12, y: 8, w: 4, count: 14 },
      { id: "h2", text: "Practice Point 2.1.1–2.1.4", type: "practice_point", channel: "F", conf: 0.85, x: 3, y: 5, w: 22, count: 4 },
      { id: "h3", text: "eGFR <30 ml/min", type: "egfr_threshold", channel: "C", conf: 0.95, x: 5, y: 32, w: 12, count: 1 },
      { id: "h4", text: "HbA1c <7% reduces microvascular complications", type: "proposition", channel: "F", conf: 0.90, x: 3, y: 14, w: 38, count: 1 },
      { id: "h5", text: "CKD G1–G3b / G4–G5 monitoring table", type: "table", channel: "D", conf: 0.95, x: 3, y: 78, w: 94, count: 15, isTable: true },
    ],
    content: [
      { type: "pp", id: "2.1.1", text: "Practice Point 2.1.1: Monitoring long-term glycemic control by HbA1c twice per year is reasonable for patients with diabetes. HbA1c may be measured as often as 4 times per year if the glycemic target is not met or after a change in glucose-lowering therapy." },
      { type: "evidence", text: "HbA1c monitoring facilitates control of diabetes to achieve glycemic targets that prevent diabetic complications. In both T1D or T2D, lower achieved levels of HbA1c <7% (<53 mmol/mol) versus 8%–9% reduce risk of overall microvascular complications..." },
      { type: "pp", id: "2.1.2", text: "Practice Point 2.1.2: Accuracy and precision of HbA1c measurement declines with advanced CKD (G4–G5), particularly among patients treated by dialysis, in whom HbA1c measurements have low reliability." },
      { type: "pp", id: "2.1.3", text: "Practice Point 2.1.3: A glucose management indicator (GMI) derived from continuous glucose monitoring (CGM) data can be used to index glycemia for individuals in whom HbA1c is not concordant with directly measured blood glucose levels or clinical symptoms." },
      { type: "pp", id: "2.1.4", text: "Practice Point 2.1.4: Daily glycemic monitoring with CGM or self-monitoring of blood glucose (SMBG) may help prevent hypoglycemia and improve glycemic control when glucose-lowering therapies associated with risk of hypoglycemia are used." },
    ],
  },
  {
    num: 2, label: "p.59", title: "CGM Glossary & Figure 12",
    sections: ["2.1.4_cont", "Glossary"],
    risk: "oracle",
    highlights: [
      { id: "h6", text: "insulin", type: "drug", channel: "B", conf: 1.0, x: 24, y: 59, w: 5, count: 7 },
      { id: "h7", text: "daily", type: "monitoring_freq", channel: "C", conf: 0.85, x: 8, y: 59, w: 3, count: 9 },
      { id: "h8", text: "avoid", type: "contraindication", channel: "E", conf: 0.95, x: 38, y: 65, w: 4, count: 3 },
      { id: "h9", text: "do not use", type: "contraindication", channel: "C", conf: 0.95, x: 5, y: 71, w: 7, count: 1 },
      { id: "h10", text: "Figure 12 – CGM thresholds (70–180 mg/dl)", type: "oracle_recovery", channel: "L1", conf: 0.70, x: 3, y: 20, w: 94, count: 6, isOracle: true },
    ],
    content: [
      { type: "cont", text: "...hypoglycemic medications such as insulin. For daily glycemic monitoring, CGM and SMBG are frequently used but relatively high-cost options..." },
      { type: "figure", id: "12", text: "Figure 12: Glossary of glucose-monitoring terms — Time in range (TIR) 70–180 mg/dl (3.9–10.0 mmol/l) at >70% of readings", isOracle: true },
      { type: "glossary", text: "Self-monitoring of blood glucose (SMBG): Self-sampling of blood via fingerstick for capillary glucose measurement using glucometers." },
    ],
  },
  {
    num: 3, label: "p.60", title: "Drug Classes & Rec 2.2.1",
    sections: ["2.1.5", "2.1.6", "Research", "2.2"],
    risk: "clean",
    highlights: [
      { id: "h11", text: "metformin, SGLT2i, GLP-1 RA, DPP-4i", type: "drug_class", channel: "B", conf: 1.0, x: 5, y: 55, w: 30, count: 8 },
      { id: "h12", text: "Insulin / Sulfonylureas / Meglitinides", type: "drug_class", channel: "B+D", conf: 1.0, x: 5, y: 49, w: 28, count: 5 },
      { id: "h13", text: "Recommendation 2.2.1", type: "recommendation_id", channel: "C", conf: 0.98, x: 3, y: 72, w: 16, count: 1 },
      { id: "h14", text: "We recommend an individualized HbA1c target ranging from <6.5% to <8.0%", type: "recommendation", channel: "F", conf: 0.85, x: 3, y: 73, w: 50, count: 1, isRec: true },
      { id: "h15", text: "Figure 13 – Drug class vs hypoglycemia risk", type: "table", channel: "D", conf: 0.95, x: 3, y: 44, w: 94, count: 6, isTable: true },
    ],
    content: [
      { type: "pp", id: "2.1.5", text: "Practice Point 2.1.5: For patients with T2D and CKD who choose not to do daily glycemic monitoring by CGM or SMBG, glucose-lowering agents that pose a lower risk of hypoglycemia are preferred..." },
      { type: "pp", id: "2.1.6", text: "Practice Point 2.1.6: CGM devices are rapidly evolving with multiple functionalities..." },
      { type: "research", text: "Research recommendations: In patients with T1D or T2D and advanced CKD, especially kidney failure treated by dialysis or kidney transplant, research is needed to..." },
      { type: "rec", id: "2.2.1", grade: "1C", text: "Recommendation 2.2.1: We recommend an individualized HbA1c target ranging from <6.5% to <8.0% in patients with diabetes and CKD not treated with dialysis (1C)." },
    ],
  },
  {
    num: 4, label: "p.61", title: "Key Information (Reparented → 2.2)",
    sections: ["Key_information"],
    risk: "reparented",
    highlights: [
      { id: "h16", text: "HbA1c", type: "lab_test", channel: "C", conf: 0.85, x: 10, y: 12, w: 4, count: 23 },
      { id: "h17", text: "ACCORD trial – mortality higher with lower HbA1c target", type: "proposition", channel: "F", conf: 0.85, x: 3, y: 28, w: 55, count: 1 },
      { id: "h18", text: "U-shaped association of HbA1c with adverse outcomes", type: "proposition", channel: "F", conf: 0.85, x: 3, y: 35, w: 50, count: 1 },
      { id: "h19", text: "eGFR <15 ml/min", type: "egfr_threshold", channel: "C", conf: 0.95, x: 20, y: 8, w: 12, count: 1 },
      { id: "h20", text: "Figure 14 – Individualized HbA1c targets", type: "table", channel: "D", conf: 0.95, x: 3, y: 74, w: 94, count: 12, isTable: true },
    ],
    content: [
      { type: "key_info", text: "Balance of benefits and harms. HbA1c targets are central to guide glucose-lowering treatment. In the general diabetes population, higher HbA1c levels have been associated with increased risk of microvascular and macrovascular complications." },
      { type: "evidence", text: "In the Action to Control Cardiovascular Risk in Diabetes (ACCORD) trial of T2D, mortality was also higher among participants assigned to the lower HbA1c target, perhaps due to hypoglycemia and related cardiovascular events." },
      { type: "evidence", text: "Among patients with diabetes and CKD, a U-shaped association of HbA1c with adverse health outcomes has been observed, suggesting risks with both inadequately controlled blood glucose and excessively lowered blood glucose." },
      { type: "qoe", text: "Quality of evidence. A systematic review with 3 comparisons examining the effects of lower (≤7.0%, ≤6.5%, ≤6.0%) versus higher (standard of care) HbA1c targets in patients with diabetes and CKD was undertaken." },
    ],
  },
];

const CHANNEL_INFO = {
  B: { name: "Drug Dictionary", color: "#3b82f6", icon: "💊" },
  C: { name: "Grammar/Regex", color: "#8b5cf6", icon: "🔬" },
  D: { name: "Table Decomp", color: "#06b6d4", icon: "📊" },
  E: { name: "GLiNER NER", color: "#f59e0b", icon: "🧬" },
  F: { name: "NuExtract LLM", color: "#10b981", icon: "🧠" },
  L1: { name: "L1 Oracle Recovery", color: "#ef4444", icon: "🔴" },
  "B+D": { name: "Drug + Table", color: "#6366f1", icon: "💊📊" },
};

const RISK_CONFIG = {
  clean: { label: "Corroborated", color: "#10b981", bg: "#ecfdf5", icon: "✓" },
  oracle: { label: "Oracle Recovery", color: "#ef4444", bg: "#fef2f2", icon: "!" },
  disagreement: { label: "Disagreement", color: "#f59e0b", bg: "#fffbeb", icon: "?" },
  reparented: { label: "Reparented §", color: "#8b5cf6", bg: "#f5f3ff", icon: "↑" },
};

// ─── COMPONENTS ──────────────────────────────────────────────────────────────

function PageNavigator({ pages, activePage, onSelect, reviewState }) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 2, padding: "0 0 12px 0" }}>
      {pages.map((p) => {
        const risk = RISK_CONFIG[p.risk];
        const isActive = activePage === p.num;
        const state = reviewState[p.num];
        return (
          <button
            key={p.num}
            onClick={() => onSelect(p.num)}
            style={{
              display: "flex", alignItems: "stretch", gap: 0,
              border: isActive ? "2px solid #1e293b" : "1px solid #e2e8f0",
              borderRadius: 8, background: isActive ? "#f8fafc" : "#fff",
              cursor: "pointer", overflow: "hidden", textAlign: "left",
              transition: "all 0.15s ease",
              boxShadow: isActive ? "0 2px 8px rgba(0,0,0,0.08)" : "none",
            }}
          >
            <div style={{
              width: 5, minHeight: "100%",
              background: state === "accepted" ? "#10b981" : state === "flagged" ? "#f59e0b" : risk.color,
              flexShrink: 0,
            }} />
            <div style={{ padding: "10px 12px", flex: 1, minWidth: 0 }}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 3 }}>
                <span style={{
                  fontFamily: "'JetBrains Mono', 'SF Mono', monospace",
                  fontSize: 11, fontWeight: 700, color: "#1e293b", letterSpacing: "0.02em",
                }}>{p.label}</span>
                <span style={{
                  fontSize: 9, fontWeight: 600,
                  padding: "2px 6px", borderRadius: 4,
                  background: risk.bg, color: risk.color,
                  letterSpacing: "0.03em", textTransform: "uppercase",
                }}>
                  {risk.icon} {risk.label}
                </span>
              </div>
              <div style={{ fontSize: 12, color: "#475569", fontWeight: 500, lineHeight: 1.3 }}>{p.title}</div>
              <div style={{ fontSize: 10, color: "#94a3b8", marginTop: 4 }}>
                {p.highlights.length} highlights · {p.sections.join(", ")}
              </div>
            </div>
          </button>
        );
      })}
    </div>
  );
}

function HighlightCard({ h, isSelected, onClick }) {
  const ch = CHANNEL_INFO[h.channel] || CHANNEL_INFO.C;
  return (
    <div
      onClick={() => onClick(h.id)}
      style={{
        padding: "10px 12px", borderRadius: 6, cursor: "pointer",
        border: isSelected ? `2px solid ${ch.color}` : "1px solid #e2e8f0",
        background: isSelected ? `${ch.color}0a` : "#fff",
        transition: "all 0.12s ease",
        marginBottom: 6,
      }}
    >
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", gap: 8 }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{
            fontSize: 12, fontWeight: 600, color: "#1e293b",
            lineHeight: 1.4, marginBottom: 4,
            overflow: "hidden", textOverflow: "ellipsis",
            display: "-webkit-box", WebkitLineClamp: 2, WebkitBoxOrient: "vertical",
          }}>
            {h.isOracle && <span style={{ color: "#ef4444", marginRight: 4 }}>⚠</span>}
            {h.isRec && <span style={{ color: "#3b82f6", marginRight: 4 }}>★</span>}
            {h.text}
          </div>
          <div style={{ display: "flex", gap: 6, flexWrap: "wrap", alignItems: "center" }}>
            <span style={{
              fontSize: 9, fontWeight: 700, padding: "1px 5px",
              borderRadius: 3, background: ch.color, color: "#fff",
              letterSpacing: "0.04em",
            }}>{h.channel}</span>
            <span style={{ fontSize: 10, color: "#64748b" }}>{h.type.replace(/_/g, " ")}</span>
            {h.count > 1 && (
              <span style={{ fontSize: 9, color: "#94a3b8", fontStyle: "italic" }}>×{h.count}</span>
            )}
          </div>
        </div>
        <div style={{
          fontSize: 20, fontWeight: 800,
          fontFamily: "'JetBrains Mono', monospace",
          color: h.conf >= 0.95 ? "#10b981" : h.conf >= 0.85 ? "#f59e0b" : "#ef4444",
          lineHeight: 1, whiteSpace: "nowrap",
        }}>
          {Math.round(h.conf * 100)}
        </div>
      </div>
    </div>
  );
}

function PDFPageViewer({ page, selectedHighlight, onSelectHighlight }) {
  return (
    <div style={{
      background: "#fff", borderRadius: 8, border: "1px solid #e2e8f0",
      overflow: "hidden", height: "100%", display: "flex", flexDirection: "column",
    }}>
      {/* Page header */}
      <div style={{
        padding: "8px 16px", background: "#f8fafc",
        borderBottom: "1px solid #e2e8f0",
        display: "flex", justifyContent: "space-between", alignItems: "center",
      }}>
        <span style={{ fontSize: 12, fontWeight: 600, color: "#1e293b" }}>
          {page.label} — {page.title}
        </span>
        <span style={{ fontSize: 10, color: "#64748b" }}>
          §{page.sections.join(" · §")}
        </span>
      </div>

      {/* Simulated PDF page with highlights */}
      <div style={{
        flex: 1, overflow: "auto", padding: 20,
        background: "#fff",
        fontFamily: "'Source Serif 4', 'Palatino Linotype', 'Book Antiqua', serif",
        fontSize: 12.5, lineHeight: 1.65, color: "#1a1a1a",
      }}>
        {page.content.map((block, i) => {
          const isOracleBlock = block.isOracle;
          const isRec = block.type === "rec";
          const isPP = block.type === "pp";
          const isKeyInfo = block.type === "key_info";
          const isQoE = block.type === "qoe";

          return (
            <div key={i} style={{
              marginBottom: 16, position: "relative",
              padding: isOracleBlock ? "12px 14px" : isRec ? "12px 14px" : isPP ? "10px 14px" : "4px 0",
              background: isOracleBlock ? "#fef2f2" : isRec ? "#eff6ff" : isKeyInfo ? "#f5f3ff" : "transparent",
              borderRadius: isOracleBlock || isRec || isPP || isKeyInfo ? 6 : 0,
              borderLeft: isOracleBlock ? "4px solid #ef4444" : isRec ? "4px solid #3b82f6" : isKeyInfo ? "4px solid #8b5cf6" : isPP ? "3px solid #10b981" : "none",
            }}>
              {/* Block type label */}
              {(isPP || isRec || isOracleBlock || isKeyInfo || isQoE) && (
                <div style={{
                  fontSize: 9, fontWeight: 700, letterSpacing: "0.06em",
                  textTransform: "uppercase", marginBottom: 6,
                  color: isOracleBlock ? "#ef4444" : isRec ? "#3b82f6" : isKeyInfo ? "#7c3aed" : isQoE ? "#6366f1" : "#059669",
                }}>
                  {isOracleBlock ? "⚠ Oracle Recovery — Figure" : isRec ? `★ Recommendation ${block.id} (${block.grade})` : isKeyInfo ? "Key Information (reparented → §2.2)" : isQoE ? "Quality of Evidence" : `Practice Point ${block.id}`}
                </div>
              )}

              {/* Text with simulated highlights */}
              <HighlightedText
                text={block.text}
                pageHighlights={page.highlights}
                selectedHighlight={selectedHighlight}
                onSelectHighlight={onSelectHighlight}
              />
            </div>
          );
        })}
      </div>
    </div>
  );
}

function HighlightedText({ text, pageHighlights, selectedHighlight, onSelectHighlight }) {
  // Find keywords in text and highlight them
  const parts = [];
  let remaining = text;
  let key = 0;

  const keywords = pageHighlights
    .filter(h => h.text.length < 40)
    .map(h => ({ pattern: h.text.split(",")[0].trim(), highlight: h }))
    .sort((a, b) => b.pattern.length - a.pattern.length);

  // Simple keyword highlighting
  const regex = keywords.length > 0
    ? new RegExp(`(${keywords.map(k => k.pattern.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join("|")})`, "gi")
    : null;

  if (regex) {
    const splits = remaining.split(regex);
    for (const segment of splits) {
      const match = keywords.find(k => k.pattern.toLowerCase() === segment.toLowerCase());
      if (match) {
        const ch = CHANNEL_INFO[match.highlight.channel] || CHANNEL_INFO.C;
        const isSelected = selectedHighlight === match.highlight.id;
        parts.push(
          <span
            key={key++}
            onClick={(e) => { e.stopPropagation(); onSelectHighlight(match.highlight.id); }}
            style={{
              background: isSelected ? `${ch.color}40` : `${ch.color}18`,
              borderBottom: `2px solid ${ch.color}`,
              padding: "1px 2px", borderRadius: 2,
              cursor: "pointer", transition: "background 0.12s",
              fontWeight: isSelected ? 600 : "inherit",
            }}
            title={`${match.highlight.channel}: ${match.highlight.type}`}
          >
            {segment}
          </span>
        );
      } else {
        parts.push(<span key={key++}>{segment}</span>);
      }
    }
  } else {
    parts.push(<span key={0}>{text}</span>);
  }

  return <span>{parts}</span>;
}

function SpanInspector({ highlight, onAction }) {
  if (!highlight) {
    return (
      <div style={{
        height: "100%", display: "flex", alignItems: "center", justifyContent: "center",
        flexDirection: "column", gap: 8, color: "#94a3b8",
      }}>
        <div style={{ fontSize: 32, opacity: 0.4 }}>⬅</div>
        <div style={{ fontSize: 12, textAlign: "center", lineHeight: 1.5 }}>
          Click a highlight in the<br />document to inspect it
        </div>
      </div>
    );
  }

  const ch = CHANNEL_INFO[highlight.channel] || CHANNEL_INFO.C;

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", gap: 0 }}>
      {/* Header */}
      <div style={{ padding: "12px 14px", borderBottom: "1px solid #e2e8f0" }}>
        <div style={{
          display: "flex", alignItems: "center", gap: 6, marginBottom: 8,
        }}>
          <span style={{
            fontSize: 10, fontWeight: 700, padding: "2px 6px",
            borderRadius: 4, background: ch.color, color: "#fff",
          }}>{ch.icon} {highlight.channel}</span>
          <span style={{ fontSize: 11, color: "#64748b", fontWeight: 500 }}>{ch.name}</span>
        </div>
        <div style={{
          fontSize: 14, fontWeight: 700, color: "#0f172a",
          lineHeight: 1.4,
        }}>
          {highlight.text}
        </div>
      </div>

      {/* Details */}
      <div style={{ flex: 1, overflow: "auto", padding: "12px 14px" }}>
        <DetailRow label="Type" value={highlight.type.replace(/_/g, " ")} />
        <DetailRow label="Confidence" value={
          <span style={{
            fontFamily: "'JetBrains Mono', monospace", fontWeight: 700,
            color: highlight.conf >= 0.95 ? "#10b981" : highlight.conf >= 0.85 ? "#f59e0b" : "#ef4444",
          }}>
            {(highlight.conf * 100).toFixed(0)}%
          </span>
        } />
        <DetailRow label="Occurrences" value={`${highlight.count} span${highlight.count !== 1 ? "s" : ""}`} />
        <DetailRow label="Channel" value={`${ch.icon} ${ch.name}`} />

        {/* Provenance chain */}
        <div style={{ marginTop: 14, marginBottom: 10 }}>
          <div style={{
            fontSize: 9, fontWeight: 700, letterSpacing: "0.06em",
            textTransform: "uppercase", color: "#94a3b8", marginBottom: 8,
          }}>Provenance Chain</div>
          <div style={{
            background: "#f8fafc", borderRadius: 6, padding: "10px 12px",
            fontFamily: "'JetBrains Mono', monospace", fontSize: 10,
            lineHeight: 1.8, color: "#475569",
          }}>
            <div>section_passage → <span style={{ color: "#3b82f6" }}>merged_span</span></div>
            <div style={{ paddingLeft: 16 }}>→ <span style={{ color: "#8b5cf6" }}>raw_span</span> (ch.{highlight.channel})</div>
            <div style={{ paddingLeft: 16 }}>→ <span style={{ color: "#059669" }}>{highlight.type}</span></div>
          </div>
        </div>

        {/* Corroboration */}
        <div style={{ marginTop: 14 }}>
          <div style={{
            fontSize: 9, fontWeight: 700, letterSpacing: "0.06em",
            textTransform: "uppercase", color: "#94a3b8", marginBottom: 8,
          }}>Channel Corroboration</div>
          {Object.entries(CHANNEL_INFO).filter(([k]) => k !== "L1" && k !== "B+D").map(([k, v]) => {
            const hasIt = k === highlight.channel || (highlight.channel === "B+D" && (k === "B" || k === "D"));
            return (
              <div key={k} style={{
                display: "flex", alignItems: "center", gap: 8, marginBottom: 4,
                opacity: hasIt ? 1 : 0.4,
              }}>
                <span style={{ fontSize: 13, width: 18 }}>{hasIt ? "✔" : "—"}</span>
                <span style={{
                  fontSize: 10, fontWeight: 600,
                  padding: "1px 5px", borderRadius: 3,
                  background: v.color, color: "#fff",
                  minWidth: 16, textAlign: "center",
                }}>{k}</span>
                <span style={{ fontSize: 11, color: hasIt ? "#1e293b" : "#94a3b8" }}>{v.name}</span>
              </div>
            );
          })}
        </div>

        {/* Clinical context warning for oracle */}
        {highlight.isOracle && (
          <div style={{
            marginTop: 16, padding: "10px 12px", borderRadius: 6,
            background: "#fef2f2", border: "1px solid #fecaca",
          }}>
            <div style={{ fontSize: 10, fontWeight: 700, color: "#dc2626", marginBottom: 4 }}>
              ⚠ Oracle Recovery Required
            </div>
            <div style={{ fontSize: 11, color: "#7f1d1d", lineHeight: 1.5 }}>
              This value was recovered from an infographic. OCR/Marker did not capture it.
              Verify numeric thresholds against the original PDF.
            </div>
          </div>
        )}

        {highlight.isRec && (
          <div style={{
            marginTop: 16, padding: "10px 12px", borderRadius: 6,
            background: "#eff6ff", border: "1px solid #bfdbfe",
          }}>
            <div style={{ fontSize: 10, fontWeight: 700, color: "#1d4ed8", marginBottom: 4 }}>
              ★ Graded Recommendation
            </div>
            <div style={{ fontSize: 11, color: "#1e3a5f", lineHeight: 1.5 }}>
              Grade 1C — strong recommendation, low certainty of evidence.
              Key Information section has been reparented under §2.2 (Fix #1b).
            </div>
          </div>
        )}
      </div>

      {/* Action buttons */}
      <div style={{
        padding: "12px 14px", borderTop: "1px solid #e2e8f0",
        display: "flex", gap: 6,
      }}>
        <ActionBtn label="✓ Accept" color="#10b981" onClick={() => onAction("accept", highlight.id)} />
        <ActionBtn label="✏ Edit" color="#f59e0b" onClick={() => onAction("edit", highlight.id)} />
        <ActionBtn label="✕ Reject" color="#ef4444" onClick={() => onAction("reject", highlight.id)} />
      </div>
    </div>
  );
}

function DetailRow({ label, value }) {
  return (
    <div style={{
      display: "flex", justifyContent: "space-between", alignItems: "center",
      padding: "5px 0", borderBottom: "1px solid #f1f5f9",
    }}>
      <span style={{ fontSize: 10, color: "#94a3b8", fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.04em" }}>{label}</span>
      <span style={{ fontSize: 12, color: "#1e293b", fontWeight: 500 }}>{typeof value === "string" ? value : value}</span>
    </div>
  );
}

function ActionBtn({ label, color, onClick }) {
  return (
    <button
      onClick={onClick}
      style={{
        flex: 1, padding: "8px 0", border: `1.5px solid ${color}`,
        borderRadius: 6, background: "transparent",
        color, fontSize: 12, fontWeight: 700, cursor: "pointer",
        transition: "all 0.12s ease",
      }}
      onMouseEnter={e => { e.target.style.background = color; e.target.style.color = "#fff"; }}
      onMouseLeave={e => { e.target.style.background = "transparent"; e.target.style.color = color; }}
    >
      {label}
    </button>
  );
}

function StatsBar({ meta }) {
  return (
    <div style={{
      display: "flex", gap: 16, padding: "0 4px", flexWrap: "wrap",
    }}>
      {[
        { label: "Raw", val: meta.raw_spans, sub: "spans" },
        { label: "Merged", val: meta.merged_spans, sub: "spans" },
        { label: "Passages", val: meta.section_passages, sub: "sections" },
        { label: "Ch.B", val: meta.channels.B, sub: "drugs" },
        { label: "Ch.C", val: meta.channels.C, sub: "grammar" },
        { label: "Ch.D", val: meta.channels.D, sub: "table" },
        { label: "Ch.F", val: meta.channels.F, sub: "LLM" },
      ].map(s => (
        <div key={s.label} style={{ textAlign: "center", minWidth: 48 }}>
          <div style={{
            fontSize: 18, fontWeight: 800, color: "#0f172a",
            fontFamily: "'JetBrains Mono', monospace", lineHeight: 1,
          }}>{s.val}</div>
          <div style={{ fontSize: 9, color: "#94a3b8", fontWeight: 600, marginTop: 2, letterSpacing: "0.03em" }}>
            {s.label}
          </div>
        </div>
      ))}
    </div>
  );
}

// ─── MAIN APP ────────────────────────────────────────────────────────────────

export default function ReviewerUI() {
  const [activePage, setActivePage] = useState(1);
  const [selectedHighlight, setSelectedHighlight] = useState(null);
  const [reviewState, setReviewState] = useState({});
  const [spanActions, setSpanActions] = useState({});
  const [showPassageMode, setShowPassageMode] = useState(false);

  const page = PAGES.find(p => p.num === activePage);
  const highlight = page?.highlights.find(h => h.id === selectedHighlight);

  const handlePageAction = useCallback((pageNum, action) => {
    setReviewState(prev => ({ ...prev, [pageNum]: action }));
  }, []);

  const handleSpanAction = useCallback((action, highlightId) => {
    setSpanActions(prev => ({ ...prev, [highlightId]: action }));
  }, []);

  const acceptedCount = Object.values(reviewState).filter(v => v === "accepted").length;
  const totalPages = PAGES.length;

  return (
    <div style={{
      width: "100%", height: "100vh", display: "flex", flexDirection: "column",
      background: "#f1f5f9",
      fontFamily: "'IBM Plex Sans', 'SF Pro Text', -apple-system, sans-serif",
    }}>
      {/* ─── Top Bar ──────────────────────────────────── */}
      <div style={{
        background: "#0f172a", color: "#fff",
        padding: "10px 20px",
        display: "flex", alignItems: "center", justifyContent: "space-between",
        boxShadow: "0 2px 8px rgba(0,0,0,0.15)",
        zIndex: 10,
      }}>
        <div style={{ display: "flex", alignItems: "center", gap: 14 }}>
          <div style={{
            width: 28, height: 28, borderRadius: 6,
            background: "linear-gradient(135deg, #3b82f6, #8b5cf6)",
            display: "flex", alignItems: "center", justifyContent: "center",
            fontSize: 14, fontWeight: 800,
          }}>⚕</div>
          <div>
            <div style={{ fontSize: 13, fontWeight: 700, letterSpacing: "0.01em" }}>
              {PIPELINE_META.guideline}
            </div>
            <div style={{ fontSize: 10, color: "#94a3b8", marginTop: 1 }}>
              {PIPELINE_META.chapter} · Pages {PIPELINE_META.pages} · Job {PIPELINE_META.job_id.slice(-8)}
            </div>
          </div>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 16 }}>
          <StatsBar meta={PIPELINE_META} />
          <div style={{
            height: 32, width: 1, background: "#334155",
          }} />
          <div style={{
            fontSize: 12, fontWeight: 700, color: acceptedCount === totalPages ? "#10b981" : "#f59e0b",
          }}>
            {acceptedCount}/{totalPages} pages reviewed
          </div>
        </div>
      </div>

      {/* ─── Main Content ─────────────────────────────── */}
      <div style={{
        flex: 1, display: "flex", overflow: "hidden", gap: 0,
      }}>
        {/* Left: Page Navigator + Highlights */}
        <div style={{
          width: 260, background: "#fff", borderRight: "1px solid #e2e8f0",
          display: "flex", flexDirection: "column", overflow: "hidden",
          flexShrink: 0,
        }}>
          <div style={{
            padding: "12px 14px 8px",
            fontSize: 9, fontWeight: 700, letterSpacing: "0.08em",
            textTransform: "uppercase", color: "#94a3b8",
            borderBottom: "1px solid #f1f5f9",
          }}>Pages</div>
          <div style={{ padding: "8px 10px", overflow: "auto", flex: "0 0 auto" }}>
            <PageNavigator
              pages={PAGES}
              activePage={activePage}
              onSelect={(n) => { setActivePage(n); setSelectedHighlight(null); }}
              reviewState={reviewState}
            />
          </div>

          <div style={{
            padding: "10px 14px 6px",
            fontSize: 9, fontWeight: 700, letterSpacing: "0.08em",
            textTransform: "uppercase", color: "#94a3b8",
            borderTop: "1px solid #e2e8f0",
            borderBottom: "1px solid #f1f5f9",
          }}>
            Highlights · {page?.label}
          </div>
          <div style={{ flex: 1, overflow: "auto", padding: "8px 10px" }}>
            {page?.highlights.map(h => (
              <HighlightCard
                key={h.id}
                h={h}
                isSelected={selectedHighlight === h.id}
                onClick={setSelectedHighlight}
              />
            ))}
          </div>
        </div>

        {/* Center: PDF Viewer */}
        <div style={{ flex: 1, display: "flex", flexDirection: "column", overflow: "hidden" }}>
          {/* View mode toggle */}
          <div style={{
            padding: "6px 16px", background: "#f8fafc",
            borderBottom: "1px solid #e2e8f0",
            display: "flex", alignItems: "center", gap: 8,
          }}>
            <button
              onClick={() => setShowPassageMode(false)}
              style={{
                padding: "4px 12px", borderRadius: 4, fontSize: 11, fontWeight: 600,
                border: "none", cursor: "pointer",
                background: !showPassageMode ? "#1e293b" : "#e2e8f0",
                color: !showPassageMode ? "#fff" : "#64748b",
              }}
            >Highlighted View</button>
            <button
              onClick={() => setShowPassageMode(true)}
              style={{
                padding: "4px 12px", borderRadius: 4, fontSize: 11, fontWeight: 600,
                border: "none", cursor: "pointer",
                background: showPassageMode ? "#1e293b" : "#e2e8f0",
                color: showPassageMode ? "#fff" : "#64748b",
              }}
            >Section Passage</button>
            <div style={{ flex: 1 }} />
            <span style={{ fontSize: 10, color: "#94a3b8" }}>
              {PIPELINE_META.version} · Fix #1b applied
            </span>
          </div>

          <div style={{ flex: 1, padding: 12, overflow: "hidden" }}>
            {!showPassageMode ? (
              <PDFPageViewer
                page={page}
                selectedHighlight={selectedHighlight}
                onSelectHighlight={setSelectedHighlight}
              />
            ) : (
              <SectionPassageView page={page} />
            )}
          </div>
        </div>

        {/* Right: Span Inspector */}
        <div style={{
          width: 280, background: "#fff", borderLeft: "1px solid #e2e8f0",
          display: "flex", flexDirection: "column", overflow: "hidden",
          flexShrink: 0,
        }}>
          <div style={{
            padding: "12px 14px 8px",
            fontSize: 9, fontWeight: 700, letterSpacing: "0.08em",
            textTransform: "uppercase", color: "#94a3b8",
            borderBottom: "1px solid #f1f5f9",
          }}>Highlight Inspector</div>
          <div style={{ flex: 1, overflow: "auto" }}>
            <SpanInspector highlight={highlight} onAction={handleSpanAction} />
          </div>
        </div>
      </div>

      {/* ─── Bottom: Decision Bar ─────────────────────── */}
      <div style={{
        background: "#fff", borderTop: "1px solid #e2e8f0",
        padding: "10px 20px",
        display: "flex", alignItems: "center", justifyContent: "space-between",
        boxShadow: "0 -2px 8px rgba(0,0,0,0.04)",
      }}>
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <span style={{ fontSize: 12, color: "#64748b", fontWeight: 500 }}>
            Page {activePage} Decision:
          </span>
          {reviewState[activePage] && (
            <span style={{
              fontSize: 11, fontWeight: 700, padding: "3px 10px", borderRadius: 4,
              background: reviewState[activePage] === "accepted" ? "#ecfdf5" : reviewState[activePage] === "flagged" ? "#fffbeb" : "#fef2f2",
              color: reviewState[activePage] === "accepted" ? "#059669" : reviewState[activePage] === "flagged" ? "#d97706" : "#dc2626",
            }}>
              {reviewState[activePage] === "accepted" ? "✓ Accepted" : reviewState[activePage] === "flagged" ? "⚑ Flagged" : "↑ Escalated"}
            </span>
          )}
        </div>
        <div style={{ display: "flex", gap: 8 }}>
          <button
            onClick={() => handlePageAction(activePage, "accepted")}
            style={{
              padding: "8px 20px", borderRadius: 6,
              background: "#10b981", color: "#fff", border: "none",
              fontSize: 12, fontWeight: 700, cursor: "pointer",
              boxShadow: "0 1px 3px rgba(16,185,129,0.3)",
            }}
          >✓ Accept Page</button>
          <button
            onClick={() => handlePageAction(activePage, "flagged")}
            style={{
              padding: "8px 20px", borderRadius: 6,
              background: "#fff", color: "#d97706",
              border: "1.5px solid #fbbf24",
              fontSize: 12, fontWeight: 700, cursor: "pointer",
            }}
          >⚑ Flag for Follow-up</button>
          <button
            onClick={() => handlePageAction(activePage, "escalated")}
            style={{
              padding: "8px 16px", borderRadius: 6,
              background: "#fff", color: "#dc2626",
              border: "1.5px solid #fca5a5",
              fontSize: 12, fontWeight: 700, cursor: "pointer",
            }}
          >↑ Escalate</button>
          <div style={{ width: 1, background: "#e2e8f0", margin: "0 4px" }} />
          <button
            onClick={() => {
              if (activePage < PAGES.length) {
                setActivePage(activePage + 1);
                setSelectedHighlight(null);
              }
            }}
            style={{
              padding: "8px 20px", borderRadius: 6,
              background: "#1e293b", color: "#fff", border: "none",
              fontSize: 12, fontWeight: 700, cursor: "pointer",
            }}
          >Save & Next →</button>
        </div>
      </div>
    </div>
  );
}

function SectionPassageView({ page }) {
  return (
    <div style={{
      background: "#fff", borderRadius: 8, border: "1px solid #e2e8f0",
      height: "100%", overflow: "auto", padding: 20,
    }}>
      <div style={{
        fontSize: 9, fontWeight: 700, letterSpacing: "0.06em",
        textTransform: "uppercase", color: "#94a3b8", marginBottom: 12,
      }}>Section Passage — What L3 Receives</div>

      <div style={{
        fontFamily: "'JetBrains Mono', monospace",
        fontSize: 11, lineHeight: 1.7,
        background: "#0f172a", color: "#e2e8f0",
        borderRadius: 8, padding: 16, whiteSpace: "pre-wrap",
      }}>
        <span style={{ color: "#94a3b8" }}>{"{"}</span>{"\n"}
        <span style={{ color: "#7dd3fc" }}>  "section_id"</span>: <span style={{ color: "#fbbf24" }}>"{page.sections[0]}"</span>,{"\n"}
        <span style={{ color: "#7dd3fc" }}>  "heading"</span>: <span style={{ color: "#fbbf24" }}>"{page.title}"</span>,{"\n"}
        <span style={{ color: "#7dd3fc" }}>  "page_number"</span>: <span style={{ color: "#a78bfa" }}>{page.num}</span>,{"\n"}
        <span style={{ color: "#7dd3fc" }}>  "span_count"</span>: <span style={{ color: "#a78bfa" }}>{page.highlights.reduce((s, h) => s + h.count, 0)}</span>,{"\n"}
        <span style={{ color: "#7dd3fc" }}>  "signals"</span>: {"{"}{"\n"}
        <span style={{ color: "#7dd3fc" }}>    "drugs"</span>: [
        {page.highlights.filter(h => h.type === "drug" || h.type === "drug_class").map((h, i) => (
          <span key={i} style={{ color: "#34d399" }}>"{h.text.split(",")[0].trim()}"</span>
        )).reduce((prev, curr, i) => i === 0 ? [curr] : [...prev, ", ", curr], [])}
        ],{"\n"}
        <span style={{ color: "#7dd3fc" }}>    "thresholds"</span>: [
        {page.highlights.filter(h => h.type === "egfr_threshold").map((h, i) => (
          <span key={i} style={{ color: "#34d399" }}>"{h.text}"</span>
        )).reduce((prev, curr, i) => i === 0 ? [curr] : [...prev, ", ", curr], [])}
        ],{"\n"}
        <span style={{ color: "#7dd3fc" }}>    "labs"</span>: [
        {page.highlights.filter(h => h.type === "lab_test").map((h, i) => (
          <span key={i} style={{ color: "#34d399" }}>"{h.text}"</span>
        )).reduce((prev, curr, i) => i === 0 ? [curr] : [...prev, ", ", curr], [])}
        ]{"\n"}
        {"  },"}{"\n"}
        <span style={{ color: "#7dd3fc" }}>  "prose_text"</span>: <span style={{ color: "#fbbf24" }}>"{page.content[0]?.text.slice(0, 120)}..."</span>{"\n"}
        <span style={{ color: "#94a3b8" }}>{"}"}</span>
      </div>

      <div style={{
        marginTop: 16, padding: 12, borderRadius: 6,
        background: "#f0fdf4", border: "1px solid #bbf7d0",
        fontSize: 11, color: "#166534", lineHeight: 1.5,
      }}>
        <strong>L3 Readiness:</strong> This passage contains complete clinical text + {page.highlights.reduce((s, h) => s + h.count, 0)} tagged entity spans.
        L3 can build dossier entries from this structure.
      </div>
    </div>
  );
}
