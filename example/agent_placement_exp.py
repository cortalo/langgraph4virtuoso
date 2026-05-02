"""Experimental ReAct agent with initial planner and LLM-based memory compaction.

Architecture
------------
    START → planner → assistant ──tools──→ tools → memory_master ─┐
                           ↑                                        │
                           └────────────────────────────────────────┘
                           │
                          END
"""

import logging

from langchain_openai import ChatOpenAI
from langchain_core.messages import HumanMessage, RemoveMessage, SystemMessage
from langgraph.graph import START, StateGraph, MessagesState
from langgraph.prebuilt import tools_condition, ToolNode

log = logging.getLogger(__name__)


def multiply(a: int, b: int) -> int:
    """Multiply a and b.

    Args:
        a: first int
        b: second int
    """
    return a * b


def add(a: int, b: int) -> int:
    """Adds a and b.

    Args:
        a: first int
        b: second int
    """
    return a + b


def divide(a: int, b: int) -> float:
    """Divide a and b.

    Args:
        a: first int
        b: second int
    """
    return a / b


tools = [add, multiply, divide]
llm = ChatOpenAI(model="gpt-4o")
llm_with_tools = llm.bind_tools(tools, parallel_tool_calls=False)

_MEMORY_MASTER_PROMPT = """\
You are a memory management system for an AI assistant.

After every tool interaction you receive the agent's current message history and \
produce a single concise working-memory summary.  This summary completely replaces \
the history on the next turn, so the agent will only see what you write — nothing else.

## Two non-negotiable requirements

1. GOAL — The original user goal must appear faithfully at the top of every summary,
   exactly as stated in the first Human message (or as carried forward from a prior
   [Working memory] block).  Never paraphrase, shorten, or drop it.

2. NEXT STEP — The summary must end with one concrete, actionable instruction telling
   the agent exactly what to do next.

## Everything else

Outside of those two requirements, compress freely.  Keep intermediate results that
are still needed to reach the goal.  Drop raw tool output, repeated reasoning, and
anything already superseded by a newer result.  If a prior attempt failed, preserve
the failure and the lesson so the agent does not repeat it.

## Format

Goal: <original user goal, verbatim>
Progress: <brief summary of what has been done and key intermediate results>
Next step: <single concrete instruction>

Write in second person, present tense.
"""


_PLANNER_PROMPT = """\
You are a planning assistant. Given a user goal, produce a concise numbered step-by-step
plan to achieve it. Each step should be a single concrete action. Do not execute anything
— only plan.
"""


def planner(state: MessagesState):
    response = llm.invoke([SystemMessage(content=_PLANNER_PROMPT)] + state["messages"])
    log.info("[planner] plan:\n%s", response.content)
    return {"messages": [response]}


def assistant(state: MessagesState):
    return {"messages": [llm_with_tools.invoke(state["messages"])]}


def memory_master(state: MessagesState):
    messages = state["messages"]
    if not messages:
        return {}

    log.info("[memory_master] current memory length: %s", len(messages))

    response = llm.invoke(
        [SystemMessage(content=_MEMORY_MASTER_PROMPT)]
        + messages
        + [HumanMessage(content="Now consolidate the above into working memory.")]
    )
    compacted = response.content.strip()
    log.info("[memory_master] compacted:\n%s", compacted)

    remove_all = [RemoveMessage(id=m.id) for m in messages]
    return {"messages": remove_all + [HumanMessage(content=f"[Working memory]\n{compacted}")]}


builder = StateGraph(MessagesState)
builder.add_node("planner", planner)
builder.add_node("assistant", assistant)
builder.add_node("tools", ToolNode(tools))
builder.add_node("memory_master", memory_master)
builder.add_edge(START, "planner")
builder.add_edge("planner", "assistant")
builder.add_conditional_edges("assistant", tools_condition)
builder.add_edge("tools", "memory_master")
builder.add_edge("memory_master", "assistant")
graph = builder.compile()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s", datefmt="%H:%M:%S")
    # suppress noisy third-party loggers
    for _noisy in ("httpx", "openai", "httpcore"):
        logging.getLogger(_noisy).setLevel(logging.WARNING)

    question = "What is (3 + 4) * 12 / 2?"
    graph.invoke({"messages": [HumanMessage(content=question)]})
