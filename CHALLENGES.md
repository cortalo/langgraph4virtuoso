# Layout Automation Agent — Challenges & Lessons Learned

## 1. Topology Inference Reliability

**Problem:** The LLM repeatedly misidentified circuit topologies from raw JSON netlists.
Examples: two cascaded inverters identified as NAND2; PMOS differential pair and NMOS current
mirror load roles swapped multiple times in a 6T OTA.

**Root cause:** LLMs are poor at precise graph traversal. Following terminal connections
across a netlist is a structured computation, not pattern matching. No amount of prompt
engineering fully compensates for this.

**Partial mitigations applied:**
- Added connection signatures (gate-sharing, drain-to-drain, tail node) to placement_guidelines.yaml
- Added hard rules in the system prompt (gate independence as primary discriminator, inverter
  disqualifier for mismatched gates, internal-net-only tail node rule)
- Passed the circuit pins list explicitly so the model could identify diff pair inputs from ports
- Introduced sub_groups in the output JSON to capture functional roles within a topology

**Remaining gap:** Still not fully reliable for complex multi-device circuits.

**Better long-term approach:** Topology classification is a graph algorithm problem —
deterministic Python code (find diode-connected devices, find shared gate nets, detect tail
nodes) would be cheaper, faster, and 100% reliable. Reserve the LLM for placement judgment
on novel circuits, not for re-deriving facts already computable from the netlist.

**Short-term pragmatic fix:** Human-in-the-loop (LangGraph Module 3) — pause after topology
identification and ask the user to confirm or correct before any placements are committed to
Virtuoso. One human confirmation catches all downstream placement errors.

---

## 2. Agent Stopping Too Early

**Problem:** The LLM would complete its analysis, describe what it intended to do next,
then emit a final response without calling any placement tools.

**Root cause:** Generating text is always the LLM's path of least resistance. Without
an explicit external check, the model self-reports completion regardless of actual state.

**Fix applied:** Added `is_placement_complete` tool that compares placed devices against
the full device set from the schematic. The system prompt requires calling this tool and
receiving COMPLETE before giving any final response. The LLM cannot lie about completion
when the tool returns a list of missing devices.

---

## 3. Device Placed Twice (Duplicate Placement)

**Problem:** The same device (e.g. P2) was placed twice — once correctly as part of the
OTA topology, then again at wrong coordinates when the LLM encountered it in a second
topology entry (current mirror).

**Root cause:** The LLM has no persistent memory of actions taken within a multi-step
session without explicit external state.

**Fix applied:** `_placement_registry` dict tracks every placed device by name. At the
start of `place_instance`, if the name is already in the registry, the tool returns the
existing coordinates and skips the SKILL call. The LLM receives the previous position as
context for placing adjacent devices.

---

## 4. Coordinate Collision (Different Devices, Same Location)

**Problem:** Multiple devices placed at identical (x, y) coordinates, either from bad
placement logic or the LLM forgetting earlier placements.

**Fix applied:** Before placing, `place_instance` builds a reverse map of the registry
(`coords → name`) and rejects any placement that would land on an already-occupied
coordinate, returning an error message so the LLM can choose different coordinates.

Coordinates stored as integer nanometers (um × 1000) to avoid floating-point comparison
failures (e.g. 1.0 vs 1.0000000001).

---

## 5. Concurrent Tool Execution Race Condition

**Problem:** When the LLM emits multiple `place_instance` calls in a single response,
LangGraph's ToolNode runs them in a thread pool concurrently. All threads read an empty
`_placement_registry` simultaneously, all pass the conflict check, then all write — so
duplicate/colliding placements slip through undetected.

**Root cause:** Module-level mutable state is not thread-safe under concurrent tool
execution. This is a fundamental tension between LangGraph's parallelism optimization
and shared state managed outside the graph.

**Fix options:**
- **Quick fix:** `threading.Lock` wrapping the check-and-register block atomically
- **Correct LangGraph approach:** Move `_placement_registry` and `_schematic_devices`
  into typed LangGraph graph state (covered in LangGraph course Module 2). State managed
  by the graph is naturally safe from this problem and survives graph interrupts/replays.

**Status:** Not yet fixed — pending.

---

## 6. Prompt Brittleness and Rule Accumulation

**Problem:** Every new circuit or edge case required adding new rules to the system prompt.
Rules sometimes contradicted each other or were silently ignored. The prompt grew long
enough that earlier rules were deprioritized by the model.

**Root cause:** Known LLM limitation — instruction following degrades with prompt length
and rule count. Prose rules cannot substitute for algorithmic constraints.

**Mitigation:** Moved circuit-specific rules into placement_guidelines.yaml (editable
without code changes, loaded at startup). Hard rules kept in a dedicated section of the
prompt with explicit "never violate" framing.

---

## 7. LLM Output Format Instability

**Problem:** The inner topology LLM sometimes returned markdown-fenced JSON (```json ...```),
breaking `json.loads` in the calling code.

**Fix applied:** Stopped parsing the topology output as JSON — return it as a raw string
embedded in the combined tool result. Added "Do not wrap output in markdown fences" to
the topology system prompt.

---

## Summary: What Worked Well

- Separating topology identification into its own inner LLM call (`read_and_analyse_schematic`)
  made it debuggable and independently tunable
- Returning structured sub_groups in topology JSON gave the placement agent the role
  assignments it needed without further inference
- External state enforcement (registry, is_complete) proved more reliable than prompt
  instructions alone for correctness guarantees
- INFO-level logging of every tool entry/exit made debugging fast

## Key Insight

The most reliable parts of the agent are those where **code enforces correctness**
(placement registry, coordinate check, is_complete). The least reliable parts are those
where the LLM is asked to perform **structured computation** (topology graph analysis).
The right architecture is: LLM for judgment and flexibility, code for facts and constraints.
