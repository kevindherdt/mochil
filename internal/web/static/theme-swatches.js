// Theme family as a row of colour cards instead of a drop-down. The real <select> stays in the page (hidden): the theme
// controller owns it, so choosing a card just sets it and fires its change event.
const rootAttr = (name) => document.documentElement.getAttribute(name);

export function mountThemeSwatches(document) {
  const select = document.querySelector("#theme-palette");
  if (!select?.options || !document.querySelector("#settings-view") || document.querySelector(".theme-swatches")) return;
  const label = select.closest("label");
  const options = [...select.options].map((option) => {
    const [, name, hint] = /^(.*?)(?: \((.*)\))?$/.exec(option.textContent) || [];
    return { id: option.value, name: name || option.textContent, hint: hint || "", group: option.parentElement.label || "" };
  });
  const cards = options.map((option) => `<button type="button" class="theme-swatch" role="radio" aria-checked="false" data-palette-choice="${option.id}"><span class="swatch-preview" data-swatch data-palette="${option.id}" data-theme="${rootAttr("data-theme") || "dark"}"><i></i><i></i><b></b></span><strong>${option.name}</strong>${option.hint ? `<small>${option.hint}</small>` : ""}</button>`).join("");
  label.insertAdjacentHTML("beforebegin", `<div class="theme-swatches" role="radiogroup" aria-label="Theme family">${cards}</div>`);
  label.classList.add("is-replaced");
  const host = document.querySelector(".theme-swatches");
  const sync = () => {
    host.querySelectorAll(".theme-swatch").forEach((card) => card.setAttribute("aria-checked", String(card.dataset.paletteChoice === select.value)));
    host.querySelectorAll("[data-swatch]").forEach((swatch) => swatch.setAttribute("data-theme", rootAttr("data-theme") || "dark"));
  };
  host.addEventListener("click", (event) => {
    const card = event.target.closest("[data-palette-choice]");
    if (!card) return;
    select.value = card.dataset.paletteChoice;
    select.dispatchEvent(new Event("change", { bubbles: true }));
    sync();
  });
  new MutationObserver(sync).observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme", "data-palette"] });
  select.addEventListener("change", sync);
  sync();
}
