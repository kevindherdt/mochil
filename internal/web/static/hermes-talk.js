// The voice conversation: listen, understand, send, wait for the answer, listen again. Pure decisions live here so they
// can be tested; the page supplies the microphone, the transcription call, sending and interrupting.

const PHANTOM = new Set(["thanks for watching", "thank you for watching", "please subscribe", "subscribe", "you", "the end", "blank audio", "silence"]);
const STOP = new Set(["stop", "stop listening", "stop talking", "stop the conversation", "end conversation"]);

export const normalize = (text) => String(text ?? "").toLowerCase().replace(/\[[^\]]*\]/g, " ").replace(/[^\p{L}\p{N}' ]/gu, " ").replace(/\s+/g, " ").trim();

/** What to do with what was heard: nothing (silence or a Whisper phantom), end the conversation, or say it to her. */
export function classify(text, ms = 0) {
  const heard = normalize(text);
  if (!heard) return "empty";
  if (STOP.has(heard)) return "stop";
  if (PHANTOM.has(heard) && ms < 2500) return "phantom";
  return "say";
}

export function createTalk({ makeListener, speaker, transcribe, send, interrupt, options, onMode = () => {}, onError = () => {} }) {
  let mode = "off";
  let listener = null;
  let interrupted = false;

  const set = (next) => { mode = next; refresh(); onMode(next); };

  /** Muted while she answers unless you may talk over her; pickier while her voice is playing. */
  function refresh() {
    if (!listener) return;
    const speaking = speaker.speaking();
    const busy = mode === "transcribing" || mode === "waiting" || speaking;
    listener.setMuted(mode === "transcribing" || (busy && !options().bargeIn));
    listener.hearAbove(speaking);
  }

  async function heard(blob, ms) {
    set("transcribing");
    if (!options().handsFree) { listener?.stop(); listener = null; }
    let text = "";
    try { text = await transcribe(blob); } catch (error) { onError(error); return back(); }
    const verdict = classify(text, ms);
    if (verdict === "stop") return stop();
    if (verdict !== "say") return back();
    set("waiting");
    const flag = interrupted;
    interrupted = false;
    await send(text, { interrupted: flag });
  }

  // Back to listening, or to rest when each turn needs a tap.
  function back() {
    if (options().handsFree && listener) set("listening"); else set("off");
  }

  function handlers() {
    return {
      onSpeechStart() {
        if (speaker.speaking() || mode === "waiting") { interrupted = true; interrupt(); }
        set("hearing");
      },
      onUtterance: (blob, ms) => heard(blob, ms),
      onError,
    };
  }

  function stop() {
    listener?.stop();
    listener = null;
    interrupted = false;
    set("off");
  }

  return {
    mode: () => mode,
    refresh,
    async toggle() {
      if (mode !== "off") return stop();
      listener = makeListener(handlers());
      try { await listener.start(); } catch (error) { listener = null; onError(error); return; }
      set("listening");
    },
    stop,
    /** The reply (and its text) is complete; the conversation goes on when it is hands-free. */
    replyDone() { if (mode === "waiting") back(); else refresh(); },
  };
}
