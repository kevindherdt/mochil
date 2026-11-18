// Play cards: what the assistant found in your library when you asked her to play something. Each card plays in the
// page with one shared audio element, so only one thing plays at a time.

let player = null;
let playing = null;

function audio() {
  if (!player) {
    player = new Audio();
    player.preload = "none";
    player.addEventListener("ended", () => reset());
    player.addEventListener("pause", () => { if (player.ended || !player.seeking) reset(); });
  }
  return player;
}

function reset() {
  if (playing) { playing.button.textContent = "Play"; playing.card.classList.remove("on"); }
  playing = null;
}

/** Draws the cards under the words of one reply. Calling it again for the same reply replaces them. */
export function showCards(row, cards, onPlay) {
  let box = row.querySelector(".hm-cards");
  if (!box) { box = document.createElement("div"); box.className = "hm-cards"; row.querySelector(".hm-text").after(box); }
  box.replaceChildren(...(cards || []).map((item) => {
    const card = document.createElement("div");
    card.className = "hm-card";
    const art = item.artwork_url ? `<img alt="" loading="lazy" src="${item.artwork_url.replace(/"/g, "")}">` : '<span class="hm-card-art"></span>';
    card.innerHTML = `${art}<span class="hm-card-text"><b></b><small></small></span><button type="button" class="hm-card-play">Play</button>`;
    const image = card.querySelector("img");
    if (image) image.onerror = () => { const tile = document.createElement("span"); tile.className = "hm-card-art"; image.replaceWith(tile); };
    card.querySelector("b").textContent = item.title || "Untitled";
    card.querySelector("small").textContent = [item.subtitle, item.kind === "music" ? "" : item.kind].filter(Boolean).join(" · ");
    const button = card.querySelector("button");
    button.onclick = () => {
      const same = playing && playing.card === card;
      const element = audio();
      if (same) { element.pause(); return; }
      reset();
      onPlay?.();
      element.src = item.stream_url;
      playing = { card, button };
      button.textContent = "Pause";
      card.classList.add("on");
      element.play().catch(() => { reset(); button.textContent = "Can't play"; });
    };
    return card;
  }));
}

/** Stops whatever a card is playing (the page calls this when she starts speaking or the chat changes). */
export function stopCards() { if (player) { player.pause(); reset(); } }
