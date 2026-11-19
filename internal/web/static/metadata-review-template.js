export const metadataReviewHTML = `
        <section class="settings-panel" data-settings-panel="metadata">
          <header>
            <h2>Metadata</h2>
            <p id="metadata-intro">
              Inspect one library at a time. Nothing changes until you review
              and confirm it.
            </p>
          </header>
          <div class="metadata-toolbar">
            <label>Shelf<select id="metadata-library"></select></label
            ><label id="metadata-filter-wrap" hidden
              >Show<select id="metadata-filter">
                <option value="pending">Needs review</option>
                <option value="all">All issues</option>
                <option value="accepted">Accepted</option>
                <option value="ignored">Ignored</option>
              </select></label
            ><label id="music-quality-filter-wrap" hidden
              >Finding<select id="music-quality-filter">
                <option value="all">All findings</option>
              </select></label
            ><button class="primary" id="inspect-metadata">
              New inspection
            </button>
          </div>
          <details class="settings-accordion metadata-ingest-preview">
            <summary>
              <span
                ><strong>Metadata path preview</strong
                ><small
                  >See what folder structure can fill in, read-only</small
                ></span
              ><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg>
            </summary>
            <div class="settings-accordion-body">
              <p>
                Read-only preview. Existing metadata, folders, and files are
                never changed. This previews guesses from folder structure for
                books already in this library — for bringing new files in,
                use the <strong>Import…</strong> link at the top of
                Libraries instead.
              </p>
              <button type="button" id="preview-ingest">
                Preview this shelf
              </button>
              <p id="ingest-preview-status" role="status"></p>
              <div id="ingest-preview-results"></div>
              <div id="ingest-preview-sentinel" aria-hidden="true"></div>
              <button type="button" id="ingest-preview-more" hidden>
                Show next proposals
              </button>
            </div>
          </details>
          <p id="metadata-status" class="catalog-status" role="status"></p>
          <div id="metadata-summary" class="metadata-summary" hidden></div>
          <div id="metadata-issues" class="metadata-issues"></div>
          <div id="metadata-apply" class="metadata-apply" hidden></div>
          <details class="settings-accordion metadata-history">
            <summary><span><strong>Previous inspections</strong><small>Reopen an earlier review</small></span><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg></summary>
            <div id="metadata-jobs"></div>
          </details>
        </section>`;

export function mountMetadataReview(document) {
  if (document.querySelector('[data-settings-panel="metadata"]')) return;
  document
    .querySelector('[data-settings-panel="libraries"]')
    .insertAdjacentHTML("beforeend", metadataReviewHTML);
}
