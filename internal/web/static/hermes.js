// Hermes chat page. Conversations are Hermes sessions saved on the server, so any device shows the same list and
// threads. Replies stream in as server-sent events from /api/hermes/chat. The assistant's name comes from Settings.
import { createSpeaker } from "./hermes-voice.js?v=2";
import { createListener } from "./hermes-listen.js?v=1";
import { createTalk } from "./hermes-talk.js?v=1";
import { createStage } from "./hermes-stage.js?v=11";
import { createActions, splitContext } from "./hermes-actions.js?v=2";
import { createStreamReader, toolWords } from "./hermes-activity.js?v=4";
import { createMemory } from "./hermes-memory.js?v=3";
import { showCards, stopCards } from "./hermes-cards.js?v=4";

const $ = (selector) => document.querySelector(selector);
const root = document.documentElement;
const read = (k) => { try { return localStorage.getItem(k); } catch (_) { return null; } };
const write = (k, v) => { try { localStorage.setItem(k, v); } catch (_) { /* the server still has the chat */ } };
const mode = read("bobobooks-theme-mode") || read("bobobooks-theme") || "system";
root.dataset.palette = read("bobobooks-palette") || "bobobooks";
root.dataset.theme = mode === "system" ? (matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark") : mode;

const escapeHTML = (value) => String(value ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);
const ICON = {
  edit: '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 20h4L19 9l-4-4L4 16v4zM13 7l4 4"/></svg>',
  trash: '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5 7h14M10 7V5h4v2M7 7l1 12h8l1-12"/></svg>',
  copy: '<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="9" y="9" width="11" height="11" rx="2"/><path d="M5 15V6a2 2 0 0 1 2-2h8"/></svg>',
  speak: '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 10v4h4l5 4V6l-5 4zM17 9a4 4 0 0 1 0 6"/></svg>',
};

/** A small, safe Markdown subset: fenced code, inline code, bold, italic, links, lists, headings. Escaped first. */
function inline(text) {
  return text
    .replace(/`([^`\n]+)`/g, "<code>$1</code>")
    .replace(/\*\*([^*\n]+)\*\*/g, "<strong>$1</strong>")
    .replace(/(^|[\s(])\*([^*\n]+)\*/g, "$1<em>$2</em>")
    .replace(/\[([^\]\n]+)\]\((https?:\/\/[^\s)]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>');
}
function renderMarkdown(text) {
  return String(text).split(/```/).map((part, index) => {
    if (index % 2 === 1) return `<pre><code>${escapeHTML(part.replace(/^[^\n]*\n/, ""))}</code></pre>`;
    return escapeHTML(part).split(/\n{2,}/).filter((block) => block.trim()).map((block) => {
      const lines = block.split("\n");
      if (lines.every((line) => /^\s*[-*•] /.test(line))) return `<ul>${lines.map((line) => `<li>${inline(line.replace(/^\s*[-*•] /, ""))}</li>`).join("")}</ul>`;
      if (lines.every((line) => /^\s*\d+[.)] /.test(line))) return `<ol>${lines.map((line) => `<li>${inline(line.replace(/^\s*\d+[.)] /, ""))}</li>`).join("")}</ol>`;
      if (/^#{1,3} /.test(block)) return `<h3>${inline(block.replace(/^#{1,3} /, ""))}</h3>`;
      return `<p>${inline(block).replace(/\n/g, "<br>")}</p>`;
    }).join("");
  }).join("");
}

const api = async (path, options) => {
  const response = await fetch(path, { credentials: "same-origin", ...options });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || `Request failed (${response.status})`);
  }
  return response.json();
};
const newId = () => "c" + crypto.randomUUID().replace(/-/g, "").slice(0, 24);

const state = { features: {}, name: "Hermes", chats: [], current: read("bobobooks-hermes-current") || newId(), messages: [], abort: null,
  voice: { on: read("bobobooks-hermes-speak") === "1", id: undefined, name: "", available: false } };
