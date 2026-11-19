export const dashboardHTML = `
  <section class="jump-rail" id="jump-rail" hidden>
    <header><h2>Jump back in</h2><p>Pick up where you left off</p></header>
    <div class="jump-track" id="jump-track"></div>
  </section>
  <details class="finished-rail" id="finished-rail" hidden>
    <summary><span>Finished</span><b id="finished-count"></b></summary>
    <div class="jump-track finished-track" id="finished-track"></div>
  </details>
  <details class="finished-rail held-rail" id="held-rail" hidden>
    <summary><span>On hold &amp; dropped</span><b id="held-count"></b></summary>
    <div class="jump-track finished-track" id="held-track"></div>
  </details>
  <div class="dashboard-lanes">
    <section class="reading-rail dashboard-reading" id="reading-rail" hidden>
      <header><h2 id="reading-heading">Continue reading</h2><button id="finished-toggle" aria-pressed="false">Finished</button></header>
      <div class="reading-track" id="reading-track"></div>
    </section>
    <div class="dashboard-media-lane">
      <section class="reading-rail dashboard-reading video-reading" id="video-reading-rail" hidden>
        <header><h2>Continue watching</h2></header>
        <div class="reading-track" id="video-reading-track"></div>
      </section>
      <section class="reading-rail dashboard-reading audio-reading" id="audio-reading-rail" hidden>
        <header><h2>Continue listening</h2></header>
        <div class="reading-track" id="audio-reading-track"></div>
      </section>
    </div>
  </div>
  <p class="dashboard-empty" id="dashboard-empty" hidden>Media you start will wait for you here.</p>`;

export const dashboardUndoHTML = `<span id="undo-message"></span><button id="undo-action">Undo</button>`;

export function mountDashboard(document) {
  const main = document.querySelector("main");
  if (!document.querySelector("#dashboard-view")) {
    const dashboard = document.createElement("section");
    dashboard.id = "dashboard-view";
    dashboard.innerHTML = dashboardHTML;
    main.prepend(dashboard);
  }
  if (!document.querySelector("#app-problem")) {
    const problem = document.createElement("div");
    problem.id = "app-problem";
    problem.hidden = true;
    main.prepend(problem);
  }
  if (!document.querySelector("#undo-toast")) {
    const toast = document.createElement("div");
    toast.className = "undo-toast";
    toast.id = "undo-toast";
    toast.hidden = true;
    toast.setAttribute("role", "status");
    toast.innerHTML = dashboardUndoHTML;
    document.body.append(toast);
  }
}
