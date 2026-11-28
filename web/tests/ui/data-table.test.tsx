// tests/ui/data-table.test.tsx — a row's action has to be a real button, reachable by keyboard and named on its own. The
// import screen used to repeat every row as a second list of a hundred buttons; this pins the shape that replaced it.
//
// Plain assertions rather than jest-dom matchers: this suite has no jest-dom setup, and adding one for two tests would be a
// dependency for a matcher's sake.
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DataTable } from "../../src/ui/DataTable";

afterEach(cleanup);

const columns = [
  { key: "title", label: "Title" },
  { key: "author", label: "Author" },
];

describe("DataTable", () => {
  it("gives each row one button, named by what it acts on", () => {
    const correct = vi.fn();
    render(
      <DataTable
        caption="Books in the drop folder"
        actionColumnLabel="Correct"
        columns={columns}
        rows={[
          {
            key: "a",
            cells: [{ text: "Dune" }, { text: "Frank Herbert" }],
            action: { label: "Correct…", ariaLabel: "Correct Dune", onClick: correct },
          },
          {
            key: "b",
            cells: [{ text: "Neuromancer" }, { text: "William Gibson" }],
            action: { label: "Correct…", ariaLabel: "Correct Neuromancer", onClick: () => {} },
          },
        ]}
      />,
    );
    // Two identical visible labels would be indistinguishable to a screen reader; the aria-label tells them apart.
    const dune = screen.getByRole("button", { name: "Correct Dune" });
    expect(screen.getByRole("button", { name: "Correct Neuromancer" })).toBeTruthy();
    expect(screen.getAllByRole("button")).toHaveLength(2);
    fireEvent.click(dune);
    expect(correct).toHaveBeenCalledOnce();
  });

  it("marks the row whose detail is open, and hides the column when no row has an action", () => {
    const { rerender } = render(
      <DataTable
        columns={columns}
        actionColumnLabel="Correct"
        rows={[
          {
            key: "a",
            cells: [{ text: "Dune" }, { text: "Frank Herbert" }],
            action: { label: "Correct…", ariaLabel: "Correct Dune", onClick: () => {} },
            selected: true,
          },
        ]}
      />,
    );
    expect(screen.getByRole("button", { name: "Correct Dune" }).getAttribute("aria-pressed")).toBe("true");

    rerender(<DataTable columns={columns} rows={[{ key: "a", cells: [{ text: "Dune" }, { text: "Frank Herbert" }] }]} />);
    expect(screen.queryByRole("button")).toBeNull();
  });
});
