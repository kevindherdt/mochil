import { favoriteTrackView } from './music-favorite-view.js?v=1'
import {patchItems} from './audio-items-patch.js?v=3'
import { loadSpokenMediaView } from './lazy-spoken-media-view.js?v=1'
import { createLazyAudiobookChapters } from './lazy-audiobook-chapters.js?v=1'
import { createMusicState } from './music-state.js?v=14'
import { createReplayGain } from './replaygain.js?v=1'
import { createGaplessPlayback } from './gapless-playback.js?v=1'
import { createAudioProgress, startPosition } from './audio-progress.js?v=4'
import { createPlayerRestore } from './audio-player-restore.js?v=2'
import { lookupPool } from './audio-item-lookup.js?v=1'
import { createContinuousPager } from './continuous-pager.js?v=4'
import { createBoundedWindow } from './bounded-window.js?v=1'
import { relativePublishedDate } from './date-format.js?v=1'
import { mountAudioPlayer } from './audio-player-template.js?v=8'
import { mountAudioLibrary } from './audio-library-template.js?v=12'
import { createMusicDestinations } from './music-destinations.js?v=10'
import { createAudiobookDestinations } from './audiobook-destinations.js?v=3'
import { installContextMenuAccelerator } from './context-menu-accelerator.js?v=1'
import { audioClock as clock, createAudioItemView } from './audio-item-view.js?v=13'
import { musicLibraryView } from './music-library-view.js?v=23'
import { musicAlphabetEnabled, musicEffectiveSort, musicNavigationEntries, musicTitleBucket } from './music-navigation-policy.js?v=3'
import { createAudioViewPreferences } from './audio-view-preferences.js?v=1'
import { createMusicGroupCatalogue } from './music-group-catalogue.js?v=2'
import { createMusicDragBindings } from './music-drag-bindings.js?v=1'
import { musicSourceFacts } from './music-source-facts.js?v=1'
import { createLazyMusicTrackInfo } from './lazy-music-track-info.js?v=2'
import { createAudioArtworkLoader } from './audio-artwork-loader.js?v=8'
import { createAudioPlaybackOrder } from './audio-playback-order.js?v=1'
import { createAudioPlayerChrome } from './audio-player-chrome.js?v=3'
import { createMusicPlayerQueue } from './audio-player-queue.js?v=4'
import { createLazyAudiobookCatalogue, createLazyPodcastCatalogue } from './lazy-spoken-catalogues.js?v=2'
import { createAudioPlaybackSession } from './audio-playback-session.js?v=6'
import { createSpatialNavigation } from './spatial-navigation.js?v=1'
import { createMusicDiscoveryLoader } from './music-discovery-loader.js?v=2'
import { bindAudioTransport } from './audio-transport-bindings.js?v=9'
import { effectivePlaybackQuality } from './playback-quality.js?v=2'
import { audioPagingDecision, canLoadPreviousMusicPage } from './audio-paging-policy.js?v=1'
import { createLazyAudioAnalysis } from './lazy-audio-analysis.js?v=1'
import { createMusicPlaylistController } from './music-playlist-controller.js?v=7'
import { musicGroupPlaybackTracks } from './music-group-artwork.js?v=1'
import { bindAudioLibraryInteractions, discoveryGroup } from './audio-library-interactions.js?v=4'
import { audioRenderPolicy } from './audio-render-policy.js?v=2'
import { createMusicCollectionWindow } from './music-collection-window.js?v=1'
import { audioGroupLabel, groupedAudioHTML } from './audio-group-view.js?v=1'
import { createMusicDestinationState, restoreAudioDestination } from './destination-scroll-memory.js?v=1'
import { createLoadingHeightGuard } from './loading-height-guard.js?v=1'
import { createAudioRootCatalogue } from './audio-root-catalogue.js?v=1'
import { createAudioSurfaceControls } from './audio-surface-controls.js?v=5'
import { applyMusicTrackWindow, musicTrackMarkup, musicTrackRenderMode } from './music-track-render.js?v=1'

