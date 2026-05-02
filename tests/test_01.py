import logging
from langgraph4virtuoso.agent.graph import run

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s — %(message)s",
    datefmt="%H:%M:%S",
)
for _noisy in ("httpx", "openai", "httpcore"):
    logging.getLogger(_noisy).setLevel(logging.WARNING)

if __name__ == '__main__':
    run("What is (3 + 4) * 12 / 2?")