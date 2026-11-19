// Listening for the Hermes chat: finds where you start and stop talking, and hands each utterance over as a small WAV.
// No audio leaves the browser until a whole utterance is finished; then it goes to this server only.

/** Voice activity detection on 20 ms loudness readings. Speech starts when it stays above the room's noise for a moment
 *  and ends after a short silence, so there is no long fixed wait. The noise floor follows the room while nobody talks. */
export function createVad({ minSpeechMs = 240, silenceMs = 800, maxMs = 55000, factor = 3, floorMin = 0.006, threshold = 0.02 } = {}) {
  let floor = floorMin, loud = 0, speaking = false, speechMs = 0, quiet = 0;
  const limit = () => Math.max(threshold, floor * factor);
  return {
    sensitivity({ factor: f, minSpeech }) { if (f) factor = f; if (minSpeech) minSpeechMs = minSpeech; },
    reset() { loud = 0; speaking = false; speechMs = 0; quiet = 0; },
    speaking: () => speaking,
    /** One loudness reading; returns "start" or "end" when the state changes. */
    feed(rms, dt) {
      if (!speaking) {
        if (rms > limit()) loud += dt; else { loud = 0; floor = Math.min(0.05, Math.max(floorMin, floor * 0.95 + rms * 0.05)); }
        if (loud >= minSpeechMs) { speaking = true; speechMs = loud; quiet = 0; return "start"; }
        return null;
      }
      speechMs += dt;
      if (rms > limit() * 0.6) quiet = 0; else quiet += dt;
      if (quiet >= silenceMs || speechMs >= maxMs) { speaking = false; loud = 0; return "end"; }
      return null;
    },
  };
}

/** Mono 16 kHz, 16-bit WAV from blocks of samples at the microphone's own rate. */
export function encodeWav(blocks, inputRate) {
  let total = 0;
  for (const block of blocks) total += block.length;
  const ratio = inputRate / 16000;
  const count = Math.floor(total / ratio);
  const pcm = new Int16Array(count);
  let index = 0, position = 0, sum = 0, taken = 0, next = ratio;
  for (const block of blocks) {
    for (let i = 0; i < block.length && index < count; i += 1) {
      sum += block[i]; taken += 1; position += 1;
      if (position >= next) {
        const value = Math.max(-1, Math.min(1, sum / taken));
        pcm[index++] = value < 0 ? value * 0x8000 : value * 0x7fff;
        sum = 0; taken = 0; next += ratio;
      }
    }
  }
  const out = new DataView(new ArrayBuffer(44 + pcm.length * 2));
  const text = (offset, value) => { for (let i = 0; i < value.length; i += 1) out.setUint8(offset + i, value.charCodeAt(i)); };
  text(0, "RIFF"); out.setUint32(4, 36 + pcm.length * 2, true); text(8, "WAVE"); text(12, "fmt ");
  out.setUint32(16, 16, true); out.setUint16(20, 1, true); out.setUint16(22, 1, true);
  out.setUint32(24, 16000, true); out.setUint32(28, 32000, true); out.setUint16(32, 2, true); out.setUint16(34, 16, true);
  text(36, "data"); out.setUint32(40, pcm.length * 2, true);
  for (let i = 0; i < pcm.length; i += 1) out.setInt16(44 + i * 2, pcm[i], true);
  return new Blob([out.buffer], { type: "audio/wav" });
}

const rmsOf = (block) => { let sum = 0; for (let i = 0; i < block.length; i += 1) sum += block[i] * block[i]; return Math.sqrt(sum / block.length); };

/** Opens the microphone and reports utterances. setMuted() ignores the microphone (for example while she speaks);
 *  hearAbove() makes the detector pickier while she speaks, so her own voice is not taken for yours. */
export function createListener({ onUtterance, onSpeechStart = () => {}, onLevel = () => {}, onError = () => {}, vad = createVad(), workletUrl = "/assets/hermes-capture-worklet.js?v=1" } = {}) {
  let context = null, stream = null, node = null, muted = false;
  let preroll = [], utterance = null, blockMs = 20, rate = 48000;
  const PREROLL = 28; // about 560 ms kept from before speech was confirmed

  function onBlock(block) {
    const level = rmsOf(block);
    onLevel(muted ? 0 : level);
    if (muted) return;
    const event = vad.feed(level, blockMs);
    if (utterance) utterance.push(block); else { preroll.push(block); if (preroll.length > PREROLL) preroll.shift(); }
    if (event === "start") { utterance = preroll.slice(); preroll = []; onSpeechStart(); }
    else if (event === "end" && utterance) {
      // Keep only a little of the closing silence.
      const keep = Math.max(1, utterance.length - Math.floor((vad.silenceTrim ?? 700) / blockMs));
      const blocks = utterance.slice(0, keep + 10);
      utterance = null;
      onUtterance(encodeWav(blocks, rate), blocks.length * blockMs);
    }
  }

  return {
    async start() {
      if (context) return;
      try {
        stream = await navigator.mediaDevices.getUserMedia({ audio: { echoCancellation: true, noiseSuppression: true, autoGainControl: true } });
        context = new AudioContext();
        rate = context.sampleRate;
        await context.audioWorklet.addModule(workletUrl);
        node = new AudioWorkletNode(context, "hermes-capture");
        blockMs = Math.max(128, Math.round(rate * 0.02)) / rate * 1000;
        node.port.onmessage = (event) => onBlock(event.data);
        context.createMediaStreamSource(stream).connect(node);
        vad.reset(); preroll = []; utterance = null;
      } catch (error) {
        this.stop();
        onError(error);
        throw error;
      }
    },
    stop() {
      node?.disconnect(); stream?.getTracks().forEach((track) => track.stop()); context?.close?.().catch?.(() => {});
      node = stream = context = null; utterance = null; preroll = []; vad.reset(); onLevel(0);
    },
    setMuted(value) { if (muted === value) return; muted = value; if (value) { vad.reset(); utterance = null; preroll = []; } },
    hearAbove(on) { vad.sensitivity(on ? { factor: 5, minSpeech: 420 } : { factor: 3, minSpeech: 240 }); },
    active: () => Boolean(context),
  };
}
