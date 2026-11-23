// features/lyana/api/speech.ts — speaking a message out loud.
//
// Two voices, in order of preference. The server's own narration voice when it is set up (`POST /api/narration/stream`
// answers with 16-bit mono PCM and an `X-Sample-Rate` header), because that is the voice the companion was given and the
// one the avatar's mouth is drawn for. Otherwise the browser's `speechSynthesis`, which needs no server and works on
// every phone — a read-aloud button that does nothing on a server without narration would be chrome, and one that only
// appears once narration is configured would be invisible to most people.
//
// The PCM player is a port of the legacy `static/hermes-voice.js` createPlayer: pieces are scheduled back to back on one
// AudioContext so audio starts with the first piece instead of after the last one arrives.
import { fetchNarrationStatus } from "./companion";

const MAX_CHUNK = 1200;

export interface Speaker {
  /** Reads the text, resolving when it has finished or was stopped. */
  speak: (text: string) => Promise<void>;
  stop: () => void;
  /** True while something is being spoken. */
  speaking: () => boolean;
  /** Which voice will be used, for the button's label. */
  source: "server" | "browser";
}

/** Splits at sentence ends so a long answer starts speaking at once and never exceeds the endpoint's body limit. */
export function speechChunks(text: string): string[] {
  const trimmed = text.trim();
  if (trimmed.length <= MAX_CHUNK) return trimmed === "" ? [] : [trimmed];
  const chunks: string[] = [];
  let rest = trimmed;
  while (rest.length > MAX_CHUNK) {
    const window = rest.slice(0, MAX_CHUNK);
    const cut = Math.max(window.lastIndexOf(". "), window.lastIndexOf("! "), window.lastIndexOf("? "), window.lastIndexOf("\n"));
    const end = cut > MAX_CHUNK / 3 ? cut + 1 : MAX_CHUNK;
    chunks.push(rest.slice(0, end).trim());
    rest = rest.slice(end).trim();
  }
  if (rest !== "") chunks.push(rest);
  return chunks;
}

interface PcmPlayer {
  play: (bytes: Uint8Array, sampleRate: number) => void;
  stop: () => void;
  idle: () => Promise<void>;
  unlock: () => void;
}

function createPcmPlayer(): PcmPlayer {
  let context: AudioContext | null = null;
  let sources: AudioBufferSourceNode[] = [];
  let next = 0;

  const ensure = (): AudioContext => {
    if (context === null) {
      context = new AudioContext();
    }
    return context;
  };

  return {
    unlock() {
      try {
        void ensure().resume();
      } catch {
        // No audio on this device: speaking simply will not produce sound.
      }
    },
    play(bytes, sampleRate) {
      const ctx = ensure();
      const count = Math.floor(bytes.length / 2);
      if (count === 0) return;
      const view = new DataView(bytes.buffer, bytes.byteOffset, count * 2);
      const buffer = ctx.createBuffer(1, count, sampleRate);
      const samples = buffer.getChannelData(0);
      for (let i = 0; i < count; i += 1) samples[i] = view.getInt16(i * 2, true) / 32768;
      const source = ctx.createBufferSource();
      source.buffer = buffer;
      source.connect(ctx.destination);
      const start = Math.max(ctx.currentTime + 0.04, next);
      source.start(start);
      next = start + buffer.duration;
      sources.push(source);
      source.onended = () => {
        sources = sources.filter((item) => item !== source);
      };
    },
    async idle() {
      // Wait for the scheduled tail to finish, so the caller can flip its button back at the right moment.
      while (context !== null && sources.length > 0) {
        await new Promise((resolve) => setTimeout(resolve, 60));
      }
    },
    stop() {
      for (const source of sources) {
        try {
          source.onended = null;
          source.stop();
        } catch {
          // Already finished.
        }
      }
      sources = [];
      next = 0;
      if (typeof speechSynthesis !== "undefined") speechSynthesis.cancel();
    },
  };
}

/** Joins whole 16-bit samples from byte pieces that may split a sample in two. */
function sampleJoiner(): (bytes: Uint8Array) => Uint8Array {
  let carry: Uint8Array | null = null;
  return (bytes) => {
    let data = bytes;
    if (carry !== null) {
      data = new Uint8Array(carry.length + bytes.length);
      data.set(carry);
      data.set(bytes, carry.length);
      carry = null;
    }
    if (data.length % 2 === 1) {
      carry = data.slice(data.length - 1);
      data = data.slice(0, data.length - 1);
    }
    return data;
  };
}

/**
 * Builds a speaker. The server is asked once whether it can narrate; the answer decides which voice is used, and the
 * browser's own synthesis is the fallback rather than an error message.
 */
export async function createSpeaker(options: { voice?: () => string } = {}): Promise<Speaker> {
  const voice = options.voice ?? (() => "");
  let server = false;
  try {
    const status = await fetchNarrationStatus();
    server = status.enabled && status.available;
  } catch {
    server = false;
  }
  const browserVoice = typeof speechSynthesis !== "undefined" && typeof SpeechSynthesisUtterance !== "undefined";
  const player = createPcmPlayer();
  let controller: AbortController | null = null;
  let active = false;

  const stop = () => {
    controller?.abort();
    controller = null;
    player.stop();
    active = false;
  };

  const speakFromBrowser = (text: string) =>
    new Promise<void>((resolve) => {
      const utterance = new SpeechSynthesisUtterance(text);
      utterance.lang = "en";
      utterance.onend = () => resolve();
      utterance.onerror = () => resolve();
      speechSynthesis.speak(utterance);
    });

  const speak = async (text: string) => {
    const chunks = speechChunks(text);
    if (chunks.length === 0) return;
    stop();
    active = true;
    controller = new AbortController();
    const mine = controller;
    try {
      if (server) {
        player.unlock();
        for (const chunk of chunks) {
          if (mine.signal.aborted) return;
          const response = await fetch("/api/narration/stream", {
            method: "POST",
            credentials: "same-origin",
            signal: mine.signal,
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ text: chunk, voice: voice() }),
          });
          if (!response.ok || response.body === null) {
            // The server says it can narrate but refused this one: fall back rather than staying silent.
            await speakFromBrowser(chunk);
            continue;
          }
          const rate = Number(response.headers.get("X-Sample-Rate") ?? "24000") || 24000;
          const reader = response.body.getReader();
          const whole = sampleJoiner();
          for (;;) {
            const { value, done } = await reader.read();
            if (done || mine.signal.aborted) break;
            if (value === undefined) continue;
            const bytes = whole(value);
            if (bytes.length > 0) player.play(bytes, rate);
          }
        }
        await player.idle();
      } else if (browserVoice) {
        for (const chunk of chunks) {
          if (mine.signal.aborted) return;
          await speakFromBrowser(chunk);
        }
      }
    } catch (error) {
      if (!(error instanceof DOMException && error.name === "AbortError")) throw error;
    } finally {
      if (controller === mine) {
        controller = null;
        active = false;
      }
    }
  };

  return {
    speak,
    stop,
    speaking: () => active,
    source: server ? "server" : "browser",
  };
}
