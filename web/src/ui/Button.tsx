// Button — primary/secondary variants (plan §4.3).
import type { ButtonHTMLAttributes } from "react";
import styles from "./Button.module.css";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "primary" | "secondary";
}

export function Button({ variant = "primary", className, ...rest }: ButtonProps) {
  const variantClass = variant === "primary" ? styles.primary : styles.secondary;
  const classes = className ? `${styles.button} ${variantClass} ${className}` : `${styles.button} ${variantClass}`;
  return <button className={classes} {...rest} />;
}
