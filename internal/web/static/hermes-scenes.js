// The scenes behind the avatar. They are plain gradients, so there is nothing to download; "image" is the one picture
// the administrator uploads (Settings of the chat). The frame says how big the model sits and where.

// A drawn scene: a small SVG (400x220, scaled to cover the stage) turned into a CSS background.
const drawn = (svg) => `center / cover no-repeat url("data:image/svg+xml,${encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 400 220" preserveAspectRatio="xMidYMid slice">${svg}</svg>`)}")`;

const room = drawn(`
<defs>
 <linearGradient id="w" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#3f2c3d"/><stop offset="1" stop-color="#5a3d4a"/></linearGradient>
 <linearGradient id="f" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#7a5239"/><stop offset="1" stop-color="#5a3b29"/></linearGradient>
 <radialGradient id="l" cx=".5" cy=".5" r=".5"><stop offset="0" stop-color="#ffd698" stop-opacity=".75"/><stop offset="1" stop-color="#ffd698" stop-opacity="0"/></radialGradient>
 <linearGradient id="n" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#1c2748"/><stop offset="1" stop-color="#3b4f7e"/></linearGradient>
</defs>
<rect width="400" height="220" fill="url(#w)"/>
<rect y="132" width="400" height="6" fill="#33232f"/>
<rect y="138" width="400" height="82" fill="url(#f)"/>
<g stroke="#4d3223" stroke-width="1" opacity=".5"><path d="M0 160H400M0 186H400M70 138L40 220M190 138L175 220M310 138L330 220"/></g>
<rect x="18" y="30" width="86" height="82" rx="4" fill="#2a1d29"/>
<rect x="23" y="35" width="76" height="72" fill="url(#n)"/>
<circle cx="78" cy="56" r="9" fill="#f6efd2"/><circle cx="82" cy="53" r="8" fill="#2c3a63"/>
<g fill="#fff"><circle cx="38" cy="50" r="1"/><circle cx="55" cy="72" r="1.2"/><circle cx="88" cy="88" r=".9"/><circle cx="44" cy="96" r=".8"/></g>
<path d="M23 107V86l14-6 12 8 14-8 14 6 10-4v25z" fill="#1a2038"/>
<path d="M61 35v72M23 71h76" stroke="#2a1d29" stroke-width="3"/>
<path d="M8 22c10 0 22 4 24 30 2 28-2 54-6 66H8z" fill="#8e4b5a"/><path d="M114 22c-10 0-22 4-24 30-2 28 2 54 6 66h18z" fill="#8e4b5a"/>
<rect x="4" y="20" width="118" height="4" rx="2" fill="#4a2b33"/>
<ellipse cx="338" cy="72" rx="78" ry="70" fill="url(#l)"/>
<g><rect x="326" y="104" width="24" height="5" rx="2" fill="#3a2a2a"/><rect x="336" y="76" width="4" height="30" fill="#3a2a2a"/><path d="M322 76l8-22h20l8 22z" fill="#ffc977"/></g>
<rect x="268" y="52" width="110" height="5" rx="2" fill="#4a3226"/>
<g><rect x="276" y="32" width="9" height="20" fill="#c25b5b"/><rect x="286" y="28" width="8" height="24" fill="#5b8cc2"/><rect x="295" y="35" width="10" height="17" fill="#e0b25a"/><rect x="306" y="30" width="7" height="22" fill="#6fae8a"/></g>
<g><path d="M350 52c-5-12-3-20 0-24 3 4 5 12 0 24zM350 52c-12-6-16-12-15-18 7 2 13 8 15 18zM350 52c12-6 16-12 15-18-7 2-13 8-15 18z" fill="#4f9a6a"/><rect x="343" y="46" width="14" height="6" rx="2" fill="#a8603f"/></g>
<ellipse cx="205" cy="196" rx="120" ry="16" fill="#8e4b5a" opacity=".85"/><ellipse cx="205" cy="196" rx="96" ry="11" fill="none" stroke="#e8c27a" stroke-width="1.5" opacity=".6"/>
`);

