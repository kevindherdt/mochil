// DataTable — a plain table for screens that are lists of facts (import, review, activity), so those screens stay free
// of layout of their own (lint law L2). Cells are data, not markup: a screen says what a cell says and how it should
// read, and this decides the type, the borders and the spacing.
//
// A row can carry one action. It is a real button in its own cell rather than a clickable <tr>: a row that is only
// clickable cannot be reached by keyboard, and a button whose visible label is "Correct…" needs its own aria-label to say
// which book it corrects.
import type { ReactNode } from "react";
import styles from "./DataTable.module.css";

export interface DataTableColumn {
  key: string;
  label: string;
  /** Numbers align right and get tabular figures, like every table that made it into a design review. */
  align?: "start" | "end";
}

export interface DataTableCell {
  text: ReactNode;
  /** `muted` for paths and asides, `warn` for a note that something needs a look, `pill` for a status word. */
  tone?: "default" | "muted" | "warn" | "pill";
  /** A status pill's own state, so "already in the library" can wear the accent and "new" cannot. */
  status?: string;
  title?: string;
}

export interface DataTableRowAction {
  label: string;
  /** What the button is for, in words a screen reader can use on its own: "Correct <title>". */
  ariaLabel: string;
  onClick: () => void;
}

export interface DataTableRow {
  key: string;
  cells: DataTableCell[];
  action?: DataTableRowAction;
  /** The row whose detail is open below the table; it stays marked when focus moves away. */
  selected?: boolean;
}

export function DataTable({
  columns,
  rows,
  caption,
  actionColumnLabel = "",
}: {
  columns: DataTableColumn[];
  rows: DataTableRow[];
  caption?: string;
  /** Heading for the action column, so its buttons are not announced under an empty header. */
  actionColumnLabel?: string;
}) {
  const hasActions = rows.some((row) => row.action !== undefined);
  return (
    <table className={styles.table}>
      {caption ? <caption className={styles.caption}>{caption}</caption> : null}
      <thead>
        <tr>
          {columns.map((column) => (
            <th key={column.key} scope="col" data-align={column.align ?? "start"}>
              {column.label}
            </th>
          ))}
          {hasActions ? (
            <th scope="col" data-align="end">
              <span className={styles.srOnly}>{actionColumnLabel || "Actions"}</span>
            </th>
          ) : null}
        </tr>
      </thead>
      <tbody>
        {rows.map((row) => (
          <tr key={row.key} data-selected={row.selected ? "true" : undefined}>
            {row.cells.map((cell, index) => (
              <td
                key={columns[index]?.key ?? index}
                data-align={columns[index]?.align ?? "start"}
                data-tone={cell.tone ?? "default"}
                title={cell.title}
              >
                {cell.tone === "pill" ? (
                  <span className={styles.pill} data-status={cell.status ?? ""}>
                    {cell.text}
                  </span>
                ) : (
                  cell.text
                )}
              </td>
            ))}
            {hasActions ? (
              <td data-align="end">
                {row.action ? (
                  <button
                    type="button"
                    className={styles.rowAction}
                    aria-label={row.action.ariaLabel}
                    aria-pressed={row.selected ?? false}
                    onClick={row.action.onClick}
                  >
                    {row.action.label}
                  </button>
                ) : null}
              </td>
            ) : null}
          </tr>
        ))}
      </tbody>
    </table>
  );
}
