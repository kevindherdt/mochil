export function bindCodeCopy(root, clipboard = navigator.clipboard) {
  root?.querySelectorAll("pre").forEach((block) => {
    if (block.querySelector(":scope > .code-copy")) return;
    const button = document.createElement("button");
    button.type = "button";
    button.className = "code-copy";
    button.textContent = "Copy";
    button.setAttribute("aria-label", "Copy code");
    button.onclick = async () => {
      const value =
        block.querySelector("code")?.textContent || block.textContent;
      try {
        await clipboard.writeText(value);
        button.textContent = "Copied";
      } catch {
        button.textContent = "Could not copy";
      }
      window.setTimeout(() => {
        button.textContent = "Copy";
      }, 1800);
    };
    block.append(button);
  });
}