const night = drawn(`
<defs>
 <linearGradient id="s" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#0d1330"/><stop offset=".7" stop-color="#26356b"/><stop offset="1" stop-color="#46508a"/></linearGradient>
 <radialGradient id="m" cx=".5" cy=".5" r=".5"><stop offset="0" stop-color="#f6efd2" stop-opacity=".5"/><stop offset="1" stop-color="#f6efd2" stop-opacity="0"/></radialGradient>
</defs>
<rect width="400" height="220" fill="url(#s)"/>
<circle cx="322" cy="52" r="50" fill="url(#m)"/><circle cx="322" cy="52" r="20" fill="#f6efd2"/><circle cx="330" cy="46" r="18" fill="#26356b" opacity=".18"/>
<g fill="#fff"><circle cx="30" cy="30" r="1.4"/><circle cx="76" cy="60" r="1"/><circle cx="120" cy="22" r="1.6"/><circle cx="170" cy="50" r="1"/><circle cx="214" cy="26" r="1.3"/><circle cx="262" cy="70" r="1"/><circle cx="372" cy="100" r="1.2"/><circle cx="52" cy="104" r="1"/><circle cx="150" cy="90" r=".9"/></g>
<path d="M0 180V150h26v-20h22v24h18v-34h26v40h24v-18h30v28h22v-36h24v30h26v-22h28v32h20v-26h26v36h24v-30h28V220H0z" fill="#12183a"/>
<g fill="#ffd27a" opacity=".85"><rect x="32" y="138" width="4" height="5"/><rect x="74" y="128" width="4" height="5"/><rect x="104" y="150" width="4" height="5"/><rect x="196" y="140" width="4" height="5"/><rect x="262" y="152" width="4" height="5"/><rect x="330" y="146" width="4" height="5"/></g>
<rect y="196" width="400" height="24" fill="#0b0f26"/>
<rect x="0" y="0" width="400" height="220" fill="none" stroke="#2a1d29" stroke-width="0"/>
<path d="M0 0h18v220H0zM382 0h18v220h-18zM0 0h400v14H0z" fill="#1c1426" opacity=".9"/>
`);

const sunset = drawn(`
<defs>
 <linearGradient id="a" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#2d1b4e"/><stop offset=".55" stop-color="#b8456b"/><stop offset="1" stop-color="#f4a261"/></linearGradient>
 <radialGradient id="g" cx=".5" cy=".5" r=".5"><stop offset="0" stop-color="#ffe3a8" stop-opacity=".9"/><stop offset="1" stop-color="#ffe3a8" stop-opacity="0"/></radialGradient>
</defs>
<rect width="400" height="220" fill="url(#a)"/>
<circle cx="300" cy="140" r="70" fill="url(#g)"/><circle cx="300" cy="140" r="26" fill="#ffe3a8"/>
<g fill="#fff" opacity=".35"><ellipse cx="70" cy="50" rx="46" ry="6"/><ellipse cx="110" cy="62" rx="36" ry="5"/><ellipse cx="250" cy="38" rx="40" ry="5"/></g>
<path d="M0 170c50-22 100-26 150-8s100 10 150-6 70-8 100 4V220H0z" fill="#7a3358"/>
<path d="M0 190c60-16 120-18 180-4s140 8 220-6V220H0z" fill="#4a2146"/>
<path d="M0 206c80-10 160-6 240 0s110 2 160-4V220H0z" fill="#2d1b4e"/>
`);

const studio = drawn(`
<defs>
 <radialGradient id="b" cx=".5" cy=".35" r=".7"><stop offset="0" stop-color="#454956"/><stop offset="1" stop-color="#14151a"/></radialGradient>
 <radialGradient id="p" cx=".5" cy=".5" r=".5"><stop offset="0" stop-color="#fff" stop-opacity=".22"/><stop offset="1" stop-color="#fff" stop-opacity="0"/></radialGradient>
</defs>
<rect width="400" height="220" fill="url(#b)"/>
<ellipse cx="200" cy="170" rx="170" ry="40" fill="url(#p)"/>
<path d="M0 150Q200 128 400 150V220H0z" fill="#1d1e25"/>
<path d="M0 150Q200 128 400 150" fill="none" stroke="#5a5e6b" stroke-width="1.5" opacity=".7"/>
<g fill="#2a2c35"><rect x="24" y="12" width="46" height="9" rx="3"/><rect x="330" y="12" width="46" height="9" rx="3"/></g>
<path d="M30 21L8 130h92L70 21zM370 21L392 130H300l30-109z" fill="#fff" opacity=".05"/>
`);

