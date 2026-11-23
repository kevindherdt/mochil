// Deliberately violates law L2 (plan §4.2: screens own no layout — features/**/screens/** may not import a
// *.module.css). Not imported anywhere; exists for scripts/check-lint-fixtures.mjs to prove the rule fires.
import styles from "./L2Violation.module.css";

export function BadScreen() {
  return <div className={styles.badLayout}>screen-owned layout, which L2 forbids</div>;
}
