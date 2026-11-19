local WidgetContainer = require("ui/widget/container/widgetcontainer")
local ConfirmBox = require("ui/widget/confirmbox")
local Device = require("device")
local Event = require("ui/event")
local InfoMessage = require("ui/widget/infomessage")
local NetworkMgr = require("ui/network/manager")
local UIManager = require("ui/uimanager")
local http = require("socket.http")
local ltn12 = require("ltn12")
local rapidjson = require("rapidjson")
local socketutil = require("socketutil")
local logger = require("logger")
local util = require("util")
local sha2 = require("ffi/sha2")
local _ = require("gettext")
local config = require("mochil_config")

local loader = ... or {}

local Mochil = WidgetContainer:extend{name = "mochil", is_doc_only = false}
-- Not called "version": KOReader copies the version from the plugin's _meta.lua over a field of that name, and that
-- file is only replaced by a manual install, so after an in-plugin update it would still show the old number.
Mochil.release = "1.27.0"
local ANNOTATION_REVISION_CURSOR_VERSION = 1

-- Unseeded math.random repeats across devices, so read real entropy when the OS has it.
function Mochil.newDeviceId()
    local file = io.open("/dev/urandom", "rb")
    local bytes = file and file:read(12)
    if file then file:close() end
    if bytes and #bytes == 12 then
        return (bytes:gsub(".", function(c) return string.format("%02x", c:byte()) end))
    end
    math.randomseed(os.time() + math.floor(os.clock() * 1000000))
    return tostring(os.time()) .. "-" .. tostring(math.random(100000000, 999999999))
end

-- KOReader's LuaJIT is Lua 5.1: table.pack and table.unpack exist only if it was built with 5.2 compatibility.
local unpack = table.unpack or unpack
local pack = table.pack or function(...) return {n = select("#", ...), ...} end

function Mochil:init()
    Mochil.instance = self
    -- A pending draw may indicate a crash, but can also follow an early exit or another plugin instance.
    -- Fall back for this instance only; never overwrite the user's choice based on this ambiguous marker.
    if G_reader_settings:readSetting("mochil_page_pending") then
        G_reader_settings:saveSetting("mochil_page_pending", false)
        G_reader_settings:saveSetting("mochil_designed_page_crashed", true)
        self.home_safe_mode = true
        logger.err("Mochil: interrupted designed page; using the list for this instance")
    end
    Mochil.hookMenus()
    self.device_id = G_reader_settings:readSetting("mochil_device_id")
    if not self.device_id then self.device_id = Mochil.newDeviceId(); G_reader_settings:saveSetting("mochil_device_id", self.device_id) end
    self.auto_push_callback = function() if self:syncPolicy() ~= "manual" and self:isOnline() then self:push(false) end end
    self.annotation_sync_callback = function() if self:syncPolicy() ~= "manual" then self:syncAnnotations(false) end end
    self.ui.menu:registerToMainMenu(self)
end

-- Lets any gesture (multiswipe, tap zone, physical button) trigger push/pull/sync-highlights without
-- opening the menu, the same registration pattern KOReader's own stock plugins use (e.g. hello.koplugin).
function Mochil:onDispatcherRegisterActions()
    local Dispatcher = require("dispatcher")
    Dispatcher:registerAction("mochil_push", {category="none", event="MochilPush", title=_("Mochil: push reading position"), general=true})
    Dispatcher:registerAction("mochil_pull", {category="none", event="MochilPull", title=_("Mochil: pull reading position"), general=true})
    Dispatcher:registerAction("mochil_sync_highlights", {category="none", event="MochilSyncHighlights", title=_("Mochil: sync highlights"), general=true})
end

function Mochil:onMochilPush() self:push(true) end
function Mochil:onMochilPull() self:pull(true) end
function Mochil:onMochilSyncHighlights() self:syncAnnotations(true) end

function Mochil:digest()
    local saved = self.ui.doc_settings:readSetting("partial_md5_checksum")
    if saved then return saved end
    local ok, value = pcall(util.partialMD5, self.ui.document.file)
    if ok and value then self.ui.doc_settings:saveSetting("partial_md5_checksum", value); return value end
end

-- A response is assembled in memory on a reader with tens of megabytes free, so
-- every fetch carries a ceiling: over it the request fails with "too_large"
-- instead of growing a table until the device is killed for memory.
local MAXIMUM_JSON_BYTES = 8 * 1024 * 1024
local MAXIMUM_PAGE_BYTES = 24 * 1024 * 1024

