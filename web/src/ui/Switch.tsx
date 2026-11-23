// ui/Switch — an on/off control, distinct from a checkbox: Settings' module rows are switches in the legacy app
// (`role="switch"` + `aria-checked`, static/modules-settings.js), and a screen reader announcing "module video,
// switch, on" is the behaviour that keeps this app's accessibility parity with it. The thumb is a real element
// rather than a background-position trick so it can animate on `transform`, which is the only property the
// component is allowed to move (law L7 bans layout-shifting animation).
//
// `disabled` exists for one specific case, not as a general "grey it out": a module option whose parent module is
// off still shows its saved choice, but the switch must report what actually runs (off) — matching the legacy
// panel's `moduleOptionHTML`, which does the same and disables the control.
import styles from "./Switch.module.css";

export interface SwitchProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  /** Accessible name. A switch with no label is a control nobody can identify out loud. */
  label: string;
  disabled?: boolean;
}

export function Switch({ checked, onChange, label, disabled = false }: SwitchProps) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      className={checked ? `${styles.track} ${styles.on}` : styles.track}
      onClick={() => onChange(!checked)}
    >
      <span className={styles.thumb} />
    </button>
  );
}
