"""Analog layout agent — reads a schematic then places instances in a new layout.

Usage:
    python example/agent_placement_demo.py
    python example/agent_placement_demo.py --debug
"""

import json
import argparse
import logging
from pathlib import Path
from typing import Annotated
from typing_extensions import TypedDict
import yaml
from langchain_openai import ChatOpenAI
from langchain_core.messages import HumanMessage, AIMessage, ToolMessage, SystemMessage
from langchain_core.tools import InjectedToolCallId
from langgraph.graph import StateGraph, START, END, add_messages
from langgraph.prebuilt import ToolNode, tools_condition, InjectedState
from langgraph.types import Command

from virtuoso_bridge import VirtuosoClient, decode_skill_output
from virtuoso_bridge.virtuoso.schematic.reader import read_schematic

log = logging.getLogger(__name__)

client = VirtuosoClient.local(port=65432)

# ---------------------------------------------------------------------------
# Graph state
# ---------------------------------------------------------------------------

def _merge_registry(old: dict, new: dict) -> dict:
    """Reducer that merges placement registry updates from concurrent tool calls."""
    return {**old, **new}


class LayoutState(TypedDict):
    messages: Annotated[list, add_messages]
    placement_registry: Annotated[dict[str, tuple[int, int]], _merge_registry]
    schematic_devices: list[str]   # set once by read_and_analyse_schematic


# ---------------------------------------------------------------------------
# Placement guidelines
# ---------------------------------------------------------------------------

with open(Path(__file__).parent / "placement_guidelines.yaml") as _f:
    _RULES = yaml.safe_load(_f)

_NMOS_PATTERNS = _RULES["device_classification"]["nmos"]
_PMOS_PATTERNS = _RULES["device_classification"]["pmos"]


def _infer_device_type(cell: str) -> str:
    c = cell.lower()
    if any(p in c for p in _NMOS_PATTERNS):
        return "NMOS"
    if any(p in c for p in _PMOS_PATTERNS):
        return "PMOS"
    return "unknown"


def _build_system_prompt(rules: dict) -> str:
    spacing = rules["spacing"]
    lines = [
        "You are an analog IC layout placement agent.",
        "",
        "## Device type identification",
        f"  NMOS: cell name contains any of {rules['device_classification']['nmos']}",
        f"  PMOS: cell name contains any of {rules['device_classification']['pmos']}",
        "",
        "## Placement guidelines",
    ]
    for name, circuit in rules["circuits"].items():
        lines.append(f"\n### {name.upper()}")
        lines.append(f"Topology: {circuit['topology'].strip()}")
        lines.append("Rules:")
        for rule in circuit["placement"]:
            rendered = rule.format(**{k.replace("-", "_"): v for k, v in spacing.items()})
            lines.append(f"  - {rendered}")
    lines += [
        "",
        "## Workflow",
        "",
        "1. Call read_and_analyse_schematic — it returns both the schematic and identified topologies.",
        "2. For each topology in the result, look up its placement rules above and call place_instance",
        "   for every device. Coordinates must follow the rules — do not use arbitrary values.",
        "   IMPORTANT: call place_instance for ONE device per turn. Wait for the result before",
        "   calling it again. This ensures the placement registry is up to date for collision checks.",
        "3. Call is_placement_complete to verify every device has been placed.",
        "   If INCOMPLETE, place the missing devices and call is_placement_complete again.",
        "   Only give a final response after is_placement_complete returns COMPLETE.",
        "",
        "## General constraints",
        "   No two devices may share the same (x, y) coordinates. Before calling place_instance,",
        "   verify that the coordinates are unique across all devices being placed.",
    ]
    return "\n".join(lines)


_SYSTEM_PROMPT = _build_system_prompt(_RULES)

llm = ChatOpenAI(model="gpt-4o")

# ---------------------------------------------------------------------------
# Topology identification
# ---------------------------------------------------------------------------

