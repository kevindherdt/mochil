import { playlistItemsAfterAction } from './music-state.js?v=14'

export function createMusicPlaylistController({
  $,
  container,
  music,
  key,
  render,
  open,
  confirmAction = globalThis.confirm,
}) {
  let selected = ''
  let selectionVersion = 0
  const loading = new Set()

  function setSelected(name = '') {
    if (selected !== name) selectionVersion++
    selected = name
    const list = music.playlists.find(value=>value.name===name)
    if (list?.detailLoaded === false && !list.detailError && !loading.has(name)) {
      loading.add(name)
      void music.ensurePlaylist(name).finally(() => {
        loading.delete(name)
        if (selected === name) render()
      })
    }
  }

  async function handleClick(event) {
    if (event.target.closest('[data-playlist-retry]')) {
      const list = music.playlists.find(value=>value.name===selected)
      if (list) delete list.detailError
      setSelected(selected)
      render()
      return true
    }
    if (event.target.closest('[data-new-playlist]')) {
      $('#playlist-create').hidden = false
      $('#playlist-name').focus()
      return true
    }
    // Favorites lives inside Playlists and Recently played inside Discover: no tabs of their own, a back arrow instead.
    const goTo = value => {
      const browse = $('#music-browse')
      browse.value = value
      browse.onchange?.()
    }
    if (event.target.closest('[data-open-favorites]')) { goTo('favorites'); return true }
    const back = event.target.closest('[data-browse-back]')
    if (back) { goTo(back.dataset.browseBack); return true }
    const openButton = event.target.closest('[data-open-playlist]')
    if (openButton) {
      setSelected(openButton.dataset.openPlaylist)
      render()
      return true
    }
    if (event.target.closest('[data-playlist-back]')) {
      setSelected()
      render()
      return true
    }
    if (event.target.closest('[data-rename-playlist]')) {
      const form = $('[data-playlist-rename-form]')
      form.hidden = false
      form.querySelector('input').focus()
      event.target.closest('details').open = false
      return true
    }
    if (event.target.closest('[data-cancel-playlist-rename]')) {
      $('[data-playlist-rename-form]').hidden = true
      return true
    }
    const group = event.target.closest('[data-playlist]')
    if (event.target.closest('[data-play-playlist]') && group) {
      const items = music.playlists.find(value => value.name === group.dataset.playlist)?.items || []
      if (items.length) {
        music.setQueue(items)
        open(items[0], true)
      }
      return true
    }
    if (event.target.closest('[data-delete-playlist]') && group) {
      const name = group.dataset.playlist
      if (confirmAction(`Delete playlist “${name}”?`)) await music.deletePlaylist(name)
      return true
    }
    return false
  }

  async function handleItemAction(event, item) {
    const action = event.target.closest('[data-action]')?.dataset.action
    if (action === 'playlist') {
      const name = event.target.closest('.audio-playlist-add').querySelector('select').value
      const list = music.playlists.find(value => value.name === name)
      if (list && !(list.items || []).some(value => key(value) === key(item))) {
        await music.addPlaylistItem(name, item)
      }
      return true
    }
    if (!['playlist-up', 'playlist-down', 'playlist-remove'].includes(action)) return false
    const list = music.playlists.find(value => value.name === selected)
    if (list) await music.editPlaylist(list.name, items => playlistItemsAfterAction(items, item, action, key))
    return true
  }

  container.addEventListener('submit', async event => {
    const form = event.target.closest('[data-playlist-rename-form]')
    if (!form) return
    event.preventDefault()
    const oldName = selected
    const newName = new FormData(form).get('name')?.trim()
    if (!oldName || !newName || oldName === newName) {
      form.hidden = true
      return
    }
    const button = event.submitter
    if (button) button.disabled = true
    setSelected(newName)
    const version = selectionVersion
    try {
      await music.renamePlaylist(oldName, newName)
      if (version === selectionVersion) render()
    } catch (error) {
      if (version !== selectionVersion) return
      selected = oldName
      render()
      const retryForm = $('[data-playlist-rename-form]')
      if (retryForm) {
        retryForm.hidden = false
        const input = retryForm.querySelector('input[name="name"]')
        if (input) input.value = newName
      }
      const status = $('[data-playlist-rename-form] p')
      if (status) status.textContent = error.message
    } finally {
      if (button) button.disabled = false
    }
  }, true)

  $('#playlist-create-cancel').onclick = () => {
    $('#playlist-create').hidden = true
    $('#playlist-status').textContent = ''
  }
  $('#playlist-create').onsubmit = async event => {
    event.preventDefault()
    const name = $('#playlist-name').value.trim()
    const button = event.submitter
    if (button) button.disabled = true
    try {
      await music.createPlaylist(name)
      $('#playlist-name').value = ''
      $('#playlist-status').textContent = ''
      $('#playlist-create').hidden = true
    } catch (error) {
      $('#playlist-status').textContent = error.message
    } finally {
      if (button) button.disabled = false
    }
  }

  return {
    get selected() { return selected },
    setSelected,
    reset: () => setSelected(),
    handleClick,
    handleItemAction,
  }
}
