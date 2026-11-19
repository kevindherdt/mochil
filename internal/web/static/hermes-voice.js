// Speaking for the Hermes chat. Replies are cut into sentences as they stream and each sentence is made into speech by
// Pocket TTS on this server (/api/narration/speak), then played in order. The next sentences are requested while the
// current one plays, so speech starts after the first sentence and keeps up (the voice makes speech a little faster
// than real time once it is loaded).

const EMOJI = /[\p{Extended_Pictographic}‍️]/gu;

/** What is worth hearing: no code, links, markdown marks, emoji, or stage directions such as *smiles softly*. */
export function speechText(raw) {
  return String(raw ?? "")
    .replace(/```[\s\S]*?```/g, " ")
    .replace(/`([^`]*)`/g, "$1")
    .replace(/\[([^\]]*)\]\([^)]*\)/g, "$1")
    .replace(/https?:\/\/\S+/g, "")
    .replace(/\*\*([^*]+)\*\*/g, "$1")
    .replace(/\*([^*]+)\*/g, (all, inner) => (inner.trim().split(/\s+/).length >= 2 ? " " : inner))
    .replace(/^\s{0,3}#{1,6}\s+/gm, "")
    .replace(/^\s*[-*•]\s+/gm, "")
    .replace(/^\s*\d+[.)]\s+/gm, "")
    .replace(EMOJI, "")
    .replace(/[_~>|]/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

/** Takes complete sentences off the front of a growing text. A sentence is complete once its end mark is followed by a
 *  space or a new line; short ones wait and join the next, so nothing tiny is spoken alone. With final, the rest is flushed. */
export function takeSentences(buffer, { final = false, minimum = 24 } = {}) {
  const sentences = [];
  const boundary = /[.!?…]+["'”)\]]*\s+|\n+/g;
  let current = "";
  let last = 0;
  let match;
  while ((match = boundary.exec(buffer))) {
    const end = match.index + match[0].length;
    current += buffer.slice(last, end);
    last = end;
    if (speechText(current).length >= minimum) { sentences.push(current); current = ""; }
  }
  let rest = current + buffer.slice(last);
  if (final) { if (speechText(rest)) sentences.push(rest); rest = ""; }
  return { sentences: sentences.map(speechText).filter(Boolean), rest };
}

/** Plays streamed speech through Web Audio: each piece is scheduled right after the one before, so sound starts with
 *  the first piece and the avatar can listen to the same output (analyser()). */
export function createPlayer({ AudioContextClass = globalThis.AudioContext } = {}) {
  let context = null, analyser = null, sources = [], next = 0;
  const ensure = () => {
    if (!context) {
      context = new AudioContextClass();
      analyser = context.createAnalyser();
      analyser.fftSize = 512;
      analyser.connect(context.destination);
    }
    return context;
  };
  return {
    /** Call from a tap or key press so the browser allows sound. */
    unlock() { try { void ensure().resume(); } catch (_) { /* no audio on this device */ } },
    analyser: () => analyser,
    /** Schedules one piece of 16-bit little-endian mono audio; resolves when it has finished playing. */
    play(bytes, sampleRate) {
      const ctx = ensure();
      const count = Math.floor(bytes.length / 2);
      if (!count) return Promise.resolve();
      const view = new DataView(bytes.buffer, bytes.byteOffset, count * 2);
      const buffer = ctx.createBuffer(1, count, sampleRate);
      const samples = buffer.getChannelData(0);
      for (let i = 0; i < count; i += 1) samples[i] = view.getInt16(i * 2, true) / 32768;
      const source = ctx.createBufferSource();
      source.buffer = buffer;
      source.connect(analyser);
      const start = Math.max(ctx.currentTime + 0.03, next);
      source.start(start);
      next = start + buffer.duration;
      sources.push(source);
      return new Promise((resolve) => { source.onended = () => { sources = sources.filter((item) => item !== source); resolve(); }; });
    },
    /** Silences everything that was scheduled. */
    stop() { for (const source of sources) { try { source.onended = null; source.stop(); } catch (_) { /* already over */ } } sources = []; next = 0; },
    /** Seconds of sound still scheduled ahead. */
    queued: () => (context ? Math.max(0, next - context.currentTime) : 0),
  };
}

/** Joins whole 16-bit samples from a stream of byte pieces that may split a sample in two. */
export function sampleSplitter() {
  let carry = null;
  return (bytes) => {
    let data = bytes;
    if (carry) { data = new Uint8Array(carry.length + bytes.length); data.set(carry); data.set(bytes, carry.length); carry = null; }
    if (data.length % 2) { carry = data.slice(data.length - 1); data = data.slice(0, data.length - 1); }
    return data;
  };
}

export function createSpeaker({ fetchImpl = (...args) => fetch(...args), player = createPlayer(), voice = () => undefined, onChange = () => {}, prebuffer = 0.25 } = {}) {
  let generation = 0;
  let controller = null;
  let queue = [];        // sentences in speaking order: { pieces, done, error, wake }
  let pumping = false;   // scheduling sentences
  let pending = 0;       // pieces scheduled that have not finished playing
  let finished = true;   // the reply has ended, so an empty queue means silence
  const speaker = { speaking: () => pumping || pending > 0 || queue.length > 0 || !finished };

  const emit = () => onChange(speaker.speaking());

  // Starts fetching a sentence at once and collects its audio as it arrives.
  function fetchSentence(text, mine) {
    const item = { pieces: [], rate: 24000, done: false, error: null, wake: null };
    const notify = () => { const wake = item.wake; item.wake = null; wake?.(); };
    (async () => {
      try {
        const response = await fetchImpl("/api/narration/stream", {
          method: "POST", credentials: "same-origin", signal: controller.signal,
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ text, voice: voice() }),
        });
        if (!response.ok) throw new Error((await response.text().catch(() => "")).trim() || `Speech failed (${response.status})`);
        item.rate = Number(response.headers.get("X-Sample-Rate")) || 24000;
        const reader = response.body.getReader(), whole = sampleSplitter();
        for (;;) {
          const { value, done } = await reader.read();
          if (done || mine !== generation) break;
          const bytes = whole(value);
          if (bytes.length) { item.pieces.push(bytes); notify(); }
        }
      } catch (error) { item.error = error; }
      item.done = true;
      notify();
    })();
    return item;
  }

  // Schedules sentences in order as their audio arrives, without waiting for earlier audio to finish playing.
  async function pump(mine) {
    if (pumping) return;
    pumping = true;
    emit();
    while (queue.length && mine === generation) {
      const item = queue[0];
      let first = true;
      for (;;) {
        if (mine !== generation) break;
        if (item.pieces.length) {
          // The first piece waits until a little is buffered, so the start does not stutter.
          if (first && !item.done && item.pieces.reduce((n, p) => n + p.length, 0) / 2 / item.rate < prebuffer) { await new Promise((resolve) => { item.wake = resolve; }); continue; }
          first = false;
          pending += 1;
          player.play(item.pieces.shift(), item.rate).then(() => {
            if (mine !== generation) return;
            pending -= 1;
            if (!pending && !pumping) emit();
          });
        } else if (item.done) break;
        else await new Promise((resolve) => { item.wake = resolve; });
      }
      queue.shift();
    }
    if (mine === generation) { pumping = false; emit(); }
  }

  function enqueue(text, mine) {
    queue.push(fetchSentence(text, mine));
    pump(mine);
  }

  Object.assign(speaker, {
    /** Call from a tap or key press, before the first sound, so the browser allows playback. */
    unlock() { player.unlock(); },
    analyser: () => player.analyser(),
    warm() { fetchImpl("/api/narration/warm", { method: "POST", credentials: "same-origin", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ voice: voice() }) }).catch(() => {}); },
    stop() {
      generation += 1;
      controller?.abort();
      controller = null;
      queue = [];
      pumping = false;
      pending = 0;
      finished = true;
      player.stop();
      emit();
    },
    /** Starts speaking a reply that is still arriving: push() text as it streams, then end(). */
    begin() {
      speaker.stop();
      const mine = generation;
      controller = new AbortController();
      finished = false;
      let buffer = "";
      return {
        push(text) { if (mine !== generation) return; buffer += text; const { sentences, rest } = takeSentences(buffer); buffer = rest; for (const s of sentences) enqueue(s, mine); },
        end() { if (mine !== generation) return; const { sentences } = takeSentences(buffer, { final: true }); buffer = ""; for (const s of sentences) enqueue(s, mine); finished = true; if (!queue.length && !pumping && !pending) emit(); },
      };
    },
    /** Reads a finished message from the start. */
    speakText(text) {
      const reply = speaker.begin();
      reply.push(text);
      reply.end();
    },
  });
  return speaker;
}
