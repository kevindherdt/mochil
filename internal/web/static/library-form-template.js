export const libraryFormHTML = `
  <details class="library-form">
    <summary>
      <i class="ico library-icon" aria-hidden="true"></i>
      <span><strong class="library-name">New library</strong><small class="library-summary-meta"></small></span>
      <label class="library-show" title="Show this library on this device"><input type="checkbox" role="switch" class="library-show-toggle" /></label>
      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg>
    </summary>
    <div class="library-form-body">
      <p class="library-evidence" data-evidence hidden></p>
      <div class="fields">
        <label>Name<input data-field="name" required placeholder="Light Novels" /></label>
        <label>Content<select data-field="kind">
          <option value="books">Books</option>
          <option value="comics">Comics</option>
          <option value="manga">Manga</option>
          <option value="mixed">Books &amp; comics</option>
          <option value="movies">Movies</option>
          <option value="tv">TV series</option>
          <option value="anime">Anime</option>
          <option value="audiobooks">Audiobooks</option>
          <option value="podcasts">Podcasts</option>
          <option value="music">Music</option>
          <option value="documents">Documents &amp; notes</option>
          <option value="vault">Obsidian vault</option>
        </select></label>
        <label>Storage<select data-field="source.type">
          <option value="filesystem">Local folder</option>
          <option value="calibre">Calibre library</option>
        </select></label>
        <label class="wide" data-source="filesystem">Folder path<input data-field="source.path" required placeholder="/media/books" /></label>
        <label class="wide" data-source="calibre">Calibre library folder<input data-field="source.path" required placeholder="/media/Calibre Library" /></label>
      </div>
      <details class="library-advanced">
        <summary>More options</summary>
        <div class="fields">
          <label>Rescan<select data-field="scan_every">
            <option value="">Same as the server default</option>
            <option value="5m">Every 5 minutes</option>
            <option value="30m">Every 30 minutes</option>
            <option value="1h">Every hour</option>
            <option value="6h">Every 6 hours</option>
            <option value="24h">Once a day</option>
            <option value="0">Only when I scan it</option>
          </select><small class="field-help">Large or rarely changing libraries can be checked less often.</small></label>
          <p class="library-metadata-line" data-metadata-line>Metadata: Off</p>
          <label class="check-field" data-page-streaming>
            <input type="checkbox" data-field="page_streaming" />
            <span><strong>OPDS page streaming</strong><small>Let compatible readers open comic pages individually.</small></span>
          </label>
          <label class="check-field" data-source="filesystem" data-vault-only>
            <input type="checkbox" data-field="vault_writes" />
            <span><strong>Allow Wiki editing</strong><small>Off by default. Saves Markdown with revision checks.</small></span>
          </label>
          <label class="check-field" data-source="filesystem" data-publication-only>
            <input type="checkbox" data-field="metadata_writes" />
            <span><strong>Allow metadata repairs</strong><small>Off by default. Previews, backs up, and asks before writing EPUB files.</small></span>
          </label>
          <label class="check-field" data-source="calibre" data-publication-only>
            <input type="checkbox" data-field="metadata_writes" />
            <span><strong>Allow Calibre repairs</strong><small>Off by default. Uses Calibre’s own tool and keeps a restore copy.</small></span>
          </label>
          <details class="library-advanced-more">
            <summary>Advanced</summary>
            <div class="fields">
              <label data-publication-only>Folder layout<select data-field="inference.folder_layout">
                <option value="auto">Automatic</option>
                <option value="series">Folders are series</option>
                <option value="metadata-only">Book metadata only</option>
              </select></label>
              <label data-publication-only>Preferred formats<input data-field="preferred_formats" placeholder="epub, pdf, mobi, azw3" /><small class="field-help">First match wins when a book exists in several formats.</small></label>
            </div>
          </details>
        </div>
      </details>
      <div class="library-form-footer">
        <button type="button" class="library-options-toggle" aria-expanded="false">Options</button>
        <button type="button" class="scan-library" title="Find new files and refresh changed metadata">Scan now</button>
        <p class="library-scan-status" role="status"></p>
        <button class="remove" aria-label="Remove library">Remove library</button>
      </div>
      <p class="media-support-note" hidden></p>
    </div>
  </details>`;

export function mountLibraryFormTemplate(document) {
  if (document.getElementById("library-form")) return;
  const template = document.createElement("template");
  template.id = "library-form";
  template.innerHTML = libraryFormHTML;
  document.body.append(template);
}
