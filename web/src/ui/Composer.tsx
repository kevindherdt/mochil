// Composer — a message input with a send button, plus an optional microphone.
//
// History worth keeping: this shipped field + send only, with a comment claiming the mic was a real gap because no
// dictation path existed on /next. That was half stale: `/api/narration/transcribe` has existed all along
// (internal/web/narration_transcribe.go — raw audio bytes in, `{ text }` out). The mic is now real: tap to record,
// tap again to stop, text lands in the field (or is sent immediately when hands-free is on). The paperclip is real
// too: files are read in the browser, shown as chips, and sent as base64 with the message (the server turns images
// into content parts and inlines plain text — see internal/web/hermes_http.go's hermesAttachmentParts).
import { useEffect, useRef, useState } from "react";
import { Mic, Paperclip, Send, Square, X } from "lucide-react";
import styles from "./Composer.module.css";

export interface ComposerProps {
  value: string;
  onChange: (value: string) => void;
  onSend: () => void;
  disabled?: boolean;
  placeholder?: string;
  /** Omitted when speech recognition is not set up on this server — the mic is then not drawn at all. */
  onTranscribe?: ((recording: Blob) => Promise<string | undefined>) | undefined;
  /**
   * Dictation through the browser's own recogniser, for a server with no speech-to-text. Used only when `onTranscribe`
   * is absent; the mic is drawn when either is given.
   */
  onListen?: (() => Promise<string | undefined>) | undefined;
  /** Receives the transcribed text; hands-free sends it immediately, otherwise it fills the field. */
  onTranscribed?: ((text: string) => void) | undefined;
  /** Pending attachments, shown as chips above the field. Omit both to hide the paperclip entirely. */
  attachments?: PendingAttachment[] | undefined;
  onAttach?: ((files: File[]) => void) | undefined;
  onRemoveAttachment?: ((index: number) => void) | undefined;
}

/** A file chosen but not yet sent. `data` is base64 without the data: prefix. */
export interface PendingAttachment {
  name: string;
  media_type: string;
  data: string;
}

