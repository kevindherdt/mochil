// Library-wide TMDB matching: preview first, then save only the matches whose title and year agree.
// It is a maintenance action over a whole library, so it lives in Settings -> Libraries beside the metadata
// provider it uses, not among the shelf's browse controls where one tap could rewrite a wall of titles.
export function mountVideoMetadataMatch({ document, container, libraryID, libraryName = "", requestJSON, escapeHTML, onApplied = () => {} }) {
	const bar = document.createElement("div");
	bar.className = "video-bulk-match";
	bar.dataset.library = libraryID;
	bar.innerHTML = `<div class="video-bulk-head"><strong>${escapeHTML(libraryName || libraryID)}</strong><small role="status">Not checked yet.</small></div><button type="button" data-bulk-preview>Match with TMDB</button><div data-bulk-results hidden></div>`;
	container.append(bar);
	const status = bar.querySelector("[role=status]");
	const results = bar.querySelector("[data-bulk-results]");
	const endpoint = (apply, offset) => `/api/video-metadata-match/${encodeURIComponent(libraryID)}?apply=${apply ? 1 : 0}&offset=${offset}`;

	async function run(apply) {
		const matched = [];
		const uncertain = [];
		let offset = 0;
		let total = 0;
		for (;;) {
			status.textContent = `${apply ? "Saving" : "Checking"}… ${matched.length + uncertain.length}${total ? ` of ${total}` : ""}`;
			const page = (await requestJSON(endpoint(apply, offset), { method: "POST" })).data;
			matched.push(...(page.matched || []));
			uncertain.push(...(page.uncertain || []));
			total = page.unmatched_total;
			if (page.next_offset == null) break;
			offset = page.next_offset;
		}
		return { matched, uncertain };
	}

	function show({ matched, uncertain }, applied) {
		const rows = (list, label) =>
			list.length
				? `<details><summary>${label} (${list.length})</summary><ul>${list.map((row) => `<li>${escapeHTML(row.name)}${row.title ? ` → ${escapeHTML(row.title)}${row.year ? ` (${row.year})` : ""}` : ""}</li>`).join("")}</ul></details>`
				: "";
		results.hidden = false;
		results.innerHTML = `${rows(matched, applied ? "Saved" : "Would save")}${rows(uncertain, "Left for you to choose")}${!applied && matched.length ? `<button type="button" data-bulk-apply>Save ${matched.length} matches</button>` : ""}`;
		status.textContent = applied
			? `Saved ${matched.length} matches. ${uncertain.length} left for you.`
			: `${matched.length} clear matches, ${uncertain.length} need a person.`;
		const apply = results.querySelector("[data-bulk-apply]");
		if (apply)
			apply.onclick = async () => {
				apply.disabled = true;
				try {
					show(await run(true), true);
					onApplied();
				} catch (error) {
					status.textContent = `Could not save: ${error.message}`;
				}
			};
	}

	bar.querySelector("[data-bulk-preview]").onclick = async (event) => {
		event.currentTarget.disabled = true;
		results.hidden = true;
		try {
			show(await run(false), false);
		} catch (error) {
			status.textContent = `Could not check: ${error.message}`;
		} finally {
			event.currentTarget.disabled = false;
		}
	};
	return { element: bar };
}
