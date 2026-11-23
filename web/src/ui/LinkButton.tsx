// LinkButton — a link that looks like a button. A navigation is a link: wrapping a <button> in an <a> nests one
// interactive control inside another (invalid, and a screen reader announces two controls for one action), and the
// anchor around it stays inline, so the thing you can actually hit is not the thing that has the 44px box.
import { Link } from "@tanstack/react-router";
import type { ReactNode } from "react";
import styles from "./Button.module.css";

export interface LinkButtonProps {
  to: string;
  variant?: "primary" | "secondary";
  children: ReactNode;
}

export function LinkButton({ to, variant = "primary", children }: LinkButtonProps) {
  const variantClass = variant === "primary" ? styles.primary : styles.secondary;
  return (
    <Link to={to} className={`${styles.button} ${variantClass}`} style={{ textDecoration: "none" }}>
      {children}
    </Link>
  );
}