export function createAudioLibrary({ document, $, escapeHTML, requestJSON, setNavigation = () => {}, isLibraryVisible = () => true }) {
  mountAudioPlayer(document)
  mountAudioLibrary(document)
  let request, items = [], total = 0, nextCursor = '', active
  let startOffset = 0, discovery = { played: [], recent: [], favorites: [], albums: [], artists: [] }
  let spokenMediaSurface, routeID = 0
  let activeLibrary = '', activeKind = '', activeSearch = '', loading = false, podcastFeeds = []
  const audioPageSize = 120, musicPageSize = 72
  const key = item => `${item.library_id}\0${item.id}`
  const player = $('#audio-player'), primaryAudio = $('#audio-element'), secondaryAudio = $('#audio-gapless-element'), seek = $('#audio-seek')
  const browseHead=$('.audio-browse-head')
  installContextMenuAccelerator({container:$('#audio-items')})
  const musicDestinations=createMusicDestinations({element:$('#music-destinations'),select:$('#music-browse'),gestureSurface:$('#audio-items'),onChange:()=>$('#music-browse').onchange?.()})
  const audiobookDestinations=createAudiobookDestinations({element:$('#audiobook-destinations'),select:$('#audiobook-browse'),gestureSurface:$('#audio-items'),onChange:()=>$('#audiobook-browse').onchange?.()})
  const surfaceControls=createAudioSurfaceControls({$,musicDestinations,audiobookDestinations})
  const artworkLoader=createAudioArtworkLoader()
  createSpatialNavigation({container:$('#audio-items'),selector:'.music-group-card,.music-playlist-card,.music-track-card>.audio-open,.audio-item>.audio-open,.podcast-card,.audiobook-series-card'})
  const trackWindow=createBoundedWindow({container:$('#audio-items'),topSpacer:$('#audio-top-spacer'),bottomSpacer:$('#audio-bottom-spacer'),maximum:360,trim:72,onRemove:nodes=>artworkLoader.release(nodes)})
  const height=createLoadingHeightGuard($('#audio-items'))
  let audio = primaryAudio
  const gapless = createGaplessPlayback({ primary: primaryAudio, secondary: secondaryAudio, key })
  const replayGain = createReplayGain({ audios: [primaryAudio, secondaryAudio], select: $('#audio-replaygain') })
  const progress=createAudioProgress({requestJSON,getActive:()=>active,getAudio:()=>audio})
  const playbackSession=createAudioPlaybackSession({setStatus:value=>{$('#audio-context').textContent=value??(active?context(active):'')},isActive:item=>item===active,updateTimeline:()=>playerChrome.updateTimeline(audio,active),resumeGain:()=>replayGain.resume(),onTerminal:failed=>{$('#audio-retry').hidden=!failed},onMode:(mode,quality)=>{$('#audio-source-facts').textContent=mode==='converted'?`AAC · ${{high:192,medium:128,low:80}[quality]||192} kbps · Converted`:active?`${musicSourceFacts(active)}${musicSourceFacts(active)?' · ':''}Original`:''}})
  $('#audio-retry').addEventListener('click',()=>playbackSession.retry())
  const natural = new Intl.Collator(undefined,{numeric:true,sensitivity:'base'})
  function artwork(el,url,immediate=false){if(!el||!url)return;el.dataset.artwork=url;artworkLoader.load(el,url,immediate)}
  const groupLabel=item=>audioGroupLabel(item,activeKind,$('#music-browse').value)
  const sortKey=()=>musicEffectiveSort(activeKind,$('#music-browse').value,$('#audio-sort').value)
  function favoriteView(){return favoriteTrackView(music.favoriteItems,{libraryID:activeLibrary,query:activeSearch,sort:$('#audio-sort').value,natural})}
  function visibleItems(){if(activeKind==='podcasts'&&podcastCatalogue.show)return podcastCatalogue.episodes;if(activeKind==='audiobooks'&&audiobookCatalogue.group)return audiobookCatalogue.books;if(activeKind==='music'&&$('#music-browse').value==='favorites')return favoriteView();if(activeKind==='music'&&$('#music-browse').value==='recently-played')return discovery.played;if(activeKind==='music'&&$('#music-browse').value==='queue')return music.queue;return items}
  function find(el, source){const article=el.closest('.audio-item'),browse=$('#music-browse')?.value,pool=source||lookupPool({kind:activeKind,browse,discovery,playlistItems:music.playlistItems,visibleItems});return pool.find(item=>item.id===article?.dataset.item&&item.library_id===article?.dataset.library)}
  let podcastManager
  async function ensurePodcastManager(){
    if(podcastManager)return podcastManager
    const {createPodcastManager}=await import('./podcast-manager.js?v=5')
    podcastManager=createPodcastManager({$,escapeHTML,requestJSON,onChanged:feeds=>{podcastFeeds=feeds;render()}})
    return podcastManager
  }
  const audioAnalysis=createLazyAudioAnalysis({container:$('#audio-analysis-action'),button:$('#audio-analyze'),status:$('#audio-analysis-status'),requestJSON})
  let playerChrome, playerQueue
  const audiobookChapters=createLazyAudiobookChapters({$,getAudio:()=>audio,claimSeek:()=>progress.claim(),player,getActive:()=>active,clock,escapeHTML,expandPlayer:()=>playerChrome.setExpanded(true),updateTimeline:()=>playerChrome.updateTimeline(audio,active),onError:value=>{$('#status').textContent=value}})
  playerChrome=createAudioPlayerChrome({document,$,player,seek,clock,onCollapse:()=>{audiobookChapters.close();playerQueue?.close()}})
  const music=createMusicState({requestJSON,key,onChanged:()=>render()})
  playerQueue=createMusicPlayerQueue({$,escapeHTML,key,music,open,onShow:audiobookChapters.close})
  const musicDiscovery=createMusicDiscoveryLoader({requestJSON,container:$('#audio-items'),setStatus:value=>{$('#status').textContent=value},getLibrary:()=>activeLibrary,getFavorites:favoriteView,onDiscovery:value=>{discovery=value;render()},onRecent:value=>{discovery.played=value;render()}})
  const trackInfo=createLazyMusicTrackInfo({document,escapeHTML,loadArtwork:artwork})
  const musicCatalogue=createMusicGroupCatalogue({requestJSON})
  const rootCatalogue=createAudioRootCatalogue({requestJSON})
  const audiobookCatalogue=createLazyAudiobookCatalogue(requestJSON,audioPageSize)
  const podcastCatalogue=createLazyPodcastCatalogue(requestJSON)
  const playlistController=createMusicPlaylistController({$,container:$('#audio-items'),music,key,render,open})
  const {playlistDrag,queueDrag}=createMusicDragBindings({container:$('#audio-items'),music,selectedPlaylist:()=>playlistController.selected,key,render,status:message=>{$('#status').textContent=message}})
  const localMusicWindow=createMusicCollectionWindow({onMore:render})
  const playbackOrder=createAudioPlaybackOrder({key,getQueue:()=>music.queue,getActive:()=>active})
  const {audiobookCard,context,musicAlbumRow,musicCard,row}=createAudioItemView({escapeHTML,relativePublishedDate,playlists:()=>music.playlists,isFavorite:item=>music.isFavorite(item),browse:()=>$('#music-browse')?.value})
  function renderMusicTrackList(newItems, append) {
    const covers = $('#music-layout').value === 'covers'
    const markup = musicTrackMarkup(newItems, covers, musicCard, row)
    const mode = musicTrackRenderMode(append)
    $('#audio-items').classList.toggle('music-track-grid', covers)
    const removed = applyMusicTrackWindow({ mode, markup, container: $('#audio-items'), trackWindow })
    if (removed.start) { items.splice(0, removed.start); startOffset += removed.start }
    if (removed.end) { items.splice(-removed.end, removed.end); nextCursor = '' }
  }
  function render(newItems = items, append = false) {
    playerQueue.sync(music.queue, active)
    $('#audio-items').removeAttribute('aria-busy')
    $('#audio-items').classList.remove('music-track-grid')
    musicDestinations.sync()
    audiobookDestinations.sync()
    $('#status').textContent=activeKind==='music'?'':items.length===total?`${total.toLocaleString()} audio items`:`${items.length.toLocaleString()} of ${total.toLocaleString()}`
    const browse=activeKind==='music'?$('#music-browse').value:activeKind==='audiobooks'?$('#audiobook-browse').value:''
    const spokenView=spokenMediaSurface?.({kind:activeKind,browse,items,podcast:{feeds:podcastFeeds,show:podcastCatalogue.show,episodes:podcastCatalogue.episodes,total:podcastCatalogue.total},audiobook:{group:audiobookCatalogue.group,books:audiobookCatalogue.books,groups:audiobookCatalogue.series,groupType:audiobookCatalogue.groupType},layouts:{podcasts:$('#podcast-layout').value,audiobookSeries:$('#audiobook-series-layout').value,audiobookItems:$('#audiobook-item-layout').value},search:activeSearch,natural,row,audiobookCard,escapeHTML})
    if(spokenView){$('#status').textContent=spokenView.status;$('#audio-items').innerHTML=spokenView.html}
    else if(activeKind==='music'&&browse!=='tracks') {const view=localMusicWindow.apply(musicLibraryView({browse,discovery,catalogue:musicCatalogue,playlists:music.playlists,selectedPlaylist:playlistController.selected,favorites:favoriteView(),queue:music.queue,items,layout:$('#music-layout').value,row,card:musicCard,albumRow:(item,index)=>musicAlbumRow(item,index),escapeHTML,loadErrors:music.loadErrors,collectionLimit:localMusicWindow.limit}));if(view){playlistController.setSelected(view.selectedPlaylist);$('#status').textContent=view.status;patchItems($('#audio-items'),view.html)}}
    else if(activeKind==='music'&&browse==='tracks') renderMusicTrackList(newItems,append)
    else $('#audio-items').innerHTML=groupedAudioHTML(items,{label:groupLabel,row,escapeHTML,music:activeKind==='music',total})
    $('#audio-items').querySelectorAll('.audio-glyph').forEach(el=>artwork(el,find(el)?.artwork_url))
    $('#audio-items').querySelectorAll('[data-art]').forEach(el=>artwork(el,el.dataset.art,el.matches('.music-group-heading-art,.music-playlist-heading>.music-playlist-art,.podcast-focus-art,.audiobook-focus-art')))
    playlistDrag.sync(activeKind==='music'&&browse==='playlists'&&!!playlistController.selected)
    queueDrag.sync(activeKind==='music'&&browse==='queue')
    const policy=audioRenderPolicy({kind:activeKind,browse,itemCount:items.length,total,startOffset,visibleItemCount:visibleItems().length,podcast:{show:podcastCatalogue.show,episodeCount:podcastCatalogue.episodes.length,total:podcastCatalogue.total},audiobook:{group:audiobookCatalogue.group,bookCount:audiobookCatalogue.books.length,groupCount:audiobookCatalogue.series.length,total:audiobookCatalogue.total},music:{selectedGroup:musicCatalogue.selectedGroup,groupCount:musicCatalogue.groups.length,total:musicCatalogue.total}})
    if(activeKind==='music'&&browse==='tracks')$('#status').textContent=''
    $('#audio-load-more').hidden=policy.loadMoreHidden;$('#audio-load-more').disabled=false;$('#audio-load-more').textContent=policy.loadMoreLabel
    const sections=[...$('#audio-items').querySelectorAll(':scope > .audio-item,:scope > .audio-group,:scope > .music-group-grid > .music-group-card,:scope > .podcast-card,:scope > .podcast-list-group,:scope > .audiobook-series-row,:scope > .audiobook-series-grid > .audiobook-series-card,:scope > .podcast-cover-grid > .podcast-card')]
    const musicSort=sortKey()
    const navigableMusic=activeKind==='music'&&musicAlphabetEnabled({browse,sort:musicSort,selectedGroup:musicCatalogue.selectedGroup,selectedArtist:musicCatalogue.selectedArtist})
    const navigationEntries=activeKind==='music'?(navigableMusic?musicNavigationEntries(sections,{browse,sort:musicSort}):[]):sections.map(element=>({element,label:element.querySelector('h2,strong')?.textContent||''}))
    setNavigation(navigationEntries,navigableMusic?{navigation:musicCatalogue.navigation,onSelect:jumpToMusicBucket}:{})
    pager?.refresh()
    height.release()
  }
  async function load(library, search, reset=true, kind=activeKind, requestedOffset=0) {
    if(!library||loading&&!reset)return
    const generation=++routeID
    if(kind==='podcasts'||kind==='audiobooks'){
      const module=await loadSpokenMediaView()
      if(generation!==routeID)return false
      spokenMediaSurface=module.spokenMediaSurface
    }
    let pendingNavigation=null
    browseHead.hidden=false;browseHead.dataset.kind=kind
    const kindChanged=kind!==activeKind,libraryChanged=library!==activeLibrary;activeLibrary=library;activeKind=kind;activeSearch=search||'';$('#audio-index').dataset.kind=kind
	if(kindChanged||libraryChanged){podcastCatalogue.reset();audiobookCatalogue.reset();playlistController.reset();musicCatalogue.reset()}
	audioAnalysis.setContext(activeLibrary,activeKind)
    if(reset){startOffset=Math.max(0,requestedOffset);localMusicWindow.reset()}
	if(kindChanged)restoreAudioDestination({kind,sort:$('#audio-sort'),savedSort:viewPreferences.savedSort(kind),music:$('#music-browse'),audiobooks:$('#audiobook-browse'),storage:localStorage,onMusic:musicDestinationState.setCurrent})
    surfaceControls.show(kind)
    if(reset&&kind==='podcasts'){
      request?.abort();items=[];total=0;$('#audio-items').innerHTML='';$('#status').textContent='Opening subscribed shows…'
      podcastFeeds=await (await ensurePodcastManager()).load();render();return
    }
    if(reset&&kind==='music'){
      const ticket=musicDiscovery.begin()
      await music.load()
      if(!musicDiscovery.isCurrent(ticket))return
      const browse=$('#music-browse').value
      musicCatalogue.reset()
      if(browse==='queue')await music.ensureQueue()
      if(!musicDiscovery.isCurrent(ticket))return
      if(browse==='discover'){await musicDiscovery.loadDiscovery(ticket);return}
      if(browse==='recently-played'){await musicDiscovery.loadRecent(ticket);return}
      if(['favorites','playlists','queue'].includes(browse)){render();return}
      if(['albums','artists'].includes(browse)){
        const [navigationChanged,groupsChanged]=await Promise.all([loadMusicNavigation(false),loadMusicGroups(true,requestedOffset,false)])
        if(musicDiscovery.isCurrent(ticket)&&(navigationChanged||groupsChanged))render()
        return
      }
      pendingNavigation=loadMusicNavigation(false)
    }
    if(reset&&kind==='audiobooks'){
      items=[];total=0;nextCursor=''
      const browse=$('#audiobook-browse').value
      if(browse==='series'){await loadAudiobookSeries();return}
      if(browse==='authors'){await loadAudiobookAuthors();return}
    }
    if(reset){request?.abort();items=[];total=0;nextCursor='';if(kind==='music'&&$('#music-browse').value==='tracks')height.lock();$('#audio-items').innerHTML='';$('#status').textContent=kind==='music'?'':'Opening the audio shelf…'}
    const current=new AbortController();request=current;loading=true
    try{const audiobookBrowse=kind==='audiobooks'?$('#audiobook-browse').value:'';const sort=audiobookBrowse==='all'?'title':audiobookBrowse==='recent'||audiobookBrowse==='continue'?'recent':sortKey();const result=await rootCatalogue.page({library,kind,search,sort,played:audiobookBrowse==='continue'?'continue':'',cursor:!reset&&items.length?nextCursor:'',offset:reset?startOffset:startOffset+items.length,signal:current.signal});if(request!==current)return;total=result.total;nextCursor=result.nextCursor;items.push(...result.items);if(pendingNavigation)await pendingNavigation;if(request!==current)return;render(result.items,!reset&&kind==='music'&&$('#music-browse').value==='tracks')}
    catch(error){if(error.status===409&&!reset){loading=false;return load(library,search,true,kind)}if(error.name!=='AbortError'){$('#status').textContent=`Could not open this shelf: ${error.message}`;height.release();return false}}
    finally{if(request===current){loading=false;height.release()}pager?.refresh()}
    return true
  }
  async function loadMusicGroups(reset=true, requestedOffset=0, renderOnChange=true){
    const browse=$('#music-browse').value
    if(reset)$('#audio-items').setAttribute('aria-busy','true')
    try{
      const changed=await musicCatalogue.loadGroups({library:activeLibrary,browse,search:activeSearch,reset,requestedOffset})
      if(changed&&renderOnChange)render()
      return changed
    }catch(error){if(error.name!=='AbortError')$('#status').textContent=`Could not open music collections: ${error.message}`;return false}
    finally{pager?.refresh()}
  }
  let musicListScroll=null
  async function openMusicGroup(group){
    const browse=$('#music-browse').value
    if(!musicCatalogue.selectedArtist&&!musicCatalogue.selectedGroup)musicListScroll=window.scrollY
    $('#audio-items').setAttribute('aria-busy','true')
    try{
      const result=await musicCatalogue.open(group,{library:activeLibrary,browse})
      if(!result)return
      items=musicGroupPlaybackTracks(result.tracks,group);total=items.length;nextCursor='';render()
    }catch(error){if(error.name!=='AbortError')$('#status').textContent=`Could not open this collection: ${error.message}`}
  }
  async function closeMusicGroup(){
    const level=await musicCatalogue.back({library:activeLibrary,search:activeSearch})
    if(level==='artist-or-albums'){items=[];total=0}
    render()
    // Back at the list the person came from: put them where they were.
    if(musicListScroll!==null&&(level==='artists'||(level==='artist-or-albums'&&!musicCatalogue.selectedArtist))){const at=musicListScroll;musicListScroll=null;requestAnimationFrame(()=>window.scrollTo(0,at))}
  }
  async function loadMusicNavigation(renderOnChange=true){const browse=$('#music-browse').value,sort=sortKey();if(!musicAlphabetEnabled({browse,sort,selectedGroup:musicCatalogue.selectedGroup,selectedArtist:musicCatalogue.selectedArtist})){musicCatalogue.clearNavigation();return false}try{const changed=await musicCatalogue.loadNavigation({library:activeLibrary,sort,mode:browse,search:activeSearch});if(changed&&renderOnChange)render();return changed}catch(error){if(error.name!=='AbortError')musicCatalogue.clearNavigation();return false}}
  async function jumpToMusicBucket(bucket){const target=musicCatalogue.navigation.find(item=>item.bucket===bucket);if(!target)return;const browse=$('#music-browse').value,offset=Math.max(0,target.offset-20);if(['albums','artists'].includes(browse))await loadMusicGroups(true,offset);else await load(activeLibrary,activeSearch,true,activeKind,offset);const element=[...$('#audio-items').querySelectorAll(':scope > .audio-item,:scope > .audio-group,:scope > .music-group-grid > .music-group-card')].find(node=>musicTitleBucket(node.querySelector('h2,strong')?.textContent)===bucket);element?.scrollIntoView({block:'start'})}
  async function loadPreviousTracks(){if(startOffset<=0||loading)return false;const offset=Math.max(0,startOffset-musicPageSize),current=new AbortController();request=current;loading=true;try{const result=await rootCatalogue.page({library:activeLibrary,kind:'music',search:activeSearch,sort:sortKey(),offset,limit:startOffset-offset,signal:current.signal});if(request!==current)return false;startOffset=offset;items.unshift(...result.items);render(result.items,'prepend');return true}catch(error){if(error.name!=='AbortError')$('#status').textContent=`Could not load earlier tracks: ${error.message}`;return false}finally{if(request===current)loading=false;previousPager?.refresh()}}
  async function loadAudiobookSeries(){request?.abort();$('#audio-items').innerHTML='<p class="empty-state">Opening audiobook series…</p>';try{if(await audiobookCatalogue.loadSeries({library:activeLibrary,search:activeSearch,sort:$('#audio-sort').value}))render()}catch(error){if(error.name!=='AbortError')$('#status').textContent=`Could not open audiobooks: ${error.message}`}}
  async function loadAudiobookAuthors(){request?.abort();$('#audio-items').innerHTML='<p class="empty-state">Opening audiobook authors…</p>';try{if(await audiobookCatalogue.loadAuthors({library:activeLibrary,search:activeSearch}))render()}catch(error){if(error.name!=='AbortError')$('#status').textContent=`Could not open audiobook authors: ${error.message}`}}
  async function loadAudiobookRoot(){return $('#audiobook-browse').value==='authors'?loadAudiobookAuthors():loadAudiobookSeries()}
  async function loadAudiobookGroup(name,reset=true){if(reset)$('#audio-items').innerHTML='<p class="empty-state">Opening audiobooks…</p>';try{if(await audiobookCatalogue.loadGroup(name,{library:activeLibrary,search:activeSearch,reset}))render()}catch(error){if(error.name!=='AbortError')$('#status').textContent=`Could not open this audiobook series: ${error.message}`}}
  async function loadPodcastShow(name,reset=true){if(reset)$('#audio-items').innerHTML='<p class="empty-state">Opening episodes…</p>';try{if(await podcastCatalogue.loadShow(name,{library:activeLibrary,search:activeSearch,sort:$('#audio-sort').value,played:$('#podcast-state').value,reset}))render()}catch(error){if(error.name!=='AbortError')$('#status').textContent=`Could not open this podcast: ${error.message}`}}
  function open(item, fromQueue=false, preparedAudio=null) {
    if(!item)return
    if(item.library_kind==='music'&&!fromQueue){const source=visibleItems(),index=source.findIndex(candidate=>key(candidate)===key(item));music.setQueue(index>=0?source.slice(index):[item])}
    playbackSession.close()
    audio=preparedAudio||gapless.reset(item);active=item;progress.reset(item);$('#audio-title').textContent=item.title;$('#audio-context').textContent=context(item);const sourceFacts=musicSourceFacts(item);$('#audio-source-facts').textContent=`${sourceFacts}${sourceFacts?' · ':''}Original`;replayGain.apply(item,audio)
    const art=$('#audio-player-art');art.querySelector('img')?.remove();artwork(art,item.player_artwork_url||item.artwork_url,true)
    const favorite=$('#audio-player-favorite'),isMusic=item.library_kind==='music';favorite.hidden=!isMusic;favorite.classList.toggle('active',music.isFavorite(item));favorite.setAttribute('aria-pressed',music.isFavorite(item));favorite.setAttribute('aria-label',music.isFavorite(item)?'Remove from favorites':'Add to favorites')
    audiobookChapters.openItem(item)
    audio.playbackRate=Number(item.progress?.playback_rate||localStorage.getItem('mochil-audio-speed')||1);$('#audio-speed').value=String(audio.playbackRate);player.hidden=false
    playbackSession.open({identity:key(item),item,audio,resumeAt:preparedAudio?0:startPosition(item),quality:effectivePlaybackQuality()})
    playerQueue.sync(music.queue, active)
    if('mediaSession'in navigator){const mediaArtwork=item.player_artwork_url||item.artwork_url;navigator.mediaSession.metadata=new MediaMetadata({title:item.title,artist:item.artist||'',album:item.album||item.series||'',artwork:mediaArtwork?[{src:mediaArtwork}]:[]})}
  }
  const save=(force=false,item=active,element=audio)=>progress.save(force,item,element)
  async function markPodcastPlayed(item, played){
    const endpoint=`/api/audio-progress/${encodeURIComponent(item.library_id)}/${encodeURIComponent(item.id)}`
    await requestJSON(endpoint,{method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify({finished:played})})
    if(podcastCatalogue.show)await loadPodcastShow(podcastCatalogue.show,true);else await load(activeLibrary,activeSearch,true,activeKind)
  }
  function adjacentItem(delta){return playbackOrder.adjacent(delta)}
  function adjacent(delta){const next=adjacentItem(delta);if(next)open(next,true)}
  const viewPreferences=createAudioViewPreferences({storage:localStorage,viewMenu:$('.audio-view-menu'),sort:$('#audio-sort'),podcastState:$('#podcast-state'),podcastLayout:$('#podcast-layout'),musicLayout:$('#music-layout'),audiobookSeriesLayout:$('#audiobook-series-layout'),audiobookItemLayout:$('#audiobook-item-layout'),activeKind:()=>activeKind,onSort:()=>podcastCatalogue.show?loadPodcastShow(podcastCatalogue.show,true):audiobookCatalogue.group?loadAudiobookGroup(audiobookCatalogue.group,true):load(activeLibrary,activeSearch,true,activeKind),onPodcastState:()=>podcastCatalogue.show?loadPodcastShow(podcastCatalogue.show,true):load(activeLibrary,activeSearch,true,activeKind),onPodcastLayout:()=>{podcastCatalogue.closeShow();render()},onMusicLayout:render,onAudiobookSeriesLayout:render,onAudiobookItemLayout:render})
  const closeViewMenu=viewPreferences.closeMenu
  const musicDestinationState=createMusicDestinationState({select:$('#music-browse'),sort:$('#audio-sort'),storage:localStorage,viewport:document.defaultView||globalThis,getLibrary:()=>activeLibrary,closeMenu:closeViewMenu,resetWindow:localMusicWindow.reset,resetPlaylist:playlistController.reset,hideCreate:()=>{$('#playlist-create').hidden=true},reload:()=>load(activeLibrary,activeSearch,true,activeKind)})
  $('#music-browse').onchange=musicDestinationState.change
  $('#audiobook-browse').onchange=async()=>{closeViewMenu();localStorage.setItem('mochil-audiobook-browse',$('#audiobook-browse').value);audiobookCatalogue.reset();await load(activeLibrary,activeSearch,true,activeKind)}
  const { setShuffle } = bindAudioTransport({
    $, primaryAudio, secondaryAudio, player, playerChrome, progress, save,
    getAudio: () => audio, setAudio: value => { audio = value },
    getActive: () => active, setActive: value => { active = value },
    music, playbackOrder, open, adjacent, adjacentItem, gapless,
    audiobookChapters, context, replayGain, playbackSession,
    directPlaybackEnabled: () => effectivePlaybackQuality() === 'maximum',
  })
  const restorer=createPlayerRestore({getAudio:()=>audio,getActive:()=>active,key,music,open})
  bindAudioLibraryInteractions({
    container:$('#audio-items'),playlistController,musicCatalogue,podcastCatalogue,
    getItems:()=>items,findItem:find,groupLabel,getActive:()=>active,getKind:()=>activeKind,getBrowse:()=>$('#music-browse').value,getDiscovery:()=>discovery,
    openMusicGroup,closeMusicGroup,loadPodcastShow,loadAudiobookGroup,loadAudiobookRoot,markPodcastPlayed,open,
    setQueue:music.setQueue,setShuffle,orderedQueue:playbackOrder.ordered,playNext:music.playNext,addToQueue:music.addToQueue,toggleFavorite:music.toggleFavorite,openTrackInfo:trackInfo.open,
    clearQueue:music.clearQueue,retryQueueSave:music.retryQueueSave,removeFromQueue:music.removeFromQueue,moveInQueue:music.moveInQueue,render,confirmClear:()=>confirm('Clear the current queue?'),
    selectBrowse:musicDestinationState.select,selectSort:musicDestinationState.selectSort,selectPlaylist:playlistController.setSelected,
    reload:()=>load(activeLibrary,activeSearch,true,activeKind),findDiscoveryGroup:discoveryGroup,
  })
  const pagingDecision=()=>audioPagingDecision({visible:!$('#audio-index').hidden,kind:activeKind,browse:activeKind==='music'?$('#music-browse').value:activeKind==='audiobooks'?$('#audiobook-browse').value:'',loading,itemCount:items.length,total,startOffset,podcast:{show:podcastCatalogue.show,count:podcastCatalogue.episodes.length,total:podcastCatalogue.total,loading:podcastCatalogue.loading},audiobook:{group:audiobookCatalogue.group,count:audiobookCatalogue.books.length,total:audiobookCatalogue.total,loading:audiobookCatalogue.loading},music:{selectedGroup:musicCatalogue.selectedGroup,count:musicCatalogue.groups.length,total:musicCatalogue.total,nextOffset:musicCatalogue.nextOffset,localHasMore:localMusicWindow.hasMore}})
  const pager=createContinuousPager({sentinel:$('#audio-sentinel'),button:$('#audio-load-more'),label:'audio',canLoad:()=>pagingDecision().canLoad,autoLoad:()=>pagingDecision().autoLoad,load:()=>{const target=pagingDecision().target;if(target==='local-music')return localMusicWindow.more();return target==='podcast-show'?loadPodcastShow(podcastCatalogue.show,false):target==='audiobook-group'?loadAudiobookGroup(audiobookCatalogue.group,false):target==='music-groups'?loadMusicGroups(false):load(activeLibrary,activeSearch,false,activeKind)}})
  const previousPager=createContinuousPager({sentinel:$('#audio-start-sentinel'),button:$('#audio-load-previous'),label:'previous tracks',canLoad:()=>canLoadPreviousMusicPage({visible:!$('#audio-index').hidden,kind:activeKind,browse:$('#music-browse').value,loading,startOffset}),load:loadPreviousTracks,rootMargin:'500px'})
  return {load,open,restore:restorer.restore,abort:()=>{routeID++;musicDiscovery.cancel();audioAnalysis.dispose();browseHead.hidden=true;request?.abort();musicCatalogue.abort();audiobookCatalogue.abort();podcastCatalogue.abort()}}
}