local function bounded_response_sink(maximum)
    local chunks, received, too_large = {}, 0, false
    local function push(chunk)
        if chunk == nil then return 1 end
        if too_large or #chunk > maximum - received then
            too_large = true
            return nil, "too_large"
        end
        received = received + #chunk
        chunks[#chunks + 1] = chunk
        return 1
    end
    return push, chunks, function() return too_large end
end

-- Background pushes (suspend, close, auto-push) must never hold the UI for long.
function Mochil:request(method, body, status_only, resource, quick, since)
    -- A status check needs no book (the file manager has none), so only look one up for book requests.
    local digest
    if not status_only then
        digest = self:digest()
        if not digest then return nil, "no_book" end
    end
    local push, sink, overflowed = bounded_response_sink(MAXIMUM_JSON_BYTES)
    local payload = nil
    if body then
        local encoded, value = pcall(rapidjson.encode, body)
        if not encoded then logger.warn("Mochil: JSON encoding failed", value); return nil, "encode_error" end
        payload = value
    end
    local path = status_only and ("/api/koreader/v1/" .. (resource or "status")) or ("/api/koreader/v1/" .. (resource or "progress") .. "/" .. digest)
    if since ~= nil then path = path .. "?since=" .. tostring(math.max(0, math.floor(tonumber(since) or 0))) end
    local req = {url=config.server_url .. path, method=method, sink=push, headers={Authorization="Bearer " .. config.token, Accept="application/json"}}
    if payload then req.source=ltn12.source.string(payload); req.headers["Content-Type"]="application/json"; req.headers["Content-Length"]=#payload end
    if quick then
        local block, total = 3, 5
        if self.deadline then
            local remaining = self.deadline - os.time()
            if remaining <= 0 then return nil, "budget" end
            block, total = math.min(block, remaining), math.min(total, remaining)
        end
        socketutil:set_timeout(block, total)
    else socketutil:set_timeout(socketutil.LARGE_BLOCK_TIMEOUT, socketutil.LARGE_TOTAL_TIMEOUT) end
    local called, result, code = pcall(http.request, req)
    socketutil:reset_timeout()
    if not called then
        self.last_error = tostring(result)
        logger.warn("Mochil: network request failed", result)
        return nil, "network"
    end
    -- LuaSocket answers nil plus the reason when it cannot connect at all.
    self.last_error = result == nil and tostring(code) or nil
    if overflowed() then
        self.last_error = "response too large"
        return nil, "too_large"
    end
    local raw=table.concat(sink); code=tonumber(code)
    if code ~= 200 then
        if code then self.last_error = "HTTP " .. tostring(code) end
        return nil, code == 401 and "expired" or ("http_" .. tostring(code or "network"))
    end
    -- Everything the server sends is an object. Anything else (a proxy's error page, a bare number) is not an answer.
    local ok, decoded = pcall(rapidjson.decode, raw); if ok and type(decoded) == "table" then return decoded end
    return nil, "invalid_response"
end

-- A plain-words reason for a failed request, from what LuaSocket or the server said.
function Mochil.describeFailure(reason)
    reason = tostring(reason or ""):lower()
    if reason == "" then return _("The connection failed.") end
    if reason:find("not found", 1, true) or reason:find("name or service", 1, true) or reason:find("resolution", 1, true) or reason:find("nodename", 1, true) then
        return _("The server name could not be looked up. This network may not provide DNS.")
    end
    if reason:find("refused", 1, true) then return _("The server refused the connection.") end
    if reason:find("timeout", 1, true) or reason:find("timed out", 1, true) then return _("The server did not answer in time.") end
    if reason:find("certificate", 1, true) or reason:find("handshake", 1, true) or reason:find("ssl", 1, true) or reason:find("tls", 1, true) then
        return _("A secure connection could not be made. Check that this device's date and time are right.")
    end
    if reason:find("unreachable", 1, true) then return _("The network cannot reach the server.") end
    if reason:find("http 5", 1, true) then return _("The server reported an error.") end
    return reason
end

function Mochil:testConnection()
    local run = function()
        UIManager:nextTick(function()
            local called, body, err = pcall(self.request, self, "GET", nil, true, nil, true)
            local text
            if called and body and body.connected then text=_("Connected to Mochil as ") .. tostring(body.account) .. "."
            elseif called and err == "expired" then text=_("This pairing has expired. Download and install a new plugin from Mochil Settings.")
            else
                text=_("Could not reach Mochil at ") .. config.server_url .. "\n" .. Mochil.describeFailure(called and self.last_error or body)
            end
            UIManager:show(InfoMessage:new{text=text,timeout=8})
        end)
    end
    if NetworkMgr:willRerunWhenConnected(run) then return end
    run()
end

function Mochil:isOnline()
    local ok, connected = pcall(function() return NetworkMgr:isConnected() end)
    return ok and connected and true or false
end

-- Four choices: when the book closes (or the reader sleeps), after a chapter change, after every page, or never on its
-- own. Close and suspend always send the final position unless the policy is manual. Unknown values read as "close".
local SYNC_POLICIES = {close = true, chapter = true, page = true, manual = true}
function Mochil:syncPolicy()
    local policy = G_reader_settings:readSetting("mochil_sync_policy")
    return SYNC_POLICIES[policy] and policy or "close"
end

-- Off by default: see the comment on the "Notes & wiki" menu entry.
function Mochil.wikiEnabled()
    return G_reader_settings:readSetting("mochil_wiki_enabled") == true
end

function Mochil:setSyncPolicy(policy)
    policy = SYNC_POLICIES[policy] and policy or "close"
    G_reader_settings:saveSetting("mochil_sync_policy", policy)
    if policy == "manual" then UIManager:unschedule(self.auto_push_callback); UIManager:unschedule(self.annotation_sync_callback) end
end

function Mochil:currentFragment()
    if not self.ui or not self.ui.document or self.ui.document.info.has_pages or not self.ui.document.getXPointer then return nil end
    local ok, xpointer = pcall(self.ui.document.getXPointer, self.ui.document)
    return ok and xpointer and xpointer:match("^/body/DocFragment%[(%d+)%]") or nil
end

function Mochil:scheduleAutoPush()
    local policy = self:syncPolicy()
    if policy == "manual" or policy == "close" then return end
    if policy == "chapter" then
        local fragment = self:currentFragment()
        if not fragment or not self.last_fragment then self.last_fragment = fragment; return end
        if fragment == self.last_fragment then return end
        self.last_fragment = fragment
    end
    UIManager:unschedule(self.auto_push_callback)
    UIManager:scheduleIn(2, self.auto_push_callback)
end

local function clean_text(value)
    if type(value) ~= "string" then return "" end
    return value:gsub("\194\173", ""):gsub("[\r\n\t ]+", " "):gsub("^%s+", ""):gsub("%s+$", "")
end

local function bound_utf8(value, maximum)
    value = clean_text(value)
    if #value <= maximum then return value end
    local out, bytes = {}, 0
    for _, character in ipairs(util.splitToChars(value)) do
        if bytes + #character > maximum then break end
        out[#out + 1], bytes = character, bytes + #character
    end
    return table.concat(out)
end

local function bound_utf8_from_end(value, maximum)
    value = clean_text(value)
    if #value <= maximum then return value end
    local characters, out, bytes = util.splitToChars(value), {}, 0
    for index = #characters, 1, -1 do
        local character = characters[index]
        if bytes + #character > maximum then break end
        table.insert(out, 1, character); bytes = bytes + #character
    end
    return table.concat(out)
end

local function metadata_text(value)
    if type(value) == "string" then return clean_text(value):sub(1, 2048) end
    if type(value) ~= "table" then return "" end
    local values = {}
    for _, item in pairs(value) do
        if type(item) == "string" and item ~= "" then values[#values + 1] = clean_text(item) end
        if #values >= 16 then break end
    end
    return table.concat(values, " | "):sub(1, 2048)
end

local function move_words(document, xpointer, count, backwards)
    local position = xpointer
    for _ = 1, count do
        local method = backwards and document.getPrevVisibleWordStart or document.getNextVisibleWordEnd
        local ok, next_position = pcall(method, document, position)
        if not ok or not next_position then break end
        position = next_position
    end
    return position
end

local function annotation_key(item)
    local start_ok, start = pcall(rapidjson.encode, item.pos0 or item.page or "")
    local finish_ok, finish = pcall(rapidjson.encode, item.pos1 or item.page or "")
    if type(item.pos0) == "string" then start_ok, start = true, item.pos0 end
    if type(item.pos1) == "string" then finish_ok, finish = true, item.pos1 end
    if not start_ok or not finish_ok then return nil end
    return tostring(start) .. "\0" .. tostring(finish) .. "\0" .. tostring(item.datetime or "")
end

local function annotation_time(value)
    if type(value) ~= "string" then return os.time() * 1000 end
    local year, month, day, hour, minute, second = value:match("^(%d%d%d%d)%-(%d%d)%-(%d%d) (%d%d):(%d%d):(%d%d)$")
    if not year then return os.time() * 1000 end
    return os.time{year=tonumber(year), month=tonumber(month), day=tonumber(day), hour=tonumber(hour), min=tonumber(minute), sec=tonumber(second)} * 1000
end

-- A bookmark is a page, not a text range: KOReader records it as an xpointer in a reflowable book and as a page
-- number in a PDF. Everything below compares bookmarks by the page they land on, so both shapes work.
local function bookmarkTarget(document, value)
    if type(value) == "number" then return value end
    if type(value) ~= "string" or value == "" then return nil end
    if document and type(document.getPageFromXPointer) == "function" then
        local ok, page = pcall(document.getPageFromXPointer, document, value)
        if ok and tonumber(page) then return tonumber(page) end
    end
    return nil
end

-- The id this plugin uploads an item under. A highlight keeps the plain key hash it has always used, so highlights
-- already on the server are not duplicated; bookmarks are namespaced, because a bookmark and a highlight can share
-- a page and must not share an id.
local function annotation_id(item)
    local key = annotation_key(item)
    if not key then return nil end
    if item.drawer then return "koreader:" .. sha2.md5(key) end
    return "koreader:" .. sha2.md5("bookmark\0" .. key)
end

local function bookmarkedPage(annotations, document, page)
    local wanted = bookmarkTarget(document, page)
    if wanted == nil then return false end
    for _, item in ipairs(annotations or {}) do
        if not item.drawer and item.page ~= nil and bookmarkTarget(document, item.page) == wanted then return true end
    end
    return false
end

function Mochil:annotationAnchor(item)
    if type(item.pos0) ~= "string" or type(item.pos1) ~= "string" then return nil end
    local document = self.ui and self.ui.document
    if not document or document.info.has_pages or not document.getTextFromXPointers then return nil end
    local prefix_start = move_words(document, item.pos0, 24, true)
    local suffix_end = move_words(document, item.pos1, 24, false)
    local before_ok, before = pcall(document.getTextFromXPointers, document, prefix_start, item.pos0)
    local after_ok, after = pcall(document.getTextFromXPointers, document, item.pos1, suffix_end)
    local fragment = tonumber(item.pos0:match("^/body/DocFragment%[(%d+)%]"))
    return {version=2, prefix=before_ok and bound_utf8_from_end(before, 2048) or "", exact=bound_utf8(item.text, 2048), suffix=after_ok and bound_utf8(after, 2048) or "", spine_index=fragment and (fragment - 1) or 0, progress=0}
end

local function annotation_datetime(value)
    value = tonumber(value)
    if not value or value <= 0 then return os.date("%Y-%m-%d %H:%M:%S") end
    return os.date("%Y-%m-%d %H:%M:%S", math.floor(value / 1000))
end

function Mochil:resolveAnnotationLocator(remote)
    local document = self.ui and self.ui.document
    if not document or document.info.has_pages then return nil end
    local encoded = remote.locators and remote.locators.koreader
    if type(encoded) == "string" and encoded ~= "" then
        local ok, native = pcall(rapidjson.decode, encoded)
        if ok and type(native)=="table" and type(native.pos0)=="string" and type(native.pos1)=="string"
                and document:isXPointerInDocument(native.pos0) and document:isXPointerInDocument(native.pos1) then
            return native.pos0, native.pos1
        end
    end
    local anchor = remote.anchor
    local exact = clean_text(anchor and (anchor.exact or anchor.text) or remote.quote)
    if exact == "" or not document.findAllText then return nil end
    local ok, hits = pcall(document.findAllText, document, exact, true, 0, 50, false)
    pcall(document.clearSelection, document)
    if not ok or type(hits) ~= "table" then return nil end
    local expected = anchor and anchor.spine_index and ("/body/DocFragment[" .. tostring(anchor.spine_index + 1) .. "]") or nil
    local best, best_score = nil, -1000
    for _, hit in ipairs(hits) do
        local finish = hit["end"] or hit.finish
        if type(hit.start)=="string" and type(finish)=="string" then
            local score = expected and hit.start:sub(1, #expected) == expected and 100 or 0
            local prefix_start = move_words(document, hit.start, 24, true)
            local suffix_end = move_words(document, finish, 24, false)
            local before_ok, before = pcall(document.getTextFromXPointers, document, prefix_start, hit.start)
            local after_ok, after = pcall(document.getTextFromXPointers, document, finish, suffix_end)
            local wanted_prefix, wanted_suffix = clean_text(anchor and anchor.prefix), clean_text(anchor and anchor.suffix)
            before, after = clean_text(before_ok and before or ""), clean_text(after_ok and after or "")
            if wanted_prefix ~= "" and before:sub(-#wanted_prefix) == wanted_prefix then score = score + 35 end
            if wanted_suffix ~= "" and after:find(wanted_suffix, 1, true) then score = score + 35 end
            if score > best_score then best, best_score = {hit.start, finish}, score end
        end
    end
    return best and best[1], best and best[2]
end

function Mochil:applyRemoteAnnotations(remotes, dirty)
    if not self.ui or not self.ui.annotation or not self.ui.rolling then return 0 end
    local annotations, changed = self.ui.annotation.annotations, 0
    self.annotation_imported_ids = {}
    for _, remote in ipairs(remotes or {}) do
        -- A highlight this plugin uploaded carries a koreader:<hash> id, so filtering those out dropped every
        -- highlight a second reader had uploaded: they never appeared, silently and permanently. Echoes are
        -- already prevented by the snapshot/dirty comparison below, not by the shape of the id.
        if type(remote.id)=="string" and remote.id ~= "" then
            if remote.kind == "bookmark" then
                -- A page bookmark has no text range to resolve, only a page, and it must be toggled only in the
                -- direction that is missing: a blind toggle would delete the bookmark that is already there.
                local encoded = remote.locators and remote.locators.koreader
                local decoded, native = false, nil
                if type(encoded) == "string" and encoded ~= "" then decoded, native = pcall(rapidjson.decode, encoded) end
                local page = decoded and type(native) == "table" and native.page or nil
                local document = self.ui and self.ui.document
                local number = bookmarkTarget(document, page)
                local bookmark = self.ui and self.ui.bookmark
                if number and type(bookmark) == "table" and type(bookmark.toggleBookmark) == "function" then
                    local present = bookmarkedPage(annotations, document, page)
                    if remote.deleted then
                        if present then pcall(bookmark.toggleBookmark, bookmark, number); changed = changed + 1 end
                    elseif not present then
                        pcall(bookmark.toggleBookmark, bookmark, number)
                        changed = changed + 1
                    end
                    if not remote.deleted and bookmarkedPage(annotations, document, page) then
                        for _, item in ipairs(annotations) do
                            if not item.drawer and item.page ~= nil and bookmarkTarget(document, item.page) == number then
                                item.mochil_id, item.mochil_revision = remote.id, tonumber(remote.revision) or 0
                                break
                            end
                        end
                        self.annotation_imported_ids[remote.id] = true
                    end
                end
            else
            local index
            for candidate, item in ipairs(annotations) do
                if item.mochil_id == remote.id then index = candidate; break end
            end
            if not index then
                -- A highlight made here has no mochil_id until it comes back from the server, so a deletion
                -- made elsewhere has to be matched by the id this plugin uploads it under.
                for candidate, item in ipairs(annotations) do
                    if annotation_id(item) == remote.id then index = candidate; break end
                end
            end
            if remote.deleted then
                if index then
                    local item = annotations[index]
                    if item.mochil_original_id then
                        item.mochil_id, item.mochil_locators, item.mochil_anchor = nil, nil, nil
                        item.mochil_revision, item.mochil_original_id = nil, nil
                    else
                        -- KOReader's own removal keeps its marks and counters in step; older versions only have the list.
                        local removed = type(self.ui.annotation.removeItemByIndex) == "function"
                            and pcall(self.ui.annotation.removeItemByIndex, self.ui.annotation, index)
                        if not removed then table.remove(annotations, index) end
                    end
                    changed = changed + 1
                end
            else
                local remote_revision = tonumber(remote.revision) or 0
                if not (index and annotations[index].mochil_revision == remote_revision) then
                    local pos0, pos1 = self:resolveAnnotationLocator(remote)
                    if pos0 and pos1 then
                        local duplicate_of_existing = false
                        if not index then
                            local quote = clean_text(remote.quote)
                            for candidate, item in ipairs(annotations) do
                                if item.drawer and item.pos0 == pos0 and item.pos1 == pos1 then
                                    if item.mochil_id and item.mochil_id ~= remote.id then
                                        duplicate_of_existing = true
                                    elseif clean_text(item.text) == quote then
                                        local local_id = item.mochil_id or ("koreader:" .. sha2.md5(annotation_key(item)))
                                        if (dirty and dirty[local_id]) or clean_text(item.note) ~= clean_text(remote.note) then
                                            duplicate_of_existing = true
                                        else
                                            index = candidate
                                            item.mochil_original_id = local_id
                                        end
                                    end
                                    break
                                end
                            end
                        end
                        local item = not duplicate_of_existing and (index and annotations[index] or {
                            page=pos0, pos0=pos0, pos1=pos1, drawer="lighten", color="yellow",
                            datetime=annotation_datetime(remote.created_at),
                        }) or nil
                        if item then
                            item.page, item.pos0, item.pos1 = pos0, pos0, pos1
                            item.text, item.note, item.chapter = clean_text(remote.quote), clean_text(remote.note), clean_text(remote.chapter)
                            if item.note == "" then item.note = nil end
                            if item.chapter == "" then item.chapter = self.ui.toc:getTocTitleByPage(pos0) end
                            item.mochil_id, item.mochil_locators, item.mochil_anchor = remote.id, remote.locators, remote.anchor
                            item.mochil_revision = remote_revision
                            self.annotation_imported_ids[remote.id] = true
                            if index then changed = changed + 1 else self.ui.annotation:addItem(item); changed = changed + 1 end
                        end
                    end
                end
            end
            end
        end
    end
    if changed > 0 then
        self.ui.doc_settings:saveSetting("annotations", annotations)
        if self.ui.view and self.ui.view.footer then self.ui.view.footer:maybeUpdateFooter() end
        UIManager:setDirty(self.ui.dialog or "all", "ui")
    end
    return changed
end

function Mochil:localAnnotations()
    local result = {}
    local annotations = self.ui and self.ui.annotation and self.ui.annotation.annotations or {}
    for _, item in ipairs(annotations) do
        -- A highlight is a text range with a drawer; a page bookmark is not. Both live in the same list, and both
        -- belong on the server: a bookmark made on the e-reader used to be invisible everywhere else.
        local is_bookmark = not item.drawer and item.page ~= nil
        if (item.drawer and clean_text(item.text) ~= "") or is_bookmark then
            local key = annotation_key(item)
            local locator_ok, native_locator
            if is_bookmark then
                locator_ok, native_locator = pcall(rapidjson.encode, {page=item.page})
            else
                locator_ok, native_locator = pcall(rapidjson.encode, {page=item.page, pos0=item.pos0, pos1=item.pos1})
            end
            if key and locator_ok then
                local identifier = item.mochil_id or annotation_id(item)
                if not is_bookmark and not item.mochil_anchor and not item.mochil_local_anchor then
                    item.mochil_local_anchor = self:annotationAnchor(item)
                end
                local locators = type(item.mochil_locators)=="table" and util.tableDeepCopy(item.mochil_locators) or {}
                locators.koreader = native_locator
                local annotation = {
                    id=identifier, kind=is_bookmark and "bookmark" or "highlight",
                    quote=is_bookmark and "" or bound_utf8(item.text, 8192),
                    note=bound_utf8(item.note, 16384),
                    color=is_bookmark and "" or bound_utf8(tostring(item.color or item.drawer or ""), 32),
                    chapter=bound_utf8(item.chapter, 1024), created_at=annotation_time(item.datetime),
                    updated_at=annotation_time(item.datetime_updated or item.datetime), deleted=false, device_id=self.device_id,
                    locators=locators,
                    anchor=is_bookmark and nil or (item.mochil_anchor or item.mochil_local_anchor),
                }
                annotation._signature = sha2.md5(key .. "\0" .. annotation.quote .. "\0" .. annotation.note .. "\0" .. annotation.color .. "\0" .. annotation.chapter)
                result[identifier] = annotation
            else
                logger.warn("Mochil: skipped an annotation whose locator could not be encoded")
            end
        end
    end
    return result
end

function Mochil:syncAnnotations(interactive)
    if not self.ui or not self.ui.document then return end
    local revision = self.ui.doc_settings:readSetting("mochil_annotation_revision") or 0
    if self.ui.doc_settings:readSetting("mochil_annotation_revision_cursor_version") ~= ANNOTATION_REVISION_CURSOR_VERSION then revision = 0 end
    local snapshot = self.ui.doc_settings:readSetting("mochil_annotation_snapshot") or {}
    -- applyRemoteAnnotations fills this, but it returns before doing so on a fixed-layout book (PDF, CBZ), so the
    -- lookup further down must never index nil: that aborted the whole sync and no change ever left the device.
    self.annotation_imported_ids = {}
    local pulled_revision = tonumber(revision) or 0
    local remote_revisions = {}
    local pulled, pull_err = self:request("GET", nil, false, "annotations", not interactive, revision)
    if not pulled then
        if interactive then UIManager:show(InfoMessage:new{text=_("Mochil highlight sync failed."), timeout=4}) end
        return
    end
    pulled_revision = tonumber(pulled.revision) or pulled_revision
    -- Never import over a local edit that has not been acknowledged by the server.
    -- Keep its old base revision so the server can report a real conflict.
    local before, dirty, safe_remotes = self:localAnnotations(), {}, {}
    for id, item in pairs(before) do
        local saved = snapshot[id]
        if type(saved) ~= "table" or not saved.signature or saved.signature ~= item._signature then dirty[id] = true end
    end
    for id, saved in pairs(snapshot) do
        if not before[id] and type(saved) == "table" and saved.signature then dirty[id] = true end
    end
    for _, remote in ipairs(pulled.applied or {}) do
        if not dirty[remote.id] then safe_remotes[#safe_remotes + 1] = remote end
    end
    self:applyRemoteAnnotations(safe_remotes, dirty)
    local current, changes = self:localAnnotations(), {}
    for _, remote in ipairs(safe_remotes) do
        if type(remote.id) == "string" then
            remote_revisions[remote.id] = tonumber(remote.revision) or 0
            local existing = snapshot[remote.id]
            if remote.deleted then
                -- Only forget a tombstone once the local highlight is really gone. Forgetting it while the item is
                -- still here made the item look unsent, and the next upload re-created it on every device.
                if not current[remote.id] then snapshot[remote.id] = nil end
            elseif existing then
                snapshot[remote.id] = {revision=remote_revisions[remote.id], signature=type(existing)=="table" and existing.signature or nil,
                    kind=remote.kind, locators=remote.locators}
            end
        end
    end
    for _, remote in ipairs(pulled.applied or {}) do
        local local_item = type(remote.id)=="string" and current[remote.id] or nil
        if local_item and not remote.deleted and self.annotation_imported_ids[remote.id] then
            snapshot[remote.id] = {revision=tonumber(remote.revision) or 0, signature=local_item._signature}
        end
    end
    -- Keep the cursor behind any remote change that was not durably reflected in
    -- local annotations/snapshot. This lets a later sync retry unresolved locators
    -- and conflicts instead of permanently skipping them.
    local cursor_safe = true
    for _, remote in ipairs(pulled.applied or {}) do
        local id = remote.id
        if type(id) ~= "string" or id == "" then
            cursor_safe = false
        elseif dirty[id] then
            cursor_safe = false
        elseif remote.deleted then
            if current[id] then cursor_safe = false end
        else
            local saved = snapshot[id]
            local saved_revision = type(saved) == "table" and (tonumber(saved.revision) or 0) or 0
            if not current[id] or saved_revision < (tonumber(remote.revision) or 0) then cursor_safe = false end
        end
    end
    for identifier, annotation in pairs(current) do
        local saved = snapshot[identifier]
        local base = remote_revisions[identifier] or (type(saved)=="table" and (tonumber(saved.revision) or 0) or (tonumber(saved) or 0))
        local signature = annotation._signature
        annotation._signature = nil
        if type(saved)~="table" or saved.signature ~= signature then
            changes[#changes + 1] = {base_revision=base, annotation=annotation, _signature=signature}
        elseif saved.missing then
            snapshot[identifier] = {revision=base, signature=signature}
        end
    end
    for identifier, saved in pairs(snapshot) do
        if not current[identifier] then
            local base = type(saved)=="table" and (tonumber(saved.revision) or 0) or (tonumber(saved) or 0)
            local missing = type(saved)=="table" and (tonumber(saved.missing) or 0) + 1 or 1
            if missing >= 2 then
                changes[#changes + 1] = {base_revision=base, annotation={id=identifier,
                    kind=(type(saved)=="table" and saved.kind) or "highlight",
                    locators=(type(saved)=="table" and saved.locators) or nil,
                    deleted=true, updated_at=os.time()*1000, device_id=self.device_id}}
            else
                -- Subtract nothing when the wait is repeated: keep what the entry knows about the annotation itself,
                -- or the reminder would forget it was a bookmark and which page it was on.
                snapshot[identifier] = {revision=base, signature=type(saved)=="table" and saved.signature or nil, missing=missing,
                    kind=type(saved)=="table" and saved.kind or nil, locators=type(saved)=="table" and saved.locators or nil}
                UIManager:unschedule(self.annotation_sync_callback)
                UIManager:scheduleIn(3, self.annotation_sync_callback)
            end
        end
    end
    if #changes > 500 then
        if interactive then UIManager:show(InfoMessage:new{text=_("Too many changed highlights to sync at once."), timeout=5}) end
        return
    end
    if #changes > 0 then
        local signatures, submitted_by_id = {}, {}
        for _, change in ipairs(changes) do
            signatures[change.annotation.id] = change._signature
            submitted_by_id[change.annotation.id] = change.annotation
            change._signature = nil
        end
        local applied, apply_err = self:request("PUT", {changes=changes}, false, "annotations", not interactive)
        if not applied then
            if interactive then UIManager:show(InfoMessage:new{text=_("Mochil highlight sync failed."), timeout=4}) end
            return
        end
        pulled_revision = math.max(pulled_revision, tonumber(applied.revision) or 0)
        for _, saved in ipairs(applied.applied or {}) do
            if saved.deleted then snapshot[saved.id] = nil
            else
                local sent = submitted_by_id[saved.id]
                snapshot[saved.id] = {revision=tonumber(saved.revision) or 0, signature=signatures[saved.id],
                    kind=sent and sent.kind or nil, locators=sent and sent.locators or nil}
            end
        end
        if #(applied.conflicts or {}) > 0 then
            -- A conflict used to have no terminal state: the identical change was re-sent, and the same message shown,
            -- on every sync for ever. Resolve each one once, by newest edit: the later version is re-sent against the
            -- winner's revision, and the older one is taken onto this device, so both sides agree afterwards.
            cursor_safe = false
            local resend, take = {}, {}
            for _, conflict in ipairs(applied.conflicts) do
                local submitted, winner = conflict.submitted or {}, conflict.current or {}
                local id = submitted.id
                if type(id) == "string" then
                    local base = tonumber(winner.revision) or 0
                    if not winner.deleted and (tonumber(submitted.updated_at) or 0) >= (tonumber(winner.updated_at) or 0) then
                        resend[#resend + 1] = {base_revision = base, annotation = submitted}
                        snapshot[id] = {revision = base, signature = signatures[id], kind = submitted.kind, locators = submitted.locators}
                    else
                        take[#take + 1] = winner
                    end
                end
            end
            if #take > 0 then
                self:applyRemoteAnnotations(take, nil)
                local after = self:localAnnotations()
                for _, remote in ipairs(take) do
                    if type(remote.id) == "string" then
                        local settled = after[remote.id]
                        if remote.deleted and not settled then
                            snapshot[remote.id] = nil
                        else
                            snapshot[remote.id] = {revision = tonumber(remote.revision) or 0, signature = settled and settled._signature or nil,
                                kind = remote.kind, locators = remote.locators}
                        end
                    end
                end
            end
            if #resend > 0 then
                local again = self:request("PUT", {changes = resend}, false, "annotations", not interactive)
                if again then
                    pulled_revision = math.max(pulled_revision, tonumber(again.revision) or 0)
                    for _, saved in ipairs(again.applied or {}) do
                        if saved.deleted then snapshot[saved.id] = nil
                        else snapshot[saved.id] = {revision = tonumber(saved.revision) or 0, signature = signatures[saved.id]} end
                    end
                    cursor_safe = #(again.conflicts or {}) == 0
                end
            end
            local text = _("Some highlights were changed elsewhere; the newer version is kept.")
            if interactive then
                UIManager:show(InfoMessage:new{text=text, timeout=6})
            else
                require("ui/widget/notification"):notify(text)
            end
        end
    end
    if cursor_safe then revision = math.max(tonumber(revision) or 0, pulled_revision) end
    self.ui.doc_settings:saveSetting("mochil_annotation_revision", revision)
    self.ui.doc_settings:saveSetting("mochil_annotation_snapshot", snapshot)
    self.ui.doc_settings:saveSetting("mochil_annotation_revision_cursor_version", ANNOTATION_REVISION_CURSOR_VERSION)
    if interactive then UIManager:show(InfoMessage:new{text=_("Highlights synchronized."), timeout=4}) end
end

function Mochil:onAnnotationsModified()
    if self:syncPolicy() == "manual" then return end
    UIManager:unschedule(self.annotation_sync_callback)
    UIManager:scheduleIn(2, self.annotation_sync_callback)
end

function Mochil:textAnchor(progress, percentage)
    local document = self.ui and self.ui.document
    if not document or document.info.has_pages or not document.getXPointer or not document.getTextFromXPointers then return nil end
    local ok, xpointer = pcall(document.getXPointer, document)
    if not ok or not xpointer then return nil end
    local prefix_start = move_words(document, xpointer, 24, true)
    local exact_end = move_words(document, xpointer, 16, false)
    local suffix_end = move_words(document, exact_end, 32, false)
    local ok_prefix, prefix = pcall(document.getTextFromXPointers, document, prefix_start, xpointer)
    local ok_exact, exact = pcall(document.getTextFromXPointers, document, xpointer, exact_end)
    local ok_suffix, suffix = pcall(document.getTextFromXPointers, document, exact_end, suffix_end)
    if not ok_exact then return nil end
    exact = clean_text(exact)
    if #exact < 24 then return nil end
    local fragment = tonumber(xpointer:match("^/body/DocFragment%[(%d+)%]"))
    return {version=2, prefix=ok_prefix and clean_text(prefix) or "", exact=exact, suffix=ok_suffix and clean_text(suffix) or "", spine_index=fragment and (fragment - 1) or 0, progress=percentage or 0}
end

function Mochil:currentPosition()
    if not self.ui or not self.ui.document then return nil, nil end
    local rolling = not self.ui.document.info.has_pages
    local progress = rolling and self.ui.rolling:getLastProgress() or tostring(self.ui.paging:getLastProgress())
    local percentage = rolling and self.ui.rolling:getLastPercent() or self.ui.paging:getLastPercent()
    return progress, percentage
end

-- Saves a full backup of every highlight/note this account has, as the server's own JSON export, to the download
-- folder. Book-independent: unlike syncing, no document needs to be open.
function Mochil:backupHighlights()
    local raw, err = self:fetchRaw("/api/koreader/v1/annotations/export", 8 * 1024 * 1024)
    if not raw then
        UIManager:show(InfoMessage:new{text=_("Could not back up highlights: ") .. Mochil.describeFailure(err), timeout=5})
        return
    end
    local root = Mochil.downloadRoot()
    if not util.pathExists(root) then util.makePath(root) end
    local path = root .. "/mochil-annotations-" .. os.date("%Y%m%d-%H%M%S") .. ".json"
    local file = io.open(path, "wb")
    if not file then
        UIManager:show(InfoMessage:new{text=_("Could not write the backup file."), timeout=5})
        return
    end
    file:write(raw)
    file:close()
    UIManager:show(InfoMessage:new{text=_("Highlights backed up to ") .. path, timeout=5})
end

-- Renders this book's highlights into a page of the server's own Notes/wiki feature (Markdown, viewed on the
-- server, not opened as a file here) via the Bearer-gated digest route, mirroring the browser-facing endpoint.
function Mochil:exportHighlightsToNotes(interactive)
    if not self.ui or not self.ui.document then
        if interactive then UIManager:show(InfoMessage:new{text=_("Open a book to export its highlights."), timeout=4}) end
        return
    end
    local digest = self:digest()
    if not digest then
        if interactive then UIManager:show(InfoMessage:new{text=_("Could not identify this book."), timeout=4}) end
        return
    end
    local sink, payload = {}, "{}"
    socketutil:set_timeout(socketutil.LARGE_BLOCK_TIMEOUT, socketutil.LARGE_TOTAL_TIMEOUT)
    local called, reply, answer = pcall(http.request, {
        url = config.server_url .. "/api/koreader/v1/annotations/" .. digest .. "/notes",
        method = "POST", sink = ltn12.sink.table(sink), source = ltn12.source.string(payload),
        headers = {Authorization = "Bearer " .. config.token, ["Content-Type"] = "application/json", ["Content-Length"] = tostring(#payload)},
    })
    socketutil:reset_timeout()
    local code = tonumber(answer)
    if called and code == 201 then
        if interactive then UIManager:show(InfoMessage:new{text=_("Highlights exported to Notes."), timeout=4}) end
        return
    end
    if interactive then
        local reason
        if code == 404 then reason = _("This book is not linked yet. Sync its highlights first.")
        elseif code == 409 then reason = _("This book has no highlights to export.")
        elseif code == 413 then reason = _("These highlights are too large to export.")
        else reason = Mochil.describeFailure(called and (code and ("http " .. code) or answer) or reply) end
        UIManager:show(InfoMessage:new{text=_("Could not export highlights: ") .. reason, timeout=5})
    end
end

-- Downloads every one of this account's Notes/wiki pages, rendered server-side to self-contained HTML, and
-- opens it directly in KOReader's own reader (crengine, the same engine EPUB/HTML/TXT use) -- unlike the raw
-- Markdown the Notes feature stores, which KOReader has no built-in way to render.
function Mochil:readWikiOnDevice()
    local raw, err = self:fetchRaw("/api/koreader/v1/wiki.html", 8 * 1024 * 1024)
    if not raw then
        UIManager:show(InfoMessage:new{text=_("Could not read your notes: ") .. Mochil.describeFailure(err), timeout=5})
        return
    end
    local root = Mochil.downloadRoot()
    if not util.pathExists(root) then util.makePath(root) end
    local path = root .. "/mochil-notes.html"
    local staged = path .. ".part"
    local file = io.open(staged, "wb")
    if not file then
        UIManager:show(InfoMessage:new{text=_("Could not write the notes file."), timeout=5})
        return
    end
    local wrote = file:write(raw)
    file:close()
    if not wrote or not os.rename(staged, path) then
        os.remove(staged)
        UIManager:show(InfoMessage:new{text=_("Could not write the notes file."), timeout=5})
        return
    end
    self:openDownloaded(path)
end

-- A book marked finished in KOReader (the end-of-book prompt, or the book status) counts as read in Mochil even
-- when the last page was not reached.
function Mochil:markedFinished()
    local summary = self.ui.doc_settings:readSetting("summary")
    return type(summary) == "table" and summary.status == "complete"
end

function Mochil:push(interactive)
    if not self.ui or not self.ui.document then return end
	local progress, percentage = self:currentPosition()
	if not progress then return end
    local finished = self:markedFinished()
    local signature = tostring(progress) .. ":" .. string.format("%.6f", percentage or 0) .. (finished and ":finished" or "")
    if not interactive and signature == self.last_sent_position then return end
    local filename = self.ui.document.file and self.ui.document.file:match("([^/]+)$") or ""
    local props = self.ui.doc_props or {}
	local timestamp = os.time()
    local result, err = self:request("PUT", {progress=progress, percentage=percentage, device=Device.model, device_id=self.device_id, timestamp=timestamp, filename=filename, title=metadata_text(props.display_title or props.title):sub(1, 1024), authors=metadata_text(props.authors), identifiers=metadata_text(props.identifiers), series=metadata_text(props.series):sub(1, 1024), series_index=tonumber(props.series_index), text_anchor=self:textAnchor(progress, percentage), finished=finished or nil}, false, nil, not interactive)
    local unmatched = result and (result.saved == false or result.linked == false)
    if unmatched then result = nil end
    if result then
		self.last_sent_position = signature
		self.ui.doc_settings:saveSetting("mochil_pushed_at", timestamp * 1000)
		self.ui.doc_settings:saveSetting("mochil_seen_at", timestamp * 1000)
	end
    if interactive then UIManager:show(InfoMessage:new{text=result and _("Progress sent to Mochil.") or (unmatched and _("This file could not be matched to the Mochil catalogue.") or (err == "expired" and _("Pairing expired. Install a newly downloaded Mochil plugin.") or _("Mochil sync failed."))), timeout=4}) end
end

function Mochil:apply(position)
	local native = position.koreader_locator
	if type(native) == "string" and native ~= "" then
		if self.ui.document.info.has_pages then
			local page = tonumber(native)
			if page then self.ui:handleEvent(Event:new("GotoPage", page)); return true end
		else
			self.ui:handleEvent(Event:new("GotoXPointer", native)); return true
		end
	end
    local anchor = position.web_anchor
    local anchor_text = anchor and (anchor.exact or anchor.text)
    if anchor_text and self.ui.document.findAllText then
        local chars = util.splitToChars(anchor_text)
        local expected = anchor.spine_index and ("/body/DocFragment[" .. tostring(anchor.spine_index + 1) .. "]") or nil
        local lengths = anchor.version == 2 and {#chars} or {#chars, 64, 48, 36}
        local tried = {}
        for _, length in ipairs(lengths) do
            length = math.min(length, #chars)
            local text = table.concat(chars, "", 1, length):gsub("%s+%S*$", ""):gsub("^%s+", ""):gsub("%s+$", "")
            if #text >= 24 and not tried[text] then
                tried[text] = true
                local ok, hits = pcall(self.ui.document.findAllText, self.ui.document, text, true, 0, 50, false)
                pcall(self.ui.document.clearSelection, self.ui.document)
                if ok and type(hits)=="table" then
                    local best, best_score = nil, -1000
                    for _, hit in ipairs(hits) do
                        if hit.start then
                            local score = 0
                            if expected and hit.start:sub(1, #expected) == expected then score = score + 100 end
                            if anchor.progress and self.ui.document.getPageFromXPointer and self.ui.document.getPageCount then
                                local page_ok, page = pcall(self.ui.document.getPageFromXPointer, self.ui.document, hit.start)
                                local count_ok, count = pcall(self.ui.document.getPageCount, self.ui.document)
                                if page_ok and count_ok and page and count and count > 0 then score = score - math.abs((page / count) - anchor.progress) * 20 end
                            end
                            local prefix_start = move_words(self.ui.document, hit.start, 24, true)
                            local suffix_end = move_words(self.ui.document, hit.start, 48, false)
                            local before_ok, before = pcall(self.ui.document.getTextFromXPointers, self.ui.document, prefix_start, hit.start)
                            local after_ok, after = pcall(self.ui.document.getTextFromXPointers, self.ui.document, hit.start, suffix_end)
                            before, after = clean_text(before_ok and before or ""), clean_text(after_ok and after or "")
                            local wanted_prefix, wanted_suffix = clean_text(anchor.prefix), clean_text(anchor.suffix)
                            if wanted_prefix ~= "" and #before >= #wanted_prefix and before:sub(-#wanted_prefix) == wanted_prefix then score = score + 35 end
                            if wanted_suffix ~= "" and after:find(wanted_suffix, 1, true) then score = score + 35 end
                            if score > best_score then best, best_score = hit.start, score end
                        end
                    end
                    if best then self.ui:handleEvent(Event:new("GotoXPointer", best)); return true end
                end
            end
        end
    end
    if position.fraction then self.ui:handleEvent(Event:new("GotoPercent", position.fraction * 100)); return true end
    return false
end

function Mochil:pull(interactive, linked_retry)
    -- Opening a book must never wait on a slow network: only a button the reader pressed may wait longer.
    local body, err = self:request("GET", nil, false, nil, not interactive)
    self.reachable = err ~= "network" and not (err or ""):find("^http_network")
    if self.reachable == false and not interactive then Mochil.unreachable_until = os.time() + 300 end
    if self.reachable then Mochil.unreachable_until = nil end
    if (not body or not body.position) and err == "http_404" and not linked_retry then
        self:push(false)
        return self:pull(interactive, true)
    end
    if not body or not body.position then if interactive then UIManager:show(InfoMessage:new{text=err == "expired" and _("Pairing expired. Install a newly downloaded Mochil plugin.") or _("This file could not be matched to the Mochil catalogue."),timeout=4}) end; return end
    local position=body.position; local seen=self.ui.doc_settings:readSetting("mochil_seen_at") or 0
    if not interactive and (position.updated_at or 0) <= seen then return end
	local local_progress, local_percentage = self:currentPosition()
	local same_native = type(position.koreader_locator) == "string" and position.koreader_locator ~= "" and position.koreader_locator == tostring(local_progress or "")
	local same_fraction = type(position.fraction) == "number" and type(local_percentage) == "number" and math.abs(position.fraction - local_percentage) <= 0.002
	if same_native or same_fraction then
		self.last_sent_position = tostring(local_progress) .. ":" .. string.format("%.6f", local_percentage or 0)
		self.ui.doc_settings:saveSetting("mochil_seen_at", position.updated_at or 0)
		return
	end
    local go=function() if self:apply(position) then self.ui.doc_settings:saveSetting("mochil_seen_at",position.updated_at or 0) end end
    if interactive then go(); return end
    UIManager:show(ConfirmBox:new{text=_("Mochil has a newer reading position. Continue there?"),ok_callback=go,cancel_callback=function() self.ui.doc_settings:saveSetting("mochil_seen_at",position.updated_at or 0) end})
end

function Mochil:onReaderReady()
    self.last_fragment = self:currentFragment()
    self.reachable = nil
    if os.time() < (Mochil.unreachable_until or 0) then self.reachable = false end
    -- A quick way to push a highlight's own change right after making it, without waiting for the usual
    -- debounce/background sync. Real KOReader hook: ReaderHighlight:addToHighlightDialog.
    if self.ui.highlight and type(self.ui.highlight.addToHighlightDialog) == "function" then
        self.ui.highlight:addToHighlightDialog("mochil_sync_highlight", function()
            return {text = _("Sync to Mochil"), callback = function() self:syncAnnotations(true) end}
        end)
    end
    -- Let the book finish drawing first, never pop a Wi-Fi prompt on open, and do the network work one short
    -- step at a time: each step has its own few seconds, and an unreachable server ends the chain after the first.
    local function step(delay, work)
        UIManager:scheduleIn(delay, function()
            if self.reachable == false or not self:isOnline() then return end
            self.deadline = os.time() + 6
            local ok, err = pcall(work)
            self.deadline = nil
            if not ok then logger.err("Mochil: sync step failed: " .. tostring(err)) end
        end)
    end
    step(1, function() if self:syncPolicy() ~= "manual" then self:pull(false) end end)
    step(8, function() if self:syncPolicy() ~= "manual" then self:syncAnnotations(false) end end)
    step(20, function() self:maybeCheckForUpdate() end)
end
-- Closing a book or going to sleep must not hold the screen: one short time budget covers everything sent.
-- Highlights are sent when a book closes; while sleeping only the position goes, edits are already sent after a pause.
function Mochil:backgroundSync(with_highlights)
    if self:syncPolicy() == "manual" or not self:isOnline() then return end
    self.deadline = os.time() + 6
    local ok, err = pcall(function()
        self:push(false)
        if with_highlights then self:syncAnnotations(false) end
    end)
    self.deadline = nil
    if not ok then logger.err("Mochil: background sync failed: " .. tostring(err)) end
end
function Mochil:onPageUpdate() self:scheduleAutoPush() end
function Mochil:onPosUpdate() self:scheduleAutoPush() end
function Mochil:onCloseDocument()
    UIManager:unschedule(self.auto_push_callback); UIManager:unschedule(self.annotation_sync_callback)
    self:backgroundSync(true)
end
function Mochil:onSuspend() self:backgroundSync(false) end

-- Updating the plugin from inside KOReader. Only mochil_core.lua is replaced (the pairing in
-- mochil_config.lua is untouched), the old core is kept as a fallback, and nothing is written unless the
-- download matches the server's checksum and compiles.
local function parse_version(text)
    local a, b, c = tostring(text or ""):match("^(%d+)%.(%d+)%.(%d+)$")
    if not a then return nil end
    return {tonumber(a), tonumber(b), tonumber(c)}
end

local function is_newer(candidate, current)
    local new, old = parse_version(candidate), parse_version(current)
    if not new or not old then return false end
    for index = 1, 3 do
        if new[index] ~= old[index] then return new[index] > old[index] end
    end
    return false
end
Mochil.isNewerVersion = is_newer

-- Executable code must never be fetched over plaintext HTTP, even on a LAN.
function Mochil.updatesAllowed(url)
    return tostring(url or ""):match("^https://") ~= nil
end

function Mochil:fetchRaw(path, maximum, quick)
    local sink, received, too_large = {}, 0, false
    maximum = math.max(0, tonumber(maximum) or 0)
    local function bounded_sink(chunk)
        if chunk == nil then return 1 end
        if too_large or #chunk > maximum - received then
            too_large = true
            return nil, "too_large"
        end
        received = received + #chunk
        sink[#sink + 1] = chunk
        return 1
    end
    local req = {url=config.server_url .. path, method="GET", redirect=false, sink=bounded_sink, headers={Authorization="Bearer " .. config.token}}
    local block, total = 10, 30
    if quick then
        block, total = 4, 8
        if self.deadline then
            local remaining = self.deadline - os.time()
            if remaining <= 0 then return nil, "budget" end
            block, total = math.min(block, remaining), math.min(total, remaining)
        end
    end
    socketutil:set_timeout(block, total)
    local called, _reply, code = pcall(http.request, req)
    socketutil:reset_timeout()
    if not called then return nil, "network" end
    if too_large then return nil, "too_large" end
    code = tonumber(code)
    if code ~= 200 then return nil, code == 401 and "expired" or ("http_" .. tostring(code or "network")) end
    local raw = table.concat(sink)
    if #raw > maximum then return nil, "too_large" end
    return raw
end

function Mochil:installUpdate(manifest)
    local dir, name = loader.dir, loader.core_file
    if not dir or not name then return false, "unsupported" end
    local raw, err = self:fetchRaw("/api/koreader/v1/plugin/core.lua", 512 * 1024)
    if not raw then return false, err end
    if #raw ~= manifest.size or sha2.sha256(raw) ~= manifest.sha256 then return false, "checksum" end
    local chunk = load(raw, "mochil_core.lua")
    if not chunk then return false, "invalid" end
    local ran, candidate = pcall(chunk, loader)
    if not ran or type(candidate) ~= "table" or type(candidate.init) ~= "function" then return false, "invalid" end
    local target, backup, staged = dir .. name, dir .. (loader.backup_file or (name .. ".bak")), dir .. name .. ".new"
    local file = io.open(staged, "wb"); if not file then return false, "write" end
    local wrote = file:write(raw); file:close()
    local check = io.open(staged, "rb"); local back = check and check:read("*a"); if check then check:close() end
    if not wrote or back ~= raw then os.remove(staged); return false, "write" end
    os.remove(backup)
    local kept = os.rename(target, backup)
    if not os.rename(staged, target) then
        if kept then os.rename(backup, target) end
        os.remove(staged)
        return false, "write"
    end
    return true
end

function Mochil:runUpdate(manifest)
    local ok, err = self:installUpdate(manifest)
    if not ok then
        logger.warn("Mochil: plugin update failed: " .. tostring(err))
        UIManager:show(InfoMessage:new{text=_("The update could not be installed. The current version is unchanged."), timeout=6})
        return
    end
    G_reader_settings:saveSetting("mochil_update_available", false)
    G_reader_settings:saveSetting("mochil_installed_version", manifest.version)
    self:offerRestart(_("Mochil Sync was updated to version ") .. tostring(manifest.version) .. _(". Restart KOReader to use it."))
end

-- The new code is on disk but this session still runs the old one, so the update
-- only takes effect after a restart. KOReader's own helper is used when the
-- version has it; otherwise a confirmation of our own drives the same restart,
-- and only a version without any restart path falls back to a message.
function Mochil:offerRestart(message)
    -- KOReader restarts itself with UIManager:restartKOReader(), which exits with
    -- the code its launcher turns into a restart. Drive that from our own
    -- confirmation rather than UIManager:askForRestart: that one goes through the
    -- device's Restart event handler and silently does nothing when the handler is
    -- absent, which is how an installed update could land with no prompt at all.
    local restart = UIManager.restartKOReader
    if type(restart) == "function" then
        UIManager:show(ConfirmBox:new{
            text = message,
            ok_text = _("Restart now"),
            cancel_text = _("Later"),
            ok_callback = function() restart(UIManager) end,
        })
        return
    end
    if type(UIManager.askForRestart) == "function" then
        UIManager:askForRestart(message)
        return
    end
    UIManager:show(InfoMessage:new{text=message, timeout=8})
end

function Mochil:checkForUpdate(interactive)
    local function say(text, timeout) if interactive then UIManager:show(InfoMessage:new{text=text, timeout=timeout or 5}) end end
    if not Mochil.updatesAllowed(config.server_url) then say(_("Updates need an https:// server address.")); return end
    local raw, err = self:fetchRaw("/api/koreader/v1/plugin/manifest", 4096, not interactive)
    if not raw then
        say(err == "expired" and _("Pairing expired. Install a newly downloaded Mochil plugin.") or _("Could not check for a plugin update."))
        return
    end
    local ok, manifest = pcall(rapidjson.decode, raw)
    if not ok or type(manifest) ~= "table" or type(manifest.version) ~= "string" or type(manifest.sha256) ~= "string" or type(manifest.size) ~= "number" then
        say(_("Could not check for a plugin update.")); return
    end
    G_reader_settings:saveSetting("mochil_update_checked_at", os.time())
    if not is_newer(manifest.version, Mochil.release) then
        G_reader_settings:saveSetting("mochil_update_available", false)
        say(_("Mochil Sync is up to date: version ") .. Mochil.release)
        return
    end
    if (tonumber(manifest.loader_api) or 1) > (loader.api or 1) then
        say(_("This update needs a fresh install. Download the plugin again from Mochil Settings."), 8); return
    end
    G_reader_settings:saveSetting("mochil_update_available", manifest.version)
    if interactive then
        UIManager:show(ConfirmBox:new{text=_("Update Mochil Sync to version ") .. manifest.version .. "?", ok_text=_("Update"), ok_callback=function() self:runUpdate(manifest) end})
    elseif G_reader_settings:readSetting("mochil_update_announced") ~= manifest.version then
        G_reader_settings:saveSetting("mochil_update_announced", manifest.version)
        UIManager:show(InfoMessage:new{text=_("A Mochil Sync update is available. Open Tools, then Mochil Sync, to install it."), timeout=5})
    end
end

-- At most once a day, when online. The old "check automatically" switch is gone; a stored value is ignored.
function Mochil:maybeCheckForUpdate()
    local checked = tonumber(G_reader_settings:readSetting("mochil_update_checked_at")) or 0
    if os.time() - checked < 86400 or not self:isOnline() then return end
    -- Count this as an attempt that failed until it succeeds: checkForUpdate() only writes the real time after a
    -- good manifest, so without this a failing server (or a non-https address) would be asked on every book opened.
    G_reader_settings:saveSetting("mochil_update_checked_at", os.time() - 64800)
    self.deadline = os.time() + 6
    local ok, err = pcall(self.checkForUpdate, self, false)
    self.deadline = nil
    if not ok then logger.err("Mochil: update check failed: " .. tostring(err)) end
end

-- The icon in KOReader's top menu bar, next to search. KOReader only looks in its user icons folder if the
-- folder existed when it started, so the very first run falls back to a built-in icon and the Mochil mark
-- appears after the next restart.
local ICON_NAME, FALLBACK_ICON = "mochil", "book.opened"
local ICON_SVG = [[<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M12 6c-1.7-1.2-4-1.8-7-1.8v12.6c3 0 5.3.6 7 1.8 1.7-1.2 4-1.8 7-1.8V4.2c-3 0-5.3.6-7 1.8z" fill="none" stroke="#000" stroke-width="1.8" stroke-linejoin="round"/><path d="M12 6v12.6" fill="none" stroke="#000" stroke-width="1.8"/></svg>
]]

function Mochil.prepareIcon()
    local ok, name = pcall(function()
        local DataStorage = require("datastorage")
        local lfs = require("libs/libkoreader-lfs")
        local dir = DataStorage:getDataDir() .. "/icons"
        local existed = lfs.attributes(dir, "mode") == "directory"
        if not existed then lfs.mkdir(dir) end
        local path = dir .. "/" .. ICON_NAME .. ".svg"
        local current = io.open(path, "rb")
        local text = current and current:read("*a")
        if current then current:close() end
        if text ~= ICON_SVG then
            local file = io.open(path, "wb")
            if not file then return FALLBACK_ICON end
            file:write(ICON_SVG); file:close()
        end
        return existed and ICON_NAME or FALLBACK_ICON
    end)
    return ok and name or FALLBACK_ICON
end

-- Adds the Mochil tab to a KOReader menu after KOReader has built it, right after the search tab. This does not
-- depend on KOReader's menu order tables: it only needs the finished list of tabs.
function Mochil.insertTab(menu)
    local plugin = Mochil.instance
    if not plugin or type(menu.tab_item_table) ~= "table" then return end
    for _index, tab in ipairs(menu.tab_item_table) do if tab.mochil_tab then return end end
    -- Tapping the tab closes KOReader's menu and opens the Mochil page, like the "reader" tab opens the last book.
    local tab = {icon = Mochil.prepareIcon(), mochil_tab = true, remember = false, callback = function()
        if menu.onCloseFileManagerMenu then menu:onCloseFileManagerMenu() elseif menu.onCloseReaderMenu then menu:onCloseReaderMenu() end
        Mochil.instance:showHome()
    end}
    local at = #menu.tab_item_table
    for index, existing in ipairs(menu.tab_item_table) do if existing.icon == "appbar.search" then at = index end end
    table.insert(menu.tab_item_table, at + 1, tab)
end

-- Wraps the file manager's and the reader's menu builders once, so the tab follows each rebuild of the menu.
function Mochil.hookMenus()
    if Mochil.hooked then return end
    Mochil.hooked = true
    for _index, module in ipairs{"apps/filemanager/filemanagermenu", "apps/reader/modules/readermenu"} do
        pcall(function()
            local Menu = require(module)
            local original = Menu.setUpdateItemTable
            if type(original) ~= "function" then return end
            Menu.setUpdateItemTable = function(self, ...)
                local results = pack(original(self, ...))
                local ok, err = pcall(Mochil.insertTab, self)
                if not ok then logger.err("Mochil: top-bar tab failed: " .. tostring(err)) end
                return unpack(results, 1, results.n)
            end
        end)
    end
end

function Mochil:menuEntries()
    local has_book = function() return self.ui and self.ui.document ~= nil end
    return {
        {text=_("Test connection"),callback=function() self:testConnection() end},
        {text=_("Server: ") .. config.server_url, enabled_func=function() return false end},
        {text=_("Automatic push"),sub_item_table={
            {text=_("When closing"),radio=true,checked_func=function() return self:syncPolicy()=="close" end,callback=function() self:setSyncPolicy("close") end},
            {text=_("After changing chapter"),radio=true,checked_func=function() return self:syncPolicy()=="chapter" end,callback=function() self:setSyncPolicy("chapter") end},
            {text=_("After every page"),radio=true,checked_func=function() return self:syncPolicy()=="page" end,callback=function() self:setSyncPolicy("page") end},
            {text=_("Manual only"),radio=true,checked_func=function() return self:syncPolicy()=="manual" end,callback=function() self:setSyncPolicy("manual") end},
        }},
        {text=_("Designed home page (experimental)"), checked_func=function() return G_reader_settings:readSetting("mochil_designed_page") == true end,
            callback=function()
                local on = G_reader_settings:readSetting("mochil_designed_page") == true
                G_reader_settings:saveSetting("mochil_designed_page", not on)
                if not on then G_reader_settings:saveSetting("mochil_designed_page_crashed", false); self.home_safe_mode = false end
                if G_reader_settings.flush then G_reader_settings:flush() end
            end},
        {text_func=function() return _("Download folder: ") .. Mochil.downloadRoot() end, callback=function() self:chooseDownloadFolder() end},
        {text=_("Download layout"),sub_item_table={
            {text=_("Series folders, then authors"),radio=true,checked_func=function() return Mochil.downloadLayout()=="series" end,callback=function() G_reader_settings:saveSetting("mochil_download_layout","series") end},
            {text=_("Author folders only"),radio=true,checked_func=function() return Mochil.downloadLayout()=="author" end,callback=function() G_reader_settings:saveSetting("mochil_download_layout","author") end},
            {text=_("One folder"),radio=true,checked_func=function() return Mochil.downloadLayout()=="flat" end,callback=function() G_reader_settings:saveSetting("mochil_download_layout","flat") end},
        }},
        {text=_("Search all libraries"),callback=function() self:searchEverything() end},
        {text=_("Pull reading position"),enabled_func=has_book,callback=function() self:pull(true) end},
        {text=_("Push reading position"),enabled_func=has_book,callback=function() self:push(true) end},
        {text=_("Sync highlights"),enabled_func=has_book,callback=function() self:syncAnnotations(true) end},
        {text=_("Export this book's highlights to Notes"),enabled_func=has_book,callback=function() self:exportHighlightsToNotes(true) end},
        {text=_("Back up all my highlights"),callback=function() self:backupHighlights() end},
        {text=_("Notes & wiki"),sub_item_table=(function()
            local items = {
                -- Off by default: a device that never opens notes should never see the action at all, not
                -- just a greyed-out one. Does not gate whether the SERVER has notes/wiki at all (m.wiki
                -- nil -> 501, handled by readWikiOnDevice itself); this is a per-device opt-in on top.
                {text=_("Enable reading notes & wiki on this device"),checked_func=function() return Mochil.wikiEnabled() end,callback=function()
                    G_reader_settings:saveSetting("mochil_wiki_enabled", not Mochil.wikiEnabled())
                end},
            }
            if Mochil.wikiEnabled() then
                items[#items + 1] = {text=_("Read my notes & wiki on this device"),callback=function() self:readWikiOnDevice() end}
                items[#items + 1] = {text=_("Read a vault library on this device"),callback=function() self:readVaultOnDevice() end}
            end
            return items
        end)()},
        {text=_("Reading statistics"),callback=function() self:readingStats() end},
        {text_func=function()
            local installed = G_reader_settings:readSetting("mochil_installed_version")
            if installed and Mochil.isNewerVersion(installed, Mochil.release) then
                return _("Plugin version ") .. Mochil.release .. _(" (update to ") .. installed .. _(" not running)")
            end
            return _("Plugin version ") .. Mochil.release
        end, callback=function() self:showInstallDetails() end, separator=true},
        {text_func=function()
            local available = G_reader_settings:readSetting("mochil_update_available")
            return available and (_("Update plugin to ") .. available) or _("Check for plugin update")
        end, callback=function() self:checkForUpdate(true) end},
    }
end

-- ===== the home page: a full-screen, text-first list (the page behind the Mochil tab) =====
local function url_encode(text)
    return (tostring(text):gsub("[^%w%-_%.~]", function(char) return string.format("%%%02X", char:byte()) end))
end
local function percent(fraction)
    fraction = tonumber(fraction) or 0
    -- A book that is open is not 0%: the bar is visibly not empty, and "0%" next to it reads as broken.
    if fraction > 0 and fraction < 0.005 then return "<1%" end
    return string.format("%d%%", math.floor(fraction * 100 + 0.5))
end
local function plural(count, one, many) return tostring(count) .. " " .. (count == 1 and one or many) end

local function continueKey(entry, authors)
    local book = entry.book
    if type(book) == "table" and book.library and book.id then
        return "book:" .. #tostring(book.library) .. ":" .. tostring(book.library) .. tostring(book.id)
    end
    return "unlinked:" .. tostring(entry.title) .. "|" .. authors
end

local function continueTitle(entry)
    local book = entry.book
    local title = tostring(entry.title or ""):match("^%s*(.-)%s*$")
    if title == "" then title = _("Untitled") end
    local series = type(book) == "table" and tostring(book.series or ""):match("^%s*(.-)%s*$") or ""
    if series ~= "" and series:lower() ~= title:lower() and title:lower():match("^chapter%s+") then
        title = series .. " — " .. title
    end
    return title .. (type(book) == "table" and book.read_along == true and ("  ·  " .. _("Read-along")) or "")
end

-- The rows of the page for what the server said (or why it could not be asked).
function Mochil:homeRows(home, failure)
    local rows = {}
    local function add(row) rows[#rows + 1] = row end
    add({text=_("Refresh"), mandatory=failure and _("Offline") or (_("Updated ") .. os.date("%H:%M")), callback=function() self:showHome(true) end})
    if failure then
        add({text=_("Could not reach Mochil"), mandatory="", callback=function() self:testConnection() end})
        add({text=Mochil.describeFailure(failure), mandatory=""})
    else
        add({text=_("Continue reading"), bold=true, mandatory=""})
        local list = type(home.continue) == "table" and home.continue or {}
        if #list == 0 then add({text=_("Nothing in progress yet"), mandatory="", dim=true}) end
        -- Collapse repeated identities, not distinct editions with the same title.
        local seen = {}
        for _index, book in ipairs(list) do
            local authors = type(book.authors) == "table" and table.concat(book.authors, ", ") or ""
            local key = continueKey(book, authors)
            if not seen[key] then
                seen[key] = true
                local detail = type(book.book) == "table" and book.book or nil
                add({text=continueTitle(book) .. (authors ~= "" and ("  ·  " .. authors) or ""), mandatory=percent(book.fraction),
                    callback=detail and function() self:openBook(detail) end or nil})
            end
        end
        add({text=_("Your library"), bold=true, mandatory=""})
        add({text=_("Search books"), mandatory=">", callback=function() self:browseKind("books", _("Books")) end})
        add({text=tonumber(home.libraries) and _("Libraries") or _("Books"), mandatory=tostring(tonumber(home.libraries) or tonumber(home.books) or 0), callback=function() self:openLibraries(home) end})
        add({text=_("Series"), mandatory=tostring(tonumber(home.series) or 0), callback=function() self:browseKind("series", _("Series")) end})
        add({text=_("Authors"), mandatory=tostring(tonumber(home.authors) or 0), callback=function() self:browseKind("authors", _("Authors")) end})
        add({text=_("Collections"), mandatory=tostring(tonumber(home.collections) or 0), callback=function() self:browseKind("collections", _("Collections")) end})
    end
    add({text=_("Settings"), bold=true, mandatory=">", callback=function() self:showSettingsPage() end})
    add({text=_("Plugin version"), mandatory=Mochil.release, callback=function() self:showInstallDetails() end})
    return rows
end

-- Load the complete Continue reading list in bounded server pages.
function Mochil:showContinue(home, library, library_name)
    local Menu = require("ui/widget/menu")
    local Screen = require("device").screen
    local state = {entries = {}, offset = 0, total = 0}
    local widget
    local fetch, render

    local function entryRows(entries)
        local rows, result, inside = self:homeRows({continue = entries}), {}, false
        for _index, row in ipairs(rows) do
            if row.text == _("Continue reading") then inside = true
            elseif row.text == _("Your library") then break
            elseif inside then result[#result + 1] = row end
        end
        return result
    end

    render = function()
        local rows = {}
        if state.failure then
            rows[#rows + 1] = {text = _("Could not load the full list. Showing available entries."), mandatory = ""}
            rows[#rows + 1] = {text = state.failure, mandatory = ""}
            rows[#rows + 1] = {text = _("Try again"), callback = function() fetch(state.offset) end}
        else
            rows[#rows + 1] = {text = _("Loaded ") .. tostring(state.offset) .. " / " .. tostring(state.total), mandatory = ""}
        end
        if state.failure and #state.entries == 0 then
            rows[#rows + 1] = {text = _("No cached home entries"), mandatory = "", dim = true}
        else
            for _, row in ipairs(entryRows(state.entries)) do rows[#rows + 1] = row end
        end
        if not state.failure and state.offset < state.total then
            rows[#rows + 1] = {text = _("Show more"), mandatory = tostring(state.total - state.offset), callback = function() fetch(state.offset) end}
        end
        if widget then widget:switchItemTable(_("Continue reading"), rows) end
        return rows
    end

    fetch = function(offset)
        local resource = "continue?limit=30&offset=" .. offset
        if library then resource = resource .. "&library=" .. url_encode(library) end
        local called, result, err = Mochil.whileLoading(_("Loading…"), function()
            self.deadline = os.time() + 10
            local worked, answer, why = pcall(self.request, self, "GET", nil, true, resource, true)
            self.deadline = nil
            return worked, answer, why
        end)
        local total = result and tonumber(result.total)
        if not called or type(result) ~= "table" or type(result.continue) ~= "table" or not total or total < 0
            or (offset < total and #result.continue == 0) then
            if offset == 0 then state.entries = (not library and type(home) == "table" and type(home.continue) == "table") and home.continue or {} end
            local reason = called and err or nil
            if reason == "http_404" or reason == "http_405" then state.failure = _("This server may not support the full list.")
            elseif reason == "expired" then state.failure = _("Authentication expired.")
            else state.failure = _("Request failed or returned an invalid response.") end
        else
            if offset == 0 then state.entries = {} end
            for _, entry in ipairs(result.continue) do state.entries[#state.entries + 1] = entry end
            state.offset = offset + #result.continue
            state.total, state.failure = total, nil
        end
        render()
    end

    fetch(0)
    local title = library_name and (_("Continue reading") .. " · " .. library_name) or _("Continue reading")
    widget = Menu:new{title = title, item_table = render(), is_borderless = true,
        is_popout = false, covers_fullscreen = true, single_line = false,
        width = Screen:getWidth(), height = Screen:getHeight(),
        close_callback = function() UIManager:close(widget) end}
    Mochil.keepOpen(widget, Menu)
    self:track(widget)
    UIManager:show(widget)
end

function Mochil:subtitleFor(home, failure)
    local server = tostring(config.server_url):gsub("^https?://", "")
    if failure or not home then return server end
    return tostring(home.account or "") .. "  ·  " .. server
end

-- The designed page: a header, three actions, a "Go to" block and Continue reading cards. It is built from KOReader's
-- own widgets; if anything in building it fails, the plain list is shown instead, so the tab never breaks.
function Mochil:buildHomeWidget(home, failure)
    local InputContainer = require("ui/widget/container/inputcontainer")
    local FrameContainer = require("ui/widget/container/framecontainer")
    local VerticalGroup = require("ui/widget/verticalgroup")
    local HorizontalGroup = require("ui/widget/horizontalgroup")
    local VerticalSpan = require("ui/widget/verticalspan")
    local HorizontalSpan = require("ui/widget/horizontalspan")
    local LineWidget = require("ui/widget/linewidget")
    local TextWidget = require("ui/widget/textwidget")
    local TitleBar = require("ui/widget/titlebar")
    local ButtonTable = require("ui/widget/buttontable")
    local ProgressWidget = require("ui/widget/progresswidget")
    local GestureRange = require("ui/gesturerange")
    local Blitbuffer = require("ffi/blitbuffer")
    local Geom = require("ui/geometry")
    local Font = require("ui/font")
    local Screen = require("device").screen
    local width, height = Screen:getWidth(), Screen:getHeight()
    -- A single spacing scale keeps the rhythm consistent: tight within a section, a little more between
    -- related rows, a full margin between the page edge and its content and between unrelated sections.
    local GAP_XS = Screen:scaleBySize(4)
    local GAP_SM = Screen:scaleBySize(8)
    local margin = Screen:scaleBySize(16)
    local GAP_MD = margin
    local inner = width - 2 * margin
    -- One radius and one padding for every card, so corners and spacing read as one design, not several.
    -- The radius stays tight/minimal (not a rounded pill); the padding is generous so text never feels cramped.
    local CARD_RADIUS = Screen:scaleBySize(8)
    local CARD_PADDING = Screen:scaleBySize(12)
    local CARD_INSET = CARD_PADDING * 2 + Screen:scaleBySize(8) -- padding on both sides plus the border and a little breathing room
    local widget = InputContainer:new{covers_fullscreen = true, dimen = Geom:new{x = 0, y = 0, w = width, h = height}}
    local function close() UIManager:close(widget) end
    local function section(label)
        return TextWidget:new{text = label, face = Font:getFace("cfont", 20), bold = true, max_width = inner}
    end
    -- A thin rule under a heading separates it from its content without the cost of another blank gap alone.
    local function divider()
        return LineWidget:new{dimen = Geom:new{w = inner, h = Screen:scaleBySize(1)}, background = Blitbuffer.COLOR_GRAY, is_divider = true}
    end
    local function tiles(rows) -- a bordered grid of buttons
        return ButtonTable:new{width = inner, buttons = rows, zero_sep = true, show_parent = widget}
    end
    local function count(value) return tostring(tonumber(value) or 0) end

    local body = VerticalGroup:new{align = "left"}
    local function add(item) body[#body + 1] = item end
    -- A section heading, its rule, and the breathing room on each side, used the same way everywhere it appears.
    local function addHeading(label)
        add(section(label))
        add(VerticalSpan:new{width = GAP_XS})
        add(divider())
        add(VerticalSpan:new{width = GAP_XS})
    end
    local titlebar = TitleBar:new{width = width, fullscreen = true, title = _("Mochil"), subtitle = self:subtitleFor(home, failure),
        with_bottom_line = true, close_callback = close}
    local function iconAction(icon, label, callback)
        return {icon = icon, icon_width = Screen:scaleBySize(24), icon_height = Screen:scaleBySize(24),
            height = Screen:scaleBySize(36), callback = callback,
            hold_callback = function() UIManager:show(InfoMessage:new{text = label, timeout = 3}) end}
    end
    add(VerticalSpan:new{width = GAP_SM})
    add(tiles{{
        iconAction("appbar.search", _("Search books"), function() self:browseKind("books", _("Books")) end),
        iconAction("cre.render.reload", _("Refresh"), function() self:showHome(true) end),
        iconAction("appbar.settings", _("Settings"), function() self:showSettingsPage() end),
        iconAction("appbar.filebrowser", _("Files"), function() self:showFiles() end),
    }})
    add(VerticalSpan:new{width = GAP_SM})
    if failure then
        addHeading(_("Could not reach Mochil"))
        add(TextWidget:new{text = Mochil.describeFailure(failure), face = Font:getFace("cfont", 18), max_width = inner})
        add(VerticalSpan:new{width = margin})
        add(tiles{{{text = _("Try again"), callback = function() self:showHome(true) end}}})
    else
        addHeading(_("Go to"))
        add(tiles{
            {{text = (tonumber(home.libraries) and _("Libraries") or _("Library")) .. "   " .. count(tonumber(home.libraries) or home.books), callback = function() self:openLibraries(home) end},
             {text = _("Authors") .. "   " .. count(home.authors), callback = function() self:browseKind("authors", _("Authors")) end}},
            {{text = _("Series") .. "   " .. count(home.series), callback = function() self:browseKind("series", _("Series")) end},
             {text = _("Collections") .. "   " .. count(home.collections), callback = function() self:browseKind("collections", _("Collections")) end}},
        })
        add(VerticalSpan:new{width = GAP_SM})
        add(tiles{{
            {text = _("Continue reading"), align = "left", callback = function() self:showContinue(home) end},
            {text = _("Show all"), callback = function() self:showContinue(home) end},
        }})
        add(VerticalSpan:new{width = GAP_SM})
        -- How many cards fit under what is already there; the same book on the shelf twice is shown once.
        local used = 0
        pcall(function() used = titlebar:getSize().h + body:getSize().h end)
        -- Measuring makes the group remember where each item goes; items added afterwards need that forgotten.
        if body.resetLayout then body:resetLayout() end
        if used >= height then error("Home controls do not fit the screen") end
        -- A card is a title, a dimmer author line, and one row holding both the bar and its percentage.
        local card_height = Screen:scaleBySize(92)
        local room = math.max(0, math.floor((height - used - GAP_MD) / (card_height + GAP_SM)))
        local list, seen, shown = type(home.continue) == "table" and home.continue or {}, {}, 0
        if #list == 0 then add(TextWidget:new{text = _("Nothing in progress yet"), face = Font:getFace("cfont", 18), max_width = inner}) end
        for _index, book in ipairs(list) do
            local authors = type(book.authors) == "table" and table.concat(book.authors, ", ") or ""
            local key = continueKey(book, authors)
            if not seen[key] and shown < room then
                seen[key] = true; shown = shown + 1
                local fraction = math.max(0, math.min(1, tonumber(book.fraction) or 0))
                local text_width = inner - CARD_INSET
                local percent_width = Screen:scaleBySize(40)
                -- The title and its percentage share one row: the title is built first so its own rendered
                -- width can be measured, and a spacer is sized to exactly fill the remainder. That pins the
                -- percentage flush to a fixed right edge no matter how short or long the title is, instead of
                -- the percentage drifting left whenever a title doesn't fill its slot.
                local title_widget = TextWidget:new{text = continueTitle(book), face = Font:getFace("cfont", 20), bold = true,
                    max_width = text_width - percent_width - GAP_SM, truncate_with_ellipsis = true, truncate_left = false}
                local title_width = 0
                if type(title_widget.getSize) == "function" then
                    local measured, size = pcall(title_widget.getSize, title_widget)
                    if measured and size and size.w then title_width = size.w end
                end
                local spacer_width = math.max(0, text_width - percent_width - GAP_SM - title_width)
                -- An explicit width and height make every card identical no matter what it holds: KOReader's
                -- TextWidget never wraps to a second line, but it also never *clips* an overlong title or
                -- author on its own, so a card would quietly draw wider/taller than its siblings on real
                -- devices without both max_width (clamps the drawn width) and a forced frame height here.
                local card = FrameContainer:new{
                    bordersize = Screen:scaleBySize(1), radius = CARD_RADIUS, padding = CARD_PADDING, margin = 0,
                    background = Blitbuffer.COLOR_WHITE, width = inner, height = card_height,
                    VerticalGroup:new{align = "left",
                        HorizontalGroup:new{is_title_row = true,
                            title_widget,
                            HorizontalSpan:new{width = spacer_width},
                            -- Solid black, fixed-width and never truncated: a short label like "42%" must
                            -- never wrap or get cut, and it carries no less emphasis than plain body text.
                            TextWidget:new{text = percent(fraction), face = Font:getFace("cfont", 14),
                                max_width = percent_width, truncate_with_ellipsis = false},
                        },
                        -- Solid black like every other card label: hierarchy comes from size/weight alone, not
                        -- from dithering the author line gray, which reads as muddy/faint on an e-ink panel.
                        TextWidget:new{text = authors, face = Font:getFace("cfont", 16),
                            max_width = text_width, truncate_with_ellipsis = true, truncate_left = false},
                        VerticalSpan:new{width = GAP_XS},
                        -- A thin, borderless bar on a light track reads as a minimal progress indicator rather
                        -- than the thicker bordered widget style used elsewhere in KOReader's UI.
                        ProgressWidget:new{width = text_width, height = Screen:scaleBySize(3), percentage = fraction,
                            ticks = nil, last = nil, bordersize = 0,
                            fillcolor = Blitbuffer.COLOR_BLACK, bgcolor = Blitbuffer.COLOR_LIGHT_GRAY},
                    },
                }
                -- A card for a book the server named can be tapped: it opens the book, or offers to download it.
                if type(book.book) == "table" then
                    local detail = book.book
                    local tappable = InputContainer:new{}
                    tappable.ges_events = {TapCard = {GestureRange:new{ges = "tap", range = function() return tappable.dimen end}}}
                    tappable.onTapCard = function() self:openBook(detail); return true end
                    tappable[1] = card
                    card = tappable
                end
                add(card)
                add(VerticalSpan:new{width = GAP_SM})
            end
        end
    end
    -- The title bar spans the screen; everything under it keeps a margin on the left (the widths already leave the right one).
    local indented = FrameContainer:new{bordersize = 0, padding = 0, margin = 0, padding_left = margin, body}
    local page = VerticalGroup:new{align = "left", titlebar, indented}
    widget[1] = FrameContainer:new{background = Blitbuffer.COLOR_WHITE, bordersize = 0, padding = 0, margin = 0, width = width, height = height, page}
    -- Measuring the page makes KOReader lay every widget out now, where a problem is caught, not while it draws.
    if type(widget[1].getSize) == "function" then widget[1]:getSize() end
    -- Draw it once into a spare buffer first: a drawing error then becomes a message here instead of ending KOReader.
    if type(widget.paintTo) == "function" then
        local spare = Blitbuffer.new(width, height, Screen.bb and Screen.bb.getType and Screen.bb:getType() or nil)
        local drawn, why = pcall(widget.paintTo, widget, spare, 0, 0)
        if spare.free then spare:free() end
        if not drawn then error(why, 0) end
    end
    widget.key_events = {Close = {{"Back"}, doc = "close"}}
    widget.onClose = function() close(); return true end
    widget.onCloseWidget = function()
        if self.home_widget == widget then
            self.home_widget = nil
            G_reader_settings:saveSetting("mochil_page_pending", false)
            if G_reader_settings.flush then G_reader_settings:flush() end
        end
    end
    return widget
end

-- KOReader's list closes itself after every row you tap (it calls close_callback). These pages are for working in,
-- so a tap only runs the row's action and refreshes what the rows show; the close button still closes the page.
function Mochil.keepOpen(widget, Menu)
    local original = Menu and Menu.onMenuSelect
    widget.onMenuSelect = function(menu, item)
        if item.sub_item_table == nil then
            if item.select_enabled == false then return true end
            if item.select_enabled_func and not item.select_enabled_func() then return true end
            menu:onMenuChoice(item)
            if menu.updateItems then menu:updateItems() end
            return true
        end
        if original then return original(menu, item) end
        return true
    end
    return widget
end

-- The plugin's menu entries as rows of that list: switches say On or Off, choices mark the current one.
function Mochil.pageItems(entries)
    local items = {}
    for _index, entry in ipairs(entries) do
        local item = {text = entry.text, text_func = entry.text_func, callback = entry.callback, separator = entry.separator,
            select_enabled_func = entry.enabled_func, dim = entry.enabled_func and not entry.enabled_func() or nil}
        if entry.checked_func then
            item.mandatory_func = function()
                local on = entry.checked_func()
                if entry.radio then return on and "✓" or "" end
                return on and _("On") or _("Off")
            end
        end
        if entry.sub_item_table then item.sub_item_table = Mochil.pageItems(entry.sub_item_table) end
        items[#items + 1] = item
    end
    return items
end

-- The plain list the page falls back to.
function Mochil:buildHomeList(home, failure)
    local Menu = require("ui/widget/menu")
    local Screen = require("device").screen
    local widget
    widget = Menu:new{
        title=_("Mochil"), subtitle=self:subtitleFor(home, failure), item_table=self:homeRows(home, failure),
        is_borderless=true, is_popout=false, covers_fullscreen=true, single_line=false,
        width=Screen:getWidth(), height=Screen:getHeight(),
        close_callback=function() UIManager:close(widget); if self.home_widget == widget then self.home_widget = nil end end,
    }
    return Mochil.keepOpen(widget, Menu)
end

-- Opens (or refreshes) the page. It always opens: when the server cannot be reached it says why, with a way to retry.
function Mochil:showHome(refresh)
    if self.home_widget then UIManager:close(self.home_widget); self.home_widget = nil end
    local run = function()
        self.deadline = os.time() + 6
        local called, home, err = pcall(self.request, self, "GET", nil, true, "home", true)
        self.deadline = nil
        local failure
        if not called then failure = home; home = nil
        elseif not home then failure = self.last_error or err or "" end
        if failure == "expired" then failure = _("This pairing has expired. Download and install a new plugin from Mochil Settings.") end
        local widget
        local screen = require("device").screen
        local compact = screen:getHeight() < screen:scaleBySize(600) or screen:getWidth() >= screen:getHeight()
        if not compact and not self.home_safe_mode and G_reader_settings:readSetting("mochil_designed_page") == true then
            local built, designed = pcall(self.buildHomeWidget, self, home, failure)
            if built then
                widget = designed
                G_reader_settings:saveSetting("mochil_designed_page_error", false)
            else
                logger.err("Mochil: home page layout failed, using the list: " .. tostring(designed))
                G_reader_settings:saveSetting("mochil_designed_page_error", tostring(designed):sub(1, 400))
            end
        end
        local pending = widget ~= nil
        if pending then
            G_reader_settings:saveSetting("mochil_page_pending", true)
            if G_reader_settings.flush then G_reader_settings:flush() end
        else
            widget = self:buildHomeList(home, failure)
        end
        self.home_widget = widget
        UIManager:show(widget)
        if pending then
            -- Reaching this timer means the page was drawn and KOReader is still running.
            UIManager:scheduleIn(3, function()
                if self.home_widget ~= widget then return end
                G_reader_settings:saveSetting("mochil_page_pending", false)
                if G_reader_settings.flush then G_reader_settings:flush() end
            end)
        end
    end
    if NetworkMgr:willRerunWhenConnected(run) then return end
    run()
end

-- ===== browsing the library and downloading books =====
local BROWSE_PAGE = 30

-- A name that is safe as a file or folder name on every file system KOReader runs on.
function Mochil.cleanName(text, maximum)
    text = tostring(text or ""):gsub("[%c\\/:*?\"<>|]", "_"):gsub("%s+", " ")
    text = text:gsub("^[%s%.]+", ""):gsub("[%s%.]+$", "")
    if text == "" then text = "Untitled" end
    return bound_utf8(text, maximum or 100)
end

-- Where downloads go: the folder the reader chose, or a Mochil folder inside KOReader's home folder.
function Mochil.downloadRoot()
    local saved = G_reader_settings:readSetting("mochil_download_dir")
    if type(saved) == "string" and saved ~= "" then return (saved:gsub("/+$", "")) end
    local home = G_reader_settings:readSetting("home_dir") or Device.home_dir or "/tmp"
    return (tostring(home):gsub("/+$", "")) .. "/Mochil"
end

function Mochil.downloadLayout()
    local layout = G_reader_settings:readSetting("mochil_download_layout")
    return (layout == "author" or layout == "flat") and layout or "series"
end

-- Series books go in Series/<series>/<NN - title>, the rest in <author>/<title>, so a shelf of files reads like the library.
function Mochil.downloadPath(root, book, layout)
    local format = Mochil.cleanName(book.format ~= nil and book.format ~= "" and book.format or "epub", 8):lower()
    local title = Mochil.cleanName(book.title)
    local series = tostring(book.series or "")
    local author = type(book.authors) == "table" and book.authors[1] or nil
    if layout == "flat" then return root .. "/" .. title .. "." .. format end
    if layout ~= "author" and series ~= "" then
        local volume = tonumber(book.series_index)
        local prefix = ""
        if volume then prefix = (volume == math.floor(volume) and string.format("%02d", volume) or tostring(volume)) .. " - " end
        return root .. "/Series/" .. Mochil.cleanName(series) .. "/" .. Mochil.cleanName(prefix .. title) .. "." .. format
    end
    return root .. "/" .. Mochil.cleanName(author or "Unknown author") .. "/" .. title .. "." .. format
end

-- Legacy/unowned files cannot be attributed safely to a server; leave them untouched.
local function downloadIdentity(book)
    return sha2.md5(tostring(config.server_url or "")) .. "\0" .. tostring(book.library) .. "\0" .. tostring(book.id)
end

function Mochil:downloadTarget(book)
    local root = Mochil.downloadRoot()
    local path = Mochil.downloadPath(root, book, Mochil.downloadLayout())
    local owners = G_reader_settings:readSetting("mochil_download_owners") or {}
    local identity = downloadIdentity(book)
    -- Metadata and layout can change; a verified existing ownership record wins
    -- over deriving a new filename. Legacy records never match this identity.
    local existing
    for owned_path, owner in pairs(owners) do
        if owner == identity and type(owned_path) == "string" and owned_path:sub(1, #root + 1) == root .. "/" and util.pathExists(owned_path)
            and (not existing or owned_path < existing) then existing = owned_path end
    end
    if existing then return existing end
    local stem, ext = path:match("^(.*)(%.[^./]+)$")
    local candidate, index = path, 1
    while (owners[candidate] and owners[candidate] ~= identity) or (util.pathExists(candidate) and owners[candidate] ~= identity) do
        candidate = (stem or path) .. " (" .. index .. ")" .. (ext or "")
        index = index + 1
    end
    return candidate
end

-- Every widget of the browsing pages that is open, so opening a book can clear them away.
function Mochil:closeBrowsing()
    for _index, widget in ipairs(self.browse_widgets or {}) do UIManager:close(widget) end
    self.browse_widgets = {}
    if self.home_widget then UIManager:close(self.home_widget); self.home_widget = nil end
end

-- "Files" means KOReader's file browser. With a book open, closing the page only shows that book again, so the
-- book is closed (its position is sent as usual) and the file browser takes its place.
function Mochil:showFiles()
    self:closeBrowsing()
    local ui = self.ui
    if ui and ui.document and ui.onHome then
        UIManager:nextTick(function() pcall(ui.onHome, ui) end)
    end
end

function Mochil:track(widget)
    self.browse_widgets = self.browse_widgets or {}
    self.browse_widgets[#self.browse_widgets + 1] = widget
    return widget
end

-- A book from Continue reading: open it if it is already on this device, otherwise offer to download it.
function Mochil:openBook(book)
    local path = self:downloadTarget(book)
    if util.pathExists(path) then self:openDownloaded(path) else self:showBook(book) end
end

function Mochil:openDownloaded(path)
    self:closeBrowsing()
    local ReaderUI = require("apps/reader/readerui")
    ReaderUI:showReader(path)
end

-- Fetches one file from the server straight to disk. A half-written file is never left under the real name.
-- onProgress(written, total) and onFailure(message) are for the download queue; the plain Download
-- button passes neither, so its behaviour and its dialogs are unchanged.
function Mochil:downloadBook(book, done, onProgress, onFailure)
    local path = self:downloadTarget(book)
    if util.pathExists(path) then if done then done(path) end return end
    local waiting = InfoMessage:new{text = _("Downloading ") .. tostring(book.title or "") .. "…"}
    UIManager:show(waiting)
    -- Let the message draw before the download holds the screen.
    UIManager:scheduleIn(0.2, function()
        local failure
        local directory = path:match("^(.*)/[^/]*$")
        local partdir = path .. ".part"
        local part = partdir .. "/download"
        if directory and not util.pathExists(directory) then util.makePath(directory) end
        -- mkdir reserves a private staging directory without truncating unknown files.
        local lfs = require("libs/libkoreader-lfs")
        local reserved, reserve_error = lfs.mkdir(partdir)
        if not reserved then
            -- KOReader can die mid-download (power off, killed, card out) and leave this directory behind, and then
            -- every later attempt for that book failed at mkdir with no way out. Clear a leftover that holds
            -- nothing but our own partial file, and try once more; anything else is left strictly alone.
            local only_ours = lfs.attributes(partdir, "mode") == "directory"
            if only_ours then
                for name in lfs.dir(partdir) do
                    if name ~= "." and name ~= ".." and name ~= "download" then only_ours = false; break end
                end
            end
            if only_ours then
                pcall(os.remove, part)
                pcall(lfs.rmdir, partdir)
                reserved, reserve_error = lfs.mkdir(partdir)
            end
        end
        local file, why
        if reserved then file, why = io.open(part, "wb") else why = reserve_error end
        if not file then
            failure = _("Could not write to ") .. tostring(directory) .. (why and (": " .. tostring(why)) or "")
        else
            socketutil:set_timeout(socketutil.FILE_BLOCK_TIMEOUT or 15, socketutil.FILE_TOTAL_TIMEOUT or 300)
            local written = 0
            local total = tonumber(book.size) or 0
            -- Counting sink: the catalogue already knows the size, so the queue can show real progress
            -- without a HEAD request. Reports are throttled to a readable rate rather than per chunk.
            local sink = ltn12.sink.file(file)
            if onProgress then
                local last_report = 0
                sink = function(chunk)
                    if chunk then
                        written = written + #chunk
                        local now = os.time()
                        if now ~= last_report then
                            last_report = now
                            pcall(onProgress, written, total)
                        end
                        return file:write(chunk) and 1 or nil
                    end
                    file:close()
                    return 1
                end
            end
            local called, reply, answer = pcall(http.request, {
                url = config.server_url .. "/api/koreader/v1/download/" .. url_encode(book.library) .. "/" .. url_encode(book.id),
                method = "GET", sink = sink, headers = {Authorization = "Bearer " .. config.token},
            })
            socketutil:reset_timeout()
            -- LuaSocket answers nil plus the reason when it cannot connect, or a result plus the HTTP status.
            local code = tonumber(answer)
            if not called then
                failure = Mochil.describeFailure(reply)
            elseif code ~= 200 then
                if code == 401 then failure = _("This pairing has expired. Download and install a new plugin from Mochil Settings.")
                elseif code == 404 then failure = _("The server no longer has this book.")
                else failure = Mochil.describeFailure(code and ("http " .. code) or answer) end
            elseif (lfs.attributes(part, "size") or 0) == 0 then
                failure = _("The server sent an empty file; the book was not saved.")
            elseif util.pathExists(path) then
                failure = _("Could not save the book because the destination already exists.")
            elseif not os.rename(part, path) then
                failure = _("Could not save the book.")
            else
                local owners = G_reader_settings:readSetting("mochil_download_owners") or {}
                owners[path] = downloadIdentity(book)
                G_reader_settings:saveSetting("mochil_download_owners", owners)
            end
            if failure then pcall(file.close, file); os.remove(part) end
        end
        if reserved then lfs.rmdir(partdir) end
        UIManager:close(waiting)
        if failure then
            -- A queue needs to hear about a failure to move on to the next book; a lone Download
            -- still just says what went wrong.
            if onFailure then onFailure(failure) else UIManager:show(InfoMessage:new{text = failure}) end
        elseif done then
            done(path)
        end
    end)
end

-- One download at a time, in the order the reader queued them. Two reasons: on a slow link parallel
-- transfers only make each one slower to finish, and one failure must not take the other books with
-- it. The queue survives a failed item and keeps going.
function Mochil:queueList()
    self.download_queue = self.download_queue or {}
    return self.download_queue
end

function Mochil:queueBook(book)
    local queue = self:queueList()
    if util.pathExists(self:downloadTarget(book)) then
        UIManager:show(InfoMessage:new{text = _("Already on this device."), timeout = 3})
        return false
    end
    for _, item in ipairs(queue) do
        if item.state ~= "done" and item.book.id == book.id and item.book.library == book.library then
            UIManager:show(InfoMessage:new{text = _("Already in the queue."), timeout = 3})
            return false
        end
    end
    queue[#queue + 1] = {book = book, state = "waiting", written = 0, total = tonumber(book.size) or 0}
    self:showQueue()
    self:runQueue()
    return true
end

-- Bytes, not percentages: with a size the reader can judge whether it is worth waiting, and with no
-- size the reader at least sees that something is arriving.
function Mochil.queueProgressText(written, total)
    local function megabytes(bytes) return string.format("%.1f MB", bytes / 1048576) end
    if total and total > 0 then return megabytes(written) .. " of " .. megabytes(total) end
    return megabytes(written)
end

function Mochil:queueItems()
    local items = {}
    for index, item in ipairs(self:queueList()) do
        local title = tostring(item.book.title or "")
        local text
        if item.state == "waiting" then
            text = title .. "  ·  " .. _("waiting (tap to remove)")
        elseif item.state == "downloading" then
            text = title .. "  ·  " .. Mochil.queueProgressText(item.written, item.total)
        elseif item.state == "done" then
            text = title .. "  ·  " .. _("downloaded (tap to open)")
        else
            text = title .. "  ·  " .. _("failed") .. (item.reason and (": " .. tostring(item.reason)) or "")
        end
        items[#items + 1] = {text = text, callback = function() self:queueTap(index) end}
    end
    if #items == 0 then
        items[1] = {text = _("Nothing queued."), select_enabled_func = function() return false end}
    end
    return items
end

function Mochil:queueTap(index)
    local item = self:queueList()[index]
    if not item then return end
    if item.state == "waiting" then
        table.remove(self:queueList(), index)
        self:refreshQueue()
    elseif item.state == "done" then
        local path = self:downloadTarget(item.book)
        if util.pathExists(path) then self:openDownloaded(path) end
    end
end

function Mochil:showQueue()
    -- Queueing a second book while the queue is already on screen must not stack a second copy.
    if self.queue_open then
        self:refreshQueue()
        return self.queue_widget
    end
    local Menu = require("ui/widget/menu")
    local Screen = require("device").screen
    local widget
    widget = Menu:new{
        title = _("Download queue"), item_table = self:queueItems(), is_borderless = true, is_popout = false,
        covers_fullscreen = true, width = Screen:getWidth(), height = Screen:getHeight(),
        close_callback = function() self.queue_open = false; self.queue_widget = nil; UIManager:close(widget) end,
    }
    self.queue_widget = widget
    self.queue_open = true
    Mochil.keepOpen(widget, Menu)
    UIManager:show(widget)
    return widget
end

-- Repainting the open queue is not worth a dialog rebuild: the list is switched in place so the
-- progress moves while the reader watches it.
function Mochil:refreshQueue()
    -- Only touch the list while its window is still on screen: a progress tick that arrives after the
    -- reader closed it must not repaint a widget that is no longer shown.
    if not self.queue_open then return end
    local widget = self.queue_widget
    if widget and widget.switchItemTable then
        pcall(widget.switchItemTable, widget, _("Download queue"), self:queueItems())
    end
end

function Mochil:runQueue()
    if self.queue_running then return end
    local current
    for _, item in ipairs(self:queueList()) do
        if item.state == "waiting" then current = item break end
    end
    if not current then return end
    self.queue_running = true
    current.state = "downloading"
    self:refreshQueue()
    self:downloadBook(current.book, function()
        current.state = "done"
        current.written = current.total
        self.queue_running = false
        self:refreshQueue()
        self:runQueue()
    end, function(written, total)
        current.written, current.total = written, total
        self:refreshQueue()
    end, function(reason)
        -- One bad book does not stop the queue.
        current.state = "failed"
        current.reason = tostring(reason or "")
        self.queue_running = false
        self:refreshQueue()
        self:runQueue()
    end)
end

-- Marks (or unmarks) a catalog book finished before it is ever downloaded or opened, via the
-- Bearer-gated /api/koreader/v1/finished/{library}/{id} route -- unlike push's digest-keyed progress
-- route, this needs no local file, since status_only skips self:digest() entirely.
function Mochil:setBookFinished(book, finished)
    local result, err = self:request("PUT", {finished = finished}, true, "finished/" .. url_encode(book.library) .. "/" .. url_encode(book.id))
    if not result then
        UIManager:show(InfoMessage:new{text=_("Could not update finished status: ") .. Mochil.describeFailure(err), timeout=5})
        return false
    end
    -- The dialog that offered the toggle was built from this table, so it has to agree with the server now.
    book.finished = finished and true or nil
    UIManager:show(InfoMessage:new{text = finished and _("Marked finished.") or _("Marked unfinished."), timeout=3})
    return true
end


-- The catalogue knows the volume order, so continuing a series is one tap instead of remembering a
-- number. Asked once per book and kept on the book table, because the dialog is rebuilt on every open.
function Mochil:seriesNext(book)
    if book._seriesNext ~= nil then return book._seriesNext or nil end
    book._seriesNext = false
    if tostring(book.series or "") == "" then return nil end
    local result = self:request("GET", nil, true, "series/" .. url_encode(book.library) .. "/" .. url_encode(book.id), true)
    if type(result) == "table" and type(result.next) == "table" and tostring(result.next.id or "") ~= "" then
        book._seriesNext = result.next
    end
    return book._seriesNext or nil
end

-- The dialog shows the cover the browser shows: one file per book, fetched once and reused. Cover
-- fetching is best-effort by design - a server without artwork, an expired pairing or a KOReader
-- too old for the widgets all fall back to the plain text dialog rather than a dead end.
function Mochil:coverTarget(book)
    return Mochil.downloadRoot() .. "/covers/" .. url_encode(book.library) .. "_" .. url_encode(book.id) .. ".cover"
end

function Mochil.readFilePrefix(path, count)
    local file = io.open(path, "rb")
    if not file then return nil end
    local prefix = file:read(count)
    file:close()
    return prefix
end

function Mochil.fileSize(path)
    local file = io.open(path, "rb")
    if not file then return 0 end
    local size = file:seek("end") or 0
    file:close()
    return size
end

-- A 200 carrying an HTML error page would otherwise be cached and "rendered" as a broken cover.
function Mochil.looksLikeImage(path)
    local prefix = Mochil.readFilePrefix(path, 12) or ""
    return prefix:sub(1, 3) == "\255\216\255" -- JPEG
        or prefix:sub(1, 8) == "\137PNG\13\10\26\10" -- PNG
        or prefix:sub(1, 3) == "GIF"
        or (prefix:sub(1, 4) == "RIFF" and prefix:sub(9, 12) == "WEBP")
end

-- A book with no artwork would otherwise be asked for again on every open; remember the miss for a
-- month, which is long enough to matter and short enough that new artwork is picked up eventually.
Mochil.COVER_MISS_SECONDS = 30 * 24 * 60 * 60

function Mochil:coverMissed(path)
    local marker = path .. ".none"
    local file = io.open(marker, "r")
    if not file then return false end
    local stamp = tonumber(file:read("*all") or "")
    file:close()
    return stamp ~= nil and (os.time() - stamp) < Mochil.COVER_MISS_SECONDS
end

function Mochil:rememberCoverMiss(path)
    local file = io.open(path .. ".none", "w")
    if not file then return end
    file:write(tostring(os.time()))
    file:close()
end

function Mochil:coverFile(book, fetch)
    if not book or not book.id or not book.library then return nil end
    local path = self:coverTarget(book)
    if Mochil.fileSize(path) > 0 then return path end
    if not fetch or self:coverMissed(path) then return nil end
    local directory = path:match("^(.*)/[^/]*$")
    if directory and not util.pathExists(directory) then util.makePath(directory) end
    local part = path .. ".part"
    local file = io.open(part, "wb")
    if not file then return nil end
    -- The cover is fetched while the dialog is being built, on the UI thread, so the budget has to be
    -- short: a server that accepts the connection and then stalls would otherwise freeze the reader
    -- for a minute before an unreadable cover is given up on.
    socketutil:set_timeout(3, 6)
    local called, reply, answer = pcall(http.request, {
        url = config.server_url .. "/api/koreader/v1/cover/" .. url_encode(book.library) .. "/" .. url_encode(book.id),
        method = "GET", sink = ltn12.sink.file(file), headers = {Authorization = "Bearer " .. config.token},
    })
    socketutil:reset_timeout()
    if not called or tonumber(answer) ~= 200 or not Mochil.looksLikeImage(part) then
        pcall(os.remove, part)
        if called and tonumber(answer) == 404 then self:rememberCoverMiss(path) end
        return nil
    end
    if not os.rename(part, path) then pcall(os.remove, part) return nil end
    return path
end

-- The cover above a title, its lines and the same buttons ButtonDialog would have shown. Returns nil
-- when this KOReader cannot build it, so the caller keeps the plain dialog it has always shown.
function Mochil:bookDialogWidget(lines, buttons, cover)
    local Screen = require("device").screen
    local width = math.floor(math.min(Screen:getWidth(), Screen:getHeight()) * 0.9)
    local ok, widget = pcall(function()
        local FrameContainer = require("ui/widget/container/framecontainer")
        local VerticalGroup = require("ui/widget/verticalgroup")
        local VerticalSpan = require("ui/widget/verticalspan")
        local ImageWidget = require("ui/widget/imagewidget")
        local TextBoxWidget = require("ui/widget/textboxwidget")
        local ButtonTable = require("ui/widget/buttontable")
        local image = ImageWidget:new{file = cover, width = math.floor(width * 0.6),
            height = math.floor(Screen:getHeight() * 0.32), scale_factor = 0}
        return FrameContainer:new{margin = Screen:scaleBySize(6), padding = Screen:scaleBySize(6), bordersize = 0,
            VerticalGroup:new{align = "center", image, VerticalSpan:new{width = Screen:scaleBySize(6)},
                TextBoxWidget:new{text = table.concat(lines, "\n"), width = width - 20, alignment = "center"},
                VerticalSpan:new{width = Screen:scaleBySize(4)},
                ButtonTable:new{buttons = buttons, width = width - 20, zero_sep = true}}}
    end)
    if not ok then return nil end
    -- Same shape ButtonDialog has: the buttons and the text, so nothing else has to know which
    -- dialog it is looking at.
    widget.buttons, widget.title = buttons, table.concat(lines, "\n")
    return widget
end

-- The reader's own label for a book, on top of finished/unfinished. "reading" clears a parked
-- label, "read" marks it finished, and both on hold and dropped take it off the continue shelf.
local STATUS_LABELS = {reading = _("Reading"), on_hold = _("On hold"), dropped = _("Dropped"), read = _("Read"), unread = _("Unread")}

function Mochil:statusLabel(status)
    return STATUS_LABELS[status or ""] or nil
end

function Mochil:setBookStatus(book, status)
    local result, err = self:request("PUT", {status = status}, true, "status/" .. url_encode(book.library) .. "/" .. url_encode(book.id))
    if not result then
        UIManager:show(InfoMessage:new{text = _("Could not set the status: ") .. Mochil.describeFailure(err), timeout = 5})
        return false
    end
    -- The dialog and the continue shelf are built from this table, so keep it honest.
    book.status = (status == "reading" or status == "unread") and nil or status
    if status == "read" then book.finished = true elseif status == "reading" or status == "unread" then book.finished = nil end
    if status == "on_hold" or status == "dropped" then book.finished = nil end
    UIManager:show(InfoMessage:new{text = _("Status: ") .. (self:statusLabel(status) or status), timeout = 3})
    return true
end

function Mochil:showStatusChoices(book)
    local ButtonDialog = require("ui/widget/buttondialog")
    local dialog
    local buttons = {}
    for _, status in ipairs({"reading", "on_hold", "dropped", "read", "unread"}) do
        local value = status
        buttons[#buttons + 1] = {{text = STATUS_LABELS[value], callback = function()
            UIManager:close(dialog)
            self:setBookStatus(book, value)
        end}}
    end
    buttons[#buttons + 1] = {{text = _("Close"), callback = function() UIManager:close(dialog) end}}
    dialog = ButtonDialog:new{title = _("Set status for ") .. tostring(book.title or ""), buttons = buttons}
    UIManager:show(dialog)
end


-- The web and the Android app both keep a shelf for parked books; this is the device's, so parking
-- something from KOReader does not make it unreachable on the device that parked it.
function Mochil:showParked()
    local result, err = Mochil.whileLoading(_("Looking for parked books…"), function()
        return Mochil:request("GET", nil, true, "continue?held=true", true)
    end)
    if not result or type(result.continue) ~= "table" then
        UIManager:show(InfoMessage:new{text = _("Could not load parked books: ") .. Mochil.describeFailure(err), timeout = 5})
        return
    end
    local Menu = require("ui/widget/menu")
    local Screen = require("device").screen
    local widget
    local items = {}
    for _, entry in ipairs(result.continue) do
        local book = entry.book
        local label = tostring(book and book.title or entry.title or "")
        local status = book and self:statusLabel(book.status) or nil
        items[#items + 1] = {text = label .. (status and ("  ·  " .. status) or ""), callback = function()
            UIManager:close(widget)
            if book then self:showBook(book) end
        end}
    end
    if #items == 0 then
        items[1] = {text = _("Nothing on hold."), select_enabled_func = function() return false end}
    end
    widget = Menu:new{
        title = _("On hold and dropped"), item_table = items, is_borderless = true, is_popout = false,
        covers_fullscreen = true, width = Screen:getWidth(), height = Screen:getHeight(),
        close_callback = function() UIManager:close(widget) end,
    }
    Mochil.keepOpen(widget, Menu)
    UIManager:show(widget)
end

-- Vault libraries (see koReaderVault in koreader_home.go) are live Obsidian-style Markdown folders the
-- server already scans on its own -- unlike the account's own Notes & wiki, there is nothing for this
-- plugin to sync or cache, only one to pick and render. Fetches /home first since the plugin keeps no
-- cached list of vaults between menu opens.
function Mochil:readVaultOnDevice()
    local ButtonDialog = require("ui/widget/buttondialog")
    local home, err = Mochil.whileLoading(_("Looking for vault libraries…"), function()
        return self:request("GET", nil, true, "home", true)
    end)
    local vaults = type(home) == "table" and home.vaults
    if not vaults or #vaults == 0 then
        UIManager:show(InfoMessage:new{text=_("No vault libraries are configured on the server.") .. (err and (" " .. Mochil.describeFailure(err)) or ""), timeout=5})
        return
    end
    if #vaults == 1 then
        self:downloadAndOpenVault(vaults[1])
        return
    end
    local dialog
    local buttons = {}
    for _, vault in ipairs(vaults) do
        buttons[#buttons + 1] = {{text=vault.name, callback=function()
            UIManager:close(dialog)
            self:downloadAndOpenVault(vault)
        end}}
    end
    buttons[#buttons + 1] = {{text=_("Close"), callback=function() UIManager:close(dialog) end}}
    dialog = ButtonDialog:new{title=_("Choose a vault"), buttons=buttons}
    UIManager:show(dialog)
end

function Mochil:downloadAndOpenVault(vault)
    local raw, err = self:fetchRaw("/api/koreader/v1/vault/" .. url_encode(vault.id) .. ".html", 16 * 1024 * 1024)
    if not raw then
        UIManager:show(InfoMessage:new{text=_("Could not read the vault: ") .. Mochil.describeFailure(err), timeout=5})
        return
    end
    local root = Mochil.downloadRoot()
    if not util.pathExists(root) then util.makePath(root) end
    local path = root .. "/" .. vault.id .. ".html"
    local staged = path .. ".part"
    local file = io.open(staged, "wb")
    if not file then
        UIManager:show(InfoMessage:new{text=_("Could not write the vault file."), timeout=5})
        return
    end
    local wrote = file:write(raw)
    file:close()
    if not wrote or not os.rename(staged, path) then
        os.remove(staged)
        UIManager:show(InfoMessage:new{text=_("Could not write the vault file."), timeout=5})
        return
    end
    self:openDownloaded(path)
end

-- One book: what it is, and the one thing to do with it.
function Mochil:showBook(book)
    local ButtonDialog = require("ui/widget/buttondialog")
    local path = self:downloadTarget(book)
    local on_device = util.pathExists(path)
    local authors = type(book.authors) == "table" and table.concat(book.authors, ", ") or ""
    local lines = {tostring(book.title or "")}
    if authors ~= "" then lines[#lines + 1] = authors end
    if tostring(book.series or "") ~= "" then
        lines[#lines + 1] = tostring(book.series) .. (tonumber(book.series_index) and (" #" .. tostring(tonumber(book.series_index))) or "")
    end
    local size = tonumber(book.size) or 0
    lines[#lines + 1] = tostring(book.format or ""):upper() .. (size > 0 and string.format("  ·  %.1f MB", size / 1048576) or "")
    local status = self:statusLabel(book.status)
    if status then lines[#lines + 1] = status end
    local dialog
    local function close() UIManager:close(dialog) end
    local action
    if on_device then
        action = {text = _("Open"), callback = function() close(); self:openDownloaded(path) end}
        lines[#lines + 1] = _("On this device")
    else
        action = {text = _("Download"), callback = function()
            close()
            self:downloadBook(book, function(saved)
                UIManager:show(ConfirmBox:new{text = _("Saved to ") .. saved, ok_text = _("Open"), cancel_text = _("Later"),
                    ok_callback = function() self:openDownloaded(saved) end})
            end)
        end}
    end
    local buttons = {}
    -- Streaming is server-gated (library PageStreaming plus the book's own page metadata; see
    -- koReaderBookOf in koreader_browse.go) and is offered alongside, never instead of, Download.
    if book.streamable then
        buttons[#buttons + 1] = {{text = _("Stream"), callback = function() close(); self:streamBook(book) end}}
    end
    buttons[#buttons + 1] = {action}
    buttons[#buttons + 1] = {{text = book.finished and _("Mark as unfinished") or _("Mark as finished"), callback = function()
        close()
        self:setBookFinished(book, not book.finished)
    end}}
    buttons[#buttons + 1] = {{text = _("Set status…"), callback = function()
        close()
        self:showStatusChoices(book)
    end}}
    if not on_device then
        buttons[#buttons + 1] = {{text = _("Add to download queue"), callback = function()
            close()
            self:queueBook(book)
        end}}
    end
    local following = self:seriesNext(book)
    if following then
        local label = tostring(following.title or "")
        if tonumber(following.series_index) then label = label .. "  ·  #" .. tostring(tonumber(following.series_index)) end
        buttons[#buttons + 1] = {{text = _("Next: ") .. label, callback = function()
            close()
            self:showBook(following)
        end}}
    end
    buttons[#buttons + 1] = {{text = _("Close"), callback = close}}
    -- The cover is fetched once per book and cached beside the downloads; without one - no artwork,
    -- expired pairing, older KOReader - the dialog is exactly what it always was.
    local cover = self:coverFile(book, true)
    dialog = (cover and self:bookDialogWidget(lines, buttons, cover))
        or ButtonDialog:new{title = table.concat(lines, "\n"), buttons = buttons}
    UIManager:show(dialog)
end

-- Opens a comic page-by-page without ever downloading or storing a file, the same way
-- KOReader's own plugins/opds.koplugin/opdspse.lua streams OPDS-PSE: a lazy table whose
-- metatable __index fetches one page's bytes (over this plugin's own Bearer-gated
-- /api/koreader/v1/pages/{library}/{id}/{pageNumber} endpoint, zero-indexed, never the
-- generic session/Basic-only /stream/ route) and decodes them with RenderImage, and
-- ui/widget/imageviewer paginates through that table with its usual tap/swipe gestures.
function Mochil:streamBook(book)
    local RenderImage = require("ui/renderimage")
    local base = config.server_url .. "/api/koreader/v1/pages/" .. url_encode(book.library) .. "/" .. url_encode(book.id) .. "/"
    local count = tonumber(book.page_count) or 0
    local page_table = {image_disposable = true}
    local notified_failures = {}
    setmetatable(page_table, {__index = function(self_table, key)
        if type(key) ~= "number" then
            return RenderImage:renderImageFile("resources/koreader.png", false)
        end
        local index = key - 1
        local push, page_data, overflowed = bounded_response_sink(MAXIMUM_PAGE_BYTES)
        socketutil:set_timeout(socketutil.LARGE_BLOCK_TIMEOUT, socketutil.LARGE_TOTAL_TIMEOUT)
        local called, reply, answer = pcall(http.request, {
            url = base .. tostring(index),
            method = "GET",
            sink = push,
            headers = {Authorization = "Bearer " .. config.token, ["Accept-Encoding"] = "identity"},
        })
        socketutil:reset_timeout()
        if overflowed() then
            logger.warn("Mochil:streamBook: page larger than the limit", index)
            called = false
        end
        local code = tonumber(answer)
        if called and code == 200 then
            local data = table.concat(page_data)
            local rendered = RenderImage:renderImageData(data, #data, false)
            if rendered then return rendered end
        end
        logger.dbg("Mochil:streamBook: page request failed:", called and answer or reply)
        if not notified_failures[key] then
            notified_failures[key] = true
            UIManager:show(InfoMessage:new{text=_("Page failed to load. Swipe to retry."), timeout=2})
        end
        return RenderImage:renderImageFile("resources/koreader.png", false)
    end})
    local ImageViewer = require("ui/widget/imageviewer")
    local viewer = ImageViewer:new{
        image = page_table,
        fullscreen = true,
        with_title_bar = false,
        image_disposable = false, -- the page table's own image_disposable covers each page instead
        images_list_nb = count,
    }
    self:closeBrowsing()
    UIManager:show(viewer)
    local last_page = tonumber(book.last_page)
    if last_page and last_page > 0 then
        viewer:switchToImageNum(last_page)
    else
        viewer:switchToImageNum(1)
    end
end

-- A page of the library: books, or series, authors and collections to open. It loads a screenful at a time.
-- Shows a short message while work holds the screen, so the device does not look frozen, and always takes it away again.
function Mochil.whileLoading(text, work)
    local message = InfoMessage:new{text = text}
    UIManager:show(message)
    pcall(function() UIManager:forceRePaint() end)
    local results = table.pack(pcall(work))
    UIManager:close(message)
    if not results[1] then error(results[2], 0) end
    return table.unpack(results, 2, results.n)
end

-- One search across every library: the Search row inside a browse list only searches the list it sits in.
function Mochil:searchEverything()
    local InputDialog = require("ui/widget/inputdialog")
    local dialog
    dialog = InputDialog:new{
        title = _("Search all libraries"),
        input_hint = _("Title, author or series"),
        buttons = {{
            {text = _("Cancel"), id = "close", callback = function() UIManager:close(dialog) end},
            {text = _("Search"), is_enter_default = true, callback = function()
                local text = tostring(dialog:getInputText() or ""):gsub("^%s+", ""):gsub("%s+$", "")
                UIManager:close(dialog)
                if text ~= "" then self:showBrowse{kind = "books", title = _("Search results"), search = text} end
            end},
        }},
    }
    UIManager:show(dialog)
    if dialog.onShowKeyboard then dialog:onShowKeyboard() end
end

function Mochil:showBrowse(spec)
    local Menu = require("ui/widget/menu")
    local Screen = require("device").screen
    local state = {groups = {}, books = {}, total = 0, search = spec.search}
    local widget

    local function resource(offset)
        local parts = {"kind=" .. url_encode(spec.kind), "limit=" .. BROWSE_PAGE, "offset=" .. offset}
        if state.search then parts[#parts + 1] = "q=" .. url_encode(state.search) end
        for key, value in pairs(spec.filter or {}) do parts[#parts + 1] = url_encode(key) .. "=" .. url_encode(value) end
        table.sort(parts)
        return "browse?" .. table.concat(parts, "&")
    end

    local function fetch(offset)
        -- The screen is held while the server answers (up to ten seconds), so say what is going on.
        local called, result, err = Mochil.whileLoading(_("Loading…"), function()
            self.deadline = os.time() + 10
            local worked, answer, why = pcall(self.request, self, "GET", nil, true, resource(offset), true)
            self.deadline = nil
            return worked, answer, why
        end)
        if not called then return nil, tostring(result) end
        if not result then return nil, self.last_error or err or "" end
        return result
    end

    local function rows()
        local items = {}
        items[#items + 1] = {text = state.search and (_("Search: ") .. state.search) or _("Search…"), mandatory = state.search and _("Clear") or "",
            callback = function()
                if state.search then state.search = nil; state.reload() return end
                local InputDialog = require("ui/widget/inputdialog")
                local dialog
                dialog = InputDialog:new{title = _("Search ") .. tostring(spec.title or ""), buttons = {{
                    {text = _("Cancel"), callback = function() UIManager:close(dialog) end},
                    {text = _("Search"), is_enter_default = true, callback = function()
                        local text = dialog:getInputText()
                        UIManager:close(dialog)
                        if text and text:match("%S") then state.search = text; state.reload() end
                    end},
                }}}
                UIManager:show(dialog)
                if dialog.onShowKeyboard then dialog:onShowKeyboard() end
            end}
        if state.failure then
            items[#items + 1] = {text = Mochil.describeFailure(state.failure), mandatory = ""}
            items[#items + 1] = {text = _("Try again"), mandatory = "", callback = function() state.reload() end}
            return items
        end
        for _index, group in ipairs(state.groups) do
            local filter
            if spec.kind == "series" then filter = {series = group.name, library = group.library}
            elseif spec.kind == "authors" then filter = {author = group.name, library = group.library}
            else filter = {collection = group.id} end
            items[#items + 1] = {text = tostring(group.name), mandatory = tostring(group.count or ""),
                callback = function()
                    if spec.kind == "libraries" then self:showLibrary(group) return end
                    self:showBrowse{kind = spec.kind == "collections" and "books" or spec.kind, title = group.name, filter = filter}
                end}
        end
        for _index, book in ipairs(state.books) do
            local volume = tonumber(book.series_index)
            local label = (volume and spec.kind == "series" and (string.format(volume == math.floor(volume) and "%d" or "%s", volume) .. ". ") or "") .. tostring(book.title or "")
            local authors = type(book.authors) == "table" and table.concat(book.authors, ", ") or ""
            if authors ~= "" and spec.kind ~= "authors" then label = label .. "  ·  " .. authors end
            items[#items + 1] = {text = label, mandatory = util.pathExists(self:downloadTarget(book)) and "✓" or tostring(book.format or ""):upper(),
                callback = function() self:showBook(book) end}
        end
        local loaded = #state.groups + #state.books
        if loaded == 0 then
            items[#items + 1] = {text = state.search and _("Nothing matches that search") or _("Nothing here yet"), mandatory = "", dim = true}
        elseif loaded < state.total then
            items[#items + 1] = {text = _("Show more"), mandatory = tostring(state.total - loaded), bold = true, callback = function() state.more() end}
        end
        return items
    end

    local function show(reset)
        local items = rows()
        if widget then widget:switchItemTable(spec.title, items, reset and 1 or nil) end
        return items
    end

    function state.reload()
        state.groups, state.books, state.total, state.failure = {}, {}, 0, nil
        local result, why = fetch(0)
        if not result then state.failure = why
        else
            state.total = tonumber(result.total) or 0
            state.groups, state.books = result.groups or {}, result.books or {}
        end
        show(true)
    end

    function state.more()
        local result, why = fetch(#state.groups + #state.books)
        if not result then
            UIManager:show(InfoMessage:new{text = Mochil.describeFailure(why), timeout = 5})
            return
        end
        for _index, group in ipairs(result.groups or {}) do state.groups[#state.groups + 1] = group end
        for _index, book in ipairs(result.books or {}) do state.books[#state.books + 1] = book end
        show(false)
    end

    local run = function()
        local first, why = fetch(0)
        if not first then state.failure = why
        else
            state.total = tonumber(first.total) or 0
            state.groups, state.books = first.groups or {}, first.books or {}
        end
        -- With one library there is nothing to choose: go straight into it.
        if spec.kind == "libraries" and not state.search and #state.groups == 1 and #state.books == 0 then
            self:showLibrary(state.groups[1])
            return
        end
        widget = Menu:new{
            title = spec.title, item_table = rows(), is_borderless = true, is_popout = false, covers_fullscreen = true, single_line = true,
            width = Screen:getWidth(), height = Screen:getHeight(),
            close_callback = function() UIManager:close(widget) end,
        }
        Mochil.keepOpen(widget, Menu)
        self:track(widget)
        UIManager:show(widget)
    end
    if NetworkMgr:willRerunWhenConnected(run) then return end
    run()
end

function Mochil:browseKind(kind, title) self:showBrowse{kind = kind, title = title} end

-- One library's front page: how to look through it. The counts come with the library list, so this needs no request.
function Mochil:showLibrary(library)
    local Menu = require("ui/widget/menu")
    local Screen = require("device").screen
    local widget
    local function open(kind, label)
        return function()
            self:showBrowse{kind = kind, title = tostring(library.name) .. " · " .. label, filter = {library = library.library or library.id}}
        end
    end
    local items = {
        {text = _("All books"), mandatory = tostring(tonumber(library.count) or 0), callback = open("books", _("Books"))},
        {text = _("Series"), mandatory = tostring(tonumber(library.series) or 0), callback = open("series", _("Series"))},
        {text = _("Authors"), mandatory = tostring(tonumber(library.authors) or 0), callback = open("authors", _("Authors"))},
    }
    if (tonumber(library.collections) or 0) > 0 then
        items[#items + 1] = {text = _("Collections"), mandatory = tostring(library.collections), callback = open("collections", _("Collections"))}
    end
    items[#items + 1] = {text = _("Continue reading here"), callback = function()
        self:showContinue(nil, library.library or library.id, tostring(library.name))
    end}
    widget = Menu:new{
        title = tostring(library.name), item_table = items, is_borderless = true, is_popout = false, covers_fullscreen = true, single_line = true,
        width = Screen:getWidth(), height = Screen:getHeight(),
        close_callback = function() UIManager:close(widget) end,
    }
    Mochil.keepOpen(widget, Menu)
    self:track(widget)
    UIManager:show(widget)
end

-- "Library" opens the list of libraries; a server that does not report libraries yet still gets the plain book list.
function Mochil:openLibraries(home)
    if home and tonumber(home.libraries) then self:browseKind("libraries", _("Libraries")) else self:browseKind("books", _("Library")) end
end

-- Pick the folder downloads go to. The picker can make a new folder, so the reader never has to leave KOReader to do it.
function Mochil:chooseDownloadFolder()
    local PathChooser = require("ui/widget/pathchooser")
    local start = Mochil.downloadRoot()
    if not util.pathExists(start) then start = G_reader_settings:readSetting("home_dir") or Device.home_dir or "/" end
    UIManager:show(PathChooser:new{
        title = _("Download folder"), path = start, select_directory = true, select_file = false, show_files = false,
        onConfirm = function(folder) G_reader_settings:saveSetting("mochil_download_dir", tostring(folder):gsub("/+$", "")) end,
    })
end

-- Labels for the values a setting currently has, so the screen says how the plugin is set up instead of making the
-- reader open every sub-menu to find out.
local function syncPolicyLabel(policy)
    if policy == "chapter" then return _("After changing chapter") end
    if policy == "page" then return _("After every page") end
    if policy == "manual" then return _("Manual only") end
    return _("When closing")
end

local function downloadLayoutLabel(layout)
    if layout == "author" then return _("Authors only") end
    if layout == "flat" then return _("One folder") end
    return _("Series, then authors")
end

-- The settings screen: a heading per group and one row per decision, with the value it currently has on the right.
-- The flat list of menu entries stays as the fallback and for the tools menu, but this is what the reader opens.
function Mochil:settingsItems(refresh)
    local items = {}
    local function heading(text) items[#items + 1] = {text = text, bold = true, select_enabled = false} end
    local function has_book() return self.ui and self.ui.document ~= nil end
    local function setSyncPolicy(policy) return function() self:setSyncPolicy(policy) end end
    local server = tostring(config.server_url):gsub("^https?://", "")

    heading(_("Syncing"))
    local function policyChoice(value, label)
        return {text = label, radio = true, keep_menu_open = true,
            checked_func = function() return self:syncPolicy() == value end,
            -- KOReader draws the tick from mandatory_func; keep it explicit so the row reads as a choice.
            mandatory_func = function() return self:syncPolicy() == value and "\226\156\147" or "" end,
            callback = setSyncPolicy(value)}
    end
    items[#items + 1] = {text = _("Automatic push"), sub_item_table = {
        policyChoice("close", _("When closing")),
        policyChoice("chapter", _("After changing chapter")),
        policyChoice("page", _("After every page")),
        policyChoice("manual", _("Manual only")),
    }, mandatory_func = function() return syncPolicyLabel(self:syncPolicy()) end}
    items[#items + 1] = {text = _("Send this book's position now"), enabled_func = has_book, callback = function() self:push(true) end}
    items[#items + 1] = {text = _("Get the newest position"), enabled_func = has_book, callback = function() self:pull(true) end}
    items[#items + 1] = {text = _("Sync highlights now"), enabled_func = has_book, callback = function() self:syncAnnotations(true) end}

    heading(_("Library"))
    items[#items + 1] = {text = _("Reading statistics"), callback = function() self:readingStats() end}
    items[#items + 1] = {text = _("On hold and dropped"), callback = function() self:showParked() end}
    local queued = #self:queueList()
    items[#items + 1] = {text = _("Download queue") .. (queued > 0 and ("  ·  " .. tostring(queued)) or ""),
        callback = function() self:showQueue() end}
    items[#items + 1] = {text = _("Notes and wiki on this device"), keep_menu_open = true,
        mandatory_func = function() return Mochil.wikiEnabled() and _("On") or _("Off") end,
        callback = function()
            G_reader_settings:saveSetting("mochil_wiki_enabled", not Mochil.wikiEnabled())
        end}

    heading(_("This device"))
    items[#items + 1] = {text = _("Download folder"), mandatory = tostring(Mochil.downloadRoot()):match("([^/]+)/*$") or Mochil.downloadRoot(),
        callback = function() self:chooseDownloadFolder() end}
    local function layoutChoice(value, label)
        return {text = label, radio = true, keep_menu_open = true,
            checked_func = function() return Mochil.downloadLayout() == value end,
            mandatory_func = function() return Mochil.downloadLayout() == value and "\226\156\147" or "" end,
            callback = function() G_reader_settings:saveSetting("mochil_download_layout", value) end}
    end
    items[#items + 1] = {text = _("Download layout"), sub_item_table = {
        layoutChoice("series", _("Series, then authors")),
        layoutChoice("author", _("Authors only")),
        layoutChoice("flat", _("One folder")),
    }, mandatory_func = function() return downloadLayoutLabel(Mochil.downloadLayout()) end}
    items[#items + 1] = {text = _("Designed home page"), keep_menu_open = true,
        mandatory_func = function() return G_reader_settings:readSetting("mochil_designed_page") == true and _("On") or _("Off") end,
        callback = function()
            local on = G_reader_settings:readSetting("mochil_designed_page") == true
            G_reader_settings:saveSetting("mochil_designed_page", not on)
            if not on then G_reader_settings:saveSetting("mochil_designed_page_crashed", false); self.home_safe_mode = false end
            if G_reader_settings.flush then G_reader_settings:flush() end
        end}
    items[#items + 1] = {text = _("Files on this device"), callback = function() self:showFiles() end}

    heading(_("Plugin"))
    items[#items + 1] = {text = _("Version"), mandatory = Mochil.release, callback = function() self:showInstallDetails() end}
    items[#items + 1] = {text = _("Server: ") .. server, select_enabled_func = function() return false end}
    items[#items + 1] = {text = _("Test connection"), callback = function() self:testConnection() end}
    items[#items + 1] = {text = _("Check for updates"), mandatory = G_reader_settings:readSetting("mochil_update_available"),
        callback = function() self:checkForUpdate(true) end}
    return items
end

-- The plugin's own settings, as a page of the same kind: sections, current values on the right, one tap to change.
function Mochil:showSettingsPage()
    local Menu = require("ui/widget/menu")
    local Screen = require("device").screen
    local widget
    local function reopen() pcall(function() UIManager:close(widget) end); self:showSettingsPage() end
    local built, items = pcall(self.settingsItems, self, reopen)
    if not built then
        logger.err("Mochil: settings page could not be built, using the plain list: " .. tostring(items))
        items = Mochil.pageItems(self:menuEntries())
    end
    widget = Menu:new{
        title=_("Mochil settings"), item_table=items, is_borderless=true, is_popout=false, covers_fullscreen=true,
        width=Screen:getWidth(), height=Screen:getHeight(), close_callback=function() UIManager:close(widget) end,
    }
    Mochil.keepOpen(widget, Menu)
    UIManager:show(widget)
end

-- Where this copy of the plugin runs from, and whether the loader had to fall back to the previous version.
function Mochil:showInstallDetails()
    local lines = {_("Running version ") .. Mochil.release, _("Folder: ") .. tostring(loader.dir or "unknown")}
    local installed = G_reader_settings:readSetting("mochil_installed_version")
    if installed then lines[#lines + 1] = _("Last update installed: ") .. tostring(installed) end
    local page_error = G_reader_settings:readSetting("mochil_designed_page_error")
    if page_error then lines[#lines + 1] = _("The designed page could not be drawn: ") .. tostring(page_error) end
    if G_reader_settings:readSetting("mochil_designed_page_crashed") then lines[#lines + 1] = _("The designed page was interrupted; the list may be used temporarily. Your preference is unchanged.") end
    if Mochil.fallback_error then lines[#lines + 1] = _("The newest version could not start, so the previous one is used: ") .. Mochil.fallback_error end
    UIManager:show(InfoMessage:new{text=table.concat(lines, "\n")})
end

-- What this reader cannot know by itself: books finished on the phone, and reading spread over several devices. The
-- numbers on this device come from KOReader's own statistics plugin, when it is running, and are added underneath.
function Mochil:readingStats()
    local waiting = InfoMessage:new{text = _("Reading statistics…")}
    UIManager:show(waiting)
    -- Let the message draw before the request holds the screen.
    UIManager:nextTick(function()
        local stats, err = self:request("GET", nil, true, "stats", true)
        UIManager:close(waiting)
        local lines = {}
        if stats then
            lines[#lines + 1] = _("On Mochil (all devices)")
            lines[#lines + 1] = "  " .. _("Books in progress: ") .. tostring(stats.in_progress or 0)
            lines[#lines + 1] = "  " .. _("Books finished: ") .. tostring(stats.finished or 0)
            lines[#lines + 1] = "  " .. _("Finished this month: ") .. tostring(stats.finished_this_month or 0)
            lines[#lines + 1] = "  " .. _("Pages read: ") .. tostring(stats.pages_read or 0)
            local recent = stats.recent_finished or {}
            if #recent > 0 then
                lines[#lines + 1] = ""
                lines[#lines + 1] = _("Recently finished")
                for _, book in ipairs(recent) do
                    local row = "  " .. tostring(book.title or "")
                    local finished_at = tonumber(book.finished_at)
                    if finished_at and finished_at > 0 then row = row .. " — " .. os.date("%d %B", math.floor(finished_at / 1000)) end
                    lines[#lines + 1] = row
                end
            end
        else
            lines[#lines + 1] = _("Mochil could not be reached: ") .. Mochil.describeFailure(err)
        end
        for _, line in ipairs(self:localReadingStats()) do lines[#lines + 1] = line end
        local TextViewer = require("ui/widget/textviewer")
        UIManager:show(TextViewer:new{title = _("Reading statistics"), text = table.concat(lines, "\n"), justified = false})
    end)
end

-- This book's numbers from KOReader's own statistics plugin. Every call is guarded: the plugin is optional, its
-- methods have changed between releases, and a missing one must simply leave the line out.
function Mochil:localReadingStats()
    local statistics = self.ui and self.ui.statistics
    if not statistics or type(statistics.getTodayBookStats) ~= "function" then return {} end
    local lines = {_("On this reader")}
    local ok, duration, pages = pcall(statistics.getTodayBookStats, statistics)
    if ok and (tonumber(duration) or tonumber(pages)) then
        lines[#lines + 1] = "  " .. _("Today: ") .. string.format("%d min", math.floor((tonumber(duration) or 0) / 60))
            .. ", " .. plural(tonumber(pages) or 0, _("page"), _("pages"))
    end
    if ok and type(statistics.getPageTimeTotalStats) == "function" and statistics.id_curr_book then
        local counted, total_pages, total_time = pcall(statistics.getPageTimeTotalStats, statistics, statistics.id_curr_book)
        if counted and tonumber(total_time) then
            lines[#lines + 1] = "  " .. _("This book: ") .. string.format("%.1f h", (tonumber(total_time) or 0) / 3600)
                .. ", " .. plural(tonumber(total_pages) or 0, _("page"), _("pages"))
        end
    end
    if #lines > 1 then return lines end
    return {}
end

function Mochil:addToMainMenu(menu_items)
    local ok, err = pcall(function()
        menu_items.mochil_tools = {text=_("Mochil Sync"),sorting_hint="tools",sub_item_table=self:menuEntries()}
    end)
    if not ok then logger.err("Mochil: menu failed: " .. tostring(err)) end
end

-- KOReader calls these from its event loop and timers, where an uncaught error ends the reading session.
-- Every entry point therefore catches its own failures, logs them, and (for a button the reader pressed) says so.
local INTERACTIVE = {push=true, pull=true, syncAnnotations=true, checkForUpdate=true}
for _index, name in ipairs{"onReaderReady", "onPageUpdate", "onPosUpdate", "onCloseDocument", "onSuspend", "onAnnotationsModified",
        "push", "pull", "syncAnnotations", "testConnection", "backgroundSync", "scheduleAutoPush", "checkForUpdate",
        "maybeCheckForUpdate", "runUpdate", "init", "showBrowse", "showBook", "downloadBook", "chooseDownloadFolder", "openDownloaded", "openBook", "showLibrary",
        "onDispatcherRegisterActions", "onMochilPush", "onMochilPull", "onMochilSyncHighlights", "backupHighlights", "exportHighlightsToNotes", "readWikiOnDevice", "setBookFinished", "setBookStatus", "showStatusChoices", "showParked", "readVaultOnDevice", "downloadAndOpenVault", "readingStats", "localReadingStats", "searchEverything", "showSettingsPage", "settingsItems"} do
    local method = Mochil[name]
    Mochil[name] = function(self, ...)
        local results = pack(pcall(method, self, ...))
        if results[1] then return unpack(results, 2, results.n) end
        logger.err("Mochil: " .. name .. " failed: " .. tostring(results[2]))
        if INTERACTIVE[name] and (...) == true then
            pcall(function() UIManager:show(InfoMessage:new{text=_("Mochil hit a problem. Details are in the KOReader log."), timeout=5}) end)
        end
    end
end

return Mochil
