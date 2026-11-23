// TextArea — label + multi-line text input (plan §4.3's control list doesn't name this; added because Notes'
// page editor needs a real multi-line field and TextField is deliberately single-line only). Shares TextField's
// shape (label above, one bordered box) so the two read as the same family.
import type { TextareaHTMLAttributes } from "react";
import styles from "./TextField.module.css";

export interface TextAreaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  label: string;
}

export function TextArea({ label, id, className, ...rest }: TextAreaProps) {
  const inputId = id ?? `field-${label.replace(/\s+/g, "-").toLowerCase()}`;
  const inputClass = className ? `${styles.input} ${className}` : styles.input;
  return (
    <div className={styles.field}>
      <label className={styles.label} htmlFor={inputId}>
        {label}
      </label>
      <textarea id={inputId} className={inputClass} {...rest} />
    </div>
  );
}