const mint = drawn(`
<defs><linearGradient id="a" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#e4f7ee"/><stop offset="1" stop-color="#a6d8c4"/></linearGradient></defs>
<rect width="400" height="220" fill="url(#a)"/>
<circle cx="70" cy="46" r="26" fill="#fff" opacity=".55"/>
<g fill="#fff" opacity=".5"><ellipse cx="300" cy="40" rx="44" ry="7"/><ellipse cx="334" cy="52" rx="30" ry="6"/></g>
<path d="M0 160c60-30 130-34 200-14s130 6 200-18V220H0z" fill="#8cc9b0"/>
<path d="M0 186c70-18 140-20 210-6s130 2 190-10V220H0z" fill="#6fb59b"/>
<g fill="#4f9a82"><path d="M40 190c-4-16 0-28 8-34 4 12 2 24-8 34zM356 192c4-18 0-30-8-36-4 12-2 26 8 36z"/></g>
`);

const rain = drawn(`
<defs>
 <linearGradient id="a" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#2b3550"/><stop offset="1" stop-color="#55688a"/></linearGradient>
 <radialGradient id="g" cx=".5" cy=".5" r=".5"><stop offset="0" stop-color="#ffd9a0" stop-opacity=".8"/><stop offset="1" stop-color="#ffd9a0" stop-opacity="0"/></radialGradient>
</defs>
<rect width="400" height="220" fill="#241f2e"/>
<rect x="60" y="14" width="280" height="160" rx="6" fill="#171421"/>
<rect x="68" y="22" width="264" height="144" fill="url(#a)"/>
<g opacity=".8"><circle cx="110" cy="120" r="16" fill="url(#g)"/><circle cx="170" cy="132" r="22" fill="url(#g)"/><circle cx="250" cy="112" r="14" fill="#9ed0ff" opacity=".4"/><circle cx="296" cy="128" r="20" fill="url(#g)"/><circle cx="214" cy="100" r="10" fill="#ff9fb5" opacity=".4"/></g>
<path d="M68 150h264v16H68z" fill="#1b1a2c" opacity=".8"/>
<g stroke="#cfe3ff" stroke-width="1" opacity=".55" stroke-linecap="round"><path d="M88 30l-4 14M120 52l-4 14M154 28l-4 14M188 70l-4 14M226 36l-4 14M262 60l-4 14M300 30l-4 14M316 86l-4 14M100 96l-4 14M240 108l-4 14M140 120l-4 14"/></g>
<g fill="#dff0ff" opacity=".6"><path d="M96 70c3 8 3 16 0 22-3-6-3-14 0-22zM204 40c3 8 3 16 0 22-3-6-3-14 0-22zM282 96c3 8 3 16 0 22-3-6-3-14 0-22z"/></g>
<path d="M200 14v152M68 90h264" stroke="#171421" stroke-width="5"/>
<rect x="48" y="170" width="304" height="10" rx="3" fill="#3a2b2f"/>
<rect y="180" width="400" height="40" fill="#2d2330"/>
<g><rect x="70" y="150" width="30" height="20" rx="3" fill="#e8d6b5"/><path d="M100 155h6a5 5 0 010 10h-6" fill="none" stroke="#e8d6b5" stroke-width="2.5"/><path d="M80 146c-3-5 3-7 0-12M90 146c-3-5 3-7 0-12" stroke="#fff" stroke-width="1.5" fill="none" opacity=".5"/></g>
<g><path d="M300 170c-6-14-4-24 0-30 4 6 6 16 0 30zM300 170c-14-4-20-12-19-20 9 2 16 10 19 20zM300 170c14-4 20-12 19-20-9 2-16 10-19 20z" fill="#4f9a6a"/><rect x="292" y="164" width="16" height="8" rx="2" fill="#a8603f"/></g>
`);

