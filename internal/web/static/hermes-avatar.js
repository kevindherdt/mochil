import { DEFAULT_FRAME, clampFrame, placeModel } from "./hermes-scenes.js?v=6";

// The avatar: a Live2D character that blinks, sways, shows a mood and moves her mouth with her voice.
// The decisions (which mood, which expression, how open the mouth is) are plain functions so they can be tested;
// the runtime part needs a browser with WebGL and the Cubism core, and is loaded only when the avatar is switched on.

/** A mood from what she just said, or null. Deliberately small: a few clear cases beat guessing at every sentence. */
export function moodFor(text) {
  const said = String(text ?? "").toLowerCase().slice(-220);
  if (!said.trim()) return null;
  if (/(love you|i love|adore you|my love|darling|❤|♥|😍)/.test(said)) return "love";
  if (/(blush|flustered|you're sweet|you are sweet|so sweet|you flatter|miss you|how cute|you're cute)/.test(said)) return "blush";
  if (/(furious|so angry|i'm angry|annoyed|how dare|that's rude)/.test(said)) return "angry";
  if (/(sorry to hear|i'm so sorry|that's sad|makes me sad|heartbroken|crying)/.test(said)) return "sad";
  if (/(oh my|no way|wow\b|you're kidding|seriously\?|what\?!)/.test(said)) return "surprise";
  if (/(hmm+|not sure what you mean|confused|what do you mean)/.test(said)) return "question";
  return null;
}

// What an expression file might be called (English, or the Chinese names this kind of model usually ships with).
const EXPRESSION_WORDS = {
  love: ["love", "heart", "爱心", "爱心眼"],
  blush: ["blush", "shy", "embarrass", "脸红"],
  angry: ["angry", "mad", "生气"],
  sad: ["tear", "cry", "sad", "流泪"],
  surprise: ["surprise", "shock", "惊讶"],
  question: ["question", "doubt", "confus", "疑惑"],
};

/** The expression file name that fits a mood, from the names a model offers; null when it has none that fits. */
export function expressionFor(mood, names) {
  const words = EXPRESSION_WORDS[mood] || [];
  return (names || []).find((name) => words.some((word) => name.toLowerCase().includes(word))) ?? null;
}

/** Mouth openness (0..1) from the loudness of her voice. It rises fast and falls slowly, which looks like speech. */
export function mouthStep(previous, level, dt) {
  const target = Math.max(0, Math.min(1, (level - 0.015) * 7));
  const rate = target > previous ? 28 : 11;
  return previous + (target - previous) * Math.min(1, rate * dt);
}

/** Seconds until the next blink: usually 2 to 6 seconds, sometimes a quick double. */
export function nextBlink(random = Math.random) {
  return 2 + random() * 4;
}

/** Eye openness (0..1) at `t` seconds into a blink that lasts `length` seconds. */
export function blinkOpenness(t, length = 0.18) {
  if (t < 0 || t > length) return 1;
  const half = length / 2;
  return t < half ? 1 - t / half : (t - half) / half;
}

const loadScript = (src) => new Promise((resolve, reject) => {
  const script = document.createElement("script");
  script.src = src;
  script.onload = resolve;
  script.onerror = () => reject(new Error(`Could not load ${src.split("?")[0]}`));
  document.head.append(script);
});

let runtime = null;
async function loadRuntime() {
  runtime ??= (async () => {
    await loadScript("/api/avatar/core");
    await loadScript("/assets/vendor/avatar/pixi.min.js?v=1");
    await loadScript("/assets/vendor/avatar/unsafe-eval.min.js?v=1");
    await loadScript("/assets/vendor/avatar/cubism4.min.js?v=1");
    window.PIXI.live2d.Live2DModel.registerTicker(window.PIXI.Ticker);
    return window.PIXI;
  })();
  try { return await runtime; } catch (error) { runtime = null; throw error; }
}

const encodePath = (path) => path.split("/").map(encodeURIComponent).join("/");

/** Draws one model into `host`. `analyser()` returns the Web Audio analyser her voice plays through, once there is one. */
export function createAvatar({ host, analyser, onError = () => {} }) {
  let app = null, model = null, watcher = null, mouth = 0, samples = null, frame = DEFAULT_FRAME;
  let blinkIn = nextBlink(), blinkAt = -1, clock = 0, loadToken = 0, reset = 0, names = [];
  const params = { sway: 0 };

  // Her voice is played through Web Audio; its analyser gives the loudness that drives the mouth.
  function level() {
    const node = analyser?.();
    if (!node) return 0;
    if (!samples || samples.length !== node.fftSize) samples = new Float32Array(node.fftSize);
    node.getFloatTimeDomainData(samples);
    let sum = 0;
    for (const value of samples) sum += value * value;
    return Math.sqrt(sum / samples.length);
  }

  // Runs every frame after the model's own motions, before it is drawn.
  function animate() {
    const core = model.internalModel.coreModel, dt = Math.min(0.1, app.ticker.deltaMS / 1000);
    clock += dt;
    mouth = mouthStep(mouth, level(), dt);
    core.setParameterValueById("ParamMouthOpenY", mouth);
    host.dataset.mouth = mouth.toFixed(2); // read by the browser test
    blinkIn -= dt;
    if (blinkIn <= 0) { blinkAt = 0; blinkIn = nextBlink(); }
    if (blinkAt >= 0) { blinkAt += dt; if (blinkAt > 0.18) blinkAt = -1; }
    const open = blinkOpenness(blinkAt);
    if (open < 1) { core.setParameterValueById("ParamEyeLOpen", open); core.setParameterValueById("ParamEyeROpen", open); }
    // A slow sway, so she is never frozen.
    core.setParameterValueById("ParamAngleX", Math.sin(clock * 0.6) * 5 + params.sway);
    core.setParameterValueById("ParamAngleY", Math.sin(clock * 0.43 + 1) * 3);
    core.setParameterValueById("ParamAngleZ", Math.sin(clock * 0.37 + 2) * 2);
    core.setParameterValueById("ParamEyeBallX", Math.sin(clock * 0.5) * 0.25);
    if (reset && clock > reset) { reset = 0; model.internalModel.motionManager.expressionManager?.resetExpression(); }
  }

  function fit() {
    if (!app || !model) return;
    const width = host.clientWidth, height = host.clientHeight;
    if (!width || !height) return;
    app.renderer.resize(width, height);
    const original = model.internalModel;
    // By default roughly the top 60% of the model (head and shoulders) shows; the frame zooms and moves it.
    const place = placeModel({ width, height, modelWidth: original.originalWidth, modelHeight: original.originalHeight, frame, corner: host.dataset.corner === "1" });
    model.scale.set(place.scale);
    model.x = place.x;
    model.y = place.y;
  }

  const api = {
    /** Shows a model. Safe to call again with another one. */
    async show(info) {
      const token = ++loadToken;
      try {
        const PIXI = await loadRuntime();
        if (token !== loadToken) return;
        if (!app) {
          const canvas = document.createElement("canvas");
          host.append(canvas);
          app = new PIXI.Application({ view: canvas, backgroundAlpha: 0, antialias: true, autoDensity: true, resolution: window.devicePixelRatio || 1, width: host.clientWidth || 300, height: host.clientHeight || 300 });
          watcher = new ResizeObserver(fit);
          watcher.observe(host);
        }
        const base = `/api/avatar/files/${encodePath(info.id === "." ? "" : info.id + "/")}`.replace(/\/$/, "/");
        const settings = await (await fetch(base + encodeURIComponent(info.model), { credentials: "same-origin" })).json();
        settings.url = base + encodeURIComponent(info.model);
        // The expression files sit next to the model but are not listed in its manifest; list them so they can be used.
        settings.FileReferences.Expressions = info.expressions.map((e) => ({ Name: e.name, File: e.file }));
        names = info.expressions.map((e) => e.name);
        const next = await PIXI.live2d.Live2DModel.from(settings, { autoInteract: false });
        if (token !== loadToken) { next.destroy(); return; }
        if (model) { app.stage.removeChild(model); model.destroy(); }
        model = next;
        model.internalModel.on("beforeModelUpdate", animate);
        app.stage.addChild(model);
        fit();
      } catch (error) { onError(error); }
    },
    /** Zoom and move the model in its frame ({zoom, x, y}). */
    setFrame(next) { frame = clampFrame(next); fit(); },
    /** Her voice is about to play: connect the mouth to it (needs a tap or key press first). */
    voiceStarting() {},
    /** Show the mood of what she is saying; it fades after a few seconds. */
    react(text) {
      const mood = moodFor(text), file = mood && expressionFor(mood, names);
      if (!file || !model) return;
      void model.expression(file).catch(() => {});
      reset = clock + 7;
    },
    calm() { reset = clock + 0.01; },
    hide() { loadToken += 1; watcher?.disconnect(); watcher = null; if (app) { app.destroy(true, { children: true }); app = null; model = null; host.replaceChildren(); } },
  };
  return api;
}