// Which "Read aloud" button is playing, so it can say Stop and go back when the speech ends.
let reading = null;
const memory = createMemory({ document, $, api, name: () => state.name, onOpen: () => closeSide() });
// The avatar beside the chat, its switcher, its own voice and the full-screen voice mode.
const stage = createStage({
  $, api, read, write, analyser: () => speaker.analyser(), setState: (kind, text) => setState(kind, text),
  onVoiceMode: (on) => voiceMode(on), onChange: () => fillVoicePick(),
});
const speaker = createSpeaker({
  voice: () => stage.voiceFor(state.voice.id),
  onChange: (speaking) => {
    $("#hm-voice-btn").classList.toggle("speaking", speaking);
    talk?.refresh();
    if (!speaking) stage.calm();
    if (!speaking && reading) { reading.lastChild.textContent = "Read aloud"; reading = null; }
  },
});
// A tap or key press lets the browser play sound; the avatar connects its mouth to the voice at the same moment.
const unlock = () => { speaker.unlock(); stage.starting(); };
let talk = null; // the voice conversation, once the server says it can recognise speech
const initial = () => (state.name.trim()[0] || "H").toUpperCase();

function setName(name) {
  state.name = name || "Hermes";
  document.title = `${state.name} · Mochil`;
  $("#hm-brand").textContent = state.name;
  $("#hm-avatar").textContent = $("#hm-avatar-side").textContent = initial();
  $("#hm-input").placeholder = `Message ${state.name}`;
}

function setState(kind, text) {
  const el = $("#hm-state");
  el.className = `hm-state ${kind}`;
  $("#hm-state-text").textContent = text;
}

// Conversations, grouped by when they were last used.
function groupOf(seconds) {
  const days = Math.floor((Date.now() / 1000 - seconds) / 86400);
  const startOfToday = new Date().setHours(0, 0, 0, 0) / 1000;
  if (seconds >= startOfToday) return "Today";
  if (seconds >= startOfToday - 86400) return "Yesterday";
  return days < 7 ? "Previous 7 days" : "Earlier";
}

function drawList() {
  const list = $("#hm-list");
  if (!state.chats.length) { list.innerHTML = '<p class="hm-nochats">Your conversations will appear here.</p>'; return; }
  const nodes = [];
  let label = "";
  for (const chat of state.chats) {
    const group = groupOf(chat.updated);
    if (group !== label) {
      label = group;
      const heading = document.createElement("div");
      heading.className = "hm-group-label"; heading.textContent = group;
      nodes.push(heading);
    }
    const item = document.createElement("div");
    item.className = "hm-item";
    if (chat.id === state.current) item.setAttribute("aria-current", "true");
    const pick = document.createElement("button");
    pick.type = "button"; pick.className = "hm-pick"; pick.textContent = chat.title; pick.title = chat.title;
    pick.onclick = () => open(chat.id);
    const acts = document.createElement("span");
    acts.className = "hm-acts";
    const edit = document.createElement("button");
    edit.type = "button"; edit.className = "hm-act"; edit.innerHTML = ICON.edit; edit.setAttribute("aria-label", `Rename ${chat.title}`);
    edit.onclick = () => rename(chat, item, pick);
    const del = document.createElement("button");
    del.type = "button"; del.className = "hm-act"; del.innerHTML = ICON.trash; del.setAttribute("aria-label", `Delete ${chat.title}`);
    del.onclick = () => remove(chat, del);
    acts.append(edit, del);
    item.append(pick, acts);
    nodes.push(item);
  }
  list.replaceChildren(...nodes);
}

function rename(chat, item, pick) {
  const input = document.createElement("input");
  input.className = "hm-rename"; input.value = chat.title; input.maxLength = 120; input.setAttribute("aria-label", "Chat name");
  item.replaceChild(input, pick);
  input.focus(); input.select();
  let done = false;
  const finish = async (save) => {
    if (done) return;
    done = true;
    const title = input.value.trim();
    if (save && title && title !== chat.title) {
      try { await api(`/api/hermes/chats/${chat.id}`, { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ title }) }); } catch (_) { /* the old name stays */ }
    }
    await refreshList();
    if (chat.id === state.current) setTitle();
  };
  input.onkeydown = (event) => { if (event.key === "Enter") finish(true); else if (event.key === "Escape") finish(false); };
  input.onblur = () => finish(true);
}