export function Composer({
  value,
  onChange,
  onSend,
  disabled = false,
  placeholder = "Message",
  onTranscribe,
  onListen,
  onTranscribed,
  attachments,
  onAttach,
  onRemoveAttachment,
}: ComposerProps) {
  const fileRef = useRef<HTMLInputElement>(null);
  const recorderRef = useRef<MediaRecorder | null>(null);
  const chunksRef = useRef<Blob[]>([]);
  const [recording, setRecording] = useState(false);
  const [listenBusy, setListenBusy] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // A recorder left running when the screen unmounts would hold the microphone open — stop it and release the
  // stream, or the browser keeps its "recording" indicator until the tab closes.
  useEffect(() => {
    return () => {
      const recorder = recorderRef.current;
      if (recorder !== null && recorder.state !== "inactive") recorder.stop();
      recorder?.stream.getTracks().forEach((track) => track.stop());
    };
  }, []);

  async function startRecording() {
    if (onTranscribe === undefined || busy) return;
    setError(null);
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const recorder = new MediaRecorder(stream);
      chunksRef.current = [];
      recorder.ondataavailable = (event) => {
        if (event.data.size > 0) chunksRef.current.push(event.data);
      };
      recorder.onstop = () => {
        stream.getTracks().forEach((track) => track.stop());
        const blob = new Blob(chunksRef.current, { type: recorder.mimeType || "audio/webm" });
        if (blob.size === 0) return;
        setBusy(true);
        void onTranscribe(blob)
          .then((text) => {
            if (text !== undefined && text.trim() !== "") onTranscribed?.(text.trim());
            else setError("Nothing was heard — try again closer to the microphone.");
          })
          .catch((caught: unknown) => setError(caught instanceof Error ? caught.message : "Could not transcribe that recording."))
          .finally(() => setBusy(false));
      };
      recorderRef.current = recorder;
      recorder.start();
      setRecording(true);
    } catch {
      setError("The microphone could not be opened. Check this site's microphone permission.");
    }
  }

  function stopRecording() {
    const recorder = recorderRef.current;
    if (recorder !== null && recorder.state !== "inactive") recorder.stop();
    setRecording(false);
  }

  const pending = attachments ?? [];

  return (
    <div>
      {pending.length > 0 && (
        <div style={{ display: "flex", flexWrap: "wrap", gap: "var(--space-8)", marginBottom: "var(--space-8)" }}>
          {pending.map((file, index) => (
            <span
              key={`${file.name}-${index}`}
              style={{
                display: "inline-flex",
                alignItems: "center",
                gap: "var(--space-8)",
                padding: "var(--space-4) var(--space-8)",
                borderRadius: "var(--radius-pill)",
                background: "var(--color-raised)",
                fontSize: "12px",
              }}
            >
              {file.name}
              {onRemoveAttachment !== undefined && (
                <button
                  type="button"
                  aria-label={`Remove ${file.name}`}
                  onClick={() => onRemoveAttachment(index)}
                  style={{ all: "unset", cursor: "pointer", display: "inline-flex" }}
                >
                  <X size={12} />
                </button>
              )}
            </span>
          ))}
        </div>
      )}
      <form
        className={styles.row}
        onSubmit={(event) => {
          event.preventDefault();
          onSend();
        }}
      >
        <input
          className={styles.input}
          value={value}
          placeholder={placeholder}
          aria-label={placeholder}
          disabled={disabled}
          onChange={(event) => onChange(event.target.value)}
        />
        {onAttach !== undefined && (
          <>
            <input
              ref={fileRef}
              type="file"
              multiple
              hidden
              aria-label="Choose a file to attach"
              // Images and the text formats the server inlines; a PDF is caught server-side with a clear reason.
              accept="image/*,text/plain,text/markdown,.md,.txt,.csv,.json,.yaml,.yml,.log"
              onChange={(event) => {
                const files = [...(event.target.files ?? [])];
                // Reset first: choosing the same file twice in a row otherwise fires no change event at all.
                event.target.value = "";
                if (files.length > 0) onAttach(files);
              }}
            />
            <button
              type="button"
              className={styles.send}
              aria-label="Attach a picture or document"
              disabled={disabled || listenBusy}
              onClick={() => fileRef.current?.click()}
            >
              <Paperclip size={16} />
            </button>
          </>
        )}
        {(onTranscribe !== undefined || onListen !== undefined) && (
          <button
            type="button"
            className={styles.send}
            aria-label={recording ? "Stop recording" : "Speak a message"}
            aria-pressed={recording || listenBusy}
            disabled={disabled || busy}
            onClick={() => {
              if (onTranscribe !== undefined) {
                if (recording) {
                  stopRecording();
                } else {
                  void startRecording();
                }
                return;
              }
              if (busy || onListen === undefined) return;
              setListenBusy(true);
              void onListen()
                .then((text) => {
                  if (text !== undefined && text.trim() !== "") onTranscribed?.(text.trim());
                })
                .catch((caught: unknown) => {
                  setError(caught instanceof Error ? caught.message : "Could not hear that.");
                })
                .finally(() => setListenBusy(false));
            }}
          >
            {recording ? <Square size={15} /> : <Mic size={16} />}
          </button>
        )}
        <button
          type="submit"
          className={styles.send}
          aria-label="Send"
          disabled={disabled || busy || (value.trim() === "" && pending.length === 0)}
        >
          <Send size={16} />
        </button>
      </form>
      {(busy || error !== null) && (
        <p style={{ margin: "var(--space-4) 0 0", fontSize: "11px", color: error === null ? "var(--color-muted)" : "var(--color-error)" }}>
          {busy ? "Transcribing…" : error}
        </p>
      )}
    </div>
  );
}
