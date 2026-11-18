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
                ><strong>Import preview</strong
                ><small
                  >See what structured paths can safely fill in</small
                ></span
              ><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg>
            </summary>
            <div class="settings-accordion-body">
              <p>
                Read-only preview. Existing metadata, folders, and files are
                never changed.
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
          <details
            class="settings-accordion metadata-ingest-managed"
            id="managed-ingest-workflow"
            hidden
          >
            <summary>
              <span
                ><strong>Managed ingest</strong
                ><small
                  >Review verified copies into managed storage</small
                ></span
              ><svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="m8 10 4 4 4-4" />
              </svg>
            </summary>
            <div class="settings-accordion-body settings-fields">
              <p>
                Paths are relative to the folders configured under Libraries → Managed ingest.
                Incoming files are never moved or deleted.
              </p>
              <div class="managed-ingest-discovery-controls">
                <label
                  >Show<select id="managed-ingest-kind">
                    <option value="">All recognized media</option>
                    <option value="publication">Books and comics</option>
                    <option value="music">Music and audio</option>
                    <option value="video">Video</option>
                  </select></label
                ><button type="button" id="managed-ingest-discover">
                  Inspect incoming folder
                </button>
              </div>
              <p id="managed-ingest-discovery-status" role="status"></p>
              <div
                id="managed-ingest-discovery-results"
                class="managed-ingest-discovery-results"
                hidden
              ></div>
              <button type="button" id="managed-ingest-discovery-more" hidden>
                Show more files
              </button>
              <div
                id="managed-ingest-batch-actions"
                class="managed-ingest-batch-actions"
                hidden
              >
                <span
                  ><strong id="managed-ingest-batch-count">0 selected</strong
                  ><small
                    >Check all destinations together before creating one
                    plan.</small
                  ></span
                ><button type="button" id="managed-ingest-batch-preview">
                  Review selected</button
                ><button type="button" id="managed-ingest-batch-clear">
                  Clear
                </button>
              </div>
              <section
                id="managed-ingest-batch-review"
                class="managed-ingest-batch-review"
                hidden
                aria-label="Batch destination review"
              >
                <header>
                  <strong>Batch review</strong
                  ><small
                    id="managed-ingest-batch-status"
                    role="status"
                  ></small>
                </header>
                <div id="managed-ingest-batch-entries"></div>
                <div>
                  <button
                    type="button"
                    class="primary"
                    id="managed-ingest-batch-plan"
                    hidden
                  >
                    Create verified plan
                  </button>
                </div>
              </section>
              <label
                >Incoming file<input
                  id="managed-ingest-source"
                  maxlength="2048"
                  autocomplete="off"
                  placeholder="Author/Book.epub"
              /></label>
              <label
                >Final managed path<input
                  id="managed-ingest-destination"
                  maxlength="2048"
                  autocomplete="off"
                  placeholder="Author/Series/Book.epub"
              /></label>
              <div>
                <button type="button" class="primary" id="managed-ingest-plan">
                  Review copy
                </button>
              </div>
              <p id="managed-ingest-status" role="status"></p>
              <section
                id="managed-ingest-review"
                hidden
                aria-label="Managed ingest review"
              >
                <p><strong id="managed-ingest-review-paths"></strong></p>
                <small
                  >Type the transaction ID below. Mochil copies, verifies,
                  and publishes only if the destination is still unused.</small
                >
                <label
                  >Transaction ID<input
                    id="managed-ingest-confirmation"
                    autocomplete="off"
                    spellcheck="false"
                /></label>
                <div>
                  <button
                    type="button"
                    class="primary"
                    id="managed-ingest-execute"
                    disabled
                  >
                    Copy verified file
                  </button>
                  <button type="button" id="managed-ingest-discard">
                    Cancel plan
                  </button>
                </div>
              </section>
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