def _build_topology_prompt(rules: dict) -> str:
    lines = [
        "You are a circuit topology analyser.",
        "Given a JSON list of instances (each with name, cell, device_type) and a JSON net map",
        "(net name → list of 'instance.terminal' strings), identify every distinct topology.",
        "",
        "Known topologies and their exact connection signatures — match against these first:",
    ]
    for name, circuit in rules["circuits"].items():
        lines.append(f"\n{name.upper()}: {circuit['topology'].strip()}")
        sig = circuit.get("signature", {})
        for k, v in sig.items():
            lines.append(f"  {k}: {str(v).strip()}")
    lines += [
        "",
        "## Hard rules — never violate these",
        "- An INVERTER requires PMOS.g == NMOS.g (same gate net). If the gates are on different",
        "  nets, it is NOT an inverter, regardless of drain connections.",
        "- A diode-connected device (gate tied to its own drain) paired with another device of",
        "  the opposite type is a current mirror load, not an inverter.",
        "- Before assigning 'inverter', explicitly verify both gate nets match.",
        "- FIRST step for any OTA: look at the 'pins' list in the schematic. The differential pair",
        "  devices are those whose GATE net appears in the circuit pins list as a signal input",
        "  (e.g. vinp, vinn, vin+, vin-). Bias or current-input pins (e.g. iin, vbias) drive",
        "  current mirror or tail bias devices, not the diff pair.",
        "  Identify the diff pair by pin membership of gate nets before any other analysis.",
        "- To distinguish a differential pair from a current mirror when both involve two devices",
        "  of the same type, check their GATE nets:",
        "    * Differential pair: each device gate is on a DIFFERENT pin net (independent inputs).",
        "    * Current mirror: all devices share the SAME gate net, one device is diode-connected.",
        "- After identifying the diff pair, assign the remaining roles using these drain-connection rules:",
        "    * Current mirror LOAD: devices whose drains connect to the diff-pair DRAIN nets (drain-to-drain",
        "      with the diff pair). These are always the OPPOSITE device type to the diff pair.",
        "    * Tail current source: the device whose drain connects to the diff-pair SOURCE net (internal net).",
        "      This is always the SAME device type as the diff pair.",
        "    * Tail bias mirror: any remaining devices of the same type as the tail whose gate is shared",
        "      with the tail gate (providing the tail bias current).",
        "- The current mirror load is ALWAYS the opposite type to the diff pair.",
        "  PMOS diff pair → NMOS current mirror load. NMOS diff pair → PMOS current mirror load.",
        "  Never assign the same type as the diff pair as the load.",
        "- A device whose gate is on the same net as its drain is diode-connected; it is a current mirror",
        "  reference, never a diff pair input.",
        "",
        "## Device dual-role note",
        "A single device may serve multiple functional roles simultaneously.",
        "For example, the current mirror load devices in a 5T OTA are also a current mirror;",
        "the tail device is also a current source. Capture this by using sub_groups inside",
        "the parent topology rather than listing the same device in multiple top-level entries.",
        "",
        "If no known topology matches a group of devices, fall back to your general circuit",
        "knowledge to infer the most likely topology name. Do not leave any device unclassified.",
        "",
        "## Output format",
        "Return ONLY a JSON array. Each element must have:",
        '  "topology":    circuit name (from the list above, or your best inference if no match)',
        '  "devices":     flat list of ALL instance names belonging to this topology',
        '  "input_nets":  list of input net names (exclude VDD/VSS supply rails)',
        '  "output_nets": list of output net names (exclude VDD/VSS supply rails)',
        '  "sub_groups":  list of functional sub-blocks within this topology (omit for simple topologies).',
        '                 Each sub_group has: "role" (e.g. differential_pair, current_mirror_load,',
        '                 tail_current_source), "devices" (list of instance names), "type" (NMOS or PMOS).',
        "",
        "Example sub_groups for a 5T OTA:",
        '  [{"role": "differential_pair", "devices": ["M1","M2"], "type": "NMOS"},',
        '   {"role": "current_mirror_load", "devices": ["M3","M4"], "type": "PMOS"},',
        '   {"role": "tail_current_source", "devices": ["M5"], "type": "NMOS"}]',
        "",
        "Every instance must appear in exactly one top-level topology entry.",
        "Do not wrap the output in markdown fences.",
    ]
    return "\n".join(lines)


_TOPOLOGY_SYSTEM_PROMPT = _build_topology_prompt(_RULES)


def _to_nanocoords(x: float, y: float) -> tuple[int, int]:
    return round(x * 1000), round(y * 1000)


# ---------------------------------------------------------------------------
# Tools
# ---------------------------------------------------------------------------

