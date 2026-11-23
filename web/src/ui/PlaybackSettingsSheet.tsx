// ui/PlaybackSettingsSheet — the player's own settings panel: how subtitles look, and how fast the film runs.
//
// It lives in ui/ and takes every value as a prop because a screen may not own a CSS module (law L2), and because this
// panel is presentation only: it does not know where preferences are stored, only that somebody above is listening.
//
// Radix Dialog rather than a hand-rolled overlay, for the two things an overlay must get right and hand-rolling
// usually gets wrong: focus is trapped while it is open and returned to the gear when it closes, and Escape closes it.
// It renders above the video, so the film keeps playing behind it — the whole point of live preview being unnecessary.
import * as Dialog from "@radix-ui/react-dialog";
import { RotateCcw, X } from "lucide-react";
import styles from "./PlaybackSettingsSheet.module.css";
import type { SubtitlePreferences } from "../lib/subtitlePreferences";
import {
  SPEED_OPTIONS,
  SUBTITLE_BACKGROUND_OPTIONS,
  SUBTITLE_COLOUR_OPTIONS,
  SUBTITLE_DEFAULTS,
  SUBTITLE_EDGE_OPTIONS,
  SUBTITLE_SIZE_OPTIONS,
  cueStyle,
} from "../lib/subtitlePreferences";

type Option = { id: string; label: string };

interface ChoiceRowProps {
  label: string;
  options: Option[];
  value: string;
  onChange: (id: string) => void;
}

function ChoiceRow({ label, options, value, onChange }: ChoiceRowProps) {
  return (
    <div className={styles.choice}>
      <span className={styles.choiceLabel}>{label}</span>
      <div className={styles.segments} role="radiogroup" aria-label={label}>
        {options.map((option) => {
          const active = option.id === value;
          return (
            <button
              key={option.id}
              type="button"
              role="radio"
              aria-checked={active}
              className={active ? `${styles.segment} ${styles.segmentActive}` : styles.segment}
              onClick={() => onChange(option.id)}
            >
              {option.label}
            </button>
          );
        })}
      </div>
    </div>
  );
}

export interface PlaybackSettingsSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  preferences: SubtitlePreferences;
  /** A *patch*, not a replacement: whoever owns the state merges it (functional update), so two choices made in the
   *  same tick cannot clobber each other by both being built from the same stale object. */
  onChange: (patch: Partial<SubtitlePreferences>) => void;
  /** Whether the film has any text subtitles at all; when it has none the appearance rows are shown as inert rather
   *  than hidden, so the panel does not change shape between films without saying why. */
  hasSubtitles: boolean;
}

export function PlaybackSettingsSheet({ open, onOpenChange, preferences, onChange, hasSubtitles }: PlaybackSettingsSheetProps) {
  const set = <K extends keyof SubtitlePreferences>(key: K, value: SubtitlePreferences[K]) => {
    onChange({ [key]: value });
  };

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Overlay className={styles.overlay} />
      <Dialog.Content className={styles.sheet} aria-describedby={undefined}>
        <div className={styles.header}>
          <Dialog.Title className={styles.title}>Playback settings</Dialog.Title>
          <Dialog.Close className={styles.close} aria-label="Close">
            <X size={18} />
          </Dialog.Close>
        </div>

        <div className={styles.preview} style={cueStyle(preferences)} aria-hidden="true">
          <span className={styles.previewCue}>The quick brown fox jumps over the lazy dog</span>
        </div>

        <section className={styles.section} aria-label="Subtitle appearance">
          <h3 className={styles.sectionTitle}>Subtitles</h3>
          {!hasSubtitles && (
            <p className={styles.note}>This film has no text subtitles, so these settings will not show here.</p>
          )}
          <ChoiceRow
            label="Size"
            options={SUBTITLE_SIZE_OPTIONS}
            value={preferences.size}
            onChange={(id) => set("size", id as SubtitlePreferences["size"])}
          />
          <ChoiceRow
            label="Colour"
            options={SUBTITLE_COLOUR_OPTIONS}
            value={preferences.colour}
            onChange={(id) => set("colour", id as SubtitlePreferences["colour"])}
          />
          <ChoiceRow
            label="Background"
            options={SUBTITLE_BACKGROUND_OPTIONS}
            value={preferences.background}
            onChange={(id) => set("background", id as SubtitlePreferences["background"])}
          />
          <ChoiceRow
            label="Edge"
            options={SUBTITLE_EDGE_OPTIONS}
            value={preferences.edge}
            onChange={(id) => set("edge", id as SubtitlePreferences["edge"])}
          />
        </section>

        <section className={styles.section} aria-label="Playback speed">
          <h3 className={styles.sectionTitle}>Speed</h3>
          <ChoiceRow
            label="Speed"
            options={SPEED_OPTIONS}
            value={String(preferences.speed)}
            onChange={(id) => set("speed", Number(id))}
          />
        </section>

        <button
          type="button"
          className={styles.reset}
          onClick={() => onChange({ ...SUBTITLE_DEFAULTS, speed: preferences.speed })}
        >
          <RotateCcw size={14} />
          Reset subtitle appearance
        </button>
</Dialog.Content>
    </Dialog.Root>
  );
}