// A second tap confirms, so deleting is never one accidental click and needs no pop-up.
function remove(chat, button) {
  if (button.dataset.sure !== "1") {
    button.dataset.sure = "1"; button.textContent = "Delete?";
    setTimeout(() => { if (button.isConnected) { button.dataset.sure = ""; button.innerHTML = ICON.trash; } }, 3000);
    return;
  }
  api(`/api/hermes/chats/${chat.id}`, { method: "DELETE" }).then(async () => {
    if (chat.id === state.current) { state.current = newId(); state.messages = []; write("bobobooks-hermes-current", state.current); drawLog(); }
    await refreshList();
  }).catch(() => { button.dataset.sure = ""; button.innerHTML = ICON.trash; });
}

async function refreshList() {
  try { state.chats = (await api("/api/hermes/chats")).chats; } catch (_) { /* keep the list we have */ }
  drawList();
}

function setTitle() {
  $("#hm-title").textContent = state.chats.find((chat) => chat.id === state.current)?.title || "New chat";
}

// Messages
const nearBottom = (el) => el.scrollHeight - el.scrollTop - el.clientHeight < 120;
function toBottom(force) { const log = $("#hm-log"); if (force || nearBottom(log)) log.scrollTop = log.scrollHeight; }

function addMessage(role, text, { working = false, error = false } = {}) {
  const log = $("#hm-log");
  log.querySelector(".hm-empty")?.remove();
  const row = document.createElement("div");
  row.className = `hm-row ${role === "user" ? "me" : "bot"}${error ? " err" : ""}`;
  if (role === "user") {
    // A message sent by a quick action carries library context; show what was asked, with the context folded away.
    const { prompt, context } = splitContext(text);
    row.innerHTML = '<div class="hm-bubble"></div>';
    row.firstChild.textContent = prompt;
    if (memory.enabled) { const tools = document.createElement("div"); tools.className = "hm-tools"; tools.append(memory.button(() => prompt)); row.append(tools); }
    if (context) {
      const note = document.createElement("details");
      note.className = "hm-ctx";
      note.innerHTML = "<summary>Library context sent with this</summary><pre></pre>";
      note.lastChild.textContent = context;
      row.firstChild.append(note);
    }
  } else {
    row.innerHTML = `<span class="hm-avatar" aria-hidden="true">${escapeHTML(initial())}</span><div class="hm-bubble"><div class="hm-name">${escapeHTML(state.name)}</div><div class="hm-text"></div><div class="hm-tools"></div></div>`;
    fill(row, text, working);
  }
  log.append(row);
  toBottom(true);
  return row;
}

function fill(row, text, working) {
  const body = row.querySelector(".hm-text");
  if (working && !text) { body.innerHTML = '<span class="hm-dots" aria-label="Typing"><i></i><i></i><i></i></span>'; return; }
  if (row.classList.contains("err")) body.textContent = text; else body.innerHTML = renderMarkdown(text);
}

// The steps she takes before answering (searching, reading, remembering), shown above her words.
function showSteps(row, steps) {
  let box = row.querySelector(".hm-activity");
  if (!box) { box = document.createElement("div"); box.className = "hm-activity"; row.querySelector(".hm-name").after(box); }
  box.replaceChildren(...steps.map((step) => {
    const line = document.createElement("div");
    line.className = `hm-step ${step.status}`;
    const label = step.detail ? `${toolWords(step.tool)}: ${step.detail}` : toolWords(step.tool);
    line.innerHTML = '<i class="hm-step-dot"></i><span></span>';
    line.lastChild.textContent = step.status === "done" ? label : step.status === "failed" ? `${label} (did not work)` : `${label}…`;
    return line;
  }));
}
function finishTools(row, text) {
  const tools = row.querySelector(".hm-tools");
  tools.replaceChildren();
  const copy = document.createElement("button");
  copy.type = "button"; copy.className = "hm-tool"; copy.innerHTML = `${ICON.copy}Copy`;
  copy.onclick = async () => {
    try { await navigator.clipboard.writeText(text); copy.lastChild.textContent = "Copied"; } catch (_) { copy.lastChild.textContent = "Press Ctrl+C"; }
    setTimeout(() => { copy.lastChild.textContent = "Copy"; }, 1500);
  };
  tools.append(copy);
  if (memory.enabled) tools.append(memory.button(() => text));
  if (state.voice.available) {
    const speak = document.createElement("button");
    speak.type = "button"; speak.className = "hm-tool"; speak.innerHTML = `${ICON.speak}Read aloud`;
    speak.onclick = () => {
      if (reading === speak) { speaker.stop(); return; }
      unlock();
      speaker.speakText(text);
      if (reading) reading.lastChild.textContent = "Read aloud";
      reading = speak; speak.lastChild.textContent = "Stop";
    };
    tools.append(speak);
  }
}