def check_layout_exists(lib: str, cell: str) -> str:
    """Check whether a layout cellview already exists for the given lib/cell.

    Args:
        lib: library name
        cell: cell name
    """
    log.info("check_layout_exists(lib=%s, cell=%s)", lib, cell)
    result = client.execute_skill(
        f'if(dbOpenCellViewByType("{lib}" "{cell}" "layout" "maskLayout" "r") "exists" "not_found")'
    )
    output = decode_skill_output(result.output)
    log.info("check_layout_exists → %s", output)
    return output


def delete_layout(
    lib: str,
    cell: str,
    tool_call_id: Annotated[str, InjectedToolCallId()],
    state: Annotated[LayoutState, InjectedState()],
) -> Command:
    """Delete the layout cellview for the given lib/cell.

    Args:
        lib: library name
        cell: cell name
    """
    log.info("delete_layout(lib=%s, cell=%s)", lib, cell)
    skill = f"""
let((ddview)
    foreach(win hiGetWindowList()
        when(win~>cellView
            && win~>cellView~>libName == "{lib}"
            && win~>cellView~>cellName == "{cell}"
            && win~>cellView~>viewName == "layout"
            dbSave(win~>cellView)
            hiCloseWindow(win)))
    ddview = ddGetObj("{lib}" "{cell}" "layout")
    if(ddview then
        ddDeleteObj(ddview)
        sprintf(nil "deleted: {lib}/{cell}/layout")
    else
        sprintf(nil "not found: {lib}/{cell}/layout")))
"""
    result = client.execute_skill(skill)
    output = decode_skill_output(result.output)
    log.info("delete_layout → %s", output)
    return Command(update={
        "messages": [ToolMessage(content=output, tool_call_id=tool_call_id)],
        "placement_registry": {},
        "schematic_devices": [],
    })


def read_and_analyse_schematic(
    lib: str,
    cell: str,
    tool_call_id: Annotated[str, InjectedToolCallId()],
    state: Annotated[LayoutState, InjectedState()],
) -> Command:
    """Read the schematic and identify all circuit topologies in one step.

    Args:
        lib: library name
        cell: cell name
    """
    log.info("read_and_analyse_schematic(lib=%s, cell=%s)", lib, cell)
    data = read_schematic(client, lib, cell)
    schematic = {
        "instances": [
            {
                "name": i["name"],
                "lib": i["lib"],
                "cell": i["cell"],
                "device_type": _infer_device_type(i["cell"]),
            }
            for i in data["instances"]
        ],
        "nets": {
            name: net["connections"]
            for name, net in data["nets"].items()
        },
        "pins": list(data["pins"].keys()),
    }
    schematic_devices = [i["name"] for i in schematic["instances"]]
    log.info("read_and_analyse_schematic schematic → %s", json.dumps(schematic, indent=2))

    response = llm.invoke([
        SystemMessage(content=_TOPOLOGY_SYSTEM_PROMPT),
        HumanMessage(content=(
            f"Circuit pins: {schematic['pins']}\n\n"
            f"Instances:\n{json.dumps(schematic['instances'], indent=2)}\n\n"
            f"Nets:\n{json.dumps(schematic['nets'], indent=2)}"
        )),
    ])
    topologies = response.content
    log.info("read_and_analyse_schematic topologies → %s", topologies)

    output = json.dumps({"schematic": schematic, "topologies": topologies}, indent=2)
    return Command(update={
        "messages": [ToolMessage(content=output, tool_call_id=tool_call_id)],
        "schematic_devices": schematic_devices,
        "placement_registry": {},
    })