const desk = drawn(`
<defs>
 <linearGradient id="w" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#1c1a2e"/><stop offset="1" stop-color="#2c2842"/></linearGradient>
 <radialGradient id="l" cx=".5" cy=".5" r=".5"><stop offset="0" stop-color="#ffcf85" stop-opacity=".7"/><stop offset="1" stop-color="#ffcf85" stop-opacity="0"/></radialGradient>
 <linearGradient id="m" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#6f9bd8"/><stop offset="1" stop-color="#b58bd6"/></linearGradient>
</defs>
<rect width="400" height="220" fill="url(#w)"/>
<g fill="#fff" opacity=".7"><circle cx="30" cy="30" r="1"/><circle cx="64" cy="58" r="1.2"/><circle cx="360" cy="40" r="1"/></g>
<rect x="24" y="22" width="70" height="52" fill="#e9d9bd" opacity=".9"/><path d="M30 68l16-20 12 12 10-8 18 16z" fill="#7b6fa8"/><circle cx="76" cy="38" r="6" fill="#ffd9a0"/>
<rect x="110" y="30" width="50" height="38" fill="#c9708a" opacity=".85"/><rect x="116" y="36" width="38" height="26" fill="#f3e3cf"/>
<rect x="150" y="150" width="130" height="8" rx="2" fill="#4a3a52"/>
<rect x="170" y="62" width="150" height="88" rx="6" fill="#0f0e18"/><rect x="176" y="68" width="138" height="76" rx="3" fill="url(#m)"/>
<g stroke="#fff" stroke-width="2" opacity=".6" stroke-linecap="round"><path d="M186 82h50M186 94h80M186 106h36M186 118h64"/></g>
<rect x="238" y="150" width="10" height="10" fill="#0f0e18"/>
<ellipse cx="344" cy="84" rx="64" ry="62" fill="url(#l)"/>
<g><rect x="332" y="148" width="26" height="6" rx="2" fill="#2a2433"/><path d="M345 148l-14-48" stroke="#2a2433" stroke-width="4" stroke-linecap="round"/><path d="M331 100l22-8 10 18z" fill="#ffcf85"/></g>
<rect y="158" width="400" height="62" fill="#3b2c3e"/><rect y="158" width="400" height="5" fill="#53405a"/>
<g><rect x="40" y="130" width="34" height="28" rx="3" fill="#6fae8a"/><path d="M74 138h6a6 6 0 010 12h-6" fill="none" stroke="#6fae8a" stroke-width="3"/></g>
<g fill="#c25b5b"><rect x="92" y="140" width="46" height="8"/></g><rect x="96" y="148" width="42" height="8" fill="#5b8cc2"/>
`);