function drawLog() {
  const log = $("#hm-log");
  log.replaceChildren();
  setTitle();
  if (!state.messages.length) {
    log.innerHTML = `<div class="hm-empty"><span class="hm-avatar" aria-hidden="true">${escapeHTML(initial())}</span><strong>${escapeHTML(state.name)}</strong><span>Say something to start the conversation.</span><div class="hm-chips"></div></div>`;
    for (const id of ["recent", "next"]) {
      const action = actions?.actions.find((item) => item.id === id);
      if (!action) continue;
      const chip = document.createElement("button");
      chip.type = "button"; chip.className = "hm-chip"; chip.textContent = action.label;
      chip.onclick = () => actions.run(action);
      log.querySelector(".hm-chips").append(chip);
    }
    return;
  }
  for (const message of state.messages) {
    const row = addMessage(message.role, message.text);
    if (message.role !== "user") finishTools(row, message.text);
    if (message.cards?.length) showCards(row, message.cards, () => speaker.stop());
  }
  toBottom(true);
}

async function open(id) {
  if (state.abort) return;
  speaker.stop();
  stopCards();
  state.current = id;
  write("bobobooks-hermes-current", id);
  closeSide();
  drawList();
  try { state.messages = (await api(`/api/hermes/chats/${id}`)).messages; } catch (error) { state.messages = []; setState("bad", error.message); }
  drawLog();
}

function startChat() {
  if (state.abort) return;
  speaker.stop();
  state.current = newId(); state.messages = [];
  write("bobobooks-hermes-current", state.current);
  closeSide(); drawList(); drawLog(); $("#hm-input").focus();
}

function setBusy(busy) {
  $("#hm-send").hidden = busy; $("#hm-stop").hidden = !busy; $("#hm-input").disabled = busy;
  if (!busy) $("#hm-input").focus();
}

async function send(text, extra = {}) {
  const spoken = (state.voice.on || state.talking) && state.voice.available;
  speaker.stop();
  if (spoken) unlock();
  const speech = spoken ? speaker.begin() : null;
  state.messages.push({ role: "user", text });
  addMessage("user", text);
  stage.caption(splitContext(text).prompt, "you");
  const row = addMessage("assistant", "", { working: true });
  let reply = "", failure = "";
  state.abort = new AbortController();
  setBusy(true);
  try {
    const response = await fetch("/api/hermes/chat", {
      method: "POST", credentials: "same-origin", signal: state.abort.signal,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ session: state.current, message: text, spoken, interrupted: Boolean(extra.interrupted) }),
    });
    if (!response.ok) throw new Error((await response.json().catch(() => ({}))).error || `Request failed (${response.status})`);
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let pending = "";
    const stream = createStreamReader();
    for (;;) {
      const { value, done } = await reader.read();
      if (done) break;
      pending += decoder.decode(value, { stream: true });
      let cut;
      while ((cut = pending.indexOf("\n")) >= 0) {
        const piece = stream.feed(pending.slice(0, cut));
        pending = pending.slice(cut + 1);
        if (piece?.steps) showSteps(row, piece.steps);
        if (piece?.cards) showCards(row, piece.cards, () => speaker.stop());
        if (piece?.error) failure = piece.error;
        if (piece?.text) { reply += piece.text; speech?.push(piece.text); stage.react(reply); stage.caption(reply, "her"); fill(row, reply, true); toBottom(false); }
      }
    }
  } catch (error) {
    if (error.name !== "AbortError") failure = error.message || `${state.name} did not answer`;
    else speaker.stop();
  }
  if (failure) speaker.stop(); else speech?.end();
  if (failure) { row.classList.add("err"); fill(row, reply ? `${reply}\n\n${failure}` : failure, false); }
  if (reply) { state.messages.push({ role: "assistant", text: reply }); finishTools(row, reply); fill(row, reply, false); }
  else if (!failure) row.remove();
  state.abort = null;
  setBusy(false);
  talk?.replyDone();
  // Hermes names the chat after the first exchange; look again to pick that up.
  setTimeout(async () => { await refreshList(); setTitle(); }, 1500);
}

