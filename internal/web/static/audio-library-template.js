export const audioLibraryHTML = `
  <div class="audio-index" id="audio-index" hidden>
    <div class="audio-browse-head" hidden>
      <div class="audio-toolbar">
      <div class="layout-toggle" id="audio-layout-toggle" role="group" aria-label="Layout" hidden>
        <button type="button" data-audio-layout="covers" aria-label="Covers" title="Covers" aria-pressed="true"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 5h6v6H4zM14 5h6v6h-6zM4 15h6v4H4zM14 15h6v4h-6z" /></svg></button>
        <button type="button" data-audio-layout="list" aria-label="List" title="List" aria-pressed="false"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 7h16M4 12h16M4 17h16" /></svg></button>
      </div>
      <details class="audio-view-menu">
        <summary aria-label="Sorting and options" title="Sorting and options">
          <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 8h9M17 8h3M4 16h3M11 16h9" /><circle cx="15" cy="8" r="2" /><circle cx="9" cy="16" r="2" /></svg>
        </summary>
        <div>
          <label id="podcast-state-field" hidden><span>Episodes</span><select id="podcast-state"><option value="all">All episodes</option><option value="unplayed">Unplayed</option><option value="played">Played</option></select></label>
          <label id="music-browse-field" hidden><span>Browse</span><select id="music-browse"><option value="discover">Discover</option><option value="recently-played">Recently played</option><option value="tracks">Tracks</option><option value="queue">Queue</option><option value="albums">Albums</option><option value="artists">Artists</option><option value="favorites">Favorites</option><option value="playlists">Playlists</option></select></label>
          <label id="music-layout-field" hidden aria-hidden="true" class="audio-layout-state"><span>Collections</span><select id="music-layout"><option value="covers">Covers</option><option value="list">List</option></select></label>
          <label id="podcast-layout-field" hidden aria-hidden="true" class="audio-layout-state"><span>Shows</span><select id="podcast-layout"><option value="covers">Covers</option><option value="list">List</option></select></label>
          <label id="audiobook-series-layout-field" hidden aria-hidden="true" class="audio-layout-state"><span>Series</span><select id="audiobook-series-layout"><option value="covers">Covers</option><option value="list">List</option></select></label>
          <label id="audiobook-item-layout-field" hidden aria-hidden="true" class="audio-layout-state"><span>Audiobooks</span><select id="audiobook-item-layout"><option value="covers">Covers</option><option value="list">List</option></select></label>
          <label><span>Sort</span><select id="audio-sort"><option value="recent">Newest first</option><option value="oldest">Oldest first</option><option value="title">Title A–Z</option><option value="artist">Artist / album</option><option value="series">Series / volume</option></select></label>
          <div class="audio-analysis-action" id="audio-analysis-action" hidden><button id="audio-analyze" type="button">Check loudness</button><small id="audio-analysis-status"></small></div>
        </div>
      </details>
      <button id="podcast-manage-toggle" hidden aria-label="Find a podcast show" title="Find a show" aria-expanded="false"><svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="10" cy="10" r="5.5" /><path d="m14 14 4.5 4.5M18.5 5v6M15.5 8h6" /></svg></button>
      </div>
      <nav class="music-destinations" id="music-destinations" aria-label="Browse music" hidden></nav>
      <nav class="music-destinations" id="audiobook-destinations" aria-label="Browse audiobooks" hidden></nav>
      <select id="audiobook-browse" hidden aria-hidden="true" tabindex="-1">
        <option value="continue">Continue</option><option value="recent">Recent</option><option value="series">Series</option><option value="authors">Authors</option><option value="all">All</option>
      </select>
    </div>
    <form class="playlist-create" id="playlist-create" hidden>
      <input id="playlist-name" maxlength="80" required placeholder="New playlist" aria-label="Playlist name" /><button>Create</button><button type="button" id="playlist-create-cancel">Cancel</button><p id="playlist-status" role="status"></p>
    </form>
    <section class="podcast-subscriptions" id="podcast-subscriptions" hidden>
      <div id="podcast-manage" hidden>
        <h2>Find a show</h2>
        <form id="podcast-search"><input id="podcast-query" type="search" required placeholder="Title or creator" aria-label="Search podcast directory" /><button type="submit" aria-label="Search podcast directory" title="Search"><svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="6.5" /><path d="m16 16 4 4" /></svg></button></form>
        <div id="podcast-results"></div>
        <details><summary>Add an RSS feed</summary><form id="podcast-add"><input id="podcast-url" type="url" required placeholder="https://example.com/feed.xml" aria-label="Podcast feed URL" /><button type="submit">Add</button></form></details>
        <details class="podcast-opml"><summary>Import or export subscriptions (OPML)</summary>
          <p>OPML is the file every podcast app uses to move subscriptions between apps. Export a file here to bring your shows into another app, or import one from somewhere else to subscribe to all of its shows at once.</p>
          <div class="podcast-opml-actions">
            <a id="podcast-opml-export" href="/api/podcast-opml" download="mochil-podcasts.opml">Export OPML</a>
            <label class="podcast-opml-import">Import OPML<input id="podcast-opml-file" type="file" accept=".opml,.xml,text/x-opml,text/xml" /></label>
          </div>
          <p id="podcast-opml-status" role="status"></p>
        </details>
        <p id="podcast-status" role="status"></p>
        <details class="podcast-current-feeds"><summary>Subscribed shows</summary><div id="podcast-feeds"></div></details>
      </div>
    </section>
    <div class="virtual-spacer" id="audio-top-spacer" aria-hidden="true"></div>
    <div class="pagination-sentinel" id="audio-start-sentinel" aria-hidden="true"></div>
    <button class="load-more" id="audio-load-previous" hidden>Load previous tracks</button>
    <div class="audio-list" id="audio-items"></div>
    <div class="pagination-sentinel" id="audio-sentinel" aria-hidden="true"></div>
    <button class="load-more" id="audio-load-more" hidden>Load more</button>
    <div class="virtual-spacer" id="audio-bottom-spacer" aria-hidden="true"></div>
  </div>`;

export function mountAudioLibrary(document) {
  if (document.querySelector("#audio-index")) return;
  document
    .querySelector("#collection-index")
    .insertAdjacentHTML("beforebegin", audioLibraryHTML);
  document.querySelector("#shell-context").append(document.querySelector(".audio-browse-head"));
}