const loft = drawn(`
<defs>
 <linearGradient id="w" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#4a2f27"/><stop offset="1" stop-color="#2f1d1c"/></linearGradient>
 <radialGradient id="l" cx=".5" cy=".5" r=".5"><stop offset="0" stop-color="#ffd48a" stop-opacity=".6"/><stop offset="1" stop-color="#ffd48a" stop-opacity="0"/></radialGradient>
</defs>
<rect width="400" height="220" fill="url(#w)"/>
<g fill="#2a1815"><rect x="0" y="0" width="400" height="6"/><rect x="0" y="62" width="400" height="5"/><rect x="0" y="126" width="400" height="5"/></g>
<g>
 <rect x="10" y="22" width="9" height="40" fill="#b5524a"/><rect x="20" y="28" width="8" height="34" fill="#d9a85b"/><rect x="29" y="18" width="10" height="44" fill="#4f7d6a"/><rect x="40" y="30" width="7" height="32" fill="#7a6aa8"/><rect x="48" y="24" width="9" height="38" fill="#c98a5b"/>
 <rect x="70" y="26" width="8" height="36" fill="#d9a85b"/><rect x="79" y="20" width="10" height="42" fill="#b5524a"/><rect x="90" y="30" width="9" height="32" fill="#5b8cc2"/>
 <rect x="300" y="24" width="10" height="38" fill="#4f7d6a"/><rect x="311" y="18" width="8" height="44" fill="#c98a5b"/><rect x="320" y="28" width="9" height="34" fill="#b5524a"/><rect x="330" y="22" width="10" height="40" fill="#d9a85b"/><rect x="342" y="30" width="8" height="32" fill="#7a6aa8"/><rect x="352" y="20" width="10" height="42" fill="#5b8cc2"/><rect x="364" y="26" width="9" height="36" fill="#b5524a"/>
 <rect x="14" y="88" width="9" height="38" fill="#7a6aa8"/><rect x="24" y="80" width="10" height="46" fill="#d9a85b"/><rect x="35" y="90" width="8" height="36" fill="#b5524a"/><rect x="44" y="84" width="10" height="42" fill="#4f7d6a"/>
 <rect x="330" y="86" width="9" height="40" fill="#c98a5b"/><rect x="340" y="80" width="10" height="46" fill="#5b8cc2"/><rect x="351" y="90" width="8" height="36" fill="#d9a85b"/><rect x="360" y="84" width="10" height="42" fill="#b5524a"/>
</g>
<g stroke="#e8b866" stroke-width="1" fill="none" opacity=".8"><path d="M110 0v36M170 0v44M230 0v30M290 0v40"/></g>
<g fill="#ffd48a"><circle cx="110" cy="38" r="4"/><circle cx="170" cy="46" r="4"/><circle cx="230" cy="32" r="4"/><circle cx="290" cy="42" r="4"/></g>
<g fill="url(#l)"><circle cx="110" cy="38" r="26"/><circle cx="170" cy="46" r="26"/><circle cx="230" cy="32" r="26"/><circle cx="290" cy="42" r="26"/></g>
<rect y="168" width="400" height="52" fill="#3a2520"/><path d="M0 168h400" stroke="#5a3a2f" stroke-width="3"/>
<ellipse cx="200" cy="198" rx="130" ry="12" fill="#8a3f4a" opacity=".8"/>
<g><rect x="30" y="148" width="36" height="20" rx="2" fill="#d9a85b"/><rect x="34" y="140" width="28" height="8" rx="2" fill="#b5524a"/></g>
`);

const balcony = drawn(`
<defs>
 <linearGradient id="a" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#3a2a63"/><stop offset=".5" stop-color="#c0517a"/><stop offset="1" stop-color="#f6a46a"/></linearGradient>
</defs>
<rect width="400" height="220" fill="url(#a)"/>
<circle cx="120" cy="120" r="24" fill="#ffe0ae" opacity=".9"/>
<g fill="#fff" opacity=".3"><ellipse cx="260" cy="44" rx="50" ry="6"/><ellipse cx="300" cy="58" rx="32" ry="5"/></g>
<path d="M0 150h20v-26h22v-20h26v30h20v-16h28v22h24v-34h24v40h30v-20h24v28h26v-24h22v30h30v-18h24V220H0z" fill="#5c2a55"/>
<path d="M0 176h30v-18h26v14h30v-26h24v30h34v-16h28v22h30v-20h28v16h30v-24h24v28h26V220H0z" fill="#3d1d44"/>
<g fill="#ffd27a" opacity=".85"><rect x="46" y="128" width="3" height="4"/><rect x="150" y="116" width="3" height="4"/><rect x="236" y="130" width="3" height="4"/><rect x="316" y="136" width="3" height="4"/></g>
<rect y="170" width="400" height="8" fill="#2a1830"/>
<g stroke="#2a1830" stroke-width="3"><path d="M0 120v50M40 120v50M80 120v50M120 120v50M160 120v50M200 120v50M240 120v50M280 120v50M320 120v50M360 120v50M400 120v50"/></g>
<rect y="118" width="400" height="5" fill="#2a1830"/>
<rect y="178" width="400" height="42" fill="#33222f"/>
<g><path d="M330 178c-4-16-2-28 2-34 4 8 4 20-2 34zM330 178c-12-4-18-12-17-20 8 2 14 10 17 20zM330 178c12-4 18-12 17-20-8 2-14 10-17 20z" fill="#4f9a6a"/><rect x="322" y="172" width="16" height="8" rx="2" fill="#a8603f"/></g>
<g fill="#ffd27a" opacity=".9"><circle cx="30" cy="112" r="2"/><circle cx="64" cy="106" r="2"/><circle cx="98" cy="112" r="2"/><circle cx="132" cy="106" r="2"/></g>
<path d="M20 112Q40 124 64 106T132 106" stroke="#2a1830" stroke-width="1" fill="none"/>
`);

