import { mountAccountsSettings } from './accounts-settings-template.js?v=1'

export function createAccountsSettings({ document, $, requestJSON }) {
  mountAccountsSettings(document)
  let currentAccountID = ''
  let accounts = []
  let recoveryEnabled = false

  const element = (tag, className, text) => {
    const node = document.createElement(tag)
    if (className) node.className = className
    if (text !== undefined) node.textContent = text
    return node
  }
  function status(message, error = false) {
    const target = $('#accounts-status')
    target.textContent = message
    target.dataset.state = error ? 'error' : ''
  }
  async function updateAccount(account, action, password = '') {
    await requestJSON('/api/accounts', { method: 'PATCH', headers: {'Content-Type':'application/json'}, body: JSON.stringify({ id: account.id, action, password }) })
    await load()
  }
  function renderAccount(account) {
    const row = element('details', 'account-row')
    const summary = element('summary')
    const identity = element('span', 'account-identity')
    identity.append(element('strong', '', account.username), element('small', '', `${account.role === 'administrator' ? 'Administrator' : 'Reader'}${account.id === currentAccountID ? ' · Current account' : ''}`))
    summary.append(element('span', 'account-avatar', account.username.trim().slice(0, 2).toUpperCase()), identity, element('span', `account-state${account.disabled ? ' disabled' : ''}`, account.disabled ? 'Disabled' : 'Active'))
    row.append(summary)
    const body = element('div', 'account-actions')
    const resetToggle = element('button', '', 'Reset password'); resetToggle.type = 'button'
    const resetForm = element('form', 'account-inline-form'); resetForm.hidden = true
    const passwordLabel = element('label', '', 'New password')
    const password = element('input'); password.type = 'password'; password.minLength = 10; password.maxLength = 1024; password.required = true; password.autocomplete = 'new-password'
    passwordLabel.append(password)
    const resetSubmit = element('button', 'primary', 'Save new password'); resetSubmit.type = 'submit'
    const cancelReset = element('button', '', 'Cancel'); cancelReset.type = 'button'
    resetForm.append(passwordLabel, resetSubmit, cancelReset)
    resetToggle.onclick = () => { resetForm.hidden = false; resetToggle.hidden = true; password.focus() }
    cancelReset.onclick = () => { resetForm.reset(); resetForm.hidden = true; resetToggle.hidden = false }
    resetForm.onsubmit = async event => {
      event.preventDefault(); if (!resetForm.reportValidity()) return
      resetSubmit.disabled = true
      try { await updateAccount(account, 'reset_password', password.value); status(`Password reset for ${account.username}.`) }
      catch (error) { status(error.message, true); resetSubmit.disabled = false }
    }
    body.append(resetToggle, resetForm)
    const download = element('a', 'settings-link', 'Download data')
    download.href = `/api/accounts/${encodeURIComponent(account.id)}/export`
    download.download = 'bobobooks-account-export.json'
    download.title = `Download ${account.username}'s Mochil data without credentials`
    body.append(download)
    if (account.id !== currentAccountID) {
      if (account.disabled) {
        const enable = element('button', '', 'Enable account'); enable.type = 'button'
        enable.onclick = async () => { enable.disabled = true; try { await updateAccount(account, 'enable'); status(`${account.username} can sign in again.`) } catch (error) { status(error.message, true); enable.disabled = false } }
        body.append(enable)
        const deleteToggle = element('button', 'text-danger', 'Delete account'); deleteToggle.type = 'button'
        const deleteForm = element('form', 'account-inline-form'); deleteForm.hidden = true
        const confirmationLabel = element('label', '', `Type ${account.username} to permanently delete this account and its data`)
        const confirmation = element('input'); confirmation.required = true; confirmation.maxLength = 128; confirmation.autocomplete = 'off'; confirmation.spellcheck = false
        confirmationLabel.append(confirmation)
        const deleteSubmit = element('button', 'danger', 'Delete permanently'); deleteSubmit.type = 'submit'
        const cancelDelete = element('button', '', 'Cancel'); cancelDelete.type = 'button'
        deleteForm.append(confirmationLabel, deleteSubmit, cancelDelete)
        deleteToggle.onclick = () => { deleteToggle.hidden = true; deleteForm.hidden = false; confirmation.focus() }
        cancelDelete.onclick = () => { deleteForm.reset(); deleteForm.hidden = true; deleteToggle.hidden = false; deleteToggle.focus() }
        deleteForm.onsubmit = async event => {
          event.preventDefault(); if (!deleteForm.reportValidity()) return
          deleteSubmit.disabled = true
          try {
            await requestJSON(`/api/accounts/${encodeURIComponent(account.id)}`, { method: 'DELETE', headers: {'Content-Type':'application/json'}, body: JSON.stringify({ confirmation: confirmation.value }) })
            await load(); status(`${account.username} and its Mochil data were permanently deleted.`)
          } catch (error) { status(error.message, true); deleteSubmit.disabled = false }
        }
        body.append(deleteToggle, deleteForm)
      } else {
        const disable = element('button', 'text-danger', 'Disable account'); disable.type = 'button'
        const confirmation = element('div', 'account-confirm'); confirmation.hidden = true
        confirmation.append(element('span', '', `Disconnect ${account.username} from Mochil? Their data will be kept.`))
        const confirm = element('button', 'danger', 'Confirm disable'); confirm.type = 'button'
        const cancel = element('button', '', 'Keep active'); cancel.type = 'button'
        confirmation.append(confirm, cancel)
        disable.onclick = () => { disable.hidden = true; confirmation.hidden = false; confirm.focus() }
        cancel.onclick = () => { confirmation.hidden = true; disable.hidden = false; disable.focus() }
        confirm.onclick = async () => { confirm.disabled = true; try { await updateAccount(account, 'disable'); status(`${account.username} is disabled. Their saved data was kept.`) } catch (error) { status(error.message, true); confirm.disabled = false } }
        body.append(disable, confirmation)
      }
    }
    row.append(body)
    return row
  }
  function render() {
    $('#accounts-list').replaceChildren(...accounts.map(renderAccount))
    $('#accounts-count').textContent = `${accounts.length} ${accounts.length === 1 ? 'account' : 'accounts'} · Registration is off`
    $('#account-recovery').hidden = !recoveryEnabled
  }
  async function load() {
    const {data} = await requestJSON('/api/accounts')
    accounts = data.items || []; currentAccountID = data.current_account_id || ''; recoveryEnabled = data.environment_recovery_enabled === true; render()
  }
  function bind() {
    const toggle = $('#add-account'), form = $('#account-create'), cancel = $('#cancel-account-create')
	const recoveryForm = $('#account-recovery-form')
	recoveryForm.onsubmit = async event => {
	  event.preventDefault(); if (!recoveryForm.reportValidity()) return
	  const button = $('#confirm-account-migration'); button.disabled = true
	  try {
		await requestJSON('/api/accounts', { method: 'PATCH', headers: {'Content-Type':'application/json'}, body: JSON.stringify({ id: currentAccountID, action: 'confirm_recovery', password: $('#account-recovery-password').value }) })
		recoveryForm.reset(); await load(); status('Account migration finished. Only passwords managed in Accounts can sign in now.')
	  } catch (error) { status(error.message, true) }
	  finally { button.disabled = false }
	}
    toggle.onclick = () => { form.hidden = false; toggle.hidden = true; $('#account-username').focus() }
    cancel.onclick = () => { form.reset(); form.hidden = true; toggle.hidden = false; status('') }
    form.onsubmit = async event => {
      event.preventDefault(); if (!form.reportValidity()) return
      const submit = $('#create-account'); submit.disabled = true; status('Creating account…')
      try {
        const username = $('#account-username').value
        await requestJSON('/api/accounts', { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify({ username, password: $('#account-password').value, role: $('#account-role').value }) })
        form.reset(); form.hidden = true; toggle.hidden = false; await load(); status(`${username} can now sign in.`)
      } catch (error) { status(error.message, true) }
      finally { submit.disabled = false }
    }
  }
  return { bind, load }
}
