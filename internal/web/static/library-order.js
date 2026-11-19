// "Order in the menu": the list that decides the order of libraries in the switcher.
//
// It is its own module because library-settings.js has a line budget and this is a self-contained
// widget: it owns its drag state, its arrows, and nothing else in the page needs to know how it
// works. The caller keeps the array and decides what a move means.
//
// Home is not a library, so it is not in this list and is always first without needing a rule.

export function createLibraryOrderPanel({ libraries, iconForKind, onChange }) {
  if (libraries.length < 2) return null;

  let dragFrom = null;

  const panel = document.createElement("section");
  panel.className = "library-order";
  const title = document.createElement("h3");
  title.textContent = "Order in the menu";
  const hint = document.createElement("p");
  hint.className = "library-order-hint";
  hint.textContent =
    "Home always comes first. Drag a row, or use the arrows, to change the order of your libraries.";
  const list = document.createElement("ol");
  list.className = "library-order-list";

  libraries.forEach((library, index) => list.append(row(library, index)));
  panel.append(title, hint, list);
  return panel;

  function row(library, index) {
    const item = document.createElement("li");
    item.className = "library-order-row";
    item.draggable = true;
    item.dataset.id = library.id;

    const handle = document.createElement("span");
    handle.className = "library-order-handle";
    handle.setAttribute("aria-hidden", "true");
    handle.textContent = "⠿";

    const icon = document.createElement("span");
    icon.className = "library-order-icon";
    icon.dataset.icon = iconForKind(library.kind);

    const name = document.createElement("span");
    name.className = "library-order-name";
    name.textContent = library.name || "New library";

    const up = arrow("↑", index === 0, `Move ${name.textContent} up`, () => onChange(index, index - 1));
    const down = arrow("↓", index === libraries.length - 1, `Move ${name.textContent} down`, () =>
      onChange(index, index + 1),
    );

    // Dragging is a shortcut, never the only way: the arrows do the same thing on a touch screen
    // and from the keyboard.
    item.ondragstart = (event) => {
      dragFrom = index;
      item.classList.add("dragging");
      event.dataTransfer.effectAllowed = "move";
      // Firefox refuses to start a drag without a payload.
      event.dataTransfer.setData("text/plain", library.id);
    };
    item.ondragend = () => {
      dragFrom = null;
      item.classList.remove("dragging");
      for (const other of list.children) other.classList.remove("drop-before", "drop-after");
    };
    item.ondragover = (event) => {
      if (dragFrom === null || dragFrom === index) return;
      event.preventDefault();
      const after = dropAfter(item, event);
      item.classList.toggle("drop-after", after);
      item.classList.toggle("drop-before", !after);
    };
    item.ondragleave = () => item.classList.remove("drop-before", "drop-after");
    item.ondrop = (event) => {
      event.preventDefault();
      if (dragFrom === null) return;
      const target = index + (dropAfter(item, event) ? 1 : 0);
      const from = dragFrom;
      dragFrom = null;
      onChange(from, from < target ? target - 1 : target);
    };

    item.append(handle, icon, name, up, down);
    return item;
  }

  function dropAfter(item, event) {
    const rect = item.getBoundingClientRect();
    return event.clientY > rect.top + rect.height / 2;
  }

  function arrow(label, disabled, ariaLabel, onClick) {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "library-order-move";
    button.textContent = label;
    button.disabled = disabled;
    button.setAttribute("aria-label", ariaLabel);
    button.onclick = onClick;
    return button;
  }
}
