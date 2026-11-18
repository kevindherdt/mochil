// The stage: the avatar beside the chat, the switcher between avatars, the voice each avatar speaks with, and the
// full-screen voice mode (a big avatar, small captions, one big microphone button).
import { createAvatar } from "./hermes-avatar.js?v=11";
import { DEFAULT_FRAME, clampFrame, imageScene, sceneCss, scenes, shrinkPicture } from "./hermes-scenes.js?v=6";

/** The last words worth showing as a caption: the end of the text, cut at a word. */
export function captionText(text, limit = 180) {
  const flat = String(text ?? "").replace(/\s+/g, " ").trim();
  if (flat.length <= limit) return flat;
  const cut = flat.slice(-limit);
  return "…" + cut.slice(cut.indexOf(" ") + 1);
}

export function createStage({ $, api, read, write, analyser, setState, onVoiceMode = () => {}, onChange = () => {} }) {
  const info = { models: [], selected: "", voices: {}, scene: "", frames: {}, image: 0 };
  let avatar = null, available = false, voiceMode = false;
  const KEY = "bobobooks-hermes-avatar";
  const current = () => info.models.find((model) => model.id === info.selected) || info.models[0];

  const frameOf = () => clampFrame(info.frames[info.selected] || DEFAULT_FRAME);
  const imageUrl = () => (info.image ? `/api/avatar/background?v=${info.image}` : "");
  const saveLook = (body) => api("/api/avatar/look", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });

  /** What is behind her, and the controls that match the saved look. */
  function paintLook() {
    $("#hm-stage").style.background = sceneCss(info.scene, imageUrl());
    $(".hm-main").classList.toggle("clear-stage", info.scene === "clear"); // transparent: she stands in the chat itself
    $("#hm-stage-host").dataset.corner = info.scene === "clear" ? "1" : "";
    const frame = frameOf();
    $("#hm-look-zoom").value = frame.zoom; $("#hm-look-x").value = frame.x; $("#hm-look-y").value = frame.y;
    $("#hm-scene-pick").value = info.scene;
    $("#hm-bg-remove").hidden = !info.image;
    avatar?.setFrame(frame);
  }

  function show(on) {
    $("#hm-stage").hidden = !on;
    $(".hm-main").classList.toggle("with-avatar", on);
    if (!on) { avatar?.hide(); avatar = null; onChange(); return; }
    const model = current();
    if (!model) return;
    avatar ??= createAvatar({ host: $("#hm-stage-host"), analyser, onError: (error) => setState("bad", `Avatar: ${error.message}`) });
    avatar.setFrame(frameOf());
    paintLook();
    void avatar.show(model);
    onChange();
  }

  function setVoiceMode(on) {
    voiceMode = on && available;
    $(".hm-main").classList.toggle("voicemode", voiceMode);
    $("#hm-vm").hidden = !voiceMode;
    if (voiceMode) show(true); else show($("#hm-avatar-on").checked);
    if (voiceMode) document.documentElement.requestFullscreen?.().catch(() => {});
    else if (document.fullscreenElement) document.exitFullscreen?.().catch(() => {});
    onVoiceMode(voiceMode);
  }

  const stage = {
    available: () => available,
    voiceMode: () => voiceMode,
    /** True while an avatar is on screen, so "her voice" means the voice of that avatar. */
    avatarActive: () => available && $("#hm-stage").hidden === false,
    voiceOf: () => info.voices[info.selected] || "",
    async setVoice(voice) {
      const model = info.selected;
      await api("/api/avatar/select", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ model, voice }) });
      if (voice) info.voices[model] = voice; else delete info.voices[model];
      stage.voiceChanged();
    },
    /** The narration voice for what she says now: the avatar's own, or the default. */
    voiceFor: (fallback) => (available && $("#hm-stage").hidden === false && info.voices[info.selected]) || fallback,
    react: (text) => avatar?.react(text),
    calm: () => avatar?.calm(),
    starting: () => avatar?.voiceStarting(),
    caption(text, who = "") {
      const line = $("#hm-caption");
      line.textContent = text ? captionText(text) : "";
      line.dataset.who = who;
    },
    setStatus(text) { $("#hm-vm-state").textContent = text; },
    mic(listening) { $("#hm-vm-mic").classList.toggle("on", listening); },
    async load() {
      try {
        const data = await api("/api/avatar/models");
        info.models = data.models || []; info.selected = data.selected; info.voices = data.voices || {};
        info.scene = data.scene || ""; info.frames = data.frames || {}; info.image = Number(data.image) || 0;
        available = Boolean(data.core) && info.models.length > 0;
      } catch (_) { available = false; }
      const box = $("#hm-avatar-on"), pick = $("#hm-avatar-pick");
      $("#hm-avatar-opt").hidden = $("#hm-avatar-pick-opt").hidden = !available;
      $("#hm-full").hidden = !available;
      $("#hm-scene-opt").hidden = $("#hm-look-opt").hidden = !available;
      if (!available) return;
      stage.wireLook();
      paintLook();
      box.checked = read(KEY) === "1";
      pick.replaceChildren(...info.models.map((model) => Object.assign(document.createElement("option"), { value: model.id, textContent: model.id === "." ? model.name : model.id.replace(/\/(runtime|[^/]+)$/, (m, last) => (last === "runtime" ? "" : m)), selected: model.id === info.selected })));
      box.onchange = () => { write(KEY, box.checked ? "1" : "0"); show(box.checked); stage.starting(); };
      pick.onchange = async () => {
        info.selected = pick.value;
        try { await api("/api/avatar/select", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ model: pick.value }) }); } catch (_) { /* it still shows for now */ }
        if (!box.checked && !voiceMode) { box.checked = true; write(KEY, "1"); }
        show(true);
        stage.voiceChanged();
      };
      $("#hm-full").onclick = () => setVoiceMode(true);
      $("#hm-vm-exit").onclick = () => setVoiceMode(false);
      document.addEventListener("keydown", (event) => { if (event.key === "Escape" && voiceMode) setVoiceMode(false); });
      document.addEventListener("fullscreenchange", () => { if (!document.fullscreenElement && voiceMode) setVoiceMode(false); });
      show(box.checked);
      stage.voiceChanged();
    },
    /** The background picker, the position sliders and the picture upload. Called once, when avatars are available. */
    wireLook() {
      if (stage.wired) return;
      stage.wired = true;
      const status = (text) => { $("#hm-look-status").textContent = text; };
      const pick = $("#hm-scene-pick");
      const options = [...scenes, imageScene].map((scene) => Object.assign(document.createElement("option"), { value: scene.id, textContent: scene.label }));
      pick.replaceChildren(...options);
      pick.onchange = async () => {
        const wanted = pick.value;
        if (wanted === "image" && !info.image) { $("#hm-bg-file").click(); pick.value = info.scene; return; }
        try { await saveLook({ scene: wanted }); info.scene = wanted; status(""); } catch (error) { status(error.message); pick.value = info.scene; }
        paintLook();
      };
      let timer = 0;
      const moved = () => {
        const frame = clampFrame({ zoom: $("#hm-look-zoom").value, x: $("#hm-look-x").value, y: $("#hm-look-y").value });
        const model = info.selected; // the model this move belongs to, even if she is switched before the save fires
        info.frames[model] = frame;
        avatar?.setFrame(frame);
        clearTimeout(timer);
        timer = setTimeout(() => saveLook({ frame: { model, ...frame } }).then(() => status(""), (error) => status(error.message)), 400);
      };
      for (const id of ["zoom", "x", "y"]) $(`#hm-look-${id}`).oninput = moved;
      $("#hm-look-reset").onclick = () => { for (const [id, value] of [["zoom", 1], ["x", 0], ["y", 0]]) $(`#hm-look-${id}`).value = value; moved(); };
      $("#hm-bg-upload").onclick = () => $("#hm-bg-file").click();
      $("#hm-bg-file").onchange = async (event) => {
        const file = event.target.files?.[0];
        event.target.value = "";
        if (!file) return;
        status("Preparing the picture…");
        try {
          const blob = await shrinkPicture(file);
          const sent = await api("/api/avatar/background", { method: "PUT", headers: { "Content-Type": blob.type }, body: blob });
          info.image = Number(sent.version) || Date.now();
          await saveLook({ scene: "image" });
          info.scene = "image";
          status("Picture saved.");
        } catch (error) { status(error.message || "Could not use that picture."); }
        paintLook();
      };
      $("#hm-bg-remove").onclick = async () => {
        try {
          const response = await fetch("/api/avatar/background", { method: "DELETE", credentials: "same-origin" });
          if (!response.ok) throw new Error(`Request failed (${response.status})`);
          info.image = 0; if (info.scene === "image") info.scene = "";
          status("Picture removed.");
        } catch (error) { status(error.message); }
        paintLook();
      };
    },
    /** Tell the person which voice the chosen avatar speaks with. */
    voiceChanged() { $("#hm-avatar-voice-note").textContent = info.voices[info.selected] ? "Speaks with the voice set for her in Settings, Avatar." : "Speaks with your default voice. Choose another in Settings, Avatar."; },
    setVoiceMode,
  };
  return stage;
}