def place_instance(
    target_lib: str,
    target_cell: str,
    inst_lib: str,
    inst_cell: str,
    inst_name: str,
    x: float,
    y: float,
    orientation: str = "R0",
    tool_call_id: Annotated[str, InjectedToolCallId()] = None,
    state: Annotated[LayoutState, InjectedState()] = None,
) -> Command:
    """Place a layout instance in the target cell at the given position.

    Args:
        target_lib: library of the layout cell to create/edit
        target_cell: cell name of the layout to create/edit
        inst_lib: library of the instance master cell
        inst_cell: cell name of the instance master
        inst_name: name to give the instance
        x: x coordinate
        y: y coordinate
        orientation: orientation string e.g. R0, R90, MY, MX
    """
    registry = state["placement_registry"]
    mc = _to_nanocoords(x, y)

    if inst_name in registry:
        x_prev, y_prev = registry[inst_name][0] / 1000, registry[inst_name][1] / 1000
        msg = f"{inst_name} already placed at ({x_prev}, {y_prev}) — not moved"
        log.info("place_instance: %s", msg)
        return Command(update={"messages": [ToolMessage(content=msg, tool_call_id=tool_call_id)]})

    occupied = {v: k for k, v in registry.items()}
    if mc in occupied:
        conflict = occupied[mc]
        msg = f"ERROR: ({x}, {y}) already occupied by {conflict} — choose different coordinates"
        log.warning("place_instance: %s", msg)
        return Command(update={"messages": [ToolMessage(content=msg, tool_call_id=tool_call_id)]})

    log.info("place_instance(%s/%s → %s/%s '%s' @ (%s, %s) %s)",
              inst_lib, inst_cell, target_lib, target_cell, inst_name, x, y, orientation)
    skill = f"""
let((cv)
    cv = dbOpenCellViewByType("{target_lib}" "{target_cell}" "layout" "maskLayout" "a")
    dbCreateParamInstByMasterName(cv "{inst_lib}" "{inst_cell}" "layout" "{inst_name}" list({x} {y}) "{orientation}")
    dbSave(cv)
    dbClose(cv)
    "placed {inst_name} at ({x} {y})")
"""
    result = client.execute_skill(skill)
    output = decode_skill_output(result.output)
    log.info("place_instance → %s", output)
    return Command(update={
        "messages": [ToolMessage(content=output, tool_call_id=tool_call_id)],
        "placement_registry": {inst_name: mc},
    })


def is_placement_complete(
    state: Annotated[LayoutState, InjectedState()],
) -> str:
    """Check whether every device from the schematic has been placed in the layout.

    Returns a summary of placed and missing devices.
    """
    all_devices = set(state["schematic_devices"])
    placed = set(state["placement_registry"].keys())
    missing = all_devices - placed
    if missing:
        log.info("is_placement_complete: missing %s", missing)
        return f"INCOMPLETE — {len(missing)} device(s) not yet placed: {sorted(missing)}. Placed so far: {sorted(placed)}"
    log.info("is_placement_complete: all %d devices placed", len(placed))
    return f"COMPLETE — all {len(placed)} devices placed: {sorted(placed)}"


tools = [check_layout_exists, delete_layout, read_and_analyse_schematic, place_instance, is_placement_complete]
llm_with_tools = llm.bind_tools(tools)


def tool_calling_llm(state: LayoutState):
    response = llm_with_tools.invoke([SystemMessage(content=_SYSTEM_PROMPT)] + state["messages"])
    if isinstance(response, AIMessage) and response.tool_calls:
        for tc in response.tool_calls:
            log.info("[LLM] tool call: %s(%s)", tc["name"], tc["args"])
    else:
        log.info("[LLM] final response")
    return {"messages": [response]}


builder = StateGraph(LayoutState)
builder.add_node("tool_calling_llm", tool_calling_llm)
builder.add_node("tools", ToolNode(tools))
builder.add_edge(START, "tool_calling_llm")
builder.add_conditional_edges("tool_calling_llm", tools_condition)
builder.add_edge("tools", "tool_calling_llm")
graph = builder.compile()


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--debug", action="store_true", help="enable debug logging")
    args = parser.parse_args()

    logging.basicConfig(
        level=logging.DEBUG if args.debug else logging.INFO,
        format="%(asctime)s [%(levelname)s] %(message)s",
        datefmt="%H:%M:%S",
    )
    if not args.debug:
        for noisy in ("httpx", "openai", "httpcore", "virtuoso_bridge"):
            logging.getLogger(noisy).setLevel(logging.WARNING)

    log.info(_SYSTEM_PROMPT)
    messages = [HumanMessage(content=(
        "Create a layout for library 'test', cell 'ota'. "
        "1. Check if the layout exists and delete it if so. "
        "2. Call read_and_analyse_schematic to get the schematic and circuit topologies. "
        "3. Call place_instance for every device using the placement guidelines. "
        "Do not stop after the analysis — place every instance before responding."
    ))]
    result = graph.invoke({
        "messages": messages,
        "placement_registry": {},
        "schematic_devices": [],
    })
    result["messages"][-1].pretty_print()
