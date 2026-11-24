// TextField — label + input (plan §4.3). Added because a real screen (login) needed a plain text/password
// field and nothing else in the control list fit.
//
// Two things a toolbar needs and a form does not: the label drawn *inside* the field (a caption above a search box
// costs a line of chrome for a word the placeholder already says), and an icon for what the field is for. Both are
// optional; a form field keeps its label above it.
import type { ComponentType, InputHTMLAttributes } from "react";
import styles from "./TextField.module.css";

export interface TextFieldProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  /** Draw the label inside the field as its placeholder-side caption instead of above it. */
  hideLabel?: boolean;
  /** A glyph at the start of the field — the magnifier that says "search" before anything is read. */
  icon?: ComponentType<{ size?: number; className?: string | undefined }>;
}

export function TextField({ label, id, className, hideLabel = false, icon: Icon, ...rest }: TextFieldProps) {
  const inputId = id ?? `field-${label.replace(/\s+/g, "-").toLowerCase()}`;
  const inputClass = [styles.input, Icon !== undefined ? styles.withIcon : "", className ?? ""].filter(Boolean).join(" ");
  return (
    <div className={styles.field}>
      <label className={hideLabel ? styles.srOnly : styles.label} htmlFor={inputId}>
        {label}
      </label>
      <div className={styles.box}>
        {Icon !== undefined && <Icon size={15} className={styles.icon} />}
        <input id={inputId} className={inputClass} {...rest} />
      </div>
    </div>
  );
}
