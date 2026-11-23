// TextField — label + input (plan §4.3). Added because a real screen (login) needed a plain text/password
// field and nothing else in the control list fit.
import type { InputHTMLAttributes } from "react";
import styles from "./TextField.module.css";

export interface TextFieldProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
}

export function TextField({ label, id, className, ...rest }: TextFieldProps) {
  const inputId = id ?? `field-${label.replace(/\s+/g, "-").toLowerCase()}`;
  const inputClass = className ? `${styles.input} ${className}` : styles.input;
  return (
    <div className={styles.field}>
      <label className={styles.label} htmlFor={inputId}>
        {label}
      </label>
      <input id={inputId} className={inputClass} {...rest} />
    </div>
  );
}
