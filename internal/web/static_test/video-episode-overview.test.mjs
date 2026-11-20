import test from 'node:test';
import assert from 'node:assert/strict';
import { bindEpisodeOverview } from '../static/video-episode-overview.js';

const episodes = [
  { displayTitle: 'First', overview: 'A longer description that overflows the box' },
  { displayTitle: 'Second', overview: 'Another story that overflows the box too' },
];

function makeClassList() {
  const calls = [];
  return { toggle: (c, on) => calls.push([c, !!on]), calls };
}

function bindWith(overflow) {
  const listeners = {};
  const title = { textContent: '' };
  const description = { textContent: '', scrollHeight: overflow ? 200 : 30, clientHeight: 70 };
  const classList = makeClassList();
  const toggle = { textContent: '', hidden: false, classList, attrs: {},
    setAttribute(k, v) { this.attrs[k] = v; }, getAttribute(k) { return this.attrs[k]; }, focus() {} };
  const overview = { classList: makeClassList() };
  const buttons = ['0', '1'].map(episodeIndex => ({ dataset: { episodeIndex } }));
  const root = {
    querySelector: s => ({ '[data-episode-title]': title, '[data-episode-description]': description, '[data-episode-toggle]': toggle, '.video-episode-overview': overview })[s],
    querySelectorAll: () => buttons,
    addEventListener: (type, fn) => { (listeners[type] ||= []).push(fn); },
  };
  bindEpisodeOverview(root, episodes);
  return { listeners, title, description, toggle, overview, buttons };
}

test('expand then collapse toggles aria-expanded and aria-label', () => {
  const { listeners, toggle } = bindWith(true);
  const clickToggle = () => listeners.click[0]({ target: { closest: s => (s === '[data-episode-toggle]' ? toggle : null) } });
  clickToggle();
  assert.equal(toggle.getAttribute('aria-expanded'), 'true');
  assert.equal(toggle.getAttribute('aria-label'), 'Hide full description');
  clickToggle();
  assert.equal(toggle.getAttribute('aria-expanded'), 'false');
  assert.equal(toggle.getAttribute('aria-label'), 'Show full description');
});

test('toggle visible when description overflows while collapsed', () => {
  const { toggle } = bindWith(true);
  assert.ok(!toggle.classList.calls.some(([c, on]) => c === 'is-hidden' && on));
});

test('toggle hidden when description fits', () => {
  const { toggle } = bindWith(false);
  assert.ok(toggle.classList.calls.some(([c, on]) => c === 'is-hidden' && on));
});

test('hovering another episode switches title and re-measures overflow', () => {
  const { listeners, title, buttons } = bindWith(true);
  listeners.pointerover[0]({ target: { closest: s => (s === '[data-episode-index]' ? buttons[1] : null) }, pointerType: 'mouse' });
  assert.equal(title.textContent, 'Second');
});
