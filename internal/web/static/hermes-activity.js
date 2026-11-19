// What she is doing while she answers. The chat stream carries "event: hermes.tool.progress" lines between the text
// chunks; this reads the stream line by line and keeps a short list of steps (searching, reading, remembering).

const WORDS = {
  web_search: "Searching the web", web_extract: "Reading a page", memory: "Remembering", session_search: "Looking back through our chats",
  ha_list_entities: "Looking at the home's devices", ha_get_state: "Checking a device", ha_list_services: "Checking what a device can do", ha_call_service: "Controlling the home",
  tool_search: "Finding a tool", tool_describe: "Reading about a tool", skill_view: "Checking a skill", skills_list: "Checking her skills", todo: "Making a list", vision_analyze: "Looking at an image",
};

/** A readable name for a tool: known ones in plain words, others from their own name. */
export function toolWords(tool) {
  return WORDS[tool] || (String(tool || "working").replace(/[_-]+/g, " ").replace(/^./, (c) => c.toUpperCase()));
}

/** Reads the chat stream one line at a time. feed() returns {text} for a piece of the answer, {steps} when the steps
 *  changed, {error} when the relay reports a failure, {cards} for the library items she offers to play, or null for anything else (comments, blank lines). A "waiting" step shows
 *  while the first words are slow and goes away when they arrive. */
export function createStreamReader() {
  let event = "";
  const steps = [];
  return {
    steps: () => steps.map((step) => ({ ...step })),
    feed(raw) {
      const line = raw.trim();
      if (!line) { event = ""; return null; }
      if (line.startsWith("event:")) { event = line.slice(6).trim(); return null; }
      if (!line.startsWith("data:")) return null;
      const data = line.slice(5).trim();
      if (!data || data === "[DONE]") return null;
      let body;
      try { body = JSON.parse(data); } catch (_) { return null; }
      if (event === "hermes.waiting") {
        let wait = steps.find((item) => item.id === "wait");
        if (!wait) { wait = { id: "wait", tool: "waiting", detail: "", status: "running" }; steps.push(wait); }
        wait.detail = `${Number(body.seconds) || 0} s`;
        return { steps: steps.map((item) => ({ ...item })) };
      }
      if (event === "hermes.cards") return { cards: Array.isArray(body.cards) ? body.cards : [] };
      if (event === "hermes.error") return { error: String(body.error || "The answer was interrupted") };
      if (event === "hermes.tool.progress") {
        const id = body.toolCallId || `${body.tool}-${steps.length}`;
        let step = steps.find((item) => item.id === id);
        if (!step) { step = { id, tool: body.tool, detail: "", status: "running" }; steps.push(step); }
        if (body.label) step.detail = String(body.label);
        step.status = body.status === "completed" ? "done" : body.status === "failed" || body.status === "error" ? "failed" : "running";
        return { steps: steps.map((item) => ({ ...item })) };
      }
      const delta = body.choices?.[0]?.delta?.content;
      if (!delta) return null;
      const at = steps.findIndex((item) => item.id === "wait");
      if (at < 0) return { text: delta };
      steps.splice(at, 1); // the answer has started, so "waiting" is over
      return { text: delta, steps: steps.map((item) => ({ ...item })) };
    },
  };
}