export const scenes = [
  { id: "", label: "Soft (default)" },
  { id: "clear", label: "Transparent (she lives in the chat)", css: "linear-gradient(transparent, transparent)" },
  { id: "room", label: "Cosy room", css: room },
  { id: "night", label: "Night window", css: night },
  { id: "sunset", label: "Sunset", css: sunset },
  { id: "studio", label: "Studio", css: studio },
  { id: "mint", label: "Mint", css: mint },
  { id: "rain", label: "Rainy window", css: rain },
  { id: "desk", label: "Desk at night", css: desk },
  { id: "loft", label: "Bookshop loft", css: loft },
  { id: "balcony", label: "Balcony dusk", css: balcony },
];

export const imageScene = { id: "image", label: "My picture" };

/** The CSS background for a scene, or "" for the default look. `imageUrl` is only used by the "image" scene. */
export function sceneCss(id, imageUrl = "") {
  if (id === "image") return imageUrl ? `center / cover no-repeat url("${imageUrl}")` : "";
  return scenes.find((scene) => scene.id === id)?.css || "";
}

export const DEFAULT_FRAME = Object.freeze({ zoom: 1, x: 0, y: 0 });

/** A frame kept inside what the server accepts; anything else becomes the default. */
export function clampFrame(frame) {
  const number = (value, low, high, fallback) => (Number.isFinite(Number(value)) ? Math.min(high, Math.max(low, Number(value))) : fallback);
  return { zoom: number(frame?.zoom, 0.3, 2.2, 1), x: number(frame?.x, -1, 1, 0), y: number(frame?.y, -1, 1, 0) };
}

/** Scale and place the model: `base` is the default fit, the frame zooms about the middle and nudges by up to half the frame. */
export function placeModel({ width, height, modelWidth, modelHeight, frame, corner = false }) {
  if (corner) {
    // Standing in the chat: sized as if she had a card of her own, and put in the bottom right corner of the whole area.
    const box = { width: Math.min(width, 340), height: Math.min(height * 0.6, 480) };
    const inner = placeModel({ width: box.width, height: box.height, modelWidth, modelHeight, frame });
    return { scale: inner.scale, x: inner.x + (width - box.width), y: inner.y + (height - box.height) };
  }
  const f = clampFrame(frame);
  const base = Math.max((width / modelWidth) * 1.45, height / (modelHeight * 0.55));
  const scale = base * f.zoom;
  return { scale, x: (width - modelWidth * scale) / 2 + f.x * width * 0.5, y: -modelHeight * scale * 0.02 + f.y * height * 0.5 };
}

/** Shrinks a picture to at most `side` pixels on its longest edge and returns it as a JPEG blob (a phone photo can be 12 MB). */
export async function shrinkPicture(file, side = 1600, quality = 0.85) {
  const bitmap = await createImageBitmap(file);
  const ratio = Math.min(1, side / Math.max(bitmap.width, bitmap.height));
  const canvas = document.createElement("canvas");
  canvas.width = Math.max(1, Math.round(bitmap.width * ratio));
  canvas.height = Math.max(1, Math.round(bitmap.height * ratio));
  canvas.getContext("2d").drawImage(bitmap, 0, 0, canvas.width, canvas.height);
  bitmap.close?.();
  return new Promise((resolve, reject) => canvas.toBlob((blob) => (blob ? resolve(blob) : reject(new Error("Could not read that picture"))), "image/jpeg", quality));
}
