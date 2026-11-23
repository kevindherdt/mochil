// SearchField — the compact search input a header reveals when its search button is pressed. Uncontrolled by
// default (`defaultValue` + `onChange`) because the screen debounces the value before it becomes a query.
import { Search } from "lucide-react";
import styles from "./SearchField.module.css";

export interface SearchFieldProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  /** Focus on mount — the header mounts it only when the user asked for search. */
  autoFocus?: boolean;
}

export function SearchField({ value, onChange, placeholder = "Search", autoFocus = true }: SearchFieldProps) {
  return (
    <div className={styles.wrap}>
      <Search size={15} aria-hidden="true" />
      <input
        className={styles.input}
        type="search"
        value={value}
        placeholder={placeholder}
        aria-label={placeholder}
        autoFocus={autoFocus}
        onChange={(event) => onChange(event.target.value)}
      />
    </div>
  );
}
