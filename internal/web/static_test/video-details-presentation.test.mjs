import test from 'node:test';
import assert from 'node:assert/strict';
import { castHTML, episodeSynopsisHTML, bindTitleDetails } from '../static/video-details-presentation.js';
import * as presentation from '../static/video-details-presentation.js';
const esc = value => String(value).replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('"','&quot;');

test('cast renders names, roles, safe proxied portraits and placeholders', () => {
  const html = castHTML([{name:'A <Actor>', character:'The "Hero"', profile_path:'/abc.jpg'}, {name:'No photo'}, {name:'Bad', profile_path:'https://evil.test/x.jpg'}], esc);
  assert.match(html, /A &lt;Actor>/); assert.match(html, /The &quot;Hero&quot;/);
  assert.match(html, /\/api\/tmdb-portrait\?path=%2Fabc.jpg/);
  assert.doesNotMatch(html, /evil\.test/); assert.match(html, /video-cast-placeholder/);
  assert.equal(castHTML([], esc), '');
});

test('episode synopsis is escaped and absent when unavailable', () => {
  assert.match(episodeSynopsisHTML({episode:1, overview:'A <story>'}, esc), /A &lt;story>/);
  assert.equal(episodeSynopsisHTML({}, esc), '');
  assert.match(episodeSynopsisHTML({overview:'Movie story'}, esc), /Movie story/);
});

test('opening details loads only the selected title and retries errors', async () => {
  const content = {innerHTML:''}, listeners = {};
  const panel = {open:false, dataset:{}, querySelector:()=>content, addEventListener:(name, cb)=>listeners[name]=cb};
  const calls = []; let fail = true;
  bindTitleDetails(panel, {libraryID:'my lib', itemID:'id/1', requestJSON:async url => {calls.push(url); if (fail) throw Error('offline'); return {data:{cast:[{name:'Actor'}]}}}, escapeHTML:esc});
  await listeners.toggle(); assert.equal(calls.length,0);
  panel.open=true; await listeners.toggle(); assert.match(content.innerHTML,/unavailable/i);
  fail=false; await listeners.toggle(); assert.match(content.innerHTML,/Actor/);
  await listeners.toggle(); assert.equal(calls.length,2);
  assert.equal(calls[0],'/api/video-details/my%20lib?item=id%2F1');
});


test('empty details are retried on reopening', async () => {
  const content = {innerHTML:''}, listeners = {}; let calls = 0;
  const panel = {open:true, querySelector:()=>content, addEventListener:(name, cb)=>listeners[name]=cb};
  bindTitleDetails(panel, {libraryID:'lib', itemID:'id', requestJSON:async () => ++calls === 1 ? {data:{cast:[]}} : {data:{cast:[{name:'Actor'}]}}, escapeHTML:esc});
  await listeners.toggle(); assert.match(content.innerHTML,/no cast/i);
  panel.open=false; await listeners.toggle(); panel.open=true; await listeners.toggle();
  assert.equal(calls,2); assert.match(content.innerHTML,/Actor/);
});


test('cast disclosures have distinct escaped accessible names', () => {
  assert.equal(typeof presentation.titleDetailsSummaryHTML, 'function');
  const {titleDetailsSummaryHTML} = presentation;
  assert.equal(titleDetailsSummaryHTML('Movie "<One>"', esc), '<summary aria-label="Cast and details for Movie &quot;&lt;One>&quot;">Cast and details</summary>');
  assert.notEqual(titleDetailsSummaryHTML('Movie One', esc), titleDetailsSummaryHTML('Movie Two', esc));
});
