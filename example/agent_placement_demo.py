"""Analog layout agent — reads a schematic then places instances in a new layout.

Usage:
    python example/create_layout_agent.py
    python example/create_layout_agent.py --debug
"""

import json
import argparse
import logging
from pathlib import Path
import yaml
from langchain_openai import ChatOpenAI
from langchain_core.messages import HumanMessage, AIMessage, ToolMessage, SystemMessage
from langgraph.graph import StateGraph, START, END, MessagesState
from langgraph.prebuilt import ToolNode, tools_condition

from virtuoso_bridge import VirtuosoClient, decode_skill_output
from virtuoso_bridge.virtuoso.schematic.reader import read_schematic

log = logging.getLogger(__name__)

client = VirtuosoClient.local(port=65432)

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
            # substitute spacing values so the LLM sees concrete numbers
            rendered = rule.format(**{k.replace("-", "_"): v for k, v in spacing.items()})
            lines.append(f"  - {rendered}")
    lines += [
        "",
        "## Analysis procedure",
        "",
        "Before placing any instance, work through these steps in order:",
        "",
        "1. Classify every instance as NMOS, PMOS, or unknown using the `device_type`",
        "   field in the read_schematic_tool output (cross-check with the cell name).",
        "",
        "2. Identify key shared nets by examining which instance terminals connect to each net:",
        "   - Shared gate net between a PMOS and an NMOS → they form a complementary pair",
        "     with a common input signal.",
        "   - Shared drain net between a PMOS and an NMOS → they share a common output node.",
        "   - A net connected only to a PMOS source (no other active device) → VDD supply rail.",
        "   - A net connected only to an NMOS source (no other active device) → VSS / GND rail.",
        "",
        "3. Match the observed connection pattern to one of the known circuit topologies",
        "   described above (e.g. inverter). Name the circuit explicitly before proceeding.",
        "",
        "4. Apply the placement rules for the matched topology.",
        "   Do not use arbitrary coordinates — every coordinate must follow from the rules.",
        "   Call place_instance for EVERY device before giving any final response.",
        "   Do not describe what you are going to do — just call the tools.",
    ]
    return "\n".join(lines)


_SYSTEM_PROMPT = _build_system_prompt(_RULES)


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


def delete_layout(lib: str, cell: str) -> str:
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
    return output


def read_schematic_tool(lib: str, cell: str) -> str:
    """Read the schematic of a cell and return its instances, nets, and pins.

    Args:
        lib: library name
        cell: cell name
    """
    log.info("read_schematic_tool(lib=%s, cell=%s)", lib, cell)
    data = read_schematic(client, lib, cell)
    compact = {
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
    output = json.dumps(compact, indent=2)
    log.info("read_schematic_tool → %s", output)
    return output


def place_instance(
    target_lib: str,
    target_cell: str,
    inst_lib: str,
    inst_cell: str,
    inst_name: str,
    x: float,
    y: float,
    orientation: str = "R0",
) -> str:
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
    return output



llm = ChatOpenAI(model="gpt-4o")
tools = [check_layout_exists, delete_layout, read_schematic_tool, place_instance]
llm_with_tools = llm.bind_tools(tools)


def tool_calling_llm(state: MessagesState):
    response = llm_with_tools.invoke([SystemMessage(content=_SYSTEM_PROMPT)] + state["messages"])
    if isinstance(response, AIMessage) and response.tool_calls:
        for tc in response.tool_calls:
            log.info("[LLM] tool call: %s(%s)", tc["name"], tc["args"])
    else:
        log.info("[LLM] final response")
    return {"messages": [response]}


builder = StateGraph(MessagesState)
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
        "Create a layout for library 'test', cell 'inverter'. "
        "1. Check if the layout exists and delete it if so. "
        "2. Read the schematic. "
        "3. Analyse the topology and device types. "
        "4. Call place_instance for every device using the placement guidelines. "
        "Do not stop after the analysis — place every instance before responding."
    ))]
    result = graph.invoke({"messages": messages})
    result["messages"][-1].pretty_print()