// The voice comes from Settings, Narration; the switch only works where narration is set up on the server.
async function voiceStatus() {
  const note = $("#hm-voice-name"), toggle = $("#hm-speak");
  try {
    const s = await api("/api/narration/status");
    state.voice.available = Boolean(s.available) && state.features.voice !== false;
    state.voice.id = s.voice;
    state.voice.listen = Boolean(s.listen);
    state.voice.list = s.voices || [];
    state.voice.name = (s.voices || []).find((voice) => voice.id === s.voice)?.name || s.voice || "";
  } catch (_) { state.voice.available = false; }
  toggle.disabled = !state.voice.available;
  toggle.checked = state.voice.on && state.voice.available;
  note.textContent = state.voice.available ? `Speaks each sentence as it arrives. Voice: ${state.voice.name}. Change it in Settings, Narration.` : "Narration is not set up on this server.";
  if (state.voice.on && state.voice.available) speaker.warm();
  fillVoicePick();
  setupTalk();
}

// "Voice" in the settings: the voice of the avatar on screen, or the default narration voice when there is none.
function fillVoicePick() {
  const pick = $("#hm-voice-pick");
  if (!pick) return;
  const viaAvatar = stage.avatarActive(), list = state.voice.list || [];
  $("#hm-voice-pick-opt").hidden = !state.voice.available || !list.length;
  const options = [...(viaAvatar ? [["", `Default (${state.voice.name || "narration voice"})`]] : []), ...list.map((voice) => [voice.id, voice.name || voice.id])];
  pick.replaceChildren(...options.map(([value, text]) => Object.assign(document.createElement("option"), { value, textContent: text })));
  pick.value = viaAvatar ? stage.voiceOf() : state.voice.id;
  $("#hm-voice-pick-note").textContent = viaAvatar ? "Her voice, kept for this avatar." : "The default narration voice.";
}
$("#hm-voice-pick").onchange = async (event) => {
  const value = event.target.value;
  try {
    if (stage.avatarActive()) await stage.setVoice(value);
    else { await api("/api/narration/default-voice", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ voice: value }) }); state.voice.id = value; state.voice.name = (state.voice.list || []).find((v) => v.id === value)?.name || value; }
    speaker.stop(); speaker.warm();
    $("#hm-voice-pick-note").textContent = "Saved. She uses it from her next sentence.";
  } catch (error) { $("#hm-voice-pick-note").textContent = error.message || "Could not save the voice."; }
};

// Talking to her: the microphone button, plus the hands-free and interrupt switches in the voice options.
function setupTalk() {
  const mic = $("#hm-mic");
  mic.hidden = !(state.voice.available && state.voice.listen);
  for (const [id, key] of [["#hm-hands", "bobobooks-hermes-hands"], ["#hm-barge", "bobobooks-hermes-barge"]]) {
    const box = $(id);
    box.disabled = mic.hidden;
    box.checked = read(key) === "1";
    box.onchange = () => { write(key, box.checked ? "1" : "0"); talk?.refresh(); };
  }
  if (mic.hidden || talk) return;
  const label = { off: "Talk", listening: "Listening… tap to stop", hearing: "Hearing you…", transcribing: "Understanding…", waiting: "Waiting for her…" };
  talk = createTalk({
    makeListener: (handlers) => createListener(handlers),
    speaker,
    transcribe: async (blob) => {
      const response = await fetch("/api/narration/transcribe", { method: "POST", credentials: "same-origin", headers: { "Content-Type": "audio/wav" }, body: blob });
      if (!response.ok) throw new Error(response.status === 503 ? "Speech recognition is not set up" : "Could not understand the recording");
      return (await response.json()).text || "";
    },
    send: (text, extra) => (state.abort ? Promise.resolve() : send(text, extra)),
    interrupt: () => { speaker.stop(); if (state.abort) state.abort.abort(); },
    options: () => ({ handsFree: $("#hm-hands").checked, bargeIn: $("#hm-barge").checked }),
    onMode: (mode) => {
      state.talking = mode !== "off";
      mic.classList.toggle("listening", mode === "listening" || mode === "hearing");
      mic.classList.toggle("busy", mode === "transcribing" || mode === "waiting");
      mic.setAttribute("aria-label", label[mode]); mic.title = label[mode];
      stage.setStatus(mode === "off" ? "Tap to talk" : label[mode]); stage.mic(mode === "listening" || mode === "hearing");
      if (mode !== "off") setState("ok", label[mode]); else setState("ok", "Online");
    },
    onError: (error) => setState("bad", error.name === "NotAllowedError" ? "Microphone blocked: allow it for this site" : error.message || "Microphone problem"),
  });
  const tap = () => { unlock(); speaker.warm(); void talk.toggle(); };
  mic.onclick = tap;
  $("#hm-vm-mic").onclick = tap;
}
// Voice mode (full screen): she reads aloud, and when the server can listen the conversation is hands-free.
function voiceMode(on) {
  if (on) {
    state.voice.on = state.voice.available; $("#hm-speak").checked = state.voice.on;
    write("bobobooks-hermes-speak", state.voice.on ? "1" : "0");
    if (!$("#hm-hands").disabled) { $("#hm-hands").checked = true; write("bobobooks-hermes-hands", "1"); }
    unlock(); speaker.warm();
    stage.setStatus(talk ? "Tap to talk" : "Reading her answers aloud");
    stage.caption("");
    if (talk && talk.mode() === "off") void talk.toggle();
  } else {
    talk?.stop(); speaker.stop(); stage.caption("");
  }
}
$("#hm-speak").onchange = (event) => {
  state.voice.on = event.target.checked;
  write("bobobooks-hermes-speak", state.voice.on ? "1" : "0");
  if (state.voice.on) { unlock(); speaker.warm(); } else speaker.stop();
};

