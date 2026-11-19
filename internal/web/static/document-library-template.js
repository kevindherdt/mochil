export const documentLibraryHTML = `
  <div class="document-index" id="document-index" hidden>
    <div class="document-toolbar" id="document-toolbar">
      <label><span class="sr-only">Sort documents</span><select id="document-sort" aria-label="Sort documents"><option value="title">Title A–Z</option><option value="recent">Recently changed</option></select></label>
    </div>
    <div class="virtual-spacer" id="document-top-spacer" aria-hidden="true"></div>
    <div class="pagination-sentinel" id="document-start-sentinel" aria-hidden="true"></div>
    <button class="load-more" id="document-load-previous" hidden>Load previous documents</button>
    <div id="documents"></div>
    <div class="virtual-spacer" id="document-bottom-spacer" aria-hidden="true"></div>
    <div class="pagination-sentinel" id="document-sentinel" aria-hidden="true"></div>
    <button class="load-more" id="document-load-more" hidden>Load more documents</button>
  </div>`;

export function mountDocumentLibrary(document) {
  if (document.querySelector("#document-index")) return;
  document
    .querySelector("#collection-index")
    .insertAdjacentHTML("beforebegin", documentLibraryHTML);
}
