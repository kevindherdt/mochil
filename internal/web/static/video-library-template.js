export const videoLibraryHTML = `
  <div class="video-index" id="video-index" hidden>
    <div class="virtual-spacer" id="video-top-spacer" aria-hidden="true"></div>
    <div class="pagination-sentinel" id="video-start-sentinel" aria-hidden="true"></div>
    <button class="load-more" id="video-load-previous" hidden>Load previous videos</button>
    <div class="video-list" id="videos"></div>
    <div class="virtual-spacer" id="video-bottom-spacer" aria-hidden="true"></div>
    <div class="pagination-sentinel" id="video-sentinel" aria-hidden="true"></div>
    <button class="load-more" id="video-load-more" hidden>Load more videos</button>
  </div>`;

export function mountVideoLibrary(document) {
  if (document.querySelector("#video-index")) return;
  document
    .querySelector("#collection-index")
    .insertAdjacentHTML("beforebegin", videoLibraryHTML);
}