async function status() {
  try {
    const s = await api("/api/hermes/status");
    setName(s.name);
    memory.setEnabled(s.memory);
    state.features = s.features || {};
    $("#hm-quick").hidden = state.features.quick === false;
    if (state.features.quick === false) $("#hm-quickmenu").hidden = true;
    if (!s.configured) setState("bad", "Not set up: open Settings, Hermes");
    else if (!s.reachable) setState("bad", s.authorized === false ? "Hermes refused the key" : `${state.name} is not reachable`);
    else setState("ok", "Online");
    await voiceStatus();
    await stage.load();
  } catch (error) { setState("bad", error.message === "Request failed (403)" ? "Administrators only" : `Cannot reach Mochil (${error.message})`); }
}

// Layout: the conversation list slides over the page on a phone.
function closeSide() { $("#hm-side").classList.remove("open"); $("#hm-scrim").hidden = true; }
$("#hm-menu").onclick = () => { $("#hm-side").classList.add("open"); $("#hm-scrim").hidden = false; };
$("#hm-scrim").onclick = closeSide;
const toggleSettings = () => {
  const panel = $("#hm-voice");
  panel.hidden = !panel.hidden;
  $("#hm-voice-btn").setAttribute("aria-expanded", String(!panel.hidden));
};
$("#hm-voice-btn").onclick = toggleSettings;
$("#hm-vm-cog").onclick = toggleSettings;
// Back goes to wherever you came from inside the app (the same window, so an installed app stays one window).
$(".hm-back").addEventListener("click", (event) => {
  if (history.length > 1 && document.referrer.startsWith(location.origin)) { event.preventDefault(); history.back(); }
});

const input = $("#hm-input");
// The box grows with what is typed, up to 40% of the window, then scrolls inside.
const grow = () => { input.style.height = "auto"; input.style.height = Math.min(input.scrollHeight, innerHeight * 0.4, 320) + "px"; };
input.addEventListener("input", grow);
input.addEventListener("keydown", (event) => {
  if (event.key === "Enter" && !event.shiftKey && !event.isComposing) { event.preventDefault(); $("#hm-form").requestSubmit(); }
});
$("#hm-form").addEventListener("submit", (event) => {
  event.preventDefault();
  const text = input.value.trim();
  if (!text || state.abort) return;
  input.value = ""; grow();
  void send(text);
});
const actions = createActions({ $, api, input, send: (text) => send(text), busy: () => Boolean(state.abort) });
$("#hm-stop").onclick = () => { state.abort?.abort(); speaker.stop(); };
$("#hm-new").onclick = startChat;
// Another device may have added or changed chats; look again when this tab comes back into view.
document.addEventListener("visibilitychange", () => { if (!document.hidden && !state.abort) void refreshList(); });

(async () => {
  await status();
  await refreshList();
  if (state.chats.some((chat) => chat.id === state.current)) await open(state.current); else drawLog();
})();
