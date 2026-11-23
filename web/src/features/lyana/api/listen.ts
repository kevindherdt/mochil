// features/lyana/api/listen.ts — hearing what you say, without a server that can.
//
// `POST /api/narration/transcribe` needs the narration build with speech recognition in it. Plenty of installs do not
// have that (this one did not), and the mic then simply did not exist — Kevin asked for a voice button twice. Chrome and
// Android ship a recogniser of their own, so the same button can work everywhere: the server's when it is there, the
// browser's otherwise. What is heard goes through the same path either way.
//
// The Web Speech API is not in TypeScript's DOM types and is prefixed (`webkitSpeechRecognition`) on Chromium, so it is
// described here rather than assumed.

interface SpeechRecognitionResultLike {
  readonly length: number;
  item: (index: number) => { transcript: string };
  [index: number]: { transcript: string };
}

interface SpeechRecognitionEventLike {
  resultIndex: number;
  results: {
    readonly length: number;
    item: (index: number) => SpeechRecognitionResultLike;
    [index: number]: SpeechRecognitionResultLike;
  };
}

interface SpeechRecognitionLike {
  lang: string;
  continuous: boolean;
  interimResults: boolean;
  start: () => void;
  stop: () => void;
  abort: () => void;
  onresult: ((event: SpeechRecognitionEventLike) => void) | null;
  onerror: ((event: { error?: string }) => void) | null;
  onend: (() => void) | null;
}

type RecognitionConstructor = new () => SpeechRecognitionLike;

function recognitionConstructor(): RecognitionConstructor | null {
  const scope = globalThis as unknown as {
    SpeechRecognition?: RecognitionConstructor;
    webkitSpeechRecognition?: RecognitionConstructor;
  };
  return scope.SpeechRecognition ?? scope.webkitSpeechRecognition ?? null;
}

/** Whether this browser can listen to you at all. */
export function browserCanListen(): boolean {
  return recognitionConstructor() !== null;
}

export interface Listener {
  /** Resolves with what was said, or "" when nothing was heard before the person stopped. */
  listen: () => Promise<string>;
  stop: () => void;
}

/**
 * One utterance at a time. `continuous` keeps the recogniser running after a pause so a call does not cut the person off
 * mid-sentence; without it the browser ends the session at the first silence, which is right for a dictation button.
 */
export function createListener(options: { continuous?: boolean; language?: string } = {}): Listener | null {
  const Constructor = recognitionConstructor();
  if (Constructor === null) return null;
  const language = options.language ?? "en-US";
  const continuous = options.continuous === true;
  let active: SpeechRecognitionLike | null = null;

  const stop = () => {
    const current = active;
    active = null;
    current?.stop();
  };

  const listen = () =>
    new Promise<string>((resolve) => {
      const recognition = new Constructor();
      recognition.lang = language;
      recognition.continuous = continuous;
      recognition.interimResults = false;
      let heard = "";
      let settled = false;
      const finish = () => {
        if (settled) return;
        settled = true;
        if (active === recognition) active = null;
        resolve(heard.trim());
      };
      recognition.onresult = (event) => {
        for (let index = event.resultIndex; index < event.results.length; index += 1) {
          const alternative = event.results[index];
          if (alternative !== undefined) heard += alternative[0]?.transcript ?? "";
        }
      };
      // "no-speech" and "aborted" are the person being quiet or tapping stop, not failures worth a red message.
      recognition.onerror = () => finish();
      recognition.onend = () => finish();
      active = recognition;
      try {
        recognition.start();
      } catch {
        // A second start before the first stops throws; treat it as "nothing heard" rather than as a crash.
        finish();
      }
    });

  return { listen, stop };
}
