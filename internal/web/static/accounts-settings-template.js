export const accountsSettingsHTML = `
  <section class="settings-panel" data-settings-panel="accounts" hidden>
    <header>
      <h2>Accounts</h2>
      <p>Choose who can sign in. New accounts can only be created here by an administrator.</p>
    </header>
    <div class="accounts-head">
      <small id="accounts-count"></small>
      <button class="add" id="add-account" type="button">Add account</button>
    </div>
    <section class="account-recovery" id="account-recovery" hidden>
      <strong>Finish account migration</strong>
      <p>Your original environment login is still available as a recovery path. Confirm the current account password to make Accounts the only sign-in source.</p>
      <form id="account-recovery-form" class="account-inline-form">
        <label>Current account password<input id="account-recovery-password" type="password" required maxlength="1024" autocomplete="current-password" /></label>
        <button class="primary" id="confirm-account-migration" type="submit">Finish migration</button>
      </form>
    </section>
    <form class="account-create fields" id="account-create" hidden>
      <label>Username<input id="account-username" required minlength="1" maxlength="128" autocomplete="off" /></label>
      <label>Role<select id="account-role"><option value="user">Reader</option><option value="administrator">Administrator</option></select></label>
      <label>Temporary password<input id="account-password" type="password" required minlength="10" maxlength="1024" autocomplete="new-password" /></label>
      <div class="account-form-actions">
        <button class="primary" id="create-account" type="submit">Create account</button>
        <button id="cancel-account-create" type="button">Cancel</button>
      </div>
    </form>
    <p id="accounts-status" class="catalog-status" role="status"></p>
    <div id="accounts-list" class="accounts-list"></div>
    <section class="account-session">
      <span><strong>This device</strong><small>Sign out of Mochil in this browser.</small></span>
      <form class="logout-form" method="post" action="/logout"><button type="submit">Sign out</button></form>
    </section>
    <small class="accounts-note">Public registration is disabled. Data exports contain personal progress and notes, but never passwords or device tokens. Disable an account to end its access; download its data before using permanent deletion.</small>
  </section>`;

export function mountAccountsSettings(document) {
  if (document.querySelector('[data-settings-panel="accounts"]')) return;
  document
    .querySelector("#settings-view")
    .insertAdjacentHTML("beforeend", accountsSettingsHTML);
}
