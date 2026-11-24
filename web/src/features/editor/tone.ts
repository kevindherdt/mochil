// tone.ts — one judgement per value, and one colour per judgement.
//
// Kevin: "if a match is 0% show it red for example or 100% is green 50% yellow … so we can make metadata editing and
// quickly glancing what is good or bad and what needs work a lot easier."
//
// The rule is three tones, not a gradient: a spreadsheet read at a glance needs "look at this", "this is fine" and
// "this is unsettled", and a fourth shade in between stops being glanceable. Colours come from the theme tokens
// (--color-error, --color-warning, --color-success, --color-muted), so e-ink gets the text colour and a dark theme
// gets its own hues without this file knowing either.
export type Tone = "poor" | "fair" | "good" | "quiet";

const TONE_COLOR: Record<Tone, string> = {
  poor: "var(--color-error)",
  fair: "var(--color-warning)",
  good: "var(--color-success)",
  quiet: "var(--color-muted)",
};

export function toneColor(tone: Tone): string {
  return TONE_COLOR[tone];
}

/** A match score, 0–1: a confident match is green, a coin flip is amber, a guess is red. */
export function scoreTone(score: number | undefined): Tone {
  if (score === undefined || Number.isNaN(score)) return "quiet";
  if (score >= 0.9) return "good";
  if (score >= 0.5) return "fair";
  return "poor";
}

// What each issue the server reports means for the book. Anything unlisted is quiet: a badge like "side_story" is a
// fact about the book, not a complaint, and colouring it red would train the eye to ignore red.
const ISSUE_TONES: Record<string, Tone> = {
  // The file and the metadata disagree, or the book says the same thing twice: both cost real time.
  isbn_differs: "poor",
  duplicate: "poor",
  no_title: "poor",
  match_none: "poor",
  hardcover_nomatch: "poor",
  hardcover_doubt: "poor",
  // Empty or unsettled: worth a look, not an error.
  no_description: "fair",
  no_isbn: "fair",
  isbn_unsure: "fair",
  match_unsure: "fair",
  hardcover_ignored: "fair",
  // Something was found and agrees with the book.
  isbn_found: "good",
  match_found: "good",
};

export function issueTone(issue: string): Tone {
  return ISSUE_TONES[issue] ?? "quiet";
}

/** Whether the found ISBN agrees with the metadata, as a tone: agreement is good, a mismatch is the thing to fix. */
export function isbnTone(state: "none" | "missing" | "same" | "differs"): Tone {
  switch (state) {
    case "same":
      return "good";
    case "differs":
      return "poor";
    case "missing":
      return "fair";
    default:
      return "quiet";
  }
}

/** How the Hardcover link on a row is doing: a linked book is good, a doubtful or absent one is the thing to fix. */
export function hardcoverTone(state: string): Tone {
  switch (state) {
    case "synced":
    case "linked":
      return "good";
    case "doubt":
    case "nomatch":
      return "poor";
    default:
      return "quiet";
  }
}

/** The worst of a set: one red badge behind a "+3" should colour the count red too. */
export function toneWorst(tones: Tone[]): Tone {
  const order: Tone[] = ["poor", "fair", "good", "quiet"];
  return tones.reduce<Tone>((worst, tone) => (order.indexOf(tone) < order.indexOf(worst) ? tone : worst), "quiet");
}

/** A tinted pill background for a tone, over whatever surface it lands on. */
export function toneWash(tone: Tone): string {
  return tone === "quiet" ? "color-mix(in srgb, var(--color-text) 10%, transparent)" : `color-mix(in srgb, ${toneColor(tone)} 20%, transparent)`;
}

/** The text colour for a tone on a wash or a raised surface.
 *
 *  A hue mixed with itself is the darkest pairing there is: the raw error red on its own 20% wash measured 3.36:1 in the
 *  mochil dark palette, and warning on warning was 4.05:1 in the light one — under the 4.5:1 that 11px text needs, in
 *  nine of the twenty-four tone/palette pairs. Mixing 40% of the text colour in keeps the hue recognisable (a red that
 *  still reads as "wrong") and clears 4.5:1 everywhere. */
export function toneInk(tone: Tone): string {
  return tone === "quiet" ? "var(--color-text)" : `color-mix(in srgb, ${toneColor(tone)} 60%, var(--color-text))`;
}

// The judgement in a form that survives a palette with one ink, or an eye that cannot separate two hues: a character,
// not an icon, so it inherits the type and prints in monochrome. E-ink maps all three tones to its single text colour,
// which made a 16% match and a 95% match physically identical until this existed.
const TONE_MARK: Record<Tone, string> = { poor: "×", fair: "~", good: "✓", quiet: "" };

export function toneMark(tone: Tone): string {
  return TONE_MARK[tone];
}

/** "94% ✓" — the score and its judgement as one run of text, so a screen reader hears both and e-ink shows both. */
export function scoreLabel(score: number | undefined): string {
  if (score === undefined || Number.isNaN(score)) return "";
  const percent = Math.round(Math.max(0, Math.min(1, score)) * 100);
  return `${percent}%${toneMark(scoreTone(score))}`;
}
