import logging
from langchain_core.messages import SystemMessage, HumanMessage, RemoveMessage
from langchain_core.tools import tool
from langchain_openai import ChatOpenAI
from langgraph.constants import START
from langgraph.graph import MessagesState, StateGraph
from langgraph.prebuilt import ToolNode, tools_condition

log = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Tools
# ---------------------------------------------------------------------------
def make_tools():
    """Create LangGraph tools"""

    @tool
    def multiply(a: int, b: int) -> int:
        """Multiply a and b.

        Args:
            a: first int
            b: second int
        """
        return a * b

    @tool
    def add(a: int, b: int) -> int:
        """Adds a and b.

        Args:
            a: first int
            b: second int
        """
        return a + b

    @tool
    def divide(a: int, b: int) -> float:
        """Divide a and b.

        Args:
            a: first int
            b: second int
        """
        return a / b

    return [
        multiply,
        add,
        divide,
    ]

# ---------------------------------------------------------------------------
# Graph
# ---------------------------------------------------------------------------

_PLANNER_PROMPT = """\
You are a planning assistant. Given a user goal, produce a concise numbered step-by-step
plan to achieve it. Do not execute anything, only plan.
"""

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

def build_graph():
    tools = make_tools()
    llm = ChatOpenAI(model="gpt-4o")
    llm_with_tools = llm.bind_tools(tools)

    def planner_node(state: MessagesState):
        response = llm.invoke([SystemMessage(content=_PLANNER_PROMPT)] + state["messages"])
        log.info("[planner] plan:\n%s", response.content)
        return {"messages": [response]}

    def agent_node(state: MessagesState):
        return {"messages": [llm_with_tools.invoke(state["messages"])]}

    def memory_master_node(state: MessagesState):
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
    builder.add_node("planner", planner_node)
    builder.add_node("agent", agent_node)
    builder.add_node("tools", ToolNode(tools))
    builder.add_node("memory_master", memory_master_node)
    builder.add_edge(START, "planner")
    builder.add_edge("planner", "agent")
    builder.add_conditional_edges("agent", tools_condition)
    builder.add_edge("tools", "memory_master")
    builder.add_edge("memory_master", "agent")
    return builder.compile()

def run(task: str):
    graph = build_graph()
    result = graph.invoke({
        "messages": [HumanMessage(content=task)]
    })
    final = result["messages"][-1]
    print("\nFinal answer:")
    print(final.content)
    return final.content